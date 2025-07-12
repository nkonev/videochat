package db

import (
	"context"
	"database/sql"
	"errors"
	"github.com/rotisserie/eris"
	"nkonev.name/storage/dto"
)

func (tx *Tx) Set(ctx context.Context, metadataCache dto.MetadataCache) error {
	_, err := tx.ExecContext(ctx, `
		insert into metadata_cache(
			chat_id,
			file_item_uuid, 
		    filename,
		    
			owner_user_id,
			correlation_id,
		    conference_recording,                        
		    message_recording,
		    original_key,
		    
		    published,
		    
		    create_date_time
		) values (
		    $1,
		    $2,
		    $3, 
		    $4,
		    $5,
		    $6,
		  	$7,
			$8,
			$9,
		    $10
		) on conflict (chat_id, file_item_uuid, filename) 
		do update set 
			owner_user_id = $4, 
			correlation_id = $5,
			conference_recording = $6, 
			message_recording = $7, 
			original_key = $8, 
			published = $9,
			create_date_time = $10
	`,
		metadataCache.ChatId,
		metadataCache.FileItemUuid,
		metadataCache.Filename,

		metadataCache.OwnerId,
		metadataCache.CorrelationId,
		metadataCache.ConferenceRecording,
		metadataCache.MessageRecording,
		metadataCache.OriginalKey,
		metadataCache.Published,
		metadataCache.CreateDateTime,
	)

	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}

	return nil
}

func (tx *Tx) Get(ctx context.Context, metadataCacheId dto.MetadataCacheId) (*dto.MetadataCache, error) {
	row := tx.QueryRowContext(ctx, `select 
			chat_id,
		    file_item_uuid,
			filename,
			
		    owner_user_id,                        
		    correlation_id,
		    conference_recording,
		    message_recording,
			original_key,
			
			published,
			
			create_date_time
		from metadata_cache 
		where (chat_id, file_item_uuid, filename) = ($1, $2, $3)
	`, metadataCacheId.ChatId, metadataCacheId.FileItemUuid, metadataCacheId.Filename)
	if row.Err() != nil {
		return nil, eris.Wrap(row.Err(), "error during interacting with db")
	}
	ucs := dto.MetadataCache{}
	err := row.Scan(&ucs.ChatId, &ucs.FileItemUuid, &ucs.Filename, &ucs.OwnerId, &ucs.CorrelationId, &ucs.ConferenceRecording, &ucs.MessageRecording, &ucs.OriginalKey, &ucs.Published, &ucs.CreateDateTime)
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

func (tx *Tx) Remove(ctx context.Context, metadataCacheId dto.MetadataCacheId) error {
	_, err := tx.ExecContext(ctx, `delete from metadata_cache 
								where (chat_id, file_item_uuid, filename) = ($1, $2, $3)`,
		metadataCacheId.ChatId, metadataCacheId.FileItemUuid, metadataCacheId.Filename)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}
