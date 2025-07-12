package db

import (
	"context"
	"database/sql"
	"errors"
	"github.com/rotisserie/eris"
	"nkonev.name/video/dto"
)

func (tx *Tx) Set(ctx context.Context, userState dto.UserCallState) error {
	_, err := tx.ExecContext(ctx, `
		insert into metadata_cache(
			token_id,
			user_id, 
		    chat_id,
			token_taken,
			owner_token_id,
		    owner_user_id,                        
		    status,
		    chat_tet_a_tet,
		    owner_avatar,
			marked_for_remove_at,
			marked_for_orphan_remove_attempt
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
			$10, 
			$11
		) on conflict (token_id, user_id) 
		do update set 
			chat_id = $3,
			token_taken = $4, 
			owner_token_id = $5,
			owner_user_id = $6, 
			status = $7, 
			chat_tet_a_tet = $8, 
			owner_avatar = $9, 
			marked_for_remove_at = $10,
			marked_for_orphan_remove_attempt = $11
	`,
		userState.TokenId,
		userState.UserId,
		userState.ChatId,
		userState.TokenTaken,
		userState.OwnerTokenId,
		userState.OwnerUserId,
		userState.Status,
		userState.ChatTetATet,
		userState.OwnerAvatar,
		userState.MarkedForRemoveAt,
		userState.MarkedForOrphanRemoveAttempt,
	)

	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}

	return nil
}

func (tx *Tx) Get(ctx context.Context, user dto.UserCallStateId) (*dto.UserCallState, error) {
	row := tx.QueryRowContext(ctx, `select 
			token_id,
			user_id, 
			chat_id,
		    token_taken,
			owner_token_id,
		    owner_user_id,                        
		    status,
		    chat_tet_a_tet,
		    owner_avatar,
			marked_for_remove_at,
			marked_for_orphan_remove_attempt,
			create_date_time
		from metadata_cache 
		where (token_id, user_id) = ($1, $2)
	`, user.TokenId, user.UserId)
	if row.Err() != nil {
		return nil, eris.Wrap(row.Err(), "error during interacting with db")
	}
	ucs := dto.UserCallState{}
	err := row.Scan(provideScanToUserCallState(&ucs)[:]...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// there were no rows, but otherwise no error occurred
			ucs.TokenId = user.TokenId
			ucs.UserId = user.UserId
			ucs.Status = CallStatusNotFound
			return &ucs, nil
		}
		return nil, eris.Wrap(err, "error during scanning from db")
	}
	return &ucs, nil
}

func (tx *Tx) Remove(ctx context.Context, user dto.UserCallStateId) error {
	_, err := tx.ExecContext(ctx, `delete from metadata_cache 
								where (token_id, user_id) = ($1, $2)`,
		user.TokenId, user.UserId)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}
