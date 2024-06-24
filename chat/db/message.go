package db

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/guregu/null"
	"github.com/rotisserie/eris"
	"nkonev.name/chat/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
	"time"
)

const MessageNotFoundId = 0

type Reaction struct {
	MessageId int64
	UserId int64
	Reaction string
}

type Message struct {
	Id             int64
	Text           string
	ChatId         int64
	OwnerId        int64
	CreateDateTime time.Time
	EditDateTime   null.Time
	FileItemUuid   *string

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
	Published   bool
	Reactions []Reaction
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
			m.blog_post,
			m.published
		FROM message_chat_%v m 
		LEFT JOIN message_chat_%v me 
			ON (m.embed_message_id = me.id AND m.embed_message_type = '%v')
		`, chatId, chatId, dto.EmbedMessageTypeReply)
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
		&message.Published,
	}
}

// see also its copy in aaa::UserListViewRepository
func getMessagesCommon(co CommonOperations, chatId int64, limit int, startingFromItemId int64, reverse, hasHash bool, searchString string) ([]*Message, error) {
	list := make([]*Message, 0)
	var err error
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

		var leftMessageId, rightMessageId *int64
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
		err = leftLimitRes.Scan(&leftMessageId)
		if err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		}

		var rightLimitRes *sql.Row
		if searchString != "" {
			rightLimitRes = co.QueryRow(fmt.Sprintf(`SELECT MAX(inn.id) + 1 FROM (SELECT m.id FROM message_chat_%v m WHERE id >= $1 AND strip_tags(m.text) ILIKE $3 ORDER BY id ASC LIMIT $2) inn`, chatId), startingFromItemId, rightLimit, searchStringPercents)
		} else {
			rightLimitRes = co.QueryRow(fmt.Sprintf(`SELECT MAX(inn.id) + 1 FROM (SELECT m.id FROM message_chat_%v m WHERE id >= $1 ORDER BY id ASC LIMIT $2) inn`, chatId), startingFromItemId, rightLimit)
		}
		err = rightLimitRes.Scan(&rightMessageId)
		if err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		}

		if leftMessageId == nil || rightMessageId == nil {
			Logger.Infof("Got leftMessageId=%v, rightMessageId=%v for chatId=%v, startingFromItemId=%v, reverse=%v, searchString=%v, fallback to simple", leftMessageId, rightMessageId, chatId, startingFromItemId, reverse, searchString)
			list, err = getMessagesSimple(co, chatId, limit, 0, reverse, searchString)
			if err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			}
			return list, err
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
				limit, *leftMessageId, *rightMessageId, searchStringPercents)
			if err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			}
			defer rows.Close()
		} else {
			rows, err = co.Query(fmt.Sprintf(`%v
					WHERE 
							m.id >= $2 
						AND m.id <= $3 
					ORDER BY m.id %s 
					LIMIT $1`, selectMessageClause(chatId), order),
				limit, *leftMessageId, *rightMessageId)
			if err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			}
			defer rows.Close()
		}
		for rows.Next() {
			message := Message{ChatId: chatId, Reactions: make([]Reaction, 0)}
			if err = rows.Scan(provideScanToMessage(&message)[:]...); err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			} else {
				list = append(list, &message)
			}
		}
	} else {
		// otherwise, startingFromItemId is used as the top or the bottom limit of the portion
		list, err = getMessagesSimple(co, chatId, limit, startingFromItemId, reverse, searchString)
		if err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		}
	}

	err = enrichMessagesWithReactions(co, chatId, list)
	if err != nil {
		return nil, fmt.Errorf("Got error during enriching messages with reactions: %v", err)
	}

	return list, nil
}

func getMessagesSimple(co CommonOperations, chatId int64, limit int, startingFromItemId int64, reverse bool, searchString string) ([]*Message, error) {
	list := make([]*Message, 0)

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
			return nil, eris.Wrap(err, "error during interacting with db")
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
			return nil, eris.Wrap(err, "error during interacting with db")
		}
		defer rows.Close()
	}

	for rows.Next() {
		message := Message{ChatId: chatId, Reactions: make([]Reaction, 0)}
		if err := rows.Scan(provideScanToMessage(&message)[:]...); err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		} else {
			list = append(list, &message)
		}
	}

	return list, nil
}

func (db *DB) GetMessages(chatId int64, limit int, startingFromItemId int64, reverse, hasHash bool, searchString string) ([]*Message, error) {
	return getMessagesCommon(db, chatId, limit, startingFromItemId, reverse, hasHash, searchString)
}

func (tx *Tx) GetMessages(chatId int64, limit int, startingFromItemId int64, reverse, hasHash bool, searchString string) ([]*Message, error) {
	return getMessagesCommon(tx, chatId, limit, startingFromItemId, reverse, hasHash, searchString)
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
			return ret, eris.New("Unexpected branch in saving in db")
		}
	}
	return ret, nil
}

func (tx *Tx) HasMessages(chatId int64) (bool, error) {
	var exists bool = false
	row := tx.QueryRow(fmt.Sprintf(`SELECT exists(SELECT * FROM message_chat_%v LIMIT 1)`, chatId))
	if err := row.Scan(&exists); err != nil {
		return false, eris.Wrap(err, "error during interacting with db")
	} else {
		return exists, nil
	}
}

func (tx *Tx) CreateMessage(m *Message) (id int64, createDatetime time.Time, editDatetime null.Time, err error) {
	if m == nil {
		return id, createDatetime, editDatetime, eris.New("message required")
	} else if m.Text == "" {
		return id, createDatetime, editDatetime, eris.New("text required")
	}

	embed, err := initEmbedMessageRequestStruct(m)
	if err != nil {
		return id, createDatetime, editDatetime, eris.Wrap(err, "error during initializing embed struct")
	}
	res := tx.QueryRow(fmt.Sprintf(`INSERT INTO message_chat_%v (text, owner_id, file_item_uuid, embed_message_id, embed_chat_id, embed_owner_id, embed_message_type, blog_post) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, create_date_time, edit_date_time`, m.ChatId), m.Text, m.OwnerId, m.FileItemUuid, embed.embedMessageId, embed.embedMessageChatId, embed.embedMessageOwnerId, embed.embedMessageType, m.BlogPost)
	if err := res.Scan(&id, &createDatetime, &editDatetime); err != nil {
		return id, createDatetime, editDatetime, eris.Wrap(err, "error during interacting with db")
	}
	return id, createDatetime, editDatetime, nil
}

func (db *DB) CountMessages() (int64, error) {
	var count int64
	row := db.QueryRow("SELECT count(*) FROM message")
	err := row.Scan(&count)
	if err != nil {
		return 0, eris.Wrap(err, "error during interacting with db")
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
	message := Message{ChatId: chatId, Reactions: make([]Reaction, 0)}
	err := row.Scan(provideScanToMessage(&message)[:]...)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil
	}
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}

	reactions, err := getMessageReactionsCommon(co, chatId, messageId)
	if err != nil {
		return nil, err
	}
	message.Reactions = reactions

	return &message, nil
}

func (db *DB) GetMessage(chatId int64, userId int64, messageId int64) (*Message, error) {
	return getMessageCommon(db, chatId, userId, messageId)
}

func (tx *Tx) GetMessage(chatId int64, userId int64, messageId int64) (*Message, error) {
	return getMessageCommon(tx, chatId, userId, messageId)
}

func getMessagePublicCommon(co CommonOperations, chatId int64, messageId int64) (*Message, error) {
	row := co.QueryRow(fmt.Sprintf(`%v
	WHERE 
	    m.id = $1 
		AND m.published = true`, selectMessageClause(chatId)),
		messageId)
	message := Message{ChatId: chatId, Reactions: make([]Reaction, 0)}
	err := row.Scan(provideScanToMessage(&message)[:]...)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil
	}
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}

	reactions, err := getMessageReactionsCommon(co, chatId, messageId)
	if err != nil {
		return nil, err
	}
	message.Reactions = reactions

	return &message, nil
}

func (db *DB) GetMessagePublic(chatId int64, messageId int64) (*Message, error) {
	return getMessagePublicCommon(db, chatId, messageId)
}

func (tx *Tx) GetMessagePublic(chatId int64, messageId int64) (*Message, error) {
	return getMessagePublicCommon(tx, chatId, messageId)
}

func (tx *Tx) SetBlogPost(chatId int64, messageId int64) error {
	_, err := tx.Exec(fmt.Sprintf("UPDATE message_chat_%v SET blog_post = false", chatId))
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}

	_, err = tx.Exec(fmt.Sprintf("UPDATE message_chat_%v SET blog_post = true WHERE id = $1", chatId), messageId)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func getMessageBasicCommon(co CommonOperations, chatId int64, messageId int64) (*string, *int64, *bool, *bool, error) {
	row := co.QueryRow(fmt.Sprintf(`SELECT 
    	m.text,
    	m.owner_id,
    	m.blog_post,
    	m.published
	FROM message_chat_%v m 
	WHERE 
	    m.id = $1 
`, chatId),
		messageId)
	var result string
	var owner int64
	var blogPost bool
	var published bool
	err := row.Scan(&result, &owner, &blogPost, &published)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil, nil, nil, nil
	}
	if err != nil {
		return nil, nil, nil, nil, eris.Wrap(err, "error during interacting with db")
	} else {
		return &result, &owner, &blogPost, &published, nil
	}
}

func (tx *Tx) GetMessageBasic(chatId int64, messageId int64) (*string, *int64, *bool, *bool, error) {
	return getMessageBasicCommon(tx, chatId, messageId)
}

func (db *DB) GetMessageBasic(chatId int64, messageId int64) (*string, *int64, *bool, *bool, error) {
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
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		return &id, nil
	}
}

func addMessageReadCommon(co CommonOperations, messageId, userId int64, chatId int64) (bool, error) {
	res, err := co.Exec(`INSERT INTO message_read (last_message_id, user_id, chat_id) VALUES ($1, $2, $3) ON CONFLICT (user_id, chat_id) DO UPDATE SET last_message_id = $1  WHERE $1 > (SELECT MAX(last_message_id) FROM message_read WHERE user_id = $2 AND chat_id = $3)`, messageId, userId, chatId)
	if err != nil {
		return false, eris.Wrap(err, "error during interacting with db")
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, eris.Wrap(err, "error during interacting with db")
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

func deleteMessageReadCommon(co CommonOperations, userId int64, chatId int64) error {
	_, err := co.Exec(`DELETE FROM message_read WHERE chat_id = $1 AND user_id = $2`, chatId, userId)
	return eris.Wrap(err, "error during interacting with db")
}

func (db *DB) DeleteMessageRead(userId int64, chatId int64) error {
	return deleteMessageReadCommon(db, userId, chatId)
}

func (tx *Tx) DeleteMessageRead(userId int64, chatId int64) error {
	return deleteMessageReadCommon(tx, userId, chatId)
}

func deleteAllMessageReadCommon(co CommonOperations, userId int64) error {
	_, err := co.Exec(`DELETE FROM message_read WHERE user_id = $1`, userId)
	return eris.Wrap(err, "error during interacting with db")
}

func (db *DB) DeleteAllMessageRead(userId int64) error {
	return deleteAllMessageReadCommon(db, userId)
}

func (tx *Tx) DeleteAllMessageRead(userId int64) error {
	return deleteAllMessageReadCommon(tx, userId)
}

func (tx *Tx) DeleteAllChatParticipantNotification(userId int64) error {
	_, err := tx.Exec(`DELETE FROM chat_participant_notification WHERE user_id = $1`, userId)
	return eris.Wrap(err, "error during interacting with db")
}

func (tx *Tx) EditMessage(m *Message) error {
	if m == nil {
		return eris.New("message required")
	} else if m.Text == "" {
		return eris.New("text required")
	} else if m.Id == 0 {
		return eris.New("id required")
	}

	embed, err := initEmbedMessageRequestStruct(m)
	if err != nil {
		return err
	}

	if res, err := tx.Exec(fmt.Sprintf(`UPDATE message_chat_%v SET text = $1, edit_date_time = utc_now(), file_item_uuid = $2, embed_message_id = $5, embed_chat_id = $6, embed_owner_id = $7, embed_message_type = $8, blog_post = $9 WHERE owner_id = $3 AND id = $4`, m.ChatId), m.Text, m.FileItemUuid, m.OwnerId, m.Id, embed.embedMessageId, embed.embedMessageChatId, embed.embedMessageOwnerId, embed.embedMessageType, m.BlogPost); err != nil {
		return eris.Wrap(err, "error during interacting with db")
	} else {
		affected, err := res.RowsAffected()
		if err != nil {
			return eris.Wrap(err, "error during interacting with db")
		}
		if affected == 0 {
			return eris.New("No rows affected")
		}
	}
	return nil
}

func (db *DB) DeleteMessage(messageId int64, ownerId int64, chatId int64) error {
	if res, err := db.Exec(fmt.Sprintf(`DELETE FROM message_chat_%v WHERE id = $1 AND owner_id = $2`, chatId), messageId, ownerId); err != nil {
		return eris.Wrap(err, "error during interacting with db")
	} else {
		affected, err := res.RowsAffected()
		if err != nil {
			return eris.Wrap(err, "error during interacting with db")
		}
		if affected == 0 {
			return eris.New("No rows affected")
		}
	}
	return nil
}

func (dbR *DB) SetFileItemUuidToNull(ownerId, chatId int64, fileItemUuid string) (int64, bool, error) {
	res := dbR.QueryRow(fmt.Sprintf(`UPDATE message_chat_%v SET file_item_uuid = NULL WHERE file_item_uuid = $1 AND owner_id = $2 RETURNING id`, chatId), fileItemUuid, ownerId)

	if res.Err() != nil {
		Logger.Errorf("Error during nulling file_item_uuid message id %v", res.Err())
		return 0, false, eris.Wrap(res.Err(), "error during interacting with db")
	}
	var messageId int64
	err := res.Scan(&messageId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// there were no rows, but otherwise no error occurred
			return 0, false, nil
		}
		return 0, false, eris.Wrap(err, "error during interacting with db")
	} else {
		return messageId, true, nil
	}
}

func (dbR *DB) SetFileItemUuidTo(ownerId, chatId, messageId int64, fileItemUuid *string) (error) {
	_, err := dbR.Exec(fmt.Sprintf(`UPDATE message_chat_%v SET file_item_uuid = $1 WHERE id = $2 AND owner_id = $3`, chatId), fileItemUuid, messageId, ownerId)

	if err != nil {
		Logger.Errorf("Error during nulling file_item_uuid message id %v", err)
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func getUnreadMessagesCountCommon(co CommonOperations, chatId int64, userId int64) (int64, error) {
	var count int64
	var unusedChatId int64
	row := co.QueryRow(getCountUnreadMessages(chatId, chatId, userId))
	err := row.Scan(&unusedChatId, &count)
	if err != nil {
		return 0, eris.Wrap(err, "error during interacting with db")
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

func getCountUnreadMessages(marker, chatId, userId int64) string {
	return fmt.Sprintf(`SELECT 
									%v, 
									CASE 
									WHEN (%v) THEN (SELECT COUNT(1) FROM message_chat_%v WHERE id > COALESCE((SELECT last_message_id FROM message_read WHERE user_id = %v AND chat_id = %v), 0))
									ELSE 0
									END		
	`, marker, getShouldConsiderMessagesAsUnread(chatId, userId), chatId, userId, chatId)
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
			builder += " UNION ALL "
		}
		builder += getCountUnreadMessages(chatId, chatId, userId)

		first = false
	}

	var rows *sql.Rows
	var err error
	rows, err = co.Query(builder)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		defer rows.Close()
		for _, cid := range chatIds {
			res[cid] = 0
		}
		for rows.Next() {
			var chatId int64
			var count int64
			if err := rows.Scan(&chatId, &count); err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
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

func getUnreadMessagesCountBatchByParticipantsCommon(co CommonOperations, userIds []int64, chatId int64) (map[int64]int64, error) {
	res := map[int64]int64{}

	if len(userIds) == 0 {
		return res, nil
	}

	var builder = ""
	var first = true
	for _, userId := range userIds {
		if !first {
			builder += " UNION ALL "
		}
		builder += getCountUnreadMessages(userId, chatId, userId)

		first = false
	}

	var rows *sql.Rows
	var err error
	rows, err = co.Query(builder)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		defer rows.Close()
		for _, uid := range userIds {
			res[uid] = 0
		}
		for rows.Next() {
			var userId int64
			var count int64
			if err := rows.Scan(&userId, &count); err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			} else {
				res[userId] = count
			}
		}
		return res, nil
	}
}

func (db *DB) GetUnreadMessagesCountBatchByParticipants(userIds []int64, chatId int64) (map[int64]int64, error) {
	return getUnreadMessagesCountBatchByParticipantsCommon(db, userIds, chatId)
}

func (tx *Tx) GetUnreadMessagesCountBatchByParticipants(userIds []int64, chatId int64) (map[int64]int64, error) {
	return getUnreadMessagesCountBatchByParticipantsCommon(tx, userIds, chatId)
}

func hasUnreadMessages(chatId, userId int64) string {
	return fmt.Sprintf(`SELECT 
									%v, 
									EXISTS (
										SELECT 1 
											FROM message_chat_%v 
											WHERE ( %v ) 
											AND id > COALESCE((SELECT last_message_id FROM message_read WHERE user_id = %v AND chat_id = %v), 0)
									) inn`, chatId, chatId, getShouldConsiderMessagesAsUnread(chatId, userId), userId, chatId,
	)
}

func hasUnreadMessagesBatchCommon(co CommonOperations, chatIds []int64, userId int64) (map[int64]bool, error) {
	res := map[int64]bool{}

	if len(chatIds) == 0 {
		return res, nil
	}

	var builder = ""
	var first = true
	for _, chatId := range chatIds {
		if !first {
			builder += " UNION ALL "
		}
		builder += hasUnreadMessages(chatId, userId)

		first = false
	}

	var rows *sql.Rows
	var err error
	rows, err = co.Query(builder)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		defer rows.Close()
		for _, cid := range chatIds {
			res[cid] = false
		}
		for rows.Next() {
			var chatId int64
			var exists bool
			if err := rows.Scan(&chatId, &exists); err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			} else {
				res[chatId] = exists
			}
		}
		return res, nil
	}
}

func hasUnreadMessagesCommon(co CommonOperations, userId int64) (bool, error) {
	shouldContinue := true
	for i := 0; shouldContinue; i++ {
		chatIds, err := getChatIdsByLimitOffsetCommon(co, userId, utils.DefaultSize, utils.DefaultSize*i)
		if err != nil {
			return false, err
		}
		if len(chatIds) < utils.DefaultSize {
			shouldContinue = false
		}
		messageUnreads, err := hasUnreadMessagesBatchCommon(co, chatIds, userId)
		if err != nil {
			return false, err
		}
		for _, hasMessageUnread := range messageUnreads {
			if hasMessageUnread {
				return true, nil
			}
		}
	}
	return false, nil
}

func (db *DB) HasUnreadMessages(userId int64) (bool, error) {
	return hasUnreadMessagesCommon(db, userId)
}

func (tx *Tx) HasUnreadMessages(userId int64) (bool, error) {
	return hasUnreadMessagesCommon(tx, userId)
}

func getShouldConsiderMessagesAsUnread(chatId, userId int64) string {
	return fmt.Sprintf(`SELECT COALESCE((SELECT consider_messages_as_unread FROM chat_participant_notification WHERE chat_id = %v AND user_id = %v), true)`, chatId, userId)
}

func (tx *Tx) ShouldSendHasUnreadMessagesCountBatchCommon(chatId int64, userIds []int64) (map[int64]bool, error) {
	res := map[int64]bool{}

	if len(userIds) == 0 {
		return res, nil
	}

	var builder = ""
	var first = true
	for _, userId := range userIds {
		if !first {
			builder += " UNION ALL "
		}
		builder += fmt.Sprintf(`SELECT %v, (%v)`, userId, getShouldConsiderMessagesAsUnread(chatId, userId))

		first = false
	}

	var rows *sql.Rows
	var err error
	rows, err = tx.Query(builder)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		defer rows.Close()
		for rows.Next() {
			var userId int64
			var should bool
			if err := rows.Scan(&userId, &should); err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			} else {
				res[userId] = should
			}
		}
		return res, nil
	}
}

func (tx *Tx) PublishMessage(chatId, messageId int64, shouldPublish bool) error {
	_, err := tx.Exec(fmt.Sprintf("UPDATE message_chat_%v SET published = $1 WHERE id = $2", chatId), shouldPublish, messageId)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func (tx *Tx) PinMessage(chatId, messageId int64, shouldPin bool) error {
	_, err := tx.Exec(fmt.Sprintf("UPDATE message_chat_%v SET pinned = $1 WHERE id = $2", chatId), shouldPin, messageId)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func (tx *Tx) GetPinnedMessages(chatId int64, limit, offset int) ([]*Message, error) {
	rows, err := tx.Query(fmt.Sprintf(`%v
			WHERE 
			    m.pinned IS TRUE
			ORDER BY m.pin_promoted DESC, m.id DESC
			LIMIT $1 OFFSET $2`, selectMessageClause(chatId)),
		limit, offset)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}

	defer rows.Close()
	list := make([]*Message, 0)
	for rows.Next() {
		message := Message{ChatId: chatId}
		if err := rows.Scan(provideScanToMessage(&message)[:]...); err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		} else {
			list = append(list, &message)
		}
	}
	return list, nil
}

func (tx *Tx) GetPublishedMessages(chatId int64, limit, offset int) ([]*Message, error) {
	rows, err := tx.Query(fmt.Sprintf(`%v
			WHERE 
			    m.published IS TRUE
			ORDER BY m.id DESC
			LIMIT $1 OFFSET $2`, selectMessageClause(chatId)),
		limit, offset)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}

	defer rows.Close()
	list := make([]*Message, 0)
	for rows.Next() {
		message := Message{ChatId: chatId}
		if err := rows.Scan(provideScanToMessage(&message)[:]...); err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		} else {
			list = append(list, &message)
		}
	}
	return list, nil
}

func (tx *Tx) GetPinnedMessagesCount(chatId int64) (int64, error) {
	row := tx.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM message_chat_%v WHERE pinned IS TRUE`, chatId))
	if row.Err() != nil {
		return 0, eris.Wrap(row.Err(), "error during interacting with db")
	}
	var res int64
	err := row.Scan(&res)
	if err != nil {
		return 0, eris.Wrap(err, "error during interacting with db")
	}
	return res, nil
}

