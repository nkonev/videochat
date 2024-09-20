package db

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"
	"nkonev.name/video/dto"
	"time"
)

const NoUser = -1

const NoChat = -1

const CallStatusBeingInvited = "beingInvited" // the status scheduler should remain,
const CallStatusInCall = "inCall"             // the status scheduler should remain
const CallStatusCancelling = "cancelling"     // will be removed after some time automatically by scheduler

const CallStatusRemoving = "removing" // will be removed after some time automatically by scheduler

const CallStatusNotFound = ""

const UserCallMarkedForRemoveAtNotSet = 0

const UserCallMarkedForOrphanRemoveAttemptNotSet = 0

const NoAvatar = ""

const NoTetATet = false

// aka Should be changed Automatically After Timeout
func IsTemporary(userCallStatus string) bool {
	return userCallStatus == CallStatusCancelling || userCallStatus == CallStatusRemoving
}

func getTemporaryStates() []string {
	return []string{
		CallStatusCancelling,
		CallStatusRemoving,
	}
}

func CanOverrideCallStatus(userCallStatus string) bool {
	return IsTemporary(userCallStatus) || userCallStatus == CallStatusNotFound
}

func getOverrideStates() []string {
	arr := getTemporaryStates()
	arr = append(arr, CallStatusNotFound)
	return arr
}

func GetStatusesToRemoveOnEnter() []string {
	arr := getOverrideStates()
	arr = append(arr, CallStatusBeingInvited)
	return arr
}

func (tx *Tx) Set(userState dto.UserCallState) error {
	_, err := tx.Exec(`
		insert into user_call_state(
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

func (tx *Tx) AddAsEntered(tokenId uuid.UUID, userId, chatId int64, tetATet bool) error {
	return tx.Set(dto.UserCallState{
		TokenId:     tokenId,
		UserId:      userId,
		ChatId:      chatId,
		TokenTaken:  true,
		Status:      CallStatusInCall,
		ChatTetATet: tetATet,
	})
}

func provideScanToUserCallState(ucs *dto.UserCallState) []any {
	return []any{
		&ucs.TokenId,
		&ucs.UserId,
		&ucs.ChatId,
		&ucs.TokenTaken,
		&ucs.OwnerTokenId,
		&ucs.OwnerUserId,
		&ucs.Status,
		&ucs.ChatTetATet,
		&ucs.OwnerAvatar,
		&ucs.MarkedForRemoveAt,
		&ucs.MarkedForOrphanRemoveAttempt,
		&ucs.CreateDateTime,
	}
}

func (tx *Tx) Get(user dto.UserCallStateId) (*dto.UserCallState, error) {
	row := tx.QueryRow(`select 
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
		from user_call_state 
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

func (tx *Tx) GetByCalleeUserIdFromAllChats(calleeUserId int64) ([]dto.UserCallState, error) {
	rows, err := tx.Query(`select 
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
		from user_call_state 
		where user_id = $1
		order by user_id, token_id
	`, calleeUserId)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	defer rows.Close()
	list := make([]dto.UserCallState, 0)
	for rows.Next() {
		ucs := dto.UserCallState{}
		if err := rows.Scan(provideScanToUserCallState(&ucs)[:]...); err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		} else {
			list = append(list, ucs)
		}
	}
	return list, nil
}

func (tx *Tx) GetBeingInvitedByOwnerAndCalleeId(owner dto.UserCallStateId, calleeUserId int64, chatId int64) ([]dto.UserCallState, error) {
	rows, err := tx.Query(`select 
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
		from user_call_state 
		where owner_token_id = $1 and owner_user_id = $2 and user_id = $3 and chat_id = $4 and status = any($5)
		order by user_id, token_id
	`, owner.TokenId, owner.UserId, calleeUserId, chatId, []string{CallStatusBeingInvited})
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	defer rows.Close()
	list := make([]dto.UserCallState, 0)
	for rows.Next() {
		ucs := dto.UserCallState{}
		if err := rows.Scan(provideScanToUserCallState(&ucs)[:]...); err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		} else {
			list = append(list, ucs)
		}
	}
	return list, nil
}

func (tx *Tx) GetBeingInvitedByCalleeIdAndChatId(calleeUserId int64, chatId int64) ([]dto.UserCallState, error) {
	rows, err := tx.Query(`select 
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
		from user_call_state 
		where user_id = $1 and chat_id = $2 and status = $3
		order by user_id, token_id
	`, calleeUserId, chatId, CallStatusBeingInvited)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	defer rows.Close()
	list := make([]dto.UserCallState, 0)
	for rows.Next() {
		ucs := dto.UserCallState{}
		if err := rows.Scan(provideScanToUserCallState(&ucs)[:]...); err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		} else {
			list = append(list, ucs)
		}
	}
	return list, nil
}

