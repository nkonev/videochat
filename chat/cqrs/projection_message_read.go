package cqrs

import (
	"context"
	"fmt"
	"time"

	"nkonev.name/chat/db"
	"nkonev.name/chat/utils"

	"github.com/georgysavva/scany/v2/sqlscan"
)

func (m *CommonProjection) OnChatUnreadMessageReaded(ctx context.Context, event *MessageReaded) error {
	if event.ReadMessagesAction == ReadMessagesActionOneMessage || event.ReadMessagesAction == ReadMessagesActionAllMessagesInOneChat {
		errOuter := db.Transact(ctx, m.db, func(tx *db.Tx) error {
			if event.ReadMessagesAction == ReadMessagesActionOneMessage {

				err := m.updateParticipantMessageReadIdBatch(ctx, tx, event.ChatId, []MessageOwner{{
					MessageId: event.MessageId,
					OwnerId:   event.AdditionalData.BehalfUserId,
					Time:      event.AdditionalData.CreatedAt,
				}})
				if err != nil {
					return err
				}

				return nil
			} else if event.ReadMessagesAction == ReadMessagesActionAllMessagesInOneChat {

				err := m.fastForwardParticipantMessageReadId(ctx, tx, event.AdditionalData.BehalfUserId, event.ChatId, event.AdditionalData.CreatedAt)
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
		offset := int64(0)
		for {

			updatedParticipantsPortion, err := m.fastForwardChatParticipantMessageReadIdInAllChats(ctx, m.db, event.AdditionalData.BehalfUserId, utils.DefaultSize, offset, event.AdditionalData.CreatedAt)
			if err != nil {
				return err
			}

			if len(updatedParticipantsPortion) < utils.DefaultSize {
				break
			}
			offset += utils.DefaultSize
		}

	} else {
		return fmt.Errorf("Unknown action: %T", event.ReadMessagesAction)
	}
	return nil
}

func (m *CommonProjection) fastForwardParticipantMessageReadId(ctx context.Context, co db.CommonOperations, userId, chatId int64, lastReadMessageDateTime time.Time) error {
	_, err := co.ExecContext(ctx, `
		with 
		max_message as (
			select coalesce((select max(id) from message where chat_id = $2), 0) as max
		)
		UPDATE chat_participant 
		SET 
		    cp_last_read_message_id = (select max from max_message)
			,cp_last_read_message_date_time = $3
		WHERE 
			(user_id, chat_id) = ($1, $2)
	`, userId, chatId, lastReadMessageDateTime)
	return err
}

// see also OnUserMessagesCreated()
func (m *CommonProjection) updateParticipantMessageReadIdBatch(ctx context.Context, co db.CommonOperations, chatId int64, messageEvents []MessageOwner) error {
	// implied that messageEvents are sorted in their natural order
	maxMessageByUser := map[int64]MessageOwner{}
	for _, me := range messageEvents {
		maxMessageByUser[me.OwnerId] = me
	}

	var messageIds = []int64{}
	var userIds = []int64{}
	var messageTimes = []time.Time{}
	for userId, ev := range maxMessageByUser {
		userIds = append(userIds, userId)
		messageIds = append(messageIds, ev.MessageId)
		messageTimes = append(messageTimes, ev.Time)
	}

	_, err := co.ExecContext(ctx, `
		with
		max_message as (
			select max(id) as max from message where chat_id = $4
		),
		max_message_normalized as (
			select coalesce((select max from max_message), 0) as max
		),
		participants_data as (
			select 
				user_id
				,chat_id
				,cp_last_read_message_id
			from chat_participant 
			where chat_id = $4 and user_id = any($1)
		),
		owner_message as (
			select * from unnest(
				 cast($1 as bigint[])
				,cast($2 as bigint[])
				,cast($3 as timestamp[])
			) as t (
				owner_id
				,message_id
				,created_at
			)
		),
		input_data as (
			select 
				cp.user_id
				,cp.chat_id
				,om.message_id
				,om.created_at
			from participants_data cp
			join owner_message om on (cp.user_id = om.owner_id)
			cross join max_message_normalized mmn
			where om.message_id > cp.cp_last_read_message_id
			and om.message_id <= mmn.max
		)
		merge into chat_participant cpa
		using input_data idt
		on (idt.chat_id, idt.user_id) = (cpa.chat_id, cpa.user_id)
		when matched then update set 
		    cp_last_read_message_id = idt.message_id
			,cp_last_read_message_date_time = idt.created_at
	`, userIds, messageIds, messageTimes, chatId)
	return err
}

// see also setNoUnreadsInAllChats()
func (m *CommonProjection) fastForwardChatParticipantMessageReadIdInAllChats(ctx context.Context, co db.CommonOperations, userId int64, size int, offset int64, lastReadMessageDateTime time.Time) ([]int64, error) {
	// here with limit and offset
	resChatIds := []int64{}
	q := `
		with
		input_data as (
			select
				uv.chat_id
				,uv.user_id
				,coalesce(cc.last_message_id, 0) as last_message_id
			from chat_participant uv
			join chat_common cc on uv.chat_id = cc.id
			left join (
				select max(id) as max_message_id, chat_id from message group by chat_id
			) mm on mm.chat_id = uv.chat_id
			where uv.user_id = $1 
				-- optimization to not process all the chats, "max(id) as max_message_id" is a part of the optimization
				and (
					mm.max_message_id is null -- corner - all the messages were removed
					or coalesce(uv.cp_last_read_message_id, 0) < mm.max_message_id
				)
			order by uv.chat_id 
			limit $2 offset $3
		)
		update chat_participant cpa 
		set 
			cp_last_read_message_id = (
				select idt.last_message_id 
				from input_data idt
				where (idt.chat_id, idt.user_id) = (cpa.chat_id, cpa.user_id)
			)
			,cp_last_read_message_date_time = $4
		where (cpa.chat_id, cpa.user_id) in (select idtt.chat_id, idtt.user_id from input_data idtt)
		returning cpa.chat_id
	`

	err := sqlscan.Select(ctx, co, &resChatIds, q, userId, size, offset, lastReadMessageDateTime)
	if err != nil {
		return nil, err
	}

	return resChatIds, nil
}

func (m *EnrichingProjection) getParticipantsReadCount(ctx context.Context, co db.CommonOperations, chatId, messageId int64) (int64, error) {
	var count int64

	err := sqlscan.Get(ctx, co, &count, `
		select 
		    count(user_id) 
		from chat_participant 
		where chat_id = $1 and cp_last_read_message_id >= $2`,
		chatId, messageId)

	return count, err
}

func (m *EnrichingProjection) getParticipantsRead(ctx context.Context, co db.CommonOperations, chatId, messageId int64, limit int32, offset int64) ([]int64, error) {
	list := make([]int64, 0)

	err := sqlscan.Select(ctx, co, &list, `
		select 
			user_id 
		from chat_participant 
		where chat_id = $1 and cp_last_read_message_id >= $2
		ORDER BY cp_last_read_message_date_time desc
		LIMIT $3 OFFSET $4`,
		chatId, messageId, limit, offset)

	if err != nil {
		return nil, err
	}
	return list, nil
}
