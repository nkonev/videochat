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

func (db *DB) GetMessages(chatId int64, userId int64, limit int, startingFromItemId int64, reverse, hasHash bool, searchString string) ([]*Message, error) {
	if hasHash {
		leftLimit := limit / 2
		rightLimit := limit / 2

		if leftLimit == 0 {
			leftLimit = 1
		}
		if rightLimit == 0 {
			rightLimit = 1
		}

		leftLimitRes := db.QueryRow(fmt.Sprintf(`SELECT MIN(inn.id) FROM (SELECT m.id FROM message_chat_%v m WHERE id <= $1 ORDER BY id DESC LIMIT $2) inn`, chatId), startingFromItemId, leftLimit)
		var leftMessageId, rightMessageId int64
		err := leftLimitRes.Scan(&leftMessageId)
		if err != nil {
			Logger.Errorf("Error during getting left messageId %v", err)
			return nil, err
		}

		rightLimitRes := db.QueryRow(fmt.Sprintf(`SELECT MAX(inn.id) + 1 FROM (SELECT m.id FROM message_chat_%v m WHERE id >= $1 ORDER BY id ASC LIMIT $2) inn`, chatId), startingFromItemId, rightLimit)
		err = rightLimitRes.Scan(&rightMessageId)
		if err != nil {
			Logger.Errorf("Error during getting right messageId %v", err)
			return nil, err
		}

		order := "asc"
		if reverse {
			order = "desc"
		}

		rows, err := db.Query(fmt.Sprintf(`SELECT m.id, m.text, m.owner_id, m.create_date_time, m.edit_date_time, m.file_item_uuid FROM message_chat_%v m WHERE $3 IN ( SELECT chat_id FROM chat_participant WHERE user_id = $1 AND chat_id = $3 ) AND id >= $4 AND id <= $5 ORDER BY id %s LIMIT $2`, chatId, order), userId, limit, chatId, leftMessageId, rightMessageId)
		if err != nil {
			Logger.Errorf("Error during get chat rows with search %v", err)
			return nil, err
		}
		defer rows.Close()
		list := make([]*Message, 0)
		for rows.Next() {
			message := Message{ChatId: chatId}
			if err := rows.Scan(&message.Id, &message.Text, &message.OwnerId, &message.CreateDateTime, &message.EditDateTime, &message.FileItemUuid); err != nil {
				Logger.Errorf("Error during scan message rows %v", err)
				return nil, err
			} else {
				list = append(list, &message)
			}
		}
		return list, nil
	} else {
		order := "asc"
		nonEquality := "m.id > $3"
		if reverse {
			order = "desc"
			nonEquality = "m.id < $3"
		}
		var err error
		var rows *sql.Rows
		if searchString != "" {
			searchString = "%" + searchString + "%"
			rows, err = db.Query(fmt.Sprintf(`SELECT m.id, m.text, m.owner_id, m.create_date_time, m.edit_date_time, m.file_item_uuid FROM message_chat_%v m WHERE $4 IN ( SELECT chat_id FROM chat_participant WHERE user_id = $1 AND chat_id = $4 ) AND %s AND strip_tags(m.text) ILIKE $5 ORDER BY id %s LIMIT $2`, chatId, nonEquality, order), userId, limit, startingFromItemId, chatId, searchString)
			if err != nil {
				Logger.Errorf("Error during get chat rows %v", err)
				return nil, err
			}
		} else {
			rows, err = db.Query(fmt.Sprintf(`SELECT m.id, m.text, m.owner_id, m.create_date_time, m.edit_date_time, m.file_item_uuid FROM message_chat_%v m WHERE $4 IN ( SELECT chat_id FROM chat_participant WHERE user_id = $1 AND chat_id = $4 ) AND %s ORDER BY id %s LIMIT $2`, chatId, nonEquality, order), userId, limit, startingFromItemId, chatId)
			if err != nil {
				Logger.Errorf("Error during get chat rows with search %v", err)
				return nil, err
			}
		}

		defer rows.Close()
		list := make([]*Message, 0)
		for rows.Next() {
			message := Message{ChatId: chatId}
			if err := rows.Scan(&message.Id, &message.Text, &message.OwnerId, &message.CreateDateTime, &message.EditDateTime, &message.FileItemUuid); err != nil {
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

	res := tx.QueryRow(fmt.Sprintf(`INSERT INTO message_chat_%v (text, owner_id, file_item_uuid) VALUES ($1, $2, $3) RETURNING id, create_date_time, edit_date_time`, m.ChatId), m.Text, m.OwnerId, m.FileItemUuid)
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
	row := co.QueryRow(fmt.Sprintf(`SELECT m.id, m.text, m.owner_id, m.create_date_time, m.edit_date_time, m.file_item_uuid FROM message_chat_%v m WHERE m.id = $1 AND $3 in (SELECT chat_id FROM chat_participant WHERE user_id = $2 AND chat_id = $3)`, chatId), messageId, userId, chatId)
	message := Message{ChatId: chatId}
	err := row.Scan(&message.Id, &message.Text, &message.OwnerId, &message.CreateDateTime, &message.EditDateTime, &message.FileItemUuid)
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

func addMessageReadCommon(co CommonOperations, messageId, userId int64, chatId int64) (bool, error) {
	res, err := co.Exec(`INSERT INTO message_read (last_message_id, user_id, chat_id) VALUES ($1, $2, $3) ON CONFLICT (user_id, chat_id) DO UPDATE SET last_message_id = $1  WHERE $1 > (SELECT MAX(last_message_id) FROM message_read WHERE user_id = $2 AND chat_id = $3)`, messageId, userId, chatId)
	if err != nil {
		Logger.Errorf("Error during marking as read message id %v", err)
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		Logger.Errorf("Error getting affected rows during marking as read message id %v", err)
		return false, err
	}
	if affected > 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func (db *DB) AddMessageRead(messageId, userId int64, chatId int64) (bool, error) {
	return addMessageReadCommon(db, messageId, userId, chatId)
}

func (tx *Tx) AddMessageRead(messageId, userId int64, chatId int64) (bool, error) {
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

	if res, err := tx.Exec(fmt.Sprintf(`UPDATE message_chat_%v SET text = $1, edit_date_time = utc_now(), file_item_uuid = $2 WHERE owner_id = $3 AND id = $4`, m.ChatId), m.Text, m.FileItemUuid, m.OwnerId, m.Id); err != nil {
		Logger.Errorf("Error during editing message id %v", err)
		return err
	} else {
		affected, err := res.RowsAffected()
		if err != nil {
			Logger.Errorf("Error during checking rows affected %v", err)
			return err
		}
		if affected == 0 {
			return errors.New("No rows affected")
		}
	}
	return nil
}

func (db *DB) DeleteMessage(messageId int64, ownerId int64, chatId int64) error {
	if res, err := db.Exec(fmt.Sprintf(`DELETE FROM message_chat_%v WHERE id = $1 AND owner_id = $2`, chatId), messageId, ownerId); err != nil {
		Logger.Errorf("Error during deleting message id %v", err)
		return err
	} else {
		affected, err := res.RowsAffected()
		if err != nil {
			Logger.Errorf("Error during checking rows affected %v", err)
			return err
		}
		if affected == 0 {
			return errors.New("No rows affected")
		}
	}
	return nil
}

func (dbR *DB) SetFileItemUuidToNull(ownerId, chatId int64, uuid string) (int64, bool, error) {
	res := dbR.QueryRow(fmt.Sprintf(`UPDATE message_chat_%v SET file_item_uuid = NULL WHERE file_item_uuid = $1 AND owner_id = $2 RETURNING id`, chatId), uuid, ownerId)

	if res.Err() != nil {
		Logger.Errorf("Error during nulling file_item_uuid message id %v", res.Err())
		return 0, false, res.Err()
	}
	var messageId int64
	err := res.Scan(&messageId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// there were no rows, but otherwise no error occurred
			return 0, false, nil
		}
		return 0, false, err
	} else {
		return messageId, true, nil
	}
}

func getUnreadMessagesCountCommon(co CommonOperations, chatId int64, userId int64) (int64, error) {
	var count int64
	row := co.QueryRow("SELECT * FROM UNREAD_MESSAGES($1, $2)", chatId, userId)
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	} else {
		return count, nil
	}
}

func (db *DB) GetUnreadMessagesCount(chatId int64, userId int64) (int64, error) {
	return getUnreadMessagesCountCommon(db, chatId, userId)
}

func (tx *Tx) GetUnreadMessagesCount(chatId int64, userId int64) (int64, error) {
	return getUnreadMessagesCountCommon(tx, chatId, userId)
}

type MessageIdsAndTextPair struct {
	Id   int64
	Text string
}

func (db *DB) IsEmbedExists(chatId int64, filenames []string) ([]*MessageIdsAndTextPair, error) {
	if len(filenames) == 0 {
		Logger.Infof("Exiting from IsEmbedExists because no filenames")
		return make([]*MessageIdsAndTextPair, 0), nil
	}

	likePart := ""
	secondFile := false
	for _, filename := range filenames {
		if secondFile {
			likePart += " OR "
		}
		likePart += " text LIKE '%" + filename + "%' "

		secondFile = true
	}
	sqlString := fmt.Sprintf(`SELECT id, text FROM message_chat_%v WHERE %v LIMIT %v`, chatId, likePart, len(filenames))
	if rows, err := db.Query(sqlString); err != nil {
		Logger.Errorf("Error during get embed files existence %v", err)
		return nil, err
	} else {
		defer rows.Close()
		list := make([]*MessageIdsAndTextPair, 0)
		for rows.Next() {
			var dtoObj = new(MessageIdsAndTextPair)
			if err := rows.Scan(&dtoObj.Id, &dtoObj.Text); err != nil {
				Logger.Errorf("Error during embed files existence rows %v", err)
				return nil, err
			} else {
				list = append(list, dtoObj)
			}
		}

		return list, nil
	}
}
