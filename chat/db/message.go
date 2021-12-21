package db

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/guregu/null"
	. "nkonev.name/chat/logger"
	"time"
)

type Message struct {
	Id             int64
	Text           string
	ChatId         int64
	OwnerId        int64
	CreateDateTime time.Time
	EditDateTime   null.Time
	FileItemUuid   *uuid.UUID
}

func (db *DB) GetMessages(chatId int64, userId int64, limit int, startingFromItemId int64, reverse bool) ([]*Message, error) {
	order := "asc"
	nonEquality := "m.id > $3"
	if reverse {
		order = "desc"
		nonEquality = "m.id < $3"
	}
	if rows, err := db.Query(fmt.Sprintf(`SELECT m.id, m.text, m.chat_id, m.owner_id, m.create_date_time, m.edit_date_time, m.file_item_uuid FROM message_chat_%v m WHERE chat_id IN ( SELECT chat_id FROM chat_participant WHERE user_id = $1 AND chat_id = $4 ) AND %s ORDER BY id %s LIMIT $2`, chatId, nonEquality, order), userId, limit, startingFromItemId, chatId); err != nil {
		Logger.Errorf("Error during get chat rows %v", err)
		return nil, err
	} else {
		defer rows.Close()
		list := make([]*Message, 0)
		for rows.Next() {
			message := Message{}
			if err := rows.Scan(&message.Id, &message.Text, &message.ChatId, &message.OwnerId, &message.CreateDateTime, &message.EditDateTime, &message.FileItemUuid); err != nil {
				Logger.Errorf("Error during scan message rows %v", err)
				return nil, err
			} else {
				list = append(list, &message)
			}
		}
		return list, nil
	}
}

func (tx *Tx) CreateMessage(m *Message) (id int64, createDatetime time.Time, editDatetime null.Time, err error) {
	if m == nil {
		return id, createDatetime, editDatetime, errors.New("message required")
	} else if m.Text == "" {
		return id, createDatetime, editDatetime, errors.New("text required")
	}

	res := tx.QueryRow(fmt.Sprintf(`INSERT INTO message_chat_%v (text, chat_id, owner_id, file_item_uuid) VALUES ($1, $2, $3, $4) RETURNING id, create_date_time, edit_date_time`, m.ChatId), m.Text, m.ChatId, m.OwnerId, m.FileItemUuid)
	if err := res.Scan(&id, &createDatetime, &editDatetime); err != nil {
		Logger.Errorf("Error during getting message id %v", err)
		return id, createDatetime, editDatetime, err
	}
	return id, createDatetime, editDatetime, nil
}

func (db *DB) CountMessages() (int64, error) {
	var count int64
	row := db.QueryRow("SELECT count(*) FROM message")
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	} else {
		return count, nil
	}
}

func getMessageCommon(co CommonOperations, chatId int64, userId int64, messageId int64) (*Message, error) {
	row := co.QueryRow(fmt.Sprintf(`SELECT m.id, m.text, m.chat_id, m.owner_id, m.create_date_time, m.edit_date_time, m.file_item_uuid FROM message_chat_%v m WHERE m.id = $1 AND chat_id in (SELECT chat_id FROM chat_participant WHERE user_id = $2 AND chat_id = $3)`, chatId), messageId, userId, chatId)
	message := Message{}
	err := row.Scan(&message.Id, &message.Text, &message.ChatId, &message.OwnerId, &message.CreateDateTime, &message.EditDateTime, &message.FileItemUuid)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil
	}
	if err != nil {
		Logger.Errorf("Error during get message row %v", err)
		return nil, err
	} else {
		return &message, nil
	}
}

func (db *DB) GetMessage(chatId int64, userId int64, messageId int64) (*Message, error) {
	return getMessageCommon(db, chatId, userId, messageId)
}

func (tx *Tx) GetMessage(chatId int64, userId int64, messageId int64) (*Message, error) {
	return getMessageCommon(tx, chatId, userId, messageId)
}

