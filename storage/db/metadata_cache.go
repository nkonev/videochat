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

func Set(ctx context.Context, co CommonOperations, metadataCache dto.MetadataCache) error {
	_, err := co.ExecContext(ctx, `
		insert into metadata_cache(
			chat_id,
			file_item_uuid, 
		    filename,
		    
			owner_user_id,
			correlation_id,
		    
		    published,
		    
		    create_date_time
		) values (
		    $1,
		    $2,
		    $3, 
		    $4,
		    $5,
		    $6,
		  	$7
		) on conflict (chat_id, file_item_uuid, filename) 
		do update set 
			correlation_id = $5,
			published = $6,
			create_date_time = $7
	`,
		metadataCache.ChatId,
		metadataCache.FileItemUuid,
		metadataCache.Filename,

		metadataCache.OwnerId,
		metadataCache.CorrelationId,
		metadataCache.Published,
		metadataCache.CreateDateTime,
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
		&ucs.CreateDateTime,
	}
}

func Get(ctx context.Context, co CommonOperations, metadataCacheId dto.MetadataCacheId) (*dto.MetadataCache, error) {
	row := co.QueryRowContext(ctx, `select 
			chat_id,
		    file_item_uuid,
			filename,
			
		    owner_user_id,                        
		    correlation_id,
			
			published,
			
			create_date_time
		from metadata_cache 
		where (chat_id, file_item_uuid, filename) = ($1, $2, $3)
	`, metadataCacheId.ChatId, metadataCacheId.FileItemUuid, metadataCacheId.Filename)
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

const selectMetadataColumns = `
			chat_id,
		    file_item_uuid,
			filename,
			
		    owner_user_id,                        
		    correlation_id,
			
			published,
			
			create_date_time
`

const selectMetadataCount = "count(*) "

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
	sqlString := `select 
    	%s
		from metadata_cache
		where chat_id = $1 and ($2 = '' or file_item_uuid = $2) %s
		order by file_item_uuid desc, filename desc
		limit $3 offset $4
	`
	sqlArgs := []any{chatId, fileItemUuid, limit, offset}

	if filterObj == nil {
		sqlString = fmt.Sprintf(sqlString, selectWhat, "")
	} else {
		switch v := filterObj.(type) {
		case *FilterBySearchString:
			sqlString = fmt.Sprintf(sqlString, selectWhat, "and lower(filename) LIKE '%' || lower($5) || '%'")
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

				sqlString = fmt.Sprintf(sqlString, selectWhat, builder)
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
