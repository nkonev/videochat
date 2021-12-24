package db

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/guregu/null"
	"nkonev.name/chat/auth"
	. "nkonev.name/chat/logger"
	"time"
)

// db model
type Chat struct {
	Id                 int64
	Title              string
	LastUpdateDateTime time.Time
	TetATet            bool
	Avatar             null.String
	AvatarBig          null.String
}

type ChatWithParticipants struct {
	Chat
	ParticipantsIds []int64
	IsAdmin         bool
}

// CreateChat creates a new chat.
// Returns an error if user is invalid or the tx fails.
func (tx *Tx) CreateChat(u *Chat) (int64, *time.Time, error) {
	// Validate the input.
	if u == nil {
		return 0, nil, errors.New("chat required")
	} else if u.Title == "" {
		return 0, nil, errors.New("title required")
	}

	// https://stackoverflow.com/questions/4547672/return-multiple-fields-as-a-record-in-postgresql-with-pl-pgsql/6085167#6085167
	res := tx.QueryRow(`SELECT chat_id, last_update_date_time FROM CREATE_CHAT($1, $2) AS (chat_id BIGINT, last_update_date_time TIMESTAMP)`, u.Title, u.TetATet)
	var id int64
	var lastUpdateDateTime time.Time
	if err := res.Scan(&id, &lastUpdateDateTime); err != nil {
		Logger.Errorf("Error during getting chat id %v", err)
		return 0, nil, err
	}

	return id, &lastUpdateDateTime, nil
}

func (tx *Tx) CreateTetATetChat(behalfUserId int64, toParticipantId int64) (int64, error) {
	tetATetChatName := fmt.Sprintf("tet_a_tet_%v_%v", behalfUserId, toParticipantId)
	chatId, _, err := tx.CreateChat(&Chat{Title: tetATetChatName, TetATet: true})
	return chatId, err
}

func (tx *Tx) IsExistsTetATet(participant1 int64, participant2 int64) (bool, int64, error) {
	res := tx.QueryRow("select b.chat_id from (select a.count >= 2 as exists, a.chat_id from ( (select cp.chat_id, count(cp.user_id) from chat_participant cp join chat ch on ch.id = cp.chat_id where ch.tet_a_tet = true and (cp.user_id = $1 or cp.user_id = $2) group by cp.chat_id)) a) b where b.exists is true;", participant1, participant2)
	var chatId int64
	if err := res.Scan(&chatId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// there were no rows, but otherwise no error occurred
			return false, 0, nil
		}
		Logger.Errorf("Error during getting chat id %v", err)
		return false, 0, err
	}
	return true, chatId, nil
}

func (db *DB) GetChats(participantId int64, limit int, offset int) ([]*Chat, error) {
	var rows *sql.Rows
	var err error
	rows, err = db.Query(`SELECT id, title, avatar, avatar_big, last_update_date_time, tet_a_tet FROM chat WHERE id IN ( SELECT chat_id FROM chat_participant WHERE user_id = $1 ) ORDER BY (last_update_date_time, id) DESC LIMIT $2 OFFSET $3`, participantId, limit, offset)
	if err != nil {
		Logger.Errorf("Error during get chat rows %v", err)
		return nil, err
	} else {
		defer rows.Close()
		list := make([]*Chat, 0)
		for rows.Next() {
			chat := Chat{}
			if err := rows.Scan(&chat.Id, &chat.Title, &chat.Avatar, &chat.AvatarBig, &chat.LastUpdateDateTime, &chat.TetATet); err != nil {
				Logger.Errorf("Error during scan chat rows %v", err)
				return nil, err
			} else {
				list = append(list, &chat)
			}
		}
		return list, nil
	}
}

func convertToWithParticipants(db CommonOperations, chat *Chat, behalfUserId int64) (*ChatWithParticipants, error) {
	if ids, err := db.GetParticipantIds(chat.Id); err != nil {
		return nil, err
	} else {
		admin, err := db.IsAdmin(behalfUserId, chat.Id)
		if err != nil {
			return nil, err
		}
		ccc := &ChatWithParticipants{
			Chat:            *chat,
			ParticipantsIds: ids,
			IsAdmin:         admin,
		}
		return ccc, nil
	}
}

