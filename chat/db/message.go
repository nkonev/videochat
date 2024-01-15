package db

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/guregu/null"
	"github.com/ztrue/tracerr"
	"nkonev.name/chat/dto"
	. "nkonev.name/chat/logger"
	"time"
)

const MessageNotFoundId = 0

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

	Pinned      bool
	PinPromoted bool
	BlogPost    bool
}

func selectMessageClause(chatId int64) string {
	return fmt.Sprintf(`SELECT 
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
			m.embed_owner_id as embedded_message_resend_owner_id,
			m.pinned,
			m.pin_promoted,
			m.blog_post
		FROM message_chat_%v m 
		LEFT JOIN message_chat_%v me 
			ON (m.embed_message_id = me.id AND m.embed_message_type = 'reply')
	`, chatId, chatId)
}

func provideScanToMessage(message *Message) []any {
	return []any{
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
		&message.Pinned,
		&message.PinPromoted,
		&message.BlogPost,
	}
}

func getMessagesCommon(co CommonOperations, chatId int64, limit int, startingFromItemId int64, reverse, hasHash bool, searchString string) ([]*Message, error) {
	if hasHash {
		// has hash means that frontend's page has message hash
		// it means we need to calculate page/2 to the top and to the bottom
		// to respond page containing from two halves
		leftLimit := limit / 2
		rightLimit := limit / 2

		if leftLimit == 0 {
			leftLimit = 1
		}
		if rightLimit == 0 {
			rightLimit = 1
		}

		var leftMessageId, rightMessageId int64
		var searchStringPercents = ""
		if searchString != "" {
			searchStringPercents = "%" + searchString + "%"
		}

		var leftLimitRes *sql.Row
		if searchString != "" {
			leftLimitRes = co.QueryRow(fmt.Sprintf(`SELECT MIN(inn.id) FROM (SELECT m.id FROM message_chat_%v m WHERE id <= $1 AND strip_tags(m.text) ILIKE $3 ORDER BY id DESC LIMIT $2) inn`, chatId), startingFromItemId, leftLimit, searchStringPercents)
		} else {
			leftLimitRes = co.QueryRow(fmt.Sprintf(`SELECT MIN(inn.id) FROM (SELECT m.id FROM message_chat_%v m WHERE id <= $1 ORDER BY id DESC LIMIT $2) inn`, chatId), startingFromItemId, leftLimit)
		}
		err := leftLimitRes.Scan(&leftMessageId)
		if err != nil {
			return nil, tracerr.Wrap(err)
		}

		var rightLimitRes *sql.Row
		if searchString != "" {
			rightLimitRes = co.QueryRow(fmt.Sprintf(`SELECT MAX(inn.id) + 1 FROM (SELECT m.id FROM message_chat_%v m WHERE id >= $1 AND strip_tags(m.text) ILIKE $3 ORDER BY id ASC LIMIT $2) inn`, chatId), startingFromItemId, rightLimit, searchStringPercents)
		} else {
			rightLimitRes = co.QueryRow(fmt.Sprintf(`SELECT MAX(inn.id) + 1 FROM (SELECT m.id FROM message_chat_%v m WHERE id >= $1 ORDER BY id ASC LIMIT $2) inn`, chatId), startingFromItemId, rightLimit)
		}
		err = rightLimitRes.Scan(&rightMessageId)
		if err != nil {
			return nil, tracerr.Wrap(err)
		}

		order := "asc"
		if reverse {
			order = "desc"
		}

		var rows *sql.Rows
		if searchString != "" {
			rows, err = co.Query(fmt.Sprintf(`%v
					WHERE 
							m.id >= $2 
						AND m.id <= $3 
						AND strip_tags(m.text) ILIKE $4
					ORDER BY m.id %s 
					LIMIT $1`, selectMessageClause(chatId), order),
				limit, leftMessageId, rightMessageId, searchStringPercents)
			if err != nil {
				return nil, tracerr.Wrap(err)
			}
			defer rows.Close()
		} else {
			rows, err = co.Query(fmt.Sprintf(`%v
					WHERE 
							m.id >= $2 
						AND m.id <= $3 
					ORDER BY m.id %s 
					LIMIT $1`, selectMessageClause(chatId), order),
				limit, leftMessageId, rightMessageId)
			if err != nil {
				return nil, tracerr.Wrap(err)
			}
			defer rows.Close()
		}
		list := make([]*Message, 0)
		for rows.Next() {
			message := Message{ChatId: chatId}
			if err := rows.Scan(provideScanToMessage(&message)[:]...); err != nil {
				return nil, tracerr.Wrap(err)
			} else {
				list = append(list, &message)
			}
		}
		return list, nil
	} else {
		order := "asc"
		nonEquality := "m.id > $2"
		if reverse {
			order = "desc"
			nonEquality = "m.id < $2"
		}
		var err error
		var rows *sql.Rows
		if searchString != "" {
			searchStringPercents := "%" + searchString + "%"
			rows, err = co.Query(fmt.Sprintf(`%v
			WHERE 
		    	    %s 
				AND strip_tags(m.text) ILIKE $3 
			ORDER BY m.id %s 
			LIMIT $1`, selectMessageClause(chatId), nonEquality, order),
				limit, startingFromItemId, searchStringPercents)
			if err != nil {
				return nil, tracerr.Wrap(err)
			}
			defer rows.Close()
		} else {
			rows, err = co.Query(fmt.Sprintf(`%v
			WHERE 
				  %s 
			ORDER BY m.id %s 
			LIMIT $1`, selectMessageClause(chatId), nonEquality, order),
				limit, startingFromItemId)
			if err != nil {
				return nil, tracerr.Wrap(err)
			}
			defer rows.Close()
		}

		list := make([]*Message, 0)
		for rows.Next() {
			message := Message{ChatId: chatId}
			if err := rows.Scan(provideScanToMessage(&message)[:]...); err != nil {
				return nil, tracerr.Wrap(err)
			} else {
				list = append(list, &message)
			}
		}
		return list, nil
	}
}

