package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgtype"
	"github.com/rotisserie/eris"
	"github.com/spf13/viper"
	"nkonev.name/storage/dto"
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
	where ($1 = -1 or chat_id = $1) and ($2 = '' or file_item_uuid = $2) %s -- filter
	%s -- keyset
	%s -- order
	%s -- offset
`
const getMetadatasCountSql = `select 
	count(*)
	from metadata_cache
	where ($1 = -1 or chat_id = $1) and ($2 = '' or file_item_uuid = $2) %s -- filter
`

const getMetadataSql = `select ` +
	metadataColumns +
	`
	from metadata_cache
	where chat_id = $1 and file_item_uuid = $2 and filename = $3 %s -- filter
`

func Set(ctx context.Context, co CommonOperations, metadataCache dto.MetadataCache) error {
	err := SetList(ctx, co, []dto.MetadataCache{metadataCache})

	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}

	return nil
}

func SetList(ctx context.Context, co CommonOperations, metadataCaches []dto.MetadataCache) error {
	chatIds := []int64{}
	fileItemUuids := []string{}
	filenames := []string{}

	ownerIds := []int64{}
	correlationIds := []*string{}
	publisheds := []bool{}
	fileSizes := []int64{}
	createDateTimes := []time.Time{}
	editDateTimes := []time.Time{}

	for _, v := range metadataCaches {
		chatIds = append(chatIds, v.ChatId)
		fileItemUuids = append(fileItemUuids, v.FileItemUuid)
		filenames = append(filenames, v.Filename)
		ownerIds = append(ownerIds, v.OwnerId)
		correlationIds = append(correlationIds, v.CorrelationId)
		publisheds = append(publisheds, v.Published)
		fileSizes = append(fileSizes, v.FileSize)
		createDateTimes = append(createDateTimes, v.CreateDateTime)
		editDateTimes = append(editDateTimes, v.EditDateTime)
	}

	_, err := co.ExecContext(ctx, `
		with input_data as (
			select * from unnest (
				cast ($1 as bigint[])
				,cast ($2 as varchar(36)[])
				,cast ($3 as varchar(255)[])
				,cast ($4 as bigint[])
				,cast ($5 as varchar(36)[])
				,cast ($6 as boolean[])
				,cast ($7 as bigint[])
				,cast ($8 as timestamp[])
				,cast ($9 as timestamp[])
			) as t (
				chat_id
				,file_item_uuid
				,filename
				,owner_user_id
				,correlation_id
				,published
				,file_size
				,create_date_time
				,edit_date_time
			)
		)
		insert into metadata_cache(
			chat_id,
			file_item_uuid,
			filename,
			owner_user_id,                        
			correlation_id,
			published,
			file_size,
			create_date_time,
			edit_date_time
		)
		select * from input_data
		on conflict (chat_id, file_item_uuid, filename) 
		do update set
			owner_user_id = excluded.owner_user_id
			,correlation_id = excluded.correlation_id
			,published = excluded.published
			,file_size = excluded.file_size
			,create_date_time = excluded.create_date_time
			,edit_date_time = excluded.edit_date_time
	`,
		chatIds,
		fileItemUuids,
		filenames,
		ownerIds,
		correlationIds,
		publisheds,
		fileSizes,
		createDateTimes,
		editDateTimes,
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

	filterSqlString, filterSqlArgs, filterNoData, err := applyFilter(filterObj, baseSqlArgs)
	if err != nil {
		return nil, eris.Wrap(err, "error during building sql")
	}
	if filterNoData {
		return nil, nil // see also below "sql.ErrNoRows"
	}
	baseSqlArgs = slices.Clone(filterSqlArgs)

	sqlString := fmt.Sprintf(getMetadataSql, filterSqlString)

	row := co.QueryRowContext(ctx, sqlString, baseSqlArgs...)
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
	apply(existingArgs []any) (string, []any, bool, error)
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

func (f *FilterBySearchString) apply(existingArgs []any) (string, []any, bool, error) {
	sqlArgs := slices.Clone(existingArgs)

	suffix := fmt.Sprintf("and filename ILIKE '%%' || $%v || '%%'", len(sqlArgs)+1)
	sqlArgs = append(sqlArgs, f.searchString)

	return suffix, sqlArgs, false, nil
}

func (f *FilterByType) apply(existingArgs []any) (string, []any, bool, error) {
	sqlArgs := slices.Clone(existingArgs)
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

		sqlString = builder
	} else {
		return sqlString, sqlArgs, true, nil // true means "no data" because of no extension
	}

	return sqlString, sqlArgs, false, nil
}

func applyFilter(filterObj Filter, existingArgs []any) (string, []any, bool, error) {
	var sqlArgs []any
	sqlString := ""
	noData := false
	var err error

	if filterObj == nil {
		sqlString = ""
		sqlArgs = existingArgs
	} else {
		sqlString, sqlArgs, noData, err = filterObj.apply(existingArgs)
		if err != nil {
			return sqlString, sqlArgs, false, err
		}
		if noData {
			return sqlString, sqlArgs, true, nil // true means "no data" because of no extension
		}
	}

	return sqlString, sqlArgs, false, nil
}

type ListPagination interface {
	apply(existingArgs []any, alreadyProvidedFileItemUuid bool, reverse bool) (
		string, // offsetString
		string, // keysetString
		string, // orderString
		[]any, // args
		bool, // noData
		error,
	)
}

type ListPaginationOffset struct {
	limit, offset int
}

type keyset struct {
	fileItemUuid   string
	createDateTime time.Time
	filename       string
}

type ListPaginationKeyset struct {
	startingFromItemId  *keyset
	includeStartingFrom bool
	size                int
}

func NewListPaginationOffset(limit, offset int) *ListPaginationOffset {
	return &ListPaginationOffset{
		limit:  limit,
		offset: offset,
	}
}

func NewListPaginationKeyset(
	startingFromFileItemUuid string,
	startingFromCreateDateTime *time.Time,
	startingFromFilename string,
	includeStartingFrom bool,
	size int,
) *ListPaginationKeyset {
	v := &ListPaginationKeyset{
		includeStartingFrom: includeStartingFrom,
		size:                size,
	}
	if len(startingFromFileItemUuid) > 0 && startingFromCreateDateTime != nil && len(startingFromFilename) > 0 {
		v.startingFromItemId = &keyset{
			fileItemUuid:   startingFromFileItemUuid,
			createDateTime: *startingFromCreateDateTime,
			filename:       startingFromFilename,
		}
	}
	return v
}

func (p *ListPaginationOffset) apply(existingArgs []any, _ bool, reverse bool) (string, string, string, []any, bool, error) {
	sqlArgs := slices.Clone(existingArgs)
	limitString := ""

	limitString = fmt.Sprintf(" limit $%v offset $%v ", len(sqlArgs)+1, len(sqlArgs)+2)
	sqlArgs = append(sqlArgs, p.limit, p.offset)

	var order string
	if reverse {
		order = "desc"
	} else {
		order = "asc"
	}

	orderStr := fmt.Sprintf(" order by chat_id, file_item_uuid desc, create_date_time %s ", order)

	return limitString, "", orderStr, sqlArgs, false, nil
}

func (p *ListPaginationKeyset) apply(existingArgs []any, alreadyProvidedFileItemUuid bool, reverse bool) (string, string, string, []any, bool, error) {
	sqlArgs := slices.Clone(existingArgs)
	keySetString := ""

	var order string
	if reverse {
		order = "desc"
	} else {
		order = "asc"
	}

	const colsTemplateF = " file_item_uuid %s, create_date_time %s, filename %s "
	const colsTemplateC = " create_date_time %s, filename %s "

	var keySetColumns string
	var orderColumns string
	if !alreadyProvidedFileItemUuid {
		keySetColumns = fmt.Sprintf(colsTemplateF, "", "", "")
		orderColumns = fmt.Sprintf(colsTemplateF, order, order, order)
	} else {
		keySetColumns = fmt.Sprintf(colsTemplateC, "", "")
		orderColumns = fmt.Sprintf(colsTemplateC, order, order)
	}

	if p.startingFromItemId != nil {
		nonEquality := ""
		if reverse {
			if p.includeStartingFrom {
				nonEquality = "<="
			} else {
				nonEquality = "<"
			}
		} else {
			if p.includeStartingFrom {
				nonEquality = ">="
			} else {
				nonEquality = ">"
			}
		}

		if !alreadyProvidedFileItemUuid {
			keySetString = fmt.Sprintf(" and (%s) %s ($%v, $%v, $%v) ", keySetColumns, nonEquality, len(sqlArgs)+1, len(sqlArgs)+2, len(sqlArgs)+3)
			sqlArgs = append(sqlArgs, p.startingFromItemId.fileItemUuid)
			sqlArgs = append(sqlArgs, p.startingFromItemId.createDateTime)
			sqlArgs = append(sqlArgs, p.startingFromItemId.filename)
		} else {
			keySetString = fmt.Sprintf(" and (%s) %s ($%v, $%v) ", keySetColumns, nonEquality, len(sqlArgs)+1, len(sqlArgs)+2)
			sqlArgs = append(sqlArgs, p.startingFromItemId.createDateTime)
			sqlArgs = append(sqlArgs, p.startingFromItemId.filename)
		}
	} else {
		emptySuffix := ""
		keySetString = emptySuffix
	}

	orderStr := fmt.Sprintf(" order by %s", orderColumns)

	limitString := fmt.Sprintf(" limit $%v ", len(sqlArgs)+1)
	sqlArgs = append(sqlArgs, p.size)

	return limitString, keySetString, orderStr, sqlArgs, false, nil
}

func applyPagination(paginationObj ListPagination, alreadyProvidedFileItemUuid bool, existingArgs []any, reverse bool) (string, string, string, []any, bool, error) {
	var sqlArgs []any
	offsetSqlString := ""
	keysetSqlString := ""
	orderStr := ""
	noData := false
	var err error

	if paginationObj == nil {
		return "", "", "", nil, false, errors.New("no pagination object")
	} else {
		offsetSqlString, keysetSqlString, orderStr, sqlArgs, noData, err = paginationObj.apply(existingArgs, alreadyProvidedFileItemUuid, reverse)
		if err != nil {
			return "", "", "", sqlArgs, false, err
		}
		if noData {
			return "", "", "", sqlArgs, true, nil // true means "no data" because of no extension
		}
	}

	return offsetSqlString, keysetSqlString, orderStr, sqlArgs, false, nil
}

func GetList(ctx context.Context, co CommonOperations, chatId int64, fileItemUuid string, filterObj Filter, paginationObj ListPagination, reverse bool) ([]dto.MetadataCache, error) {
	list := make([]dto.MetadataCache, 0)

	hasFileItemUuid := len(fileItemUuid) > 0

	baseSqlArgs := []any{chatId, fileItemUuid}

	offsetSqlString, keysetSqlString, orderString, paginationSqlArgs, paginationNoData, err := applyPagination(paginationObj, hasFileItemUuid, baseSqlArgs, reverse)
	if err != nil {
		return nil, eris.Wrap(err, "error during building sql")
	}
	if paginationNoData {
		return []dto.MetadataCache{}, nil
	}
	baseSqlArgs = slices.Clone(paginationSqlArgs)

	filterSqlString, filterSqlArgs, filterNoData, err := applyFilter(filterObj, baseSqlArgs)
	if err != nil {
		return nil, eris.Wrap(err, "error during building sql")
	}
	if filterNoData {
		return []dto.MetadataCache{}, nil
	}
	baseSqlArgs = slices.Clone(filterSqlArgs)

	sqlString := fmt.Sprintf(getMetadatasSql, filterSqlString, keysetSqlString, orderString, offsetSqlString)

	rows, err := co.QueryContext(ctx, sqlString, baseSqlArgs...)
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
	}

	return list, nil
}

func GetCount(ctx context.Context, co CommonOperations, chatId int64, fileItemUuid string, filterObj Filter) (int64, error) {
	var count int64

	baseSqlArgs := []any{chatId, fileItemUuid}

	filterSqlString, filterSqlArgs, filterNoData, err := applyFilter(filterObj, baseSqlArgs)
	if err != nil {
		return 0, eris.Wrap(err, "error during building sql")
	}
	if filterNoData {
		return 0, nil
	}

	sqlString := fmt.Sprintf(getMetadatasCountSql, filterSqlString)

	baseSqlArgs = slices.Clone(filterSqlArgs)

	row := co.QueryRowContext(ctx, sqlString, baseSqlArgs...)
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
		order by outp.file_item_uuid desc
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
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
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

func RemoveList(ctx context.Context, co CommonOperations, metadataCacheIds []dto.MetadataCacheId) error {
	chatIds := []int64{}
	fileItemUuids := []string{}
	filenames := []string{}

	for _, v := range metadataCacheIds {
		chatIds = append(chatIds, v.ChatId)
		fileItemUuids = append(fileItemUuids, v.FileItemUuid)
		filenames = append(filenames, v.Filename)
	}

	_, err := co.ExecContext(ctx, `
		with input_data as (
			select * from unnest (
				cast ($1 as bigint[])
				,cast ($2 as varchar(36)[])
				,cast ($3 as varchar(255)[])
			) as t (
			    chat_id
			    ,file_item_uuid
			    ,filename
			)
		)
		delete from metadata_cache 
		where (chat_id, file_item_uuid, filename) in (
			select chat_id, file_item_uuid, filename from input_data
		)
	`,
		chatIds, fileItemUuids, filenames)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func RemoveFileItem(ctx context.Context, co CommonOperations, chatId int64, fileItemUuid string) error {
	_, err := co.ExecContext(ctx, `delete from metadata_cache 
								where (chat_id, file_item_uuid) = ($1, $2)`,
		chatId, fileItemUuid)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}