func (db *DB) GetChatsWithParticipants(participantId int64, limit int, offset int, userPrincipalDto *auth.AuthResult) ([]*ChatWithParticipants, error) {
	chats, err := db.GetChats(participantId, limit, offset)
	if err != nil {
		return nil, err
	} else {
		list := make([]*ChatWithParticipants, 0)
		for _, cc := range chats {
			if ccc, err := convertToWithParticipants(db, cc, userPrincipalDto.UserId); err != nil {
				return nil, err
			} else {
				list = append(list, ccc)
			}
		}
		return list, nil
	}
}

func (tx *Tx) GetChatWithParticipants(behalfParticipantId, chatId int64) (*ChatWithParticipants, error) {
	return getChatWithParticipantsCommon(tx, behalfParticipantId, chatId)
}

func (db *DB) GetChatWithParticipants(behalfParticipantId, chatId int64) (*ChatWithParticipants, error) {
	return getChatWithParticipantsCommon(db, behalfParticipantId, chatId)
}

func getChatWithParticipantsCommon(commonOps CommonOperations, behalfParticipantId, chatId int64) (*ChatWithParticipants, error) {
	if chat, err := commonOps.GetChat(behalfParticipantId, chatId); err != nil {
		return nil, err
	} else if chat == nil {
		return nil, nil
	} else {
		return convertToWithParticipants(commonOps, chat, behalfParticipantId)
	}
}

func (db *DB) CountChats() (int64, error) {
	var count int64
	row := db.QueryRow("SELECT count(*) FROM chat")
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	} else {
		return count, nil
	}
}

func (db *DB) CountChatsPerUser(userId int64) (int64, error) {
	var count int64
	row := db.QueryRow("SELECT count(*) FROM chat_participant WHERE user_id = $1", userId)
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	} else {
		return count, nil
	}
}

func (tx *Tx) DeleteChat(id int64) error {
	if _, err := tx.Exec(fmt.Sprintf(`DROP TABLE message_chat_%v;`, id)); err != nil {
		Logger.Errorf("Error during drop message table %v %v", id, err)
		return err
	}

	if _, err := tx.Exec("DELETE FROM chat WHERE id = $1", id); err != nil {
		Logger.Errorf("Error during delete chat %v %v", id, err)
		return err
	} else {
		// OK
		return nil
	}
}

func (tx *Tx) EditChat(id int64, newTitle string, avatar, avatarBig null.String) (*time.Time, error) {
	var lastUpdateDateTime time.Time
	res := tx.QueryRow(`UPDATE chat SET title = $2, avatar = $3, avatar_big = $4, last_update_date_time = utc_now() WHERE id = $1 RETURNING id, last_update_date_time`, id, newTitle, avatar, avatarBig)
	if err := res.Scan(&id, &lastUpdateDateTime); err != nil {
		Logger.Errorf("Error during getting chat id %v", err)
		return nil, err
	}
	return &lastUpdateDateTime, nil
}

func getChatCommon(co CommonOperations, participantId, chatId int64) (*Chat, error) {
	row := co.QueryRow(`SELECT id, title, avatar, avatar_big, last_update_date_time, tet_a_tet FROM chat WHERE chat.id in (SELECT chat_id FROM chat_participant WHERE user_id = $2 AND chat_id = $1)`, chatId, participantId)
	chat := Chat{}
	err := row.Scan(&chat.Id, &chat.Title, &chat.Avatar, &chat.AvatarBig, &chat.LastUpdateDateTime, &chat.TetATet)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil
	}
	if err != nil {
		Logger.Errorf("Error during get chat row %v", err)
		return nil, err
	} else {
		return &chat, nil
	}
}

func (db *DB) GetChat(participantId, chatId int64) (*Chat, error) {
	return getChatCommon(db, participantId, chatId)
}

func (tx *Tx) GetChat(participantId, chatId int64) (*Chat, error) {
	return getChatCommon(tx, participantId, chatId)
}

func (tx *Tx) UpdateChatLastDatetimeChat(id int64) error {
	if _, err := tx.Exec("UPDATE chat SET last_update_date_time = utc_now() WHERE id = $1", id); err != nil {
		Logger.Errorf("Error during update chat %v %v", id, err)
		return err
	} else {
		return nil
	}
}

func (db *DB) IsChatExists(chatId int64) (bool, error) {
	row := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM chat WHERE id = $1)`, chatId)
	exists := false
	err := row.Scan(&exists)
	if err != nil {
		Logger.Errorf("Error during get chat exists %v", err)
		return false, err
	} else {
		return exists, nil
	}

}