func (db *DB) GetMessages(chatId int64, limit int, startingFromItemId int64, reverse, hasHash bool, searchString string) ([]*Message, error) {
	return getMessagesCommon(db, chatId, limit, startingFromItemId, reverse, hasHash, searchString)
}

func (tx *Tx) GetMessages(chatId int64, limit int, startingFromItemId int64, reverse, hasHash bool, searchString string) ([]*Message, error) {
	return getMessagesCommon(tx, chatId, limit, startingFromItemId, reverse, hasHash, searchString)
}

func getCommentsCommon(co CommonOperations, chatId int64, blogPostId int64, limit int, startingFromItemId int64, reverse bool) ([]*Message, error) {
	order := "asc"
	nonEquality := "m.id > $2"
	if reverse {
		order = "desc"
		nonEquality = "m.id < $2"
	}
	var err error
	var rows *sql.Rows
	var preparedSql = fmt.Sprintf(`%v
			WHERE
				  %s 
				  AND m.id > $3 
			ORDER BY m.id %s 
			LIMIT $1`, selectMessageClause(chatId), nonEquality, order)
	rows, err = co.Query(preparedSql,
		limit, startingFromItemId, blogPostId)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	defer rows.Close()
	list := make([]*Message, 0)
	for rows.Next() {
		message := Message{ChatId: chatId}
		if err := rows.Scan(provideScanToMessage(&message)[:]...); err != nil {
			return nil, tracerr.Wrap(err)
		} else {
			list = append(list, &message)
		}
	}
	return list, nil
}

func (db *DB) GetComments(chatId int64, blogPostId int64, limit int, startingFromItemId int64, reverse bool) ([]*Message, error) {
	return getCommentsCommon(db, chatId, blogPostId, limit, startingFromItemId, reverse)
}

