package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgtype"
	"github.com/rotisserie/eris"
	"github.com/spf13/viper"
	"nkonev.name/storage/dto"
	"strings"
)

const metadataColumns = `
	chat_id,
	file_item_uuid,
	filename,
	
	owner_user_id,                        
	correlation_id,
	
	published,
	
	file_size,
	
	create_date_time,
	edit_date_time
`

// "" <=> dto.NoFileItemUuid
// -1 <=> dto.NoChatId
const getMetadatasSql = `select ` +
	metadataColumns +
	`
	from metadata_cache
	where ($1 = -1 or chat_id = $1) and ($2 = '' or file_item_uuid = $2) %s
	order by chat_id, file_item_uuid asc, create_date_time %s
	limit $3 offset $4
`
const getMetadatasCountSql = `select 
	count(*)
	from metadata_cache
	where ($1 = -1 or chat_id = $1) and ($2 = '' or file_item_uuid = $2) %s
`

const getMetadataSql = `select ` +
	metadataColumns +
	`
	from metadata_cache
	where chat_id = $1 and file_item_uuid = $2 and filename = $3 %s
`

func Set(ctx context.Context, co CommonOperations, metadataCache dto.MetadataCache) error {
	_, err := co.ExecContext(ctx, fmt.Sprintf(`
		insert into metadata_cache(
			%s
		) values (
		    $1,
		    $2,
		    $3, 
		    $4,
		    $5,
		    $6,
		  	$7,
		    $8,
		    $9
		) on conflict (chat_id, file_item_uuid, filename) 
		do update set 
			published = $6,
		    file_size = $7,
			edit_date_time = $9
	`, metadataColumns),
		metadataCache.ChatId,
		metadataCache.FileItemUuid,
		metadataCache.Filename,

		metadataCache.OwnerId,
		metadataCache.CorrelationId,
		metadataCache.Published,
		metadataCache.FileSize,
		metadataCache.CreateDateTime,
		metadataCache.EditDateTime,
	)

	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}

	return nil
}

func provideScanToMetadataCache(ucs *dto.MetadataCache) []any {
	return []any{
		&ucs.ChatId,
		&ucs.FileItemUuid,
		&ucs.Filename,
		&ucs.OwnerId,
		&ucs.CorrelationId,
		&ucs.Published,
		&ucs.FileSize,
		&ucs.CreateDateTime,
		&ucs.EditDateTime,
	}
}

func Get(ctx context.Context, co CommonOperations, metadataCacheId dto.MetadataCacheId, filterObj Filter) (*dto.MetadataCache, error) {
	baseSqlArgs := []any{metadataCacheId.ChatId, metadataCacheId.FileItemUuid, metadataCacheId.Filename}

	sqlString, sqlArgs, noData, err := applyFilter(filterObj, getMetadataSql, baseSqlArgs)
	if err != nil {
		return nil, eris.Wrap(err, "error during building sql")
	}
	if noData {
		return nil, err // see also below "sql.ErrNoRows"
	}

	row := co.QueryRowContext(ctx, sqlString, sqlArgs...)
	if row.Err() != nil {
		return nil, eris.Wrap(row.Err(), "error during interacting with db")
	}
	ucs := dto.MetadataCache{}
	err = row.Scan(provideScanToMetadataCache(&ucs)...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// there were no rows, but otherwise no error occurred
			return nil, nil // see also above "noData"
		}
		return nil, eris.Wrap(err, "error during scanning from db")
	}
	return &ucs, nil
}

func CheckFileItemBelongsToUser(ctx context.Context, co CommonOperations, chatId int64, fileItemUuid string, ownerId int64) (bool, error) {
	row := co.QueryRowContext(ctx, "select not exists(select 1 from metadata_cache where chat_id = $1 and file_item_uuid = $2 and owner_user_id != $3)", chatId, fileItemUuid, ownerId)
	if row.Err() != nil {
		return false, eris.Wrap(row.Err(), "error during interacting with db")
	}
	var belongs bool
	err := row.Scan(&belongs)
	if err != nil {
		return false, eris.Wrap(err, "error during scanning from db")
	}
	return belongs, nil
}

type Filter interface {
	apply(baseSqlTemplate string, existingArgs []any) (string, []any, bool, error)
}

type FilterBySearchString struct {
	searchString string
}

type FilterByType struct {
	typeExtensions []string
}

func NewFilterBySearchString(searchString string) *FilterBySearchString {
	return &FilterBySearchString{
		searchString: searchString,
	}
}

func NewFilterByType(typeExtensions []string) *FilterByType {
	return &FilterByType{
		typeExtensions: typeExtensions,
	}
}

func (f *FilterBySearchString) apply(baseSqlTemplate string, existingArgs []any) (string, []any, bool, error) {
	sqlArgs := existingArgs
	sqlString := ""

	suffix := fmt.Sprintf("and lower(filename) LIKE '%%' || lower($%v) || '%%'", len(sqlArgs)+1)
	sqlString = fmt.Sprintf(baseSqlTemplate, suffix)
	sqlArgs = append(sqlArgs, f.searchString)

	return sqlString, sqlArgs, false, nil
}

func (f *FilterByType) apply(baseSqlTemplate string, existingArgs []any) (string, []any, bool, error) {
	sqlArgs := existingArgs
	sqlString := ""

	if len(f.typeExtensions) > 0 {
		builder := " and ( "
		for i, dotExt := range f.typeExtensions {
			orClause := ""
			if i != 0 {
				orClause = "or"
			}
			builder += fmt.Sprintf(" %v lower(filename) like '%%%v'", orClause, strings.ToLower(dotExt))
		}
		builder += ") "

		sqlString = fmt.Sprintf(baseSqlTemplate, builder)
	} else {
		return sqlString, sqlArgs, true, nil // true means "no data" because of no extension
	}

	return sqlString, sqlArgs, false, nil
}

