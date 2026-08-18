package cqrs

import (
	"context"

	"nkonev.name/chat/db"

	"github.com/georgysavva/scany/v2/sqlscan"
	"nkonev.name/chat/dto"
)

func CanCreateThread(chatCanCreateThread, cfgCanCreateThread, isParticipant, parentThreadIsRoot bool) bool {
	return isParticipant && chatCanCreateThread && cfgCanCreateThread && parentThreadIsRoot
}

// TODO тред может удалить только автор сообщения или админ (по настройке) - подумать
func CanDeleteThread(chatCanCreateThread, cfgCanCreateThread, isParticipant, parentThreadIsRoot bool) bool {
	return isParticipant && chatCanCreateThread && cfgCanCreateThread && parentThreadIsRoot
}

func (m *CommonProjection) GetThreadDataForAuthorization(ctx context.Context, co db.CommonOperations, userId, chatId, parentThreadId int64) (dto.ThreadAuthorizationData, error) {
	d := dto.ThreadAuthorizationData{}
	err := sqlscan.Get(ctx, co, &d, `
		with
		provided as (
			select 
				 cast($2 as bigint) as chat_id
		),
		chat_participant_row as (
			SELECT user_id, chat_admin FROM chat_participant WHERE user_id = $1 AND chat_id = $2 LIMIT 1
		),
		chat_info as (
			select * from chat where id = $2
		),
		parent_thread_info as (
			select * from thread where chat_id = $2 and id = $3
		),
		SELECT 
			cc.id is not null as is_chat_found
			,(SELECT exists(SELECT * FROM chat_participant_row) as is_chat_participant)
			,coalesce(cc.can_create_thread, false) as chat_can_create_thread
			-- right now we can create thread only first level nesting
			-- so here we check if parentThreadId is root thread
			,coalesce((pti.parent_thread_id = $4), false) as parent_thread_is_root 
		FROM provided pr
		LEFT JOIN chat_info cc on pr.chat_id = cc.id
		LEFT JOIN parent_thread_info pti on pr.chat_id = pti.chat_id
	`, userId, chatId, parentThreadId, dto.RootThreadId)
	if err != nil {
		return d, err
	}
	return d, nil
}

func (m *CommonProjection) FindRootThread(ctx context.Context, co db.CommonOperations, chatId int64) (*int64, error) {
	var res *int64

	err := sqlscan.Get(ctx, co, &res, "select id from thread where chat_id = $1 and parent_thread_id = $2", chatId, dto.RootThreadId)

	return res, err
}