func (tx *Tx) GetComments(chatId int64, blogPostId int64, limit int, startingFromItemId int64, reverse bool) ([]*Message, error) {
	return getCommentsCommon(tx, chatId, blogPostId, limit, startingFromItemId, reverse)
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
			return ret, tracerr.Wrap(errors.New("Unexpected branch in saving in db"))
		}
	}
	return ret, nil
}

func (tx *Tx) HasMessages(chatId int64) (bool, error) {
	var exists bool = false
	row := tx.QueryRow(fmt.Sprintf(`SELECT exists(SELECT * FROM message_chat_%v LIMIT 1)`, chatId))
	if err := row.Scan(&exists); err != nil {
		return false, tracerr.Wrap(err)
	} else {
		return exists, nil
	}
}

func (tx *Tx) CreateMessage(m *Message) (id int64, createDatetime time.Time, editDatetime null.Time, err error) {
	if m == nil {
		return id, createDatetime, editDatetime, tracerr.Wrap(errors.New("message required"))
	} else if m.Text == "" {
		return id, createDatetime, editDatetime, tracerr.Wrap(errors.New("text required"))
	}

	embed, err := initEmbedMessageRequestStruct(m)
	if err != nil {
		return id, createDatetime, editDatetime, tracerr.Wrap(errors.New("error during initializing embed struct"))
	}
	res := tx.QueryRow(fmt.Sprintf(`INSERT INTO message_chat_%v (text, owner_id, file_item_uuid, embed_message_id, embed_chat_id, embed_owner_id, embed_message_type, blog_post) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, create_date_time, edit_date_time`, m.ChatId), m.Text, m.OwnerId, m.FileItemUuid, embed.embedMessageId, embed.embedMessageChatId, embed.embedMessageOwnerId, embed.embedMessageType, m.BlogPost)
	if err := res.Scan(&id, &createDatetime, &editDatetime); err != nil {
		return id, createDatetime, editDatetime, tracerr.Wrap(err)
	}
	return id, createDatetime, editDatetime, nil
}

func (db *DB) CountMessages() (int64, error) {
	var count int64
	row := db.QueryRow("SELECT count(*) FROM message")
	err := row.Scan(&count)
	if err != nil {
		return 0, tracerr.Wrap(err)
	} else {
		return count, nil
	}
}

