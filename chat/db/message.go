package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/guregu/null"
	"github.com/rotisserie/eris"
	"math"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/utils"
	"time"
)

const MessageNotFoundId = 0

type Reaction struct {
	MessageId int64
	UserId    int64
	Reaction  string
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
	Reactions   []Reaction
}

func selectMessageClause() string {
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
		FROM message m 
		LEFT JOIN message me 
			ON (m.chat_id = me.chat_id and m.embed_message_id = me.id AND m.embed_message_type = '%v')
		`, dto.EmbedMessageTypeReply)
}

func messageChatWhere(chatId int64) string {
	return fmt.Sprintf("m.chat_id = %v", chatId)
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

func selectMessageReactionsClause() string {
	return fmt.Sprintf("SELECT user_id, message_id, reaction FROM message_reaction ")
}

func messageReactionsChatWhere(chatId int64) string {
	return fmt.Sprintf("chat_id = %v", chatId)
}

// see also its copy in aaa::UserListViewRepository
func getMessagesCommon(ctx context.Context, co CommonOperations, chatId int64, limit int, startingFromItemId *int64, includeStartingFrom, reverse bool, searchString string) ([]*Message, error) {
	list := make([]*Message, 0)
	var err error

	// startingFromItemId is used as the top or the bottom limit of the portion
	list, err = getMessagesSimple(ctx, co, chatId, limit, startingFromItemId, includeStartingFrom, reverse, searchString)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}

	err = enrichMessagesWithReactions(ctx, co, chatId, list)
	if err != nil {
		return nil, fmt.Errorf("Got error during enriching messages with reactions: %v", err)
	}

	return list, nil
}

// implements keyset pagination
func getMessagesSimple(ctx context.Context, co CommonOperations, chatId int64, limit int, startingFromItemId0 *int64, includeStartingFrom, reverse bool, searchString string) ([]*Message, error) {
	list := make([]*Message, 0)

	// see also getSafeDefaultUserId() in aaa
	var startingFromItemIdVal int64
	if startingFromItemId0 == nil {
		if reverse {
			startingFromItemIdVal = math.MaxInt64
		} else {
			startingFromItemIdVal = 0
		}
	} else {
		startingFromItemIdVal = *startingFromItemId0
	}

	order := ""
	nonEquality := ""
	if reverse {
		order = "desc"
		s := ""
		if includeStartingFrom {
			s = "<="
		} else {
			s = "<"
		}
		nonEquality = fmt.Sprintf("m.id %v $2", s)
	} else {
		order = "asc"
		s := ""
		if includeStartingFrom {
			s = ">="
		} else {
			s = ">"
		}
		nonEquality = fmt.Sprintf("m.id %v $2", s)
	}
	var err error
	var rows *sql.Rows
	if searchString != "" {
		searchStringPercents := "%" + searchString + "%"
		rows, err = co.QueryContext(ctx, fmt.Sprintf(`%v
			WHERE 	%s AND
		    	    %s 
				AND strip_tags(m.text) ILIKE $3 
			ORDER BY m.id %s 
			LIMIT $1`, selectMessageClause(), messageChatWhere(chatId), nonEquality, order),
			limit, startingFromItemIdVal, searchStringPercents)
		if err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		}
		defer rows.Close()
	} else {
		rows, err = co.QueryContext(ctx, fmt.Sprintf(`%v
			WHERE %s AND
				  %s 
			ORDER BY m.id %s 
			LIMIT $1`, selectMessageClause(), messageChatWhere(chatId), nonEquality, order),
			limit, startingFromItemIdVal)
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

func (db *DB) GetMessages(ctx context.Context, chatId int64, limit int, startingFromItemId *int64, includeStartingFrom, reverse bool, searchString string) ([]*Message, error) {
	return getMessagesCommon(ctx, db, chatId, limit, startingFromItemId, includeStartingFrom, reverse, searchString)
}

func (tx *Tx) GetMessages(ctx context.Context, chatId int64, limit int, startingFromItemId *int64, includeStartingFrom, reverse bool, searchString string) ([]*Message, error) {
	return getMessagesCommon(ctx, tx, chatId, limit, startingFromItemId, includeStartingFrom, reverse, searchString)
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

func (tx *Tx) HasMessages(ctx context.Context, chatId int64) (bool, error) {
	var exists bool = false
	row := tx.QueryRowContext(ctx, fmt.Sprintf(`SELECT exists(SELECT * FROM message WHERE chat_id = %v LIMIT 1)`, chatId))
	if err := row.Scan(&exists); err != nil {
		return false, eris.Wrap(err, "error during interacting with db")
	} else {
		return exists, nil
	}
}

func (tx *Tx) CreateMessage(ctx context.Context, m *Message) (id int64, err error) {
	if m == nil {
		return id, eris.New("message required")
	} else if m.Text == "" {
		return id, eris.New("text required")
	}

	var messageId int64
	res := tx.QueryRowContext(ctx, "UPDATE chat SET last_generated_message_id = last_generated_message_id + 1 WHERE id = $1 RETURNING last_generated_message_id;", m.ChatId)
	if err := res.Scan(&messageId); err != nil {
		return id, eris.Wrap(err, "error during generating message id")
	}

	embed, err := initEmbedMessageRequestStruct(m)
	if err != nil {
		return id, eris.Wrap(err, "error during initializing embed struct")
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO message (id, chat_id, text, owner_id, file_item_uuid, embed_message_id, embed_chat_id, embed_owner_id, embed_message_type, blog_post) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		messageId, m.ChatId, m.Text, m.OwnerId, m.FileItemUuid, embed.embedMessageId, embed.embedMessageChatId, embed.embedMessageOwnerId, embed.embedMessageType, m.BlogPost)
	if err != nil {
		return id, eris.Wrap(err, "error during creating the message")
	}
	return messageId, nil
}

func (db *DB) CountMessages(ctx context.Context, chatId int64) (int64, error) {
	var count int64
	row := db.QueryRowContext(ctx, "SELECT count(*) FROM message WHERE chat_id = $1", chatId)
	err := row.Scan(&count)
	if err != nil {
		return 0, eris.Wrap(err, "error during interacting with db")
	} else {
		return count, nil
	}
}
func getMessageCommon(ctx context.Context, co CommonOperations, chatId int64, userId int64, messageId int64) (*Message, error) {
	q := fmt.Sprintf(`%v
		WHERE %s AND m.id = $1
		AND $3 in (SELECT chat_id FROM chat_participant WHERE user_id = $2 AND chat_id = $3)
		LIMIT 1`,
		selectMessageClause(), messageChatWhere(chatId))

	row := co.QueryRowContext(ctx, q, messageId, userId, chatId)

	message := Message{ChatId: chatId, Reactions: make([]Reaction, 0)}
	err := row.Scan(provideScanToMessage(&message)[:]...)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil
	}
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}

	reactions, err := getMessageReactionsCommon(ctx, co, chatId, messageId)
	if err != nil {
		return nil, err
	}
	message.Reactions = reactions

	return &message, nil
}

func (db *DB) GetMessage(ctx context.Context, chatId int64, userId int64, messageId int64) (*Message, error) {
	return getMessageCommon(ctx, db, chatId, userId, messageId)
}

func (tx *Tx) GetMessage(ctx context.Context, chatId int64, userId int64, messageId int64) (*Message, error) {
	return getMessageCommon(ctx, tx, chatId, userId, messageId)
}

func getMessagePublicCommon(ctx context.Context, co CommonOperations, chatId int64, messageId int64) (*Message, error) {
	row := co.QueryRowContext(ctx, fmt.Sprintf(`%v
	WHERE 
		%s AND
	    m.id = $1 
		AND m.published = true`, selectMessageClause(), messageChatWhere(chatId)),
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

	reactions, err := getMessageReactionsCommon(ctx, co, chatId, messageId)
	if err != nil {
		return nil, err
	}
	message.Reactions = reactions

	return &message, nil
}

func (db *DB) GetMessagePublic(ctx context.Context, chatId int64, messageId int64) (*Message, error) {
	return getMessagePublicCommon(ctx, db, chatId, messageId)
}

func (tx *Tx) GetMessagePublic(ctx context.Context, chatId int64, messageId int64) (*Message, error) {
	return getMessagePublicCommon(ctx, tx, chatId, messageId)
}

func (tx *Tx) SetBlogPost(ctx context.Context, chatId int64, messageId int64, desiredValue bool) error {
	_, err := tx.ExecContext(ctx, fmt.Sprintf("UPDATE message SET blog_post = $2 WHERE chat_id = %v AND id = $1", chatId), messageId, desiredValue)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func getMessageBasicCommon(ctx context.Context, co CommonOperations, chatId int64, messageId int64) (*MessageBasic, error) {
	row := co.QueryRowContext(ctx, fmt.Sprintf(`SELECT 
    	m.text,
    	m.owner_id,
    	m.blog_post,
    	m.published,
    	m.file_item_uuid
	FROM message m 
	WHERE 
	    m.chat_id = %v AND
	    m.id = $1 
`, chatId),
		messageId)
	var mb = MessageBasic{}
	err := row.Scan(&mb.Text, &mb.OwnerId, &mb.BlogPost, &mb.Published, &mb.FileItemUuid)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil
	}
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		return &mb, nil
	}
}

type MessageBasic struct {
	Text         string
	OwnerId      int64
	BlogPost     bool
	Published    bool
	FileItemUuid *string
}

func (tx *Tx) GetMessageBasic(ctx context.Context, chatId int64, messageId int64) (*MessageBasic, error) {
	return getMessageBasicCommon(ctx, tx, chatId, messageId)
}

func (db *DB) GetMessageBasic(ctx context.Context, chatId int64, messageId int64) (*MessageBasic, error) {
	return getMessageBasicCommon(ctx, db, chatId, messageId)
}

func (tx *Tx) GetBlogPostMessageId(ctx context.Context, chatId int64) (*int64, error) {
	row := tx.QueryRowContext(ctx, fmt.Sprintf(`
							SELECT 
								m.id 
							FROM message m 
							WHERE 
							    m.chat_id = %v AND
								m.blog_post IS TRUE
							ORDER BY id LIMIT 1
						`, chatId),
	)
	var id *int64
	err := row.Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil
	}
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		return id, nil
	}
}

func (tx *Tx) MarkMessageAsRead(ctx context.Context, chatId int64, participantId int64, messageId *int64) error {
	r := tx.QueryRowContext(ctx, fmt.Sprintf(`SELECT COALESCE((SELECT max(id) from message WHERE chat_id = %v), 0)`, chatId))
	if r.Err() != nil {
		return eris.Wrap(r.Err(), "error during interacting with db")
	}
	var lastMessageId int64
	err := r.Scan(&lastMessageId)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}

	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO message_read (last_message_id, user_id, chat_id)
			SELECT %v, $1, $2
		ON CONFLICT (user_id, chat_id) DO UPDATE SET last_message_id = (
			CASE 
				WHEN ($3::bigint <= %v) THEN (
					CASE 
						WHEN ($3::bigint > message_read.last_message_id) THEN $3::bigint
						ELSE message_read.last_message_id
					END
				)
				ELSE %v
			END
		) 
			WHERE message_read.user_id = $1 AND message_read.chat_id = $2
		`, lastMessageId, lastMessageId, lastMessageId),
		participantId, chatId, messageId)
	return err
}

