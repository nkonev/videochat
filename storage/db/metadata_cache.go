package db

import (
	"context"
	"database/sql"
	"errors"
	"github.com/rotisserie/eris"
	"nkonev.name/storage/dto"
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
	err := row.Scan(&ucs.ChatId, &ucs.FileItemUuid, &ucs.Filename, &ucs.OwnerId, &ucs.CorrelationId, &ucs.Published, &ucs.CreateDateTime)
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

func Remove(ctx context.Context, co CommonOperations, metadataCacheId dto.MetadataCacheId) error {
	_, err := co.ExecContext(ctx, `delete from metadata_cache 
								where (chat_id, file_item_uuid, filename) = ($1, $2, $3)`,
		metadataCacheId.ChatId, metadataCacheId.FileItemUuid, metadataCacheId.Filename)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}
