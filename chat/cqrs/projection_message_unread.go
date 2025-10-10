package cqrs

import (
	"context"
	"fmt"

	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/utils"

	"github.com/georgysavva/scany/v2/sqlscan"
)

func (m *CommonProjection) OnUserUnreadMessageReaded(ctx context.Context, event *UserMessageReaded, allChatsReadedConsumer func([]dto.ChatUserViewBasic)) error {
	if event.ReadMessagesAction == ReadMessagesActionOneMessage || event.ReadMessagesAction == ReadMessagesActionAllMessagesInOneChat {
		errOuter := db.Transact(ctx, m.db, func(tx *db.Tx) error {
			if event.ReadMessagesAction == ReadMessagesActionOneMessage {

				err := m.setUnreadMessages(ctx, tx, event.AdditionalData.BehalfUserId, event.ChatId, event.MessageId, SetUnreadedMessagesActionCalculateUnreadsFromTheProvidedMessage) // includes updateHasUnreads()
				if err != nil {
					return err
				}

				return nil
			} else if event.ReadMessagesAction == ReadMessagesActionAllMessagesInOneChat {

				err := m.fastForwardUnreadMessages(ctx, tx, event.AdditionalData.BehalfUserId, event.ChatId)
				if err != nil {
					return err
				}

				err = m.updateHasUnreads(ctx, tx, event.AdditionalData.BehalfUserId)
				if err != nil {
					return err
				}

				return nil
			} else {
				return fmt.Errorf("Unknown action: %T", event.ReadMessagesAction)
			}
		})
		if errOuter != nil {
			return fmt.Errorf("error during read messages: %w", errOuter)
		}
	} else if event.ReadMessagesAction == ReadMessagesActionAllChats {
		for {
			// deliberately don't use transaction in order not to span transaction over all the loop iterations
			updatedChatsPortion, err := m.setNoUnreadsInAllChats(ctx, m.db, event.AdditionalData.BehalfUserId, utils.DefaultSize)
			if err != nil {
				return err
			}

			allChatsReadedConsumer(updatedChatsPortion)

			// we cannot use offset-limit because we update what we return
			// so we iteratevily update it by portions until we have zero returned rows
			if len(updatedChatsPortion) == 0 {
				break
			}
		}

		err := m.setHasNoUnreadsInAllChats(ctx, m.db, event.AdditionalData.BehalfUserId)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Unknown action: %T", event.ReadMessagesAction)
	}
	return nil
}

type SetUnreadedMessagesAction int16

const (
	SetUnreadedMessagesActionUnspecified SetUnreadedMessagesAction = iota
	SetUnreadedMessagesActionInitialize
	SetUnreadedMessagesActionCalculateUnreadsFromTheUsersLastSavedReadedMessage
	SetUnreadedMessagesActionCalculateUnreadsFromTheProvidedMessage
)

func (m *CommonProjection) setUnreadMessages(ctx context.Context, co db.CommonOperations, participantId int64, chatId, messageId int64, setUnreadedMessagesAction SetUnreadedMessagesAction) error {
	queryArgs := []any{participantId, chatId}

	var inputOptionClause string

	switch setUnreadedMessagesAction {
	case SetUnreadedMessagesActionInitialize:
		inputOptionClause = `
		normalized_considerable_message as (
			select 
				n.user_id,
				0 as normalized_read_message_id
			from normalized_user n
		)
		`
	case SetUnreadedMessagesActionCalculateUnreadsFromTheProvidedMessage:
		queryArgs = append(queryArgs, messageId)
		// to calculate against just from the message
		inputOptionClause = `
		input_option_considerable_existing_message as (
			select coalesce(
				(select m.id from chat_messages m where m.id = $3),
				(select max from max_message),
				0
			) as normalized_read_message_id
		),
		normalized_considerable_message as (
			select 
				n.user_id,
				(select normalized_read_message_id from input_option_considerable_existing_message) as normalized_read_message_id
			from normalized_user n
		)
		`
	case SetUnreadedMessagesActionCalculateUnreadsFromTheUsersLastSavedReadedMessage:
		// to calculate from the last saved readed
		inputOptionClause = `
		input_option_considerable_last_saved_readed_message as (
			select 
				coalesce(ww.last_message_id, 0) as last_message_id,
				nu.user_id
			from (
				select
					w.cuv_last_read_message_id as last_message_id,
					w.user_id
				from chat_user_view w
				where w.id = $2 and w.user_id = $1
			) ww
			right join normalized_user nu on ww.user_id = nu.user_id
		),
		normalized_considerable_message as (
			select 
				n.user_id,
				(select l.last_message_id from input_option_considerable_last_saved_readed_message l where l.user_id = n.user_id) as normalized_read_message_id
			from normalized_user n
		)
		`
	default:
		return fmt.Errorf("Unknown action: %v", setUnreadedMessagesAction)
	}

	q := fmt.Sprintf(`
		with 
		chat_messages as (
			select m.id from message m where m.chat_id = $2
		),
		max_message as (
			select max(m.id) as max from chat_messages m
		),
		normalized_user as (
			select cast ($1 as bigint) as user_id
		),
		%s,
		input_data as (
			select
				ngm.user_id as user_id,
				cast ($2 as bigint) as chat_id,
				(
					SELECT count(m.id) FILTER(WHERE m.id > (select normalized_read_message_id from normalized_considerable_message n where n.user_id = ngm.user_id))
					FROM chat_messages m
				) as unread_messages,
				ngm.normalized_read_message_id as last_read_message_id
			from normalized_considerable_message ngm
		)
		merge into chat_user_view cuv
		using input_data idt
		on (idt.chat_id, idt.user_id) = (cuv.id, cuv.user_id)
		when matched then update set 
		   unread_messages = idt.unread_messages
		  ,cuv_last_read_message_id = idt.last_read_message_id
	`, inputOptionClause)

	_, err := co.ExecContext(ctx, q, queryArgs...)
	if err != nil {
		return err
	}

	err = m.updateHasUnreads(ctx, co, participantId)
	if err != nil {
		return err
	}

	return nil
}

