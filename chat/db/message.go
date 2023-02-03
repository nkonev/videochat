package db

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/guregu/null"
	"nkonev.name/chat/dto"
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

	RequestEmbeddedMessageId      *int64
	RequestEmbeddedMessageType    *string
	RequestEmbeddedMessageChatId  *int64
	RequestEmbeddedMessageOwnerId *int64

	ResponseEmbeddedMessageType *string

	ResponseEmbeddedMessageReplyId      *int64
	ResponseEmbeddedMessageReplyOwnerId *int64
	ResponseEmbeddedMessageReplyText    *string

	ResponseEmbeddedMessageResendId      *int64
	ResponseEmbeddedMessageResendOwnerId *int64
	ResponseEmbeddedMessageResendChatId  *int64
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

		rows, err := db.Query(fmt.Sprintf(`SELECT 
    		m.id, 
    		m.text, 
    		m.owner_id,
    		m.create_date_time, 
    		m.edit_date_time, 
    		m.file_item_uuid,
			m.embed_message_type as embedded_message_type,
			me.id as embedded_message_reply_id,
			me.text as embedded_message_reply_text,
			me.owner_id as embedded_message_reply_owner_id,
			m.embed_message_id as embedded_message_resend_id,
			m.embed_chat_id as embedded_message_resend_chat_id,
			m.embed_owner_id as embedded_message_resend_owner_id
		FROM message_chat_%v m 
		LEFT JOIN message_chat_%v me 
			ON (m.embed_message_id = me.id AND m.embed_message_type = 'reply')
		WHERE 
		    $3 IN ( SELECT chat_id FROM chat_participant WHERE user_id = $1 AND chat_id = $3 ) 
			AND m.id >= $4 
			AND m.id <= $5 
		ORDER BY m.id %s 
		LIMIT $2`, chatId, chatId, order),
			userId, limit, chatId, leftMessageId, rightMessageId)

		if err != nil {
			Logger.Errorf("Error during get chat rows with search %v", err)
			return nil, err
		}
		defer rows.Close()
		list := make([]*Message, 0)
		for rows.Next() {
			message := Message{ChatId: chatId}
			if err := rows.Scan(
				&message.Id,
				&message.Text,
				&message.OwnerId,
				&message.CreateDateTime,
				&message.EditDateTime,
				&message.FileItemUuid,
				&message.ResponseEmbeddedMessageType,
				&message.ResponseEmbeddedMessageReplyId,
				&message.ResponseEmbeddedMessageReplyText,
				&message.ResponseEmbeddedMessageReplyOwnerId,
				&message.ResponseEmbeddedMessageResendId,
				&message.ResponseEmbeddedMessageResendChatId,
				&message.ResponseEmbeddedMessageResendOwnerId,
			); err != nil {
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
			searchStringPercents := "%" + searchString + "%"
			rows, err = db.Query(fmt.Sprintf(`SELECT 
    			m.id, 
    			m.text,
    			m.owner_id, 
    			m.create_date_time, 
    			m.edit_date_time, 
    			m.file_item_uuid,
				m.embed_message_type as embedded_message_type,
				me.id as embedded_message_reply_id,
				me.text as embedded_message_reply_text,
				me.owner_id as embedded_message_reply_owner_id,
				m.embed_message_id as embedded_message_resend_id,
				m.embed_chat_id as embedded_message_resend_chat_id,
				m.embed_owner_id as embedded_message_resend_owner_id
			FROM message_chat_%v m 
			LEFT JOIN message_chat_%v me 
				ON (m.embed_message_id = me.id AND m.embed_message_type = 'reply')
			WHERE 
		    	$4 IN (SELECT chat_id FROM chat_participant WHERE user_id = $1 AND chat_id = $4) 
				AND %s 
				AND strip_tags(m.text) ILIKE $5 
			ORDER BY m.id %s 
			LIMIT $2`, chatId, chatId, nonEquality, order), userId, limit, startingFromItemId, chatId, searchStringPercents)
			if err != nil {
				Logger.Errorf("Error during get chat rows %v", err)
				return nil, err
			}
		} else {
			rows, err = db.Query(fmt.Sprintf(`SELECT 
    			m.id, 
    			m.text,
    			m.owner_id, 
    			m.create_date_time,
    			m.edit_date_time, 
    			m.file_item_uuid,
				m.embed_message_type as embedded_message_type,
				me.id as embedded_message_reply_id,
				me.text as embedded_message_reply_text,
				me.owner_id as embedded_message_reply_owner_id,
				m.embed_message_id as embedded_message_resend_id,
				m.embed_chat_id as embedded_message_resend_chat_id,
				m.embed_owner_id as embedded_message_resend_owner_id
			FROM message_chat_%v m 
			LEFT JOIN message_chat_%v me 
				ON (m.embed_message_id = me.id AND m.embed_message_type = 'reply')
			WHERE 
			    $4 IN ( SELECT chat_id FROM chat_participant WHERE user_id = $1 AND chat_id = $4 ) 
				AND %s 
			ORDER BY m.id %s 
			LIMIT $2`, chatId, chatId, nonEquality, order),
				userId, limit, startingFromItemId, chatId)
			if err != nil {
				Logger.Errorf("Error during get chat rows with search %v", err)
				return nil, err
			}
		}

		defer rows.Close()
		list := make([]*Message, 0)
		for rows.Next() {
			message := Message{ChatId: chatId}
			if err := rows.Scan(
				&message.Id,
				&message.Text,
				&message.OwnerId,
				&message.CreateDateTime,
				&message.EditDateTime,
				&message.FileItemUuid,
				&message.ResponseEmbeddedMessageType,
				&message.ResponseEmbeddedMessageReplyId,
				&message.ResponseEmbeddedMessageReplyText,
				&message.ResponseEmbeddedMessageReplyOwnerId,
				&message.ResponseEmbeddedMessageResendId,
				&message.ResponseEmbeddedMessageResendChatId,
				&message.ResponseEmbeddedMessageResendOwnerId,
			); err != nil {
				Logger.Errorf("Error during scan message rows %v", err)
				return nil, err
			} else {
				list = append(list, &message)
			}
		}
		return list, nil
	}
}