func (tx *Tx) GetBeingInvitedByCalleeId(calleeUserId int64) ([]dto.UserCallState, error) {
	rows, err := tx.Query(`select 
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
		from user_call_state 
		where user_id = $1 and status = $2
		order by user_id, token_id
	`, calleeUserId, CallStatusBeingInvited)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	defer rows.Close()
	list := make([]dto.UserCallState, 0)
	for rows.Next() {
		ucs := dto.UserCallState{}
		if err := rows.Scan(provideScanToUserCallState(&ucs)[:]...); err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		} else {
			list = append(list, ucs)
		}
	}
	return list, nil
}

func (tx *Tx) Remove(user dto.UserCallStateId) error {
	_, err := tx.Exec(`delete from user_call_state 
								where (token_id, user_id) = ($1, $2)`,
		user.TokenId, user.UserId)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func (tx *Tx) RemoveOwnedAndOwner(owner dto.UserCallStateId) error {
	// 1. remove my own states
	_, err := tx.Exec(`delete from user_call_state where (owner_token_id, owner_user_id) = ($1, $2)`,
		owner.TokenId, owner.UserId)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}

	// 2. remove myself
	return tx.Remove(owner)
}

func (tx *Tx) RemoveByUserCallStates(ids []dto.UserCallStateId) error {
	if len(ids) == 0 {
		return nil
	}

	var bldr string
	var first = true
	for _, u := range ids {
		if !first {
			bldr += ", "
		}
		bldr += fmt.Sprintf("('%v', %v)", u.TokenId, u.UserId)

		first = false
	}
	_, err := tx.Exec(fmt.Sprintf(`delete from user_call_state
		 where (token_id, user_id) in (%v)`, bldr))
	return err
}

func (tx *Tx) SetUserStatus(user dto.UserCallStateId, status string) error {
	_, err := tx.Exec(`update user_call_state 
								set status = $3
								where (token_id, user_id) = ($1, $2)`,
		user.TokenId, user.UserId, status)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func (tx *Tx) SetCurrentTimeForRemoving(user dto.UserCallStateId) error {
	_, err := tx.Exec(`update user_call_state 
								set marked_for_remove_at = $3
								where (token_id, user_id) = ($1, $2)`,
		user.TokenId, user.UserId, time.Now().UTC())
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func (tx *Tx) SetMarkedForOrphanRemoveAttempt(user dto.UserCallStateId, attempt int) error {
	_, err := tx.Exec(`update user_call_state 
								set marked_for_orphan_remove_attempt = $3
								where (token_id, user_id) = ($1, $2)`,
		user.TokenId, user.UserId, attempt)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func (tx *Tx) GetUserOwnedBeingInvitedCallees(owner dto.UserCallStateId) ([]dto.UserCallState, error) {
	rows, err := tx.Query(`select 
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
		from user_call_state where owner_token_id = $1 and owner_user_id = $2 and status = any($3)
		order by user_id, token_id
		`, owner.TokenId, owner.UserId, []string{CallStatusBeingInvited})
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	defer rows.Close()

	list := []dto.UserCallState{}
	for rows.Next() {
		ucs := dto.UserCallState{}
		if err := rows.Scan(provideScanToUserCallState(&ucs)[:]...); err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		} else {
			list = append(list, ucs)
		}
	}
	return list, nil
}

// for red dot
func (tx *Tx) GetUserStatesFiltered(userIdsToFilter []int64) ([]dto.UserCallState, error) {
	var rows *sql.Rows
	var err error
	rows, err = tx.Query(`select 
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
		from user_call_state where user_id = any($1)
		order by user_id, token_id
		`, userIdsToFilter)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	defer rows.Close()

	list := []dto.UserCallState{}
	for rows.Next() {
		ucs := dto.UserCallState{}
		if err := rows.Scan(provideScanToUserCallState(&ucs)[:]...); err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		} else {
			list = append(list, ucs)
		}
	}
	return list, nil
}

func (tx *Tx) GetAllUserStates(limit, offset int64) ([]dto.UserCallState, error) {
	var rows *sql.Rows
	var err error
	rows, err = tx.Query(`select 
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
		from user_call_state 
		order by user_id, token_id
		limit $1 offset $2`, limit, offset)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	defer rows.Close()

	list := []dto.UserCallState{}
	for rows.Next() {
		ucs := dto.UserCallState{}
		if err := rows.Scan(provideScanToUserCallState(&ucs)[:]...); err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		} else {
			list = append(list, ucs)
		}
	}
	return list, nil
}

func (tx *Tx) GetAllUserStatesOrderByOwnerAndChat(limit, offset int64) ([]dto.UserCallState, error) {
	var rows *sql.Rows
	var err error
	rows, err = tx.Query(`select 
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
		from user_call_state 
		order by owner_user_id, chat_id, user_id, token_id
		limit $1 offset $2`, limit, offset)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	defer rows.Close()

	list := []dto.UserCallState{}
	for rows.Next() {
		ucs := dto.UserCallState{}
		if err := rows.Scan(provideScanToUserCallState(&ucs)[:]...); err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		} else {
			list = append(list, ucs)
		}
	}
	return list, nil
}