// see also fastForwardChatParticipantMessageReadIdInAllChats()
func (m *CommonProjection) setNoUnreadsInAllChats(ctx context.Context, co db.CommonOperations, userId int64, size int) ([]dto.ChatUserViewBasic, error) {
	updatedChatsPortion := []dto.ChatUserViewBasic{}

	const noOffset = 0

	q := `
		with
		input_data as (
			select
				uv.id as chat_id
				,uv.user_id
				,coalesce(cc.last_message_id, 0) as last_message_id
			from chat_user_view uv
			join chat_common cc on uv.id = cc.id
			where uv.user_id = $1 
				-- optimization to not process all the chats and
				-- inn.unread_messages > 0 is required to always return pass pages to uv.id and, consequently, to return the full pages in returning
				and uv.unread_messages > 0 
			order by uv.id 
			limit $2 offset $3
		)
		update chat_user_view cuv 
		set (unread_messages, cuv_last_read_message_id) = (
			select 0, idt.last_message_id 
			from input_data idt
			where (idt.chat_id, idt.user_id) = (cuv.id, cuv.user_id)
		)
		where (cuv.id, cuv.user_id) in (select idtt.chat_id, idtt.user_id from input_data idtt) -- to avoid null. merge with return isn't supported
		returning cuv.id, cuv.unread_messages, cuv.update_date_time
	`

	err := sqlscan.Select(ctx, co, &updatedChatsPortion, q, userId, size, noOffset)
	if err != nil {
		return nil, err
	}

	return updatedChatsPortion, nil
}

func (m *CommonProjection) fastForwardUnreadMessages(ctx context.Context, co db.CommonOperations, userId, chatId int64) error {
	_, err := co.ExecContext(ctx, `
		UPDATE chat_user_view 
		SET unread_messages = 0, cuv_last_read_message_id = (select max(id) from message where chat_id = $2)
		WHERE (user_id, id) = ($1, $2);
	`, userId, chatId)
	return err
}

// should be called after upserting into unread_messages_user_view otherwise it's going to reset has to false
func (m *CommonProjection) updateHasUnreads(ctx context.Context, co db.CommonOperations, participantId int64) error {
	_, err := co.ExecContext(ctx, `
	with
	normalized_user as (
		select cast ($1 as bigint) as user_id
	),	
	users_hases as (
		select 
			uv.user_id, 
			(any_value(uv.unread_messages) filter (where uv.unread_messages > 0 and uv.consider_messages_as_unread)) != 0 as has 
		from chat_user_view uv
		where uv.user_id = $1
		group by (uv.user_id)
	),
	input_data as (
		select 
			nu.user_id,
			coalesce(uh.has, false) as has
		from normalized_user nu
		left join users_hases uh on nu.user_id = uh.user_id
	)
	insert into has_unread_messages(user_id, has)
	select user_id, has from input_data
	on conflict (user_id) do update
	set has = excluded.has
	`, participantId)
	return err
}

func (m *CommonProjection) setHasNoUnreadsInAllChats(ctx context.Context, co db.CommonOperations, userId int64) error {
	_, err := co.ExecContext(ctx, "update has_unread_messages set has = false where user_id = $1", userId)
	if err != nil {
		return err
	}
	return nil
}

func (m *CommonProjection) getLastMessageUnreadReaded(ctx context.Context, chatId, userId int64) (int64, bool, int64, error) {
	type lastMessageReaded struct {
		LastReadedMessageId int64 `db:"last_readed_message_id"`
		Has                 bool  `db:"has"`
		MaxMessageId        int64 `db:"max_message_id"`
	}

	res := lastMessageReaded{}

	err := sqlscan.Get(ctx, m.db, &res, `
	with
	chat_messages as (
		select m.id from message m where m.chat_id = $2
	),
	user_last_read_message as (
		select cuv_last_read_message_id as last_read_message_id from chat_user_view um 
		where (um.user_id, um.id) = ($1, $2)
	)
	select 
	    (select last_read_message_id from user_last_read_message) as last_readed_message_id, 
	    exists(select * from chat_messages m where m.id = cc.last_message_id) as has,
	    (select max(m.id) from chat_messages m) as max_message_id
	from chat_common cc 
    where cc.id = $2
	`, userId, chatId)
	if err != nil {
		return 0, false, 0, err
	}
	return res.LastReadedMessageId, res.Has, res.MaxMessageId, nil
}