func applyFilter(filterObj Filter, baseSqlTemplate string, existingArgs []any) (string, []any, bool, error) {
	var sqlArgs []any
	sqlString := ""
	noData := false
	var err error

	if filterObj == nil {
		sqlString = fmt.Sprintf(baseSqlTemplate, "")
		sqlArgs = existingArgs
	} else {
		sqlString, sqlArgs, noData, err = filterObj.apply(baseSqlTemplate, existingArgs)
		if err != nil {
			return sqlString, sqlArgs, false, err
		}
		if noData {
			return sqlString, sqlArgs, true, nil // true means "no data" because of no extension
		}
	}

	return sqlString, sqlArgs, false, nil
}

func GetList(ctx context.Context, co CommonOperations, chatId int64, fileItemUuid string, filterObj Filter, reverse bool, limit, offset int) ([]dto.MetadataCache, error) {
	list := make([]dto.MetadataCache, 0)

	baseSqlArgs := []any{chatId, fileItemUuid, limit, offset}

	var order string
	if reverse {
		order = "asc"
	} else {
		order = "desc"
	}
	tmpSql := fmt.Sprintf(getMetadatasSql, "%s", order)

	sqlString, sqlArgs, noData, err := applyFilter(filterObj, tmpSql, baseSqlArgs)
	if err != nil {
		return nil, eris.Wrap(err, "error during building sql")
	}
	if noData {
		return []dto.MetadataCache{}, err
	}

	rows, err := co.QueryContext(ctx, sqlString, sqlArgs...)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	defer rows.Close()
	for rows.Next() {
		ucs := dto.MetadataCache{}
		if err = rows.Scan(provideScanToMetadataCache(&ucs)[:]...); err != nil {
			return nil, eris.Wrap(err, "error during scanning")
		} else {
			list = append(list, ucs)
		}
		if err != nil {
			return nil, eris.Wrap(err, "error during mapping")
		}
	}

	return list, nil
}

func GetCount(ctx context.Context, co CommonOperations, chatId int64, fileItemUuid string, filterObj Filter) (int64, error) {
	var count int64

	baseSqlArgs := []any{chatId, fileItemUuid}

	sqlString, sqlArgs, noData, err := applyFilter(filterObj, getMetadatasCountSql, baseSqlArgs)
	if err != nil {
		return 0, eris.Wrap(err, "error during building sql")
	}
	if noData {
		return 0, err
	}

	row := co.QueryRowContext(ctx, sqlString, sqlArgs...)
	if row.Err() != nil {
		return 0, eris.Wrap(row.Err(), "error during interacting with db")
	}

	err = row.Scan(&count)
	if err != nil {
		return 0, eris.Wrap(row.Err(), "error during interacting with db")
	}

	return count, nil
}

type SimpleFileItem struct {
	Filename string `json:"filename"`
}

type GroupedByFileItemUuid struct {
	FileItemUuid string           `json:"fileItemUuid"`
	Files        []SimpleFileItem `json:"files"`
}

func GetListFilesItemUuids(ctx context.Context, co CommonOperations, chatId int64, limit, offset int) ([]GroupedByFileItemUuid, error) {
	res := []GroupedByFileItemUuid{}

	maxFiles := viper.GetInt32("fileItemUuid.maxFiles")

	rows, err := co.QueryContext(ctx, `
		select 
		    outp.file_item_uuid,
		    array_agg(outp.filename)
		from (
		    select 
		        inn.file_item_uuid,
		        inn.filename,
		        row_number() over (partition by inn.file_item_uuid order by inn.filename) as seqnum
			from metadata_cache inn 
			where inn.chat_id = $1
		) outp
		where outp.seqnum <= $4
		group by outp.file_item_uuid
		order by outp.file_item_uuid
		limit $2 offset $3
`, chatId, limit, offset, maxFiles)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	defer rows.Close()
	for rows.Next() {
		var item GroupedByFileItemUuid
		var filenames = pgtype.TextArray{}
		err = rows.Scan(&item.FileItemUuid, &filenames)
		if err != nil {
			return nil, eris.Wrap(err, "error during mapping")
		}

		item.Files = make([]SimpleFileItem, 0)
		for _, fn := range filenames.Elements {
			item.Files = append(item.Files, SimpleFileItem{
				Filename: fn.String,
			})
		}

		res = append(res, item)
	}
	return res, nil
}

func GetCountFilesItemUuids(ctx context.Context, co CommonOperations, chatId int64) (int64, error) {
	var count int64

	row := co.QueryRowContext(ctx, `
		select 
		    count(t.file_item_uuid)
		from metadata_cache t
		where t.chat_id = $1
		group by t.file_item_uuid
	`, chatId)
	if row.Err() != nil {
		return 0, eris.Wrap(row.Err(), "error during interacting with db")
	}

	err := row.Scan(&count)
	if err != nil {
		return 0, eris.Wrap(err, "error during mapping")
	}

	return count, nil
}

func Remove(ctx context.Context, co CommonOperations, metadataCacheId dto.MetadataCacheId) error {
	_, err := co.ExecContext(ctx, `delete from metadata_cache 
								where (chat_id, file_item_uuid, filename) = ($1, $2, $3)`,
		metadataCacheId.ChatId, metadataCacheId.FileItemUuid, metadataCacheId.Filename)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}