type embedMessage struct {
	embedMessageId      *int64
	embedMessageChatId  *int64
	embedMessageOwnerId *int64
	embedMessageType    *string
}

func initEmbedMessageRequestStruct(m *Message) (embedMessage, error) {
	ret := embedMessage{}
	if m.RequestEmbeddedMessageType != nil {
		if *m.RequestEmbeddedMessageType == dto.EmbedMessageTypeReply {
			ret.embedMessageId = m.RequestEmbeddedMessageId
			ret.embedMessageType = m.RequestEmbeddedMessageType
		} else if *m.RequestEmbeddedMessageType == dto.EmbedMessageTypeResend {
			ret.embedMessageId = m.RequestEmbeddedMessageId
			ret.embedMessageChatId = m.RequestEmbeddedMessageChatId
			ret.embedMessageOwnerId = m.RequestEmbeddedMessageOwnerId
			ret.embedMessageType = m.RequestEmbeddedMessageType
		} else {
			return ret, errors.New("Unexpected branch in saving in db")
		}
	}
	return ret, nil
}

func (tx *Tx) CreateMessage(m *Message) (id int64, createDatetime time.Time, editDatetime null.Time, err error) {
	if m == nil {
		return id, createDatetime, editDatetime, errors.New("message required")
	} else if m.Text == "" {
		return id, createDatetime, editDatetime, errors.New("text required")
	}

	embed, err := initEmbedMessageRequestStruct(m)
	if err != nil {
		return id, createDatetime, editDatetime, errors.New("error during initializing embed struct")
	}
	res := tx.QueryRow(fmt.Sprintf(`INSERT INTO message_chat_%v (text, owner_id, file_item_uuid, embed_message_id, embed_chat_id, embed_owner_id, embed_message_type) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, create_date_time, edit_date_time`, m.ChatId), m.Text, m.OwnerId, m.FileItemUuid, embed.embedMessageId, embed.embedMessageChatId, embed.embedMessageOwnerId, embed.embedMessageType)
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
	row := co.QueryRow(fmt.Sprintf(`SELECT 
    	m.id, 
    	m.text,
    	m.owner_id,
    	m.create_date_time, 
    	m.edit_date_time,
    	m.file_item_uuid,
		m.embed_message_type as embedded_message_type,
		me.id as embedded_message_reply_id,
		me.text as embedded_message_reply_text,
		me.owner_id as embedded_message_reply_owner_id,
		m.embed_message_id as embedded_message_resend_id,
		m.embed_chat_id as embedded_message_resend_chat_id,
		m.embed_owner_id as embedded_message_resend_owner_id
	FROM message_chat_%v m 
	LEFT JOIN message_chat_%v me 
		ON (m.embed_message_id = me.id AND m.embed_message_type = 'reply')
	WHERE 
	    m.id = $1 
		AND $3 in (SELECT chat_id FROM chat_participant WHERE user_id = $2 AND chat_id = $3)`, chatId, chatId),
		messageId, userId, chatId)
	message := Message{ChatId: chatId}
	err := row.Scan(
		&message.Id,
		&message.Text,
		&message.OwnerId,
		&message.CreateDateTime,
		&message.EditDateTime,
		&message.FileItemUuid,
		&message.ResponseEmbeddedMessageType,
		&message.ResponseEmbeddedMessageReplyId,
		&message.ResponseEmbeddedMessageReplyText,
		&message.ResponseEmbeddedMessageReplyOwnerId,
		&message.ResponseEmbeddedMessageResendId,
		&message.ResponseEmbeddedMessageResendChatId,
		&message.ResponseEmbeddedMessageResendOwnerId,
	)
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

func (tx *Tx) GetMessageBasic(chatId int64, messageId int64) (*string, *int64, error) {
	row := tx.QueryRow(fmt.Sprintf(`SELECT 
    	m.text,
    	m.owner_id
	FROM message_chat_%v m 
	WHERE 
	    m.id = $1 
`, chatId),
		messageId)
	var result string
	var owner int64
	err := row.Scan(&result, &owner)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil, nil
	}
	if err != nil {
		Logger.Errorf("Error during get message row %v", err)
		return nil, nil, err
	} else {
		return &result, &owner, nil
	}
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

	embed, err := initEmbedMessageRequestStruct(m)
	if err != nil {
		return errors.New("error during initializing embed struct")
	}

	if res, err := tx.Exec(fmt.Sprintf(`UPDATE message_chat_%v SET text = $1, edit_date_time = utc_now(), file_item_uuid = $2, embed_message_id = $5, embed_chat_id = $6, embed_owner_id = $7, embed_message_type = $8 WHERE owner_id = $3 AND id = $4`, m.ChatId), m.Text, m.FileItemUuid, m.OwnerId, m.Id, embed.embedMessageId, embed.embedMessageChatId, embed.embedMessageOwnerId, embed.embedMessageType); err != nil {
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