func addMessageReadCommon(co CommonOperations, messageId, userId int64, chatId int64) error {
	_, err := co.Exec(`INSERT INTO message_read (last_message_id, user_id, chat_id) VALUES ($1, $2, $3) ON CONFLICT (user_id, chat_id) DO UPDATE SET last_message_id = $1  WHERE $1 > (SELECT MAX(last_message_id) FROM message_read WHERE user_id = $2 AND chat_id = $3)`, messageId, userId, chatId)
	return err
}

func (db *DB) AddMessageRead(messageId, userId int64, chatId int64) error {
	return addMessageReadCommon(db, messageId, userId, chatId)
}

func (tx *Tx) AddMessageRead(messageId, userId int64, chatId int64) error {
	return addMessageReadCommon(tx, messageId, userId, chatId)
}

func (tx *Tx) EditMessage(m *Message) error {
	if m == nil {
		return errors.New("message required")
	} else if m.Text == "" {
		return errors.New("text required")
	} else if m.Id == 0 {
		return errors.New("id required")
	}

	if _, err := tx.Exec(fmt.Sprintf(`UPDATE message_chat_%v SET text = $1, edit_date_time = utc_now(), file_item_uuid = $2 WHERE owner_id = $3 AND id = $4`, m.ChatId), m.Text, m.FileItemUuid, m.OwnerId, m.Id); err != nil {
		Logger.Errorf("Error during editing message id %v", err)
		return err
	}
	return nil
}

func (db *DB) DeleteMessage(messageId int64, ownerId int64, chatId int64) error {
	if _, err := db.Exec(fmt.Sprintf(`DELETE FROM message_chat_%v WHERE id = $1 AND owner_id = $2 AND chat_id = $3`, chatId), messageId, ownerId, chatId); err != nil {
		Logger.Errorf("Error during deleting message id %v", err)
		return err
	}
	return nil
}

func (dbR *DB) SetFileItemUuidToNull(ownerId, chatId int64, uuid string) (int64, error) {
	res := dbR.QueryRow(fmt.Sprintf(`UPDATE message_chat_%v SET file_item_uuid = NULL WHERE file_item_uuid = $1 AND owner_id = $2 AND chat_id = $3 RETURNING id`, chatId), uuid, ownerId, chatId)

	if res.Err() != nil {
		Logger.Errorf("Error during nulling file_item_uuid message id %v", res.Err())
		return 0, res.Err()
	}
	var messageId int64
	err := res.Scan(&messageId)
	if err != nil {
		return 0, err
	} else {
		return messageId, nil
	}
}

func getUnreadMessagesCountCommon(co CommonOperations, chatId int64, userId int64) (int64, error) {
	var count int64
	row := co.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM message_chat_%v WHERE id > COALESCE((SELECT last_message_id FROM message_read WHERE user_id = $2 AND chat_id = $1), 0)", chatId), chatId, userId)
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	} else {
		return count, nil
	}
}

func getAllUnreadMessagesCountCommon(co CommonOperations, userId int64) (int64, error) {
	//	var count int64
	//	row := co.QueryRow(`
	//SELECT COALESCE(
	//	(SELECT SUM(unread) FROM (
	//		SELECT chp.chat_id, (
	//			SELECT COUNT(*) FROM message WHERE chat_id = chp.chat_id AND id > COALESCE((SELECT last_message_id FROM message_read WHERE user_id = $1 AND chat_id = chp.chat_id), 0)
	//			) AS unread FROM chat_participant chp WHERE chp.user_id = $1
	//	) AS alias_ignored),
	//	0
	//)
	//`, userId)
	//	err := row.Scan(&count)
	//	if err != nil {
	//		return 0, err
	//	} else {
	//		return count, nil
	//	}
	return 0, nil // TODO fix it
}

func (db *DB) GetUnreadMessagesCount(chatId int64, userId int64) (int64, error) {
	return getUnreadMessagesCountCommon(db, chatId, userId)
}

func (tx *Tx) GetUnreadMessagesCount(chatId int64, userId int64) (int64, error) {
	return getUnreadMessagesCountCommon(tx, chatId, userId)
}

func (db *DB) GetAllUnreadMessagesCount(userId int64) (int64, error) {
	return getAllUnreadMessagesCountCommon(db, userId)
}

func (tx *Tx) GetAllUnreadMessagesCount(userId int64) (int64, error) {
	return getAllUnreadMessagesCountCommon(tx, userId)
}