func getMessageCommon(co CommonOperations, chatId int64, userId int64, messageId int64) (*Message, error) {
	row := co.QueryRow(fmt.Sprintf(`%v
	WHERE 
	    m.id = $1 
		AND $3 in (SELECT chat_id FROM chat_participant WHERE user_id = $2 AND chat_id = $3)`, selectMessageClause(chatId)),
		messageId, userId, chatId)
	message := Message{ChatId: chatId}
	err := row.Scan(provideScanToMessage(&message)[:]...)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil
	}
	if err != nil {
		return nil, tracerr.Wrap(err)
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

func (tx *Tx) SetBlogPost(chatId int64, messageId int64) error {
	_, err := tx.Exec(fmt.Sprintf("UPDATE message_chat_%v SET blog_post = false", chatId))
	if err != nil {
		return tracerr.Wrap(err)
	}

	_, err = tx.Exec(fmt.Sprintf("UPDATE message_chat_%v SET blog_post = true WHERE id = $1", chatId), messageId)
	if err != nil {
		return tracerr.Wrap(err)
	}
	return nil
}

func getMessageBasicCommon(co CommonOperations, chatId int64, messageId int64) (*string, *int64, error) {
	row := co.QueryRow(fmt.Sprintf(`SELECT 
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
		return nil, nil, tracerr.Wrap(err)
	} else {
		return &result, &owner, nil
	}
}

func (tx *Tx) GetMessageBasic(chatId int64, messageId int64) (*string, *int64, error) {
	return getMessageBasicCommon(tx, chatId, messageId)
}

func (db *DB) GetMessageBasic(chatId int64, messageId int64) (*string, *int64, error) {
	return getMessageBasicCommon(db, chatId, messageId)
}

func (tx *Tx) GetBlogPostMessageId(chatId int64) (*int64, error) {
	row := tx.QueryRow(fmt.Sprintf(`
							SELECT 
								m.id 
							FROM message_chat_%v m 
							WHERE 
								m.blog_post IS TRUE
							ORDER BY id LIMIT 1
						`, chatId),
	)
	var id int64
	err := row.Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil
	}
	if err != nil {
		return nil, tracerr.Wrap(err)
	} else {
		return &id, nil
	}
}

func addMessageReadCommon(co CommonOperations, messageId, userId int64, chatId int64) (bool, error) {
	res, err := co.Exec(`INSERT INTO message_read (last_message_id, user_id, chat_id) VALUES ($1, $2, $3) ON CONFLICT (user_id, chat_id) DO UPDATE SET last_message_id = $1  WHERE $1 > (SELECT MAX(last_message_id) FROM message_read WHERE user_id = $2 AND chat_id = $3)`, messageId, userId, chatId)
	if err != nil {
		return false, tracerr.Wrap(err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, tracerr.Wrap(err)
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
		return tracerr.Wrap(errors.New("message required"))
	} else if m.Text == "" {
		return tracerr.Wrap(errors.New("text required"))
	} else if m.Id == 0 {
		return tracerr.Wrap(errors.New("id required"))
	}

	embed, err := initEmbedMessageRequestStruct(m)
	if err != nil {
		return err
	}

	if res, err := tx.Exec(fmt.Sprintf(`UPDATE message_chat_%v SET text = $1, edit_date_time = utc_now(), file_item_uuid = $2, embed_message_id = $5, embed_chat_id = $6, embed_owner_id = $7, embed_message_type = $8, blog_post = $9 WHERE owner_id = $3 AND id = $4`, m.ChatId), m.Text, m.FileItemUuid, m.OwnerId, m.Id, embed.embedMessageId, embed.embedMessageChatId, embed.embedMessageOwnerId, embed.embedMessageType, m.BlogPost); err != nil {
		return tracerr.Wrap(err)
	} else {
		affected, err := res.RowsAffected()
		if err != nil {
			return tracerr.Wrap(err)
		}
		if affected == 0 {
			return tracerr.Wrap(errors.New("No rows affected"))
		}
	}
	return nil
}

func (db *DB) DeleteMessage(messageId int64, ownerId int64, chatId int64) error {
	if res, err := db.Exec(fmt.Sprintf(`DELETE FROM message_chat_%v WHERE id = $1 AND owner_id = $2`, chatId), messageId, ownerId); err != nil {
		return tracerr.Wrap(err)
	} else {
		affected, err := res.RowsAffected()
		if err != nil {
			return tracerr.Wrap(err)
		}
		if affected == 0 {
			return tracerr.Wrap(errors.New("No rows affected"))
		}
	}
	return nil
}

func (dbR *DB) SetFileItemUuidToNull(ownerId, chatId int64, uuid string) (int64, bool, error) {
	res := dbR.QueryRow(fmt.Sprintf(`UPDATE message_chat_%v SET file_item_uuid = NULL WHERE file_item_uuid = $1 AND owner_id = $2 RETURNING id`, chatId), uuid, ownerId)

	if res.Err() != nil {
		Logger.Errorf("Error during nulling file_item_uuid message id %v", res.Err())
		return 0, false, tracerr.Wrap(res.Err())
	}
	var messageId int64
	err := res.Scan(&messageId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// there were no rows, but otherwise no error occurred
			return 0, false, nil
		}
		return 0, false, tracerr.Wrap(err)
	} else {
		return messageId, true, nil
	}
}

func (dbR *DB) SetFileItemUuidTo(ownerId, chatId, messageId int64, uuid *string) (error) {
	_, err := dbR.Exec(fmt.Sprintf(`UPDATE message_chat_%v SET file_item_uuid = $1 WHERE id = $2 AND owner_id = $3`, chatId), uuid, messageId, ownerId)

	if err != nil {
		Logger.Errorf("Error during nulling file_item_uuid message id %v", err)
		return tracerr.Wrap(err)
	}
	return nil
}

func getUnreadMessagesCountCommon(co CommonOperations, chatId int64, userId int64) (int64, error) {
	var count int64
	row := co.QueryRow("SELECT * FROM UNREAD_MESSAGES($1, $2)", chatId, userId)
	err := row.Scan(&count)
	if err != nil {
		return 0, tracerr.Wrap(err)
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

func getUnreadMessagesCountBatchCommon(co CommonOperations, chatIds []int64, userId int64) (map[int64]int64, error) {
	res := map[int64]int64{}

	if len(chatIds) == 0 {
		return res, nil
	}

	var builder = ""
	var first = true
	for _, chatId := range chatIds {
		if !first {
			builder += " union "
		}
		builder += fmt.Sprintf("(SELECT %v, * FROM UNREAD_MESSAGES(%v, %v))", chatId, chatId, userId)

		first = false
	}

	var rows *sql.Rows
	var err error
	rows, err = co.Query(builder)
	if err != nil {
		return nil, tracerr.Wrap(err)
	} else {
		defer rows.Close()
		for _, cid := range chatIds {
			res[cid] = 0
		}
		for rows.Next() {
			var chatId int64
			var count int64
			if err := rows.Scan(&chatId, &count); err != nil {
				return nil, tracerr.Wrap(err)
			} else {
				res[chatId] = count
			}
		}
		return res, nil
	}
}

func (db *DB) GetUnreadMessagesCountBatch(chatIds []int64, userId int64) (map[int64]int64, error) {
	return getUnreadMessagesCountBatchCommon(db, chatIds, userId)
}

func (tx *Tx) GetUnreadMessagesCountBatch(chatIds []int64, userId int64) (map[int64]int64, error) {
	return getUnreadMessagesCountBatchCommon(tx, chatIds, userId)
}

func (tx *Tx) HasPinnedMessages(chatId int64) (hasPinnedMessages bool, err error) {
	row := tx.QueryRow(fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM message_chat_%v WHERE pinned IS TRUE)", chatId))
	err = tracerr.Wrap(row.Scan(&hasPinnedMessages))
	return
}

func (tx *Tx) PinMessage(chatId, messageId int64, shouldPin bool) error {
	_, err := tx.Exec(fmt.Sprintf("UPDATE message_chat_%v SET pinned = $1 WHERE id = $2", chatId), shouldPin, messageId)
	if err != nil {
		return tracerr.Wrap(err)
	}
	return nil
}

func (tx *Tx) GetPinnedMessages(chatId int64, limit, offset int) ([]*Message, error) {
	rows, err := tx.Query(fmt.Sprintf(`%v
			WHERE 
			    m.pinned IS TRUE
			ORDER BY m.id desc
			LIMIT $1 OFFSET $2`, selectMessageClause(chatId)),
		limit, offset)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	defer rows.Close()
	list := make([]*Message, 0)
	for rows.Next() {
		message := Message{ChatId: chatId}
		if err := rows.Scan(provideScanToMessage(&message)[:]...); err != nil {
			return nil, tracerr.Wrap(err)
		} else {
			list = append(list, &message)
		}
	}
	return list, nil
}

func (tx *Tx) GetPinnedMessagesCount(chatId int64) (int64, error) {
	row := tx.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM message_chat_%v WHERE pinned IS TRUE`, chatId))
	if row.Err() != nil {
		return 0, tracerr.Wrap(row.Err())
	}
	var res int64
	err := row.Scan(&res)
	if err != nil {
		return 0, tracerr.Wrap(err)
	}
	return res, nil
}

func (tx *Tx) UnpromoteMessages(chatId int64) error {
	_, err := tx.Exec(fmt.Sprintf(`UPDATE message_chat_%v SET pin_promoted = FALSE`, chatId))
	return tracerr.Wrap(err)
}

func (tx *Tx) PromoteMessage(chatId, messageId int64) error {
	_, err := tx.Exec(fmt.Sprintf(`UPDATE message_chat_%v SET pin_promoted = TRUE WHERE id = $1`, chatId), messageId)
	return tracerr.Wrap(err)
}

func (tx *Tx) PromotePreviousMessage(chatId int64) error {
	_, err := tx.Exec(fmt.Sprintf(`UPDATE message_chat_%v SET pin_promoted = TRUE WHERE id IN (SELECT id FROM message_chat_%v WHERE pinned IS TRUE ORDER BY id DESC LIMIT 1)`, chatId, chatId))
	return tracerr.Wrap(err)
}

func (tx *Tx) GetPinnedPromoted(chatId int64) (*Message, error) {
	row := tx.QueryRow(fmt.Sprintf(`%v
			WHERE 
			    m.pinned IS TRUE AND m.pin_promoted IS TRUE
			ORDER BY m.id desc
			LIMIT 1`, selectMessageClause(chatId)),
	)
	if row.Err() != nil {
		Logger.Errorf("Error during get pinned messages %v", row.Err())
		return nil, tracerr.Wrap(row.Err())
	}

	message := Message{ChatId: chatId}
	err := row.Scan(provideScanToMessage(&message)[:]...)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil
	}
	if err != nil {
		return nil, tracerr.Wrap(err)
	} else {
		return &message, nil
	}
}

