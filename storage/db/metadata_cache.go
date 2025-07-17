package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/rotisserie/eris"
	"nkonev.name/storage/dto"
	"strings"
)

const selectMetadataColumns = `
	chat_id,
	file_item_uuid,
	metadataCacheId.FileItemUuid,
	
	owner_user_id,                        
	correlation_id,
	
	published,
	
	file_size,
	
	create_date_time,
	edit_date_time
`

const selectMetadataCount = " count(*) "

const getMetadatasSql = `select 
	%s
	from metadata_cache
	where chat_id = $1 and ($2 = '' or file_item_uuid = $2) %s
	order by file_item_uuid desc, filename desc
	limit $3 offset $4
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
	} else if v, ok := filterObj.(FilterBySearchString); ok {
		sqlString = fmt.Sprintf(getMetadatasSql, selectMetadataColumns, " and filename = $5 and lower(filename) LIKE '%' || lower($6) || '%'")
		sqlArgs = append(sqlArgs, v.searchString)
	}

	row := co.QueryRowContext(ctx, fmt.Sprintf(sqlString, sqlArgs...))
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
	row := co.QueryRowContext(ctx, "select not exists(select 1 from metadata_cashe where chat_id = $1 and file_item_uuid = $2 and owner_user_id != $3)", chatId, fileItemUuid, ownerId)
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

	errOuter := getMetadatas(ctx, co, func(rows *sql.Rows) error {
		ucs := dto.MetadataCache{}
		if err := rows.Scan(provideScanToMetadataCache(&ucs)[:]...); err != nil {
			return eris.Wrap(err, "error during scanning")
		} else {
			list = append(list, ucs)
		}
		return nil
	}, selectMetadataColumns, chatId, fileItemUuid, filterObj, limit, offset)
	if errOuter != nil {
		return nil, errOuter
	}

	return list, nil
}

func GetCount(ctx context.Context, co CommonOperations, chatId int64, fileItemUuid string, filterObj Filter) (int64, error) {
	list := make([]int64, 0)

	errOuter := getMetadatas(ctx, co, func(rows *sql.Rows) error {
		var count int64
		if err := rows.Scan(&count); err != nil {
			return eris.Wrap(err, "error during scanning")
		} else {
			list = append(list, count)
		}
		return nil
	}, selectMetadataCount, chatId, fileItemUuid, filterObj, 1, 0)
	if errOuter != nil {
		return 0, errOuter
	}
	if len(list) != 1 {
		return 0, errors.New("Expected 1 row for count")
	}

	return list[0], nil
}

func getMetadatas(ctx context.Context, co CommonOperations, rowMapper func(rows *sql.Rows) error, selectWhat string, chatId int64, fileItemUuid string, filterObj Filter, limit, offset int) error {
	sqlArgs := []any{chatId, fileItemUuid, limit, offset}

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
				return nil
			}
		default:
			return fmt.Errorf("unknown filter type %T", filterObj)
		}
	}

	rows, err := co.QueryContext(ctx, sqlString, sqlArgs...)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	defer rows.Close()
	for rows.Next() {
		err = rowMapper(rows)
		if err != nil {
			return eris.Wrap(err, "error during mapping")
		}
	}
	return nil
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