func (tx *Tx) GetPublishedMessagesCount(chatId int64) (int64, error) {
	row := tx.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM message_chat_%v WHERE published IS TRUE`, chatId))
	if row.Err() != nil {
		return 0, eris.Wrap(row.Err(), "error during interacting with db")
	}
	var res int64
	err := row.Scan(&res)
	if err != nil {
		return 0, eris.Wrap(err, "error during interacting with db")
	}
	return res, nil
}

func (tx *Tx) UnpromoteMessages(chatId int64) error {
	_, err := tx.Exec(fmt.Sprintf(`UPDATE message_chat_%v SET pin_promoted = FALSE`, chatId))
	return eris.Wrap(err, "error during interacting with db")
}

func (tx *Tx) PromoteMessage(chatId, messageId int64) error {
	_, err := tx.Exec(fmt.Sprintf(`UPDATE message_chat_%v SET pin_promoted = TRUE WHERE id = $1`, chatId), messageId)
	return eris.Wrap(err, "error during interacting with db")
}

func (tx *Tx) PromotePreviousMessage(chatId int64) error {
	_, err := tx.Exec(fmt.Sprintf(`UPDATE message_chat_%v SET pin_promoted = TRUE WHERE id IN (SELECT id FROM message_chat_%v WHERE pinned IS TRUE ORDER BY id DESC LIMIT 1)`, chatId, chatId))
	return eris.Wrap(err, "error during interacting with db")
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
		return nil, eris.Wrap(row.Err(), "error during interacting with db")
	}

	message := Message{ChatId: chatId}
	err := row.Scan(provideScanToMessage(&message)[:]...)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil
	}
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
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
		return nil, eris.Wrap(err, "error during interacting with db")
	}

	defer rows.Close()
	list := make([]int64, 0)
	for rows.Next() {
		var anUserId int64
		if err := rows.Scan(&anUserId); err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
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
		return 0, eris.Wrap(row.Err(), "error during interacting with db")
	}

	var count int
	err := row.Scan(&count)
	if err != nil {
		return 0, eris.Wrap(err, "error during interacting with db")
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
		return 0, eris.Wrap(row.Err(), "error during interacting with db")
	}

	var messageId int64
	err := row.Scan(&messageId)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return MessageNotFoundId, nil
	}
	if err != nil {
		return 0, eris.Wrap(err, "error during interacting with db")
	} else {
		return messageId, nil
	}
}

func flipReactionCommon(co CommonOperations, userId int64, chatId int64, messageId int64, reaction string) (bool, error) {
	var wasAdded bool

	var exists bool
	row := co.QueryRow(fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM message_reaction_chat_%v WHERE user_id = $1 AND message_id = $2 AND reaction = $3)", chatId), userId, messageId, reaction)
	err := eris.Wrap(row.Scan(&exists), "error during interacting with db")
	if err != nil {
		return false, err
	}

	if exists {
		// if reaction exists - remove it
		_, err2 := co.Exec(fmt.Sprintf("DELETE FROM message_reaction_chat_%v WHERE user_id = $1 AND message_id = $2 AND reaction = $3", chatId), userId, messageId, reaction)
		err = eris.Wrap(err2, "error during interacting with db")
		if err != nil {
			return false, err
		}
	} else {
		// else insert reaction
		_, err2 := co.Exec(fmt.Sprintf("INSERT INTO message_reaction_chat_%v(user_id, message_id, reaction) VALUES ($1, $2, $3)", chatId), userId, messageId, reaction)
		err = eris.Wrap(err2, "error during interacting with db")
		if err != nil {
			return false, err
		}
		wasAdded = true
	}
	return wasAdded, nil
}

func (db *DB) FlipReaction(userId int64, chatId int64, messageId int64, reaction string) (bool, error) {
	return flipReactionCommon(db, userId, chatId, messageId, reaction)
}

func (tx *Tx) FlipReaction(userId int64, chatId int64, messageId int64, reaction string) (bool, error) {
	return flipReactionCommon(tx, userId, chatId, messageId, reaction)
}


func getReactionUsersCommon(co CommonOperations, chatId int64, messageId int64, reaction string) ([]int64, error) {
	rows, err := co.Query(fmt.Sprintf("SELECT user_id FROM message_reaction_chat_%v WHERE message_id = $1 AND reaction = $2", chatId), messageId, reaction)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}

	defer rows.Close()
	list := make([]int64, 0)
	for rows.Next() {
		var anUserId int64
		if err := rows.Scan(&anUserId); err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		} else {
			list = append(list, anUserId)
		}
	}
	return list, nil
}

func (db *DB) GetReactionUsers(chatId int64, messageId int64, reaction string) ([]int64, error) {
	return getReactionUsersCommon(db, chatId, messageId, reaction)
}

func (tx *Tx) GetReactionUsers(chatId int64, messageId int64, reaction string) ([]int64, error) {
	return getReactionUsersCommon(tx, chatId, messageId, reaction)
}

func getMessageReactionsCommon(co CommonOperations, chatId, messageId int64) ([]Reaction, error) {
	var reactions []Reaction = make([]Reaction, 0)

	rows, err := co.Query(fmt.Sprintf("SELECT user_id, message_id, reaction FROM message_reaction_chat_%v WHERE message_id = $1", chatId), messageId)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	defer rows.Close()

	for rows.Next() {
		reaction := Reaction{}
		if err := rows.Scan(&reaction.UserId, &reaction.MessageId, &reaction.Reaction); err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		}

		reactions = append(reactions, reaction)
	}
	return reactions, nil
}

func (db *DB) GetReactionsOnMessage(chatId, messageId int64) ([]Reaction, error) {
	return getMessageReactionsCommon(db, chatId, messageId)
}

func (tx *Tx) GetReactionsOnMessage(chatId, messageId int64) ([]Reaction, error) {
	return getMessageReactionsCommon(tx, chatId, messageId)
}

func enrichMessagesWithReactions(co CommonOperations, chatId int64, list []*Message) error {
	messageIds := make([]int64, 0)
	for _, message := range list {
		messageIds = append(messageIds, message.Id)
	}

	rows, err := co.Query(fmt.Sprintf("SELECT user_id, message_id, reaction FROM message_reaction_chat_%v WHERE message_id = ANY ($1)", chatId), messageIds)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	defer rows.Close()

	for rows.Next() { // iterate by reactions
		reaction := Reaction{}
		if err = rows.Scan(&reaction.UserId, &reaction.MessageId, &reaction.Reaction); err != nil {
			return eris.Wrap(err, "error during interacting with db")
		}

		for _, message := range list { // iterate by messages
			if message.Id == reaction.MessageId {
				message.Reactions = append(message.Reactions, reaction)
			}
		}
	}
	return nil
}

func (tx *Tx) MessageFilter(chatId int64, searchString string, messageId int64) (bool, error) {
	searchStringWithPercents := "%" + searchString + "%"
	row := tx.QueryRow(fmt.Sprintf("SELECT EXISTS (SELECT * FROM message_chat_%v m WHERE m.id = $1 AND strip_tags(m.text) ILIKE $2)", chatId), messageId, searchStringWithPercents)
	if row.Err() != nil {
		Logger.Errorf("Error during get Search %v", row.Err())
		return false, eris.Wrap(row.Err(), "error during interacting with db")
	}

	var found bool
	err := row.Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, eris.Wrap(err, "error during interacting with db")
	}
	return found, nil
}

func (tx *Tx) ChatFilter(searchString string, chatId int64) (bool, error) {
	searchStringWithPercents := "%" + searchString + "%"
	row := tx.QueryRow(fmt.Sprintf("SELECT EXISTS (SELECT * FROM chat ch WHERE ch.id = $1 AND strip_tags(ch.title) ILIKE $2)"), chatId, searchStringWithPercents)
	if row.Err() != nil {
		Logger.Errorf("Error during get Search %v", row.Err())
		return false, eris.Wrap(row.Err(), "error during interacting with db")
	}

	var found bool
	err := row.Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, eris.Wrap(err, "error during interacting with db")
	}
	return found, nil
}

func getCommentsCommon(co CommonOperations, chatId int64, blogPostId int64, limit int, offset int, reverse bool) ([]*Message, error) {
	order := "asc"
	if reverse {
		order = "desc"
	}
	var err error
	var rows *sql.Rows
	var preparedSql = fmt.Sprintf(`%v
			WHERE
				  m.id > $3 
			ORDER BY m.id %s 
			LIMIT $1 OFFSET $2`, selectMessageClause(chatId), order)
	rows, err = co.Query(preparedSql,
		limit, offset, blogPostId)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}

	defer rows.Close()
	list := make([]*Message, 0)
	for rows.Next() {
		message := Message{ChatId: chatId}
		if err := rows.Scan(provideScanToMessage(&message)[:]...); err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		} else {
			list = append(list, &message)
		}
	}

	err = enrichMessagesWithReactions(co, chatId, list)
	if err != nil {
		return nil, fmt.Errorf("Got error during enriching messages with reactions: %v", err)
	}

	return list, nil
}

func (db *DB) GetComments(chatId int64, blogPostId int64, limit int, offset int, reverse bool) ([]*Message, error) {
	return getCommentsCommon(db, chatId, blogPostId, limit, offset, reverse)
}

func (tx *Tx) GetComments(chatId int64, blogPostId int64, limit int, offset int, reverse bool) ([]*Message, error) {
	return getCommentsCommon(tx, chatId, blogPostId, limit, offset, reverse)
}

func countCommentsCommon(co CommonOperations, chatId int64, messageId int64) (int64, error) {
	res := co.QueryRow(fmt.Sprintf("SELECT count(*) FROM message_chat_%v m WHERE m.id > $1", chatId), messageId)
	var count int64
	if err := res.Scan(&count); err != nil {
		return 0, eris.Wrap(err, "error during interacting with db")
	}
	return count, nil
}

func (db *DB) CountComments(chatId int64, messageId int64) (int64, error) {
	return countCommentsCommon(db, chatId, messageId)
}

func (tx *Tx) CountComments(chatId int64, messageId int64) (int64, error) {
	return countCommentsCommon(tx, chatId, messageId)
}