func (db *DB) GetParticipantsRead(chatId, messageId int64, limit, offset int) ([]int64, error) {
	rows, err := db.Query(fmt.Sprintf(`
			select user_id from message_read where chat_id = $1 and last_message_id >= $2
			ORDER BY user_id asc
			LIMIT $3 OFFSET $4`,
	),
		chatId, messageId,
		limit, offset)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	defer rows.Close()
	list := make([]int64, 0)
	for rows.Next() {
		var anUserId int64
		if err := rows.Scan(&anUserId); err != nil {
			return nil, tracerr.Wrap(err)
		} else {
			list = append(list, anUserId)
		}
	}
	return list, nil
}

func (db *DB) GetParticipantsReadCount(chatId, messageId int64) (int, error) {
	row := db.QueryRow(fmt.Sprintf(`
			select count(user_id) from message_read where chat_id = $1 and last_message_id >= $2
			`,
		),
		chatId, messageId)
	if row.Err() != nil {
		Logger.Errorf("Error during get count of participants read the message %v", row.Err())
		return 0, tracerr.Wrap(row.Err())
	}

	var count int
	err := row.Scan(&count)
	if err != nil {
		return 0, tracerr.Wrap(err)
	} else {
		return count, nil
	}
}


func (db *DB) FindMessageByFileItemUuid(chatId int64, fileItemUuid string) (int64, error) {
	if len(fileItemUuid) == 0 {
		return MessageNotFoundId, nil
	}
	fileItemUuidWithPercents := "%" + fileItemUuid + "%"
	sqlFormatted := fmt.Sprintf(`
			select id from message_chat_%v where file_item_uuid = $1 or text ilike $2 order by id limit 1
			`, chatId,
	)
	row := db.QueryRow(sqlFormatted, fileItemUuid, fileItemUuidWithPercents)
	if row.Err() != nil {
		Logger.Errorf("Error during get MessageByFileItemUuid %v", row.Err())
		return 0, tracerr.Wrap(row.Err())
	}

	var messageId int64
	err := row.Scan(&messageId)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return MessageNotFoundId, nil
	}
	if err != nil {
		return 0, tracerr.Wrap(err)
	} else {
		return messageId, nil
	}
}