func deleteMessageReadCommon(ctx context.Context, co CommonOperations, userId int64, chatId int64) error {
	_, err := co.ExecContext(ctx, `DELETE FROM message_read WHERE chat_id = $1 AND user_id = $2`, chatId, userId)
	return eris.Wrap(err, "error during interacting with db")
}

func (db *DB) DeleteMessageRead(ctx context.Context, userId int64, chatId int64) error {
	return deleteMessageReadCommon(ctx, db, userId, chatId)
}

func (tx *Tx) DeleteMessageRead(ctx context.Context, userId int64, chatId int64) error {
	return deleteMessageReadCommon(ctx, tx, userId, chatId)
}

func deleteAllMessageReadCommon(ctx context.Context, co CommonOperations, userId int64) error {
	_, err := co.ExecContext(ctx, `DELETE FROM message_read WHERE user_id = $1`, userId)
	return eris.Wrap(err, "error during interacting with db")
}

func (db *DB) DeleteAllMessageRead(ctx context.Context, userId int64) error {
	return deleteAllMessageReadCommon(ctx, db, userId)
}

func (tx *Tx) DeleteAllMessageRead(ctx context.Context, userId int64) error {
	return deleteAllMessageReadCommon(ctx, tx, userId)
}

func (tx *Tx) DeleteAllChatParticipantNotification(ctx context.Context, userId int64) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM chat_participant_notification WHERE user_id = $1`, userId)
	return eris.Wrap(err, "error during interacting with db")
}

func (tx *Tx) EditMessage(ctx context.Context, m *Message) error {
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

	r := tx.QueryRowContext(ctx, `SELECT utc_now()`)
	if r.Err() != nil {
		return eris.Wrap(r.Err(), "error during interacting with db")
	}
	var dt time.Time
	err = r.Scan(&dt)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}

	if res, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE message SET text = $1, edit_date_time = $10, file_item_uuid = $2, embed_message_id = $5, embed_chat_id = $6, embed_owner_id = $7, embed_message_type = $8, blog_post = $9 WHERE chat_id = %v AND owner_id = $3 AND id = $4`, m.ChatId), m.Text, m.FileItemUuid, m.OwnerId, m.Id, embed.embedMessageId, embed.embedMessageChatId, embed.embedMessageOwnerId, embed.embedMessageType, m.BlogPost, dt); err != nil {
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

func deleteMessageCommon(ctx context.Context, co CommonOperations, messageId int64, ownerId int64, chatId int64) error {
	if res, err := co.ExecContext(ctx, fmt.Sprintf(`DELETE FROM message WHERE chat_id = %v AND id = $1 AND owner_id = $2`, chatId), messageId, ownerId); err != nil {
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

func (db *DB) DeleteMessage(ctx context.Context, messageId int64, ownerId int64, chatId int64) error {
	return deleteMessageCommon(ctx, db, messageId, ownerId, chatId)
}

func (tx *Tx) DeleteMessage(ctx context.Context, messageId int64, ownerId int64, chatId int64) error {
	return deleteMessageCommon(ctx, tx, messageId, ownerId, chatId)
}

func (dbR *DB) SetFileItemUuidToNull(ctx context.Context, ownerId, chatId int64, fileItemUuid string) (int64, bool, error) {
	res := dbR.QueryRowContext(ctx, fmt.Sprintf(`UPDATE message SET file_item_uuid = NULL WHERE chat_id = %v AND file_item_uuid = $1 AND owner_id = $2 RETURNING id`, chatId), fileItemUuid, ownerId)

	if res.Err() != nil {
		dbR.lgr.WithTracing(ctx).Errorf("Error during nulling file_item_uuid message id %v", res.Err())
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

func (dbR *DB) SetFileItemUuidTo(ctx context.Context, ownerId, chatId, messageId int64, fileItemUuid *string) error {
	_, err := dbR.ExecContext(ctx, fmt.Sprintf(`UPDATE message SET file_item_uuid = $1 WHERE chat_id = %v AND id = $2 AND owner_id = $3`, chatId), fileItemUuid, messageId, ownerId)

	if err != nil {
		dbR.lgr.WithTracing(ctx).Errorf("Error during nulling file_item_uuid message id %v", err)
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func getUnreadMessagesCountCommon(ctx context.Context, co CommonOperations, chatId int64, userId int64) (int64, error) {
	aMap, err := getUnreadMessagesCountByChatsBatchCommon(ctx, co, []int64{chatId}, userId)
	if err != nil {
		return 0, err
	}
	count, ok := aMap[chatId]
	if !ok {
		return 0, errors.New("something wrong with getting from map by chat id")
	}
	return count, nil
}

func (db *DB) GetUnreadMessagesCount(ctx context.Context, chatId int64, userId int64) (int64, error) {
	return getUnreadMessagesCountCommon(ctx, db, chatId, userId)
}

func (tx *Tx) GetUnreadMessagesCount(ctx context.Context, chatId int64, userId int64) (int64, error) {
	return getUnreadMessagesCountCommon(ctx, tx, chatId, userId)
}

func getNotificationSettingsByChatsBatch(ctx context.Context, co CommonOperations, chatIds []int64, userId int64) (map[int64]bool, error) {
	res := map[int64]bool{}

	if len(chatIds) == 0 {
		return res, nil
	}

	for _, chatId := range chatIds {
		res[chatId] = true // default value
	}

	// true is a default value
	rows, err := co.QueryContext(ctx, "select chat_id, coalesce(consider_messages_as_unread, true) from chat_participant_notification where user_id = $1 and chat_id = any($2)", userId, chatIds)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	defer rows.Close()

	for rows.Next() {
		var chatId int64
		var consider bool
		if err := rows.Scan(&chatId, &consider); err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		}
		res[chatId] = consider
	}
	return res, nil
}

func getNotificationSettingsByUsersBatch(ctx context.Context, co CommonOperations, chatId int64, userIds []int64) (map[int64]bool, error) {
	res := map[int64]bool{}

	if len(userIds) == 0 {
		return res, nil
	}

	for _, userId := range userIds {
		res[userId] = true // default value
	}

	// true is a default value
	rows, err := co.QueryContext(ctx, "select user_id, coalesce(consider_messages_as_unread, true) from chat_participant_notification where chat_id = $1 and user_id = any($2)", chatId, userIds)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	defer rows.Close()

	for rows.Next() {
		var userId int64
		var consider bool
		if err := rows.Scan(&userId, &consider); err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		}
		res[userId] = consider
	}
	return res, nil
}

func getUnreadMessagesCountByChatsBatchCommon(ctx context.Context, co CommonOperations, chatIds []int64, userId int64) (map[int64]int64, error) {
	res := map[int64]int64{}

	if len(chatIds) == 0 {
		return res, nil
	}

	for _, cid := range chatIds {
		res[cid] = 0
	}

	chatAllowCounting, err := getNotificationSettingsByChatsBatch(ctx, co, chatIds, userId)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}

	allowedChatIds := make([]int64, 0)
	for cid, a := range chatAllowCounting {
		if a {
			allowedChatIds = append(allowedChatIds, cid)
		}
	}

	// reads cte in order to process a cornercase when for an user there is still no record in message_read
	// we query chat_participant because of
	// 1) there can be no rows in message_read, but we need to get user_id or chat_id with coalesce(last_message_id, 0)
	// 2) in order to filter out only rows which correspond to the real user_id or chat_id, e.g. to filter with caution in case adjacent user_id or chat_id
	var q = `
		with reads as (
		  select chat_id, coalesce(last_message_id, 0) as normalized_last_message_id from 
		  (select chat_id from chat_participant where user_id = $2 and chat_id = ANY($1)) all_chats
		  left join (
		    select chat_id as read_chat_id, last_message_id from message_read WHERE user_id = $2 AND chat_id = any($1) order by user_id
		  ) rds on all_chats.chat_id = rds.read_chat_id
		)
		select m.chat_id, count(m.id) FILTER(WHERE m.id > reads.normalized_last_message_id) as new_messages_for_user 
		from message m
		join reads
		on m.chat_id = reads.chat_id
		where m.chat_id = ANY($1) 
		group by m.chat_id 
		order by m.chat_id;
	`

	var rows *sql.Rows
	rows, err = co.QueryContext(ctx, q, allowedChatIds, userId)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	defer rows.Close()
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

func (db *DB) GetUnreadMessagesCountByChatsBatch(ctx context.Context, chatIds []int64, userId int64) (map[int64]int64, error) {
	return getUnreadMessagesCountByChatsBatchCommon(ctx, db, chatIds, userId)
}

func (tx *Tx) GetUnreadMessagesCountByChatsBatch(ctx context.Context, chatIds []int64, userId int64) (map[int64]int64, error) {
	return getUnreadMessagesCountByChatsBatchCommon(ctx, tx, chatIds, userId)
}

func getUnreadMessagesCountBatchByParticipantsCommon(ctx context.Context, co CommonOperations, userIds []int64, chatId int64) (map[int64]int64, error) {
	res := map[int64]int64{}

	if len(userIds) == 0 {
		return res, nil
	}

	for _, uid := range userIds {
		res[uid] = 0
	}

	userAllowCounting, err := getNotificationSettingsByUsersBatch(ctx, co, chatId, userIds)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	allowedUserIds := make([]int64, 0)
	for uid, a := range userAllowCounting {
		if a {
			allowedUserIds = append(allowedUserIds, uid)
		}
	}

	// reads cte in order to process a cornercase when for an user there is still no record in message_read
	// we query chat_participant because of
	// 1) there can be no rows in message_read, but we need to get user_id or chat_id with coalesce(last_message_id, 0)
	// 2) in order to filter out only rows which correspond to the real user_id or chat_id, e.g. to filter with caution in case adjacent user_id or chat_id
	var q = `
		with reads as (
		  select user_id, $1 as normalized_chat_id, coalesce(last_message_id, 0) as normalized_last_message_id from
		  (select user_id from chat_participant where chat_id = $1 and user_id = ANY($2)) all_users
		  left join (
		   select user_id as read_user_id, chat_id, last_message_id from message_read WHERE user_id = ANY($2) AND chat_id = $1 order by user_id
		  ) rds on all_users.user_id = rds.read_user_id
		)
		SELECT user_id, (select count(m.id) FILTER(WHERE m.id > reads.normalized_last_message_id)) 
		FROM message m 
		join reads on m.chat_id = reads.normalized_chat_id
		where m.chat_id = $1
		group by user_id;
	`

	var rows *sql.Rows
	rows, err = co.QueryContext(ctx, q, chatId, allowedUserIds)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	defer rows.Close()
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

func (db *DB) GetUnreadMessagesCountBatchByParticipants(ctx context.Context, userIds []int64, chatId int64) (map[int64]int64, error) {
	return getUnreadMessagesCountBatchByParticipantsCommon(ctx, db, userIds, chatId)
}

func (tx *Tx) GetUnreadMessagesCountBatchByParticipants(ctx context.Context, userIds []int64, chatId int64) (map[int64]int64, error) {
	return getUnreadMessagesCountBatchByParticipantsCommon(ctx, tx, userIds, chatId)
}

func hasUnreadMessagesBatchByChatIdsCommon(ctx context.Context, co CommonOperations, chatIds []int64, userId int64) (map[int64]bool, error) {
	res := map[int64]bool{}

	if len(chatIds) == 0 {
		return res, nil
	}

	for _, cid := range chatIds {
		res[cid] = false
	}

	chatAllowCounting, err := getNotificationSettingsByChatsBatch(ctx, co, chatIds, userId)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}

	allowedChatIds := make([]int64, 0)
	for cid, a := range chatAllowCounting {
		if a {
			allowedChatIds = append(allowedChatIds, cid)
		}
	}

	// reads cte in order to process a cornercase when for an user there is still no record in message_read
	// we query chat_participant because of
	// 1) there can be no rows in message_read, but we need to get user_id or chat_id with coalesce(last_message_id, 0)
	// 2) in order to filter out only rows which correspond to the real user_id or chat_id, e.g. to filter with caution in case adjacent user_id or chat_id
	var q = `
		with reads as (
		  select chat_id, coalesce(last_message_id, 0) as normalized_last_message_id from 
		  (select chat_id from chat_participant where user_id = $2 and chat_id = ANY($1)) all_chats
		  left join (
		    select chat_id as read_chat_id, last_message_id from message_read WHERE user_id = $2 AND chat_id = any($1) order by user_id
		  ) rds on all_chats.chat_id = rds.read_chat_id
		)
		select maxes.chat_id, max_message_id > normalized_last_message_id as has_new_message_for_user from
		(select chat_id, max(id) as max_message_id from message where chat_id = ANY($1) group by chat_id order by chat_id) maxes 
		join reads
		on maxes.chat_id = reads.chat_id;
	`

	var rows *sql.Rows
	rows, err = co.QueryContext(ctx, q, allowedChatIds, userId)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	defer rows.Close()
	for rows.Next() {
		var chatId int64
		var has bool
		if err := rows.Scan(&chatId, &has); err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		} else {
			res[chatId] = has
		}
	}
	return res, nil
}

func (db *DB) HasUnreadMessagesByChatIdsBatch(ctx context.Context, chatIds []int64, userId int64) (map[int64]bool, error) {
	return hasUnreadMessagesBatchByChatIdsCommon(ctx, db, chatIds, userId)
}

func (tx *Tx) HasUnreadMessagesByChatIdsBatch(ctx context.Context, chatIds []int64, userId int64) (map[int64]bool, error) {
	return hasUnreadMessagesBatchByChatIdsCommon(ctx, tx, chatIds, userId)
}

func hasUnreadMessagesCommon(ctx context.Context, co CommonOperations, userId int64) (bool, error) {
	shouldContinue := true

	const size = utils.PaginationMaxSize

	for i := 0; shouldContinue; i++ {
		chatIds, err := getChatIdsByParticipantIdCommon(ctx, co, userId, size, size*i)
		if err != nil {
			return false, err
		}
		if len(chatIds) < size {
			shouldContinue = false
		}
		messageUnreads, err := hasUnreadMessagesBatchByChatIdsCommon(ctx, co, chatIds, userId)
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

func (db *DB) HasUnreadMessages(ctx context.Context, userId int64) (bool, error) {
	return hasUnreadMessagesCommon(ctx, db, userId)
}

func (tx *Tx) HasUnreadMessages(ctx context.Context, userId int64) (bool, error) {
	return hasUnreadMessagesCommon(ctx, tx, userId)
}

func (tx *Tx) ShouldSendHasUnreadMessagesCountBatch(ctx context.Context, chatId int64, userIds []int64) (map[int64]bool, error) {

	userAllowCounting, err := getNotificationSettingsByUsersBatch(ctx, tx, chatId, userIds)
	if err != nil {
		return nil, err
	}

	return userAllowCounting, err
}

func (tx *Tx) PublishMessage(ctx context.Context, chatId, messageId int64, shouldPublish bool) error {
	_, err := tx.ExecContext(ctx, fmt.Sprintf("UPDATE message SET published = $1 WHERE chat_id = %v AND id = $2", chatId), shouldPublish, messageId)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func (tx *Tx) PinMessage(ctx context.Context, chatId, messageId int64, shouldPin bool) error {
	_, err := tx.ExecContext(ctx, fmt.Sprintf("UPDATE message SET pinned = $1 WHERE chat_id = %v AND id = $2", chatId), shouldPin, messageId)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func (tx *Tx) GetPinnedMessages(ctx context.Context, chatId int64, limit, offset int) ([]*Message, error) {
	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`%v
			WHERE %s AND
			    m.pinned IS TRUE
			ORDER BY m.pin_promoted DESC, m.id DESC
			LIMIT $1 OFFSET $2`, selectMessageClause(), messageChatWhere(chatId)),
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

func (tx *Tx) GetPublishedMessages(ctx context.Context, chatId int64, limit, offset int) ([]*Message, error) {
	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`%v
			WHERE %s AND
			    m.published IS TRUE
			ORDER BY m.id DESC
			LIMIT $1 OFFSET $2`, selectMessageClause(), messageChatWhere(chatId)),
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

func commonGetPinnedMessagesCount(ctx context.Context, co CommonOperations, chatId int64) (int64, error) {
	row := co.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM message WHERE chat_id = %v AND pinned IS TRUE`, chatId))
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

func (tx *Tx) GetPinnedMessagesCount(ctx context.Context, chatId int64) (int64, error) {
	return commonGetPinnedMessagesCount(ctx, tx, chatId)
}

func (db *DB) GetPinnedMessagesCount(ctx context.Context, chatId int64) (int64, error) {
	return commonGetPinnedMessagesCount(ctx, db, chatId)
}

func commonGetPublishedMessagesCount(ctx context.Context, co CommonOperations, chatId int64) (int64, error) {
	row := co.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM message WHERE chat_id = %v AND published IS TRUE`, chatId))
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

func (tx *Tx) GetPublishedMessagesCount(ctx context.Context, chatId int64) (int64, error) {
	return commonGetPublishedMessagesCount(ctx, tx, chatId)
}

func (db *DB) GetPublishedMessagesCount(ctx context.Context, chatId int64) (int64, error) {
	return commonGetPublishedMessagesCount(ctx, db, chatId)
}

func (tx *Tx) UnpromoteMessages(ctx context.Context, chatId int64) error {
	_, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE message SET pin_promoted = FALSE WHERE chat_id = %v`, chatId))
	return eris.Wrap(err, "error during interacting with db")
}

func (tx *Tx) PromoteMessage(ctx context.Context, chatId, messageId int64) error {
	_, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE message SET pin_promoted = TRUE WHERE chat_id = %v AND id = $1`, chatId), messageId)
	return eris.Wrap(err, "error during interacting with db")
}

func (tx *Tx) PromotePreviousMessage(ctx context.Context, chatId int64) error {
	_, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE message SET pin_promoted = TRUE WHERE chat_id = %v AND id IN (SELECT id FROM message WHERE chat_id = %v AND pinned IS TRUE ORDER BY id DESC LIMIT 1)`, chatId, chatId))
	return eris.Wrap(err, "error during interacting with db")
}

func (tx *Tx) GetPinnedPromoted(ctx context.Context, chatId int64) (*Message, error) {
	row := tx.QueryRowContext(ctx, fmt.Sprintf(`%v
			WHERE %s AND
			    m.pinned IS TRUE AND m.pin_promoted IS TRUE
			ORDER BY m.id desc
			LIMIT 1`, selectMessageClause(), messageChatWhere(chatId)),
	)
	if row.Err() != nil {
		tx.lgr.WithTracing(ctx).Errorf("Error during get pinned messages %v", row.Err())
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

func (db *DB) GetParticipantsRead(ctx context.Context, chatId, messageId int64, limit, offset int) ([]int64, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf(`
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

func (db *DB) GetParticipantsReadCount(ctx context.Context, chatId, messageId int64) (int, error) {
	row := db.QueryRowContext(ctx, fmt.Sprintf(`
			select count(user_id) from message_read where chat_id = $1 and last_message_id >= $2
			`,
	),
		chatId, messageId)
	if row.Err() != nil {
		db.lgr.WithTracing(ctx).Errorf("Error during get count of participants read the message %v", row.Err())
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

func (db *DB) FindMessageByFileItemUuid(ctx context.Context, chatId int64, fileItemUuid string) (int64, error) {
	if len(fileItemUuid) == 0 {
		return MessageNotFoundId, nil
	}
	fileItemUuidWithPercents := "%" + fileItemUuid + "%"
	sqlFormatted := fmt.Sprintf(`
			select id from message where chat_id = %v AND file_item_uuid = $1 or text ilike $2 order by id limit 1
			`, chatId,
	)
	row := db.QueryRowContext(ctx, sqlFormatted, fileItemUuid, fileItemUuidWithPercents)
	if row.Err() != nil {
		db.lgr.WithTracing(ctx).Errorf("Error during get MessageByFileItemUuid %v", row.Err())
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

func flipReactionCommon(ctx context.Context, co CommonOperations, userId int64, chatId int64, messageId int64, reaction string) (bool, error) {
	var wasAdded bool

	var exists bool
	row := co.QueryRowContext(ctx, fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM message_reaction WHERE chat_id = %v AND user_id = $1 AND message_id = $2 AND reaction = $3)", chatId), userId, messageId, reaction)
	err := eris.Wrap(row.Scan(&exists), "error during interacting with db")
	if err != nil {
		return false, err
	}

	if exists {
		// if reaction exists - remove it
		_, err2 := co.ExecContext(ctx, fmt.Sprintf("DELETE FROM message_reaction WHERE chat_id = %v AND user_id = $1 AND message_id = $2 AND reaction = $3", chatId), userId, messageId, reaction)
		err = eris.Wrap(err2, "error during interacting with db")
		if err != nil {
			return false, err
		}
	} else {
		// else insert reaction
		_, err2 := co.ExecContext(ctx, fmt.Sprintf("INSERT INTO message_reaction(user_id, message_id, reaction, chat_id) VALUES ($1, $2, $3, %v)", chatId), userId, messageId, reaction)
		err = eris.Wrap(err2, "error during interacting with db")
		if err != nil {
			return false, err
		}
		wasAdded = true
	}
	return wasAdded, nil
}

func (db *DB) FlipReaction(ctx context.Context, userId int64, chatId int64, messageId int64, reaction string) (bool, error) {
	return flipReactionCommon(ctx, db, userId, chatId, messageId, reaction)
}

func (tx *Tx) FlipReaction(ctx context.Context, userId int64, chatId int64, messageId int64, reaction string) (bool, error) {
	return flipReactionCommon(ctx, tx, userId, chatId, messageId, reaction)
}

func getReactionUsersCommon(ctx context.Context, co CommonOperations, chatId int64, messageId int64, reaction string) ([]int64, error) {
	rows, err := co.QueryContext(ctx, fmt.Sprintf("SELECT user_id FROM message_reaction WHERE chat_id = %v AND message_id = $1 AND reaction = $2", chatId), messageId, reaction)
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

func (db *DB) GetReactionUsers(ctx context.Context, chatId int64, messageId int64, reaction string) ([]int64, error) {
	return getReactionUsersCommon(ctx, db, chatId, messageId, reaction)
}

func (tx *Tx) GetReactionUsers(ctx context.Context, chatId int64, messageId int64, reaction string) ([]int64, error) {
	return getReactionUsersCommon(ctx, tx, chatId, messageId, reaction)
}

func getMessageReactionsCommon(ctx context.Context, co CommonOperations, chatId, messageId int64) ([]Reaction, error) {
	var reactions []Reaction = make([]Reaction, 0)

	rows, err := co.QueryContext(ctx, fmt.Sprintf("%s WHERE %s AND message_id = $1", selectMessageReactionsClause(), messageReactionsChatWhere(chatId)), messageId)
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

func (db *DB) GetReactionsOnMessage(ctx context.Context, chatId, messageId int64) ([]Reaction, error) {
	return getMessageReactionsCommon(ctx, db, chatId, messageId)
}

func (tx *Tx) GetReactionsOnMessage(ctx context.Context, chatId, messageId int64) ([]Reaction, error) {
	return getMessageReactionsCommon(ctx, tx, chatId, messageId)
}

func enrichMessagesWithReactions(ctx context.Context, co CommonOperations, chatId int64, list []*Message) error {
	messageIds := make([]int64, 0)
	for _, message := range list {
		messageIds = append(messageIds, message.Id)
	}

	rows, err := co.QueryContext(ctx, fmt.Sprintf("%s WHERE %s AND message_id = ANY ($1)", selectMessageReactionsClause(), messageReactionsChatWhere(chatId)), messageIds)
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

func (tx *Tx) MessageFilter(ctx context.Context, chatId int64, searchString string, messageId int64) (bool, error) {
	searchStringWithPercents := "%" + searchString + "%"
	row := tx.QueryRowContext(ctx, fmt.Sprintf("SELECT EXISTS (SELECT * FROM message m WHERE m.chat_id = %v AND m.id = $1 AND strip_tags(m.text) ILIKE $2)", chatId), messageId, searchStringWithPercents)
	if row.Err() != nil {
		tx.lgr.WithTracing(ctx).Errorf("Error during get Search %v", row.Err())
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

func getCommentsCommon(ctx context.Context, co CommonOperations, chatId int64, blogPostId int64, limit int, offset int, reverse bool) ([]*Message, error) {
	order := "asc"
	if reverse {
		order = "desc"
	}
	var err error
	var rows *sql.Rows
	var preparedSql = fmt.Sprintf(`%v
			WHERE %s AND
				  m.id > $3 
			ORDER BY m.id %s 
			LIMIT $1 OFFSET $2`, selectMessageClause(), messageChatWhere(chatId), order)
	rows, err = co.QueryContext(ctx, preparedSql,
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

	err = enrichMessagesWithReactions(ctx, co, chatId, list)
	if err != nil {
		return nil, fmt.Errorf("Got error during enriching messages with reactions: %v", err)
	}

	return list, nil
}

func (db *DB) GetComments(ctx context.Context, chatId int64, blogPostId int64, limit int, offset int, reverse bool) ([]*Message, error) {
	return getCommentsCommon(ctx, db, chatId, blogPostId, limit, offset, reverse)
}

func (tx *Tx) GetComments(ctx context.Context, chatId int64, blogPostId int64, limit int, offset int, reverse bool) ([]*Message, error) {
	return getCommentsCommon(ctx, tx, chatId, blogPostId, limit, offset, reverse)
}

func countCommentsCommon(ctx context.Context, co CommonOperations, chatId int64, messageId int64) (int64, error) {
	res := co.QueryRowContext(ctx, fmt.Sprintf("SELECT count(*) FROM message m WHERE m.chat_id = %v AND m.id > $1", chatId), messageId)
	var count int64
	if err := res.Scan(&count); err != nil {
		return 0, eris.Wrap(err, "error during interacting with db")
	}
	return count, nil
}

func (db *DB) CountComments(ctx context.Context, chatId int64, messageId int64) (int64, error) {
	return countCommentsCommon(ctx, db, chatId, messageId)
}

func (tx *Tx) CountComments(ctx context.Context, chatId int64, messageId int64) (int64, error) {
	return countCommentsCommon(ctx, tx, chatId, messageId)
}
