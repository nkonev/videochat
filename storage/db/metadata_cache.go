package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgtype"
	"github.com/rotisserie/eris"
	"nkonev.name/storage/dto"
	"strings"
)

const selectMetadataColumns = `
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

const getMetadatasSql = `select 
	%s
	from metadata_cache
	where chat_id = $1 and ($2 = '' or file_item_uuid = $2) %s
	order by file_item_uuid asc, filename desc
	limit $3 offset $4
`

const getMetadatasCountSql = `select 
	count(*)
	from metadata_cache
	where chat_id = $1 and ($2 = '' or file_item_uuid = $2) %s
`

func Set(ctx context.Context, co CommonOperations, metadataCache dto.MetadataCache) error {
	_, err := co.ExecContext(ctx, `
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
	`,
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
	const limit = 1
	const offset = 0
	sqlArgs := []any{metadataCacheId.ChatId, metadataCacheId.FileItemUuid, limit, offset, metadataCacheId.Filename}
	var sqlString string

	if filterObj == nil {
		sqlString = fmt.Sprintf(getMetadatasSql, selectMetadataColumns, " and filename = $5 ")
	} else if v, ok := filterObj.(*FilterBySearchString); ok {
		sqlString = fmt.Sprintf(getMetadatasSql, selectMetadataColumns, " and filename = $5 and lower(filename) LIKE '%' || lower($6) || '%'")
		sqlArgs = append(sqlArgs, v.searchString)
	} else {
		return nil, fmt.Errorf("Unknown filter: %T", filterObj)
	}

	row := co.QueryRowContext(ctx, sqlString, sqlArgs...)
	if row.Err() != nil {
		return nil, eris.Wrap(row.Err(), "error during interacting with db")
	}
	ucs := dto.MetadataCache{}
	err := row.Scan(provideScanToMetadataCache(&ucs)...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// there were no rows, but otherwise no error occurred
			ucs.ChatId = metadataCacheId.ChatId
			ucs.FileItemUuid = metadataCacheId.FileItemUuid
			ucs.Filename = metadataCacheId.Filename
			return &ucs, nil
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

func GetList(ctx context.Context, co CommonOperations, chatId int64, fileItemUuid string, filterObj Filter, limit, offset int) ([]dto.MetadataCache, error) {
	list := make([]dto.MetadataCache, 0)

	sqlArgs := []any{chatId, fileItemUuid, limit, offset}

	selectWhat := selectMetadataColumns

	var sqlString string
	if filterObj == nil {
		sqlString = fmt.Sprintf(getMetadatasSql, selectWhat, "")
	} else {
		switch v := filterObj.(type) {
		case *FilterBySearchString:
			sqlString = fmt.Sprintf(getMetadatasSql, selectWhat, "and lower(filename) LIKE '%' || lower($5) || '%'")
			sqlArgs = append(sqlArgs, v.searchString)
		case *FilterByType: // we define extensions, it isn't an user input, so it is safe
			if len(v.typeExtensions) > 0 {
				builder := " and ( "
				for i, dotExt := range v.typeExtensions {
					orClause := ""
					if i != 0 {
						orClause = "or"
					}
					builder += fmt.Sprintf(" %v lower(filename) like '%%%v'", orClause, strings.ToLower(dotExt))
				}
				builder += ") "

				sqlString = fmt.Sprintf(getMetadatasSql, selectWhat, builder)
			} else {
				return []dto.MetadataCache{}, nil
			}
		default:
			return nil, fmt.Errorf("unknown filter type %T", filterObj)
		}
	}

	rows, err := co.QueryContext(ctx, sqlString, sqlArgs...)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	defer rows.Close()
	for rows.Next() {
		ucs := dto.MetadataCache{}
		if err := rows.Scan(provideScanToMetadataCache(&ucs)[:]...); err != nil {
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

	sqlArgs := []any{chatId, fileItemUuid}

	var sqlString string
	if filterObj == nil {
		sqlString = fmt.Sprintf(getMetadatasCountSql, "")
	} else {
		switch v := filterObj.(type) {
		case *FilterBySearchString:
			sqlString = fmt.Sprintf(getMetadatasCountSql, "and lower(filename) LIKE '%' || lower($3) || '%'")
			sqlArgs = append(sqlArgs, v.searchString)
		case *FilterByType: // we define extensions, it isn't an user input, so it is safe
			if len(v.typeExtensions) > 0 {
				builder := " and ( "
				for i, dotExt := range v.typeExtensions {
					orClause := ""
					if i != 0 {
						orClause = "or"
					}
					builder += fmt.Sprintf(" %v lower(filename) like '%%%v'", orClause, strings.ToLower(dotExt))
				}
				builder += ") "

				sqlString = fmt.Sprintf(getMetadatasCountSql, builder)
			} else {
				return 0, nil
			}
		default:
			return 0, fmt.Errorf("unknown filter type %T", filterObj)
		}
	}

	row := co.QueryRowContext(ctx, sqlString, sqlArgs...)
	if row.Err() != nil {
		return 0, eris.Wrap(row.Err(), "error during interacting with db")
	}

	err := row.Scan(&count)
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

	rows, err := co.QueryContext(ctx, `
		select 
		    t.file_item_uuid,
		    array_agg(l.filename)
		from metadata_cache t
		inner join lateral (
			select inc.* from metadata_cache inc
			where inc.chat_id = t.chat_id and 
			order by inc.filename 
			limit 10
		) l on true
		group by t.file_item_uuid
		order by t.file_item_uuid asc 
		where t.chat_id = $1 
		limit $2 offset $3`, chatId, limit, offset)
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

		for _, fn := range filenames.Elements {
			item.Files = make([]SimpleFileItem, 0)
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
		group by t.file_item_uuid
		where t.chat_id = $1 
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
