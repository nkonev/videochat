package cqrs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/preview"
	"nkonev.name/chat/utils"

	"github.com/georgysavva/scany/v2/sqlscan"
)

func (m *CommonProjection) isMessagePinned(ctx context.Context, co db.CommonOperations, chatId, messageId int64) (bool, error) {
	var isMessagePinned bool
	err := sqlscan.Get(ctx, co, &isMessagePinned, "select exists (select * from message_pinned where chat_id = $1 and message_id = $2)", chatId, messageId)
	if err != nil {
		return false, err
	}

	return isMessagePinned, nil
}

func (m *CommonProjection) isMessagePromoted(ctx context.Context, co db.CommonOperations, chatId, messageId int64) (bool, error) {

	var isMessagePromoted bool
	err := sqlscan.Get(ctx, co, &isMessagePromoted, "select exists (select * from message_pinned where chat_id = $1 and message_id = $2 and promoted = true)", chatId, messageId)
	if err != nil {
		return false, err
	}

	return isMessagePromoted, nil
}

func (m *CommonProjection) setMessagePinned(ctx context.Context, tx *db.Tx, chatId, messageId int64, pinned bool) error {
	_, err := tx.ExecContext(ctx, `update message set pinned = $3 where chat_id = $1 and id = $2`, chatId, messageId, pinned)
	if err != nil {
		return err
	}
	return nil
}

func (m *EnrichingProjection) GetPinnedPromotedMessage(ctx context.Context, chatId, behalfUserId int64) (*dto.PinnedMessageDto, bool, error) {
	type resDto struct {
		promoted        *dto.PinnedMessageDto
		notAParticipant bool
	}

	res, errOuter := db.TransactWithResult(ctx, m.cp.db, func(tx *db.Tx) (*resDto, error) {
		participant, err := m.cp.IsParticipant(ctx, tx, behalfUserId, chatId)
		if err != nil {
			return nil, err
		}

		if !participant {
			return &resDto{
				notAParticipant: true,
			}, nil
		}

		type promotedDto struct {
			MessageId int64 `db:"message_id"`
			OwnerId   int64 `db:"owner_id"`
		}

		var promoted promotedDto
		var promotedP *promotedDto
		err = sqlscan.Get(ctx, tx, &promoted, "select message_id, owner_id from message_pinned where chat_id = $1 and promoted = true order by create_date_time desc limit 1", chatId)
		if errors.Is(err, sql.ErrNoRows) {
			// there were no rows, but otherwise no error occurred
		} else if err != nil {
			return nil, err
		} else {
			// ok
			promotedP = &promoted
		}

		var pr *dto.PinnedMessageDto
		if promotedP != nil {
			users, err := m.aaaRestClient.GetUsers(ctx, []int64{promotedP.OwnerId})
			if err != nil {
				m.lgr.WarnContext(ctx, "unable to get users", logger.AttributeError, err)
			}

			usersMap := utils.ToMap(users)

			enricheds, err := m.GetPinnedMessageEnriched(ctx, tx, chatId, promotedP.MessageId, []int64{behalfUserId}, usersMap)
			if err != nil {
				return nil, err
			}

			pr = enricheds[behalfUserId]
		}

		return &resDto{
			promoted: pr,
		}, nil
	})
	if errOuter != nil {
		return nil, false, errOuter
	}

	return res.promoted, res.notAParticipant, nil
}

func (m *EnrichingProjection) GetPinnedMessagesEnriched(ctx context.Context, chatId, userId, offset int64, size int32) ([]dto.PinnedMessageDto, int64, error) {
	type txRes struct {
		list  []dto.PinnedMessageDto
		count int64
	}
	res, errOuter := db.TransactWithResult(ctx, m.cp.db, func(tx *db.Tx) (*txRes, error) {
		rs := txRes{
			list:  []dto.PinnedMessageDto{},
			count: 0,
		}

		participant, err := m.cp.IsParticipant(ctx, tx, userId, chatId)
		if err != nil {
			return nil, err
		}
		if !participant {
			return nil, NewUnauthorizedError(fmt.Sprintf("user %v is not a participant of chat %v", userId, chatId))
		}

		pinnedMessages, err := m.cp.GetPinnedMessages(ctx, tx, chatId, offset, size)
		if err != nil {
			return nil, err
		}

		cb, err := m.cp.GetChatBasic(ctx, tx, chatId)
		if err != nil {
			return nil, err
		}

		if cb == nil {
			m.lgr.InfoContext(ctx, "chat is not found", logger.AttributeChatId, chatId)
			return &rs, nil
		}

		areAdmins, err := m.cp.getAreAdminsOfUserIds(ctx, tx, []int64{userId}, chatId)
		if err != nil {
			return nil, err
		}

		messageOwners := map[int64]struct{}{}
		for _, msg := range pinnedMessages {
			messageOwners[msg.OwnerId] = struct{}{}
		}

		messageOwnerUsers, err := m.aaaRestClient.GetUsers(ctx, utils.SetMapIdStructToSlice(messageOwners))
		if err != nil {
			m.lgr.WarnContext(ctx, "unable to get users", logger.AttributeError, err)
		}

		messageOwnerUsersMap := utils.ToMap(messageOwnerUsers)

		for _, pm := range pinnedMessages {
			pinnedEnriched := m.enrichMessagePinned(ctx, &pm, cb.RegularParticipantCanPinMessage, areAdmins[userId], messageOwnerUsersMap)
			rs.list = append(rs.list, *pinnedEnriched)
		}

		cnt, err := m.cp.GetPinnedMessageCount(ctx, tx, chatId)
		if err != nil {
			return nil, err
		}

		rs.count = cnt

		return &rs, nil
	})

	if errOuter != nil {
		return nil, 0, errOuter
	}

	return res.list, res.count, nil
}

func (m *EnrichingProjection) GetPinnedMessageEnriched(ctx context.Context, co db.CommonOperations, chatId, messageId int64, behalfUserIds []int64, messageOwnerUsersMap map[int64]*dto.User) (map[int64]*dto.PinnedMessageDto, error) {
	pinned, err := m.cp.GetPinnedMessage(ctx, co, chatId, messageId)
	if err != nil {
		return nil, err
	}
	if pinned != nil {
		areAdmins, err := m.cp.getAreAdminsOfUserIds(ctx, co, behalfUserIds, chatId)
		if err != nil {
			return nil, err
		}

		cb, err := m.cp.GetChatBasic(ctx, co, chatId)
		if err != nil {
			return nil, err
		}

		resMap := map[int64]*dto.PinnedMessageDto{}

		if cb != nil {
			for _, participantId := range behalfUserIds {
				pinnedEnriched := m.enrichMessagePinned(ctx, pinned, cb.RegularParticipantCanPinMessage, areAdmins[participantId], messageOwnerUsersMap)
				resMap[participantId] = pinnedEnriched
			}
		} else {
			m.lgr.ErrorContext(ctx, "Chat isn't found", logger.AttributeChatId, chatId)
		}

		return resMap, nil
	} else {
		return nil, nil
	}
}

const pinnedMessageCols = `
		message_id
		,chat_id
		,owner_id
		,create_date_time
		,preview
		,promoted
`

func (m *CommonProjection) GetPinnedMessage(ctx context.Context, co db.CommonOperations, chatId, messageId int64) (*dto.PinnedMessage, error) {
	var pm dto.PinnedMessage
	err := sqlscan.Get(ctx, co, &pm, fmt.Sprintf(`
	select 
		%s
	from message_pinned 
	where chat_id = $1 and message_id = $2
	`, pinnedMessageCols), chatId, messageId)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
	} else if err != nil {
		return nil, err
	}

	return &pm, nil
}

func (m *CommonProjection) GetPinnedMessages(ctx context.Context, co db.CommonOperations, chatId int64, offset int64, size int32) ([]dto.PinnedMessage, error) {
	var pm = []dto.PinnedMessage{}

	err := sqlscan.Select(ctx, co, &pm, fmt.Sprintf(`
	select 
		%s
	from message_pinned 
	where chat_id = $1 
	order by promoted desc, create_date_time desc
	limit $2 offset $3
	`, pinnedMessageCols),
		chatId, size, offset)
	if err != nil {
		return nil, err
	}

	return pm, nil
}

func (m *CommonProjection) GetPinnedMessageCount(ctx context.Context, co db.CommonOperations, chatId int64) (int64, error) {
	var count int64
	err := sqlscan.Get(ctx, co, &count, "select count (*) from message_pinned where chat_id = $1", chatId)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (m *CommonProjection) createMessagePinnedText(content string) string {
	return preview.CreateMessagePreviewWithoutLogin(m.stripTags, m.cfg.Message.PreviewMaxTextSize, m.stripTags.Sanitize(content))
}

func (m *CommonProjection) tryNominatePreviousToPromote(ctx context.Context, co db.CommonOperations, chatId int64) (*int64, error) {

	var previousPinned *int64
	err := sqlscan.Get(ctx, co, &previousPinned, "select message_id from message_pinned where chat_id = $1 order by create_date_time desc limit 1", chatId)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
	} else if err != nil {
		return nil, err
	}

	if previousPinned != nil {
		_, err := co.ExecContext(ctx, "update message_pinned set promoted = true where chat_id = $1 and message_id = $2", chatId, *previousPinned)
		if err != nil {
			return nil, err
		}
	}

	return previousPinned, nil
}

func (m *CommonProjection) OnMessagePinned(ctx context.Context, event *MessagePinned) (*int64, int64, error) {
	type resDto struct {
		count             int64
		promotedMessageId *int64
	}
	res, errOuter := db.TransactWithResult(ctx, m.db, func(tx *db.Tx) (*resDto, error) {
		var pinnedCount int64
		var promotedMessageId *int64
		if event.Pinned {
			mb, err := m.GetMessageBasic(ctx, tx, event.ChatId, event.MessageId)
			if err != nil {
				return nil, err
			}

			if mb != nil {
				previewTxt := m.createMessagePinnedText(mb.Content)

				_, err = tx.ExecContext(ctx, `
					insert into message_pinned (chat_id, message_id, owner_id, create_date_time, update_date_time, preview, promoted)
					values ($1, $2, $3, $4, $5, $6, true)
					on conflict (chat_id, message_id) do update set
					preview = excluded.preview
					,promoted = excluded.promoted
					,update_date_time = excluded.update_date_time
				`,
					event.ChatId, event.MessageId, mb.OwnerId, event.AdditionalData.CreatedAt, event.AdditionalData.CreatedAt, previewTxt)
				if err != nil {
					return nil, err
				}

				// set pinned
				err = m.setMessagePinned(ctx, tx, event.ChatId, event.MessageId, true)
				if err != nil {
					return nil, err
				}

				// unpromote previous
				_, err = tx.ExecContext(ctx, `update message_pinned set promoted = false where chat_id = $1 and message_id != $2`, event.ChatId, event.MessageId)
				if err != nil {
					return nil, err
				}

				promotedMessageId = &event.MessageId
			} else {
				m.lgr.InfoContext(ctx, "Skipping pinning the mesage because it is not exists", logger.AttributeChatId, event.ChatId, logger.AttributeMessageId, event.MessageId)
			}
		} else {
			// unpin
			isPromoted, err := m.isMessagePromoted(ctx, tx, event.ChatId, event.MessageId)
			if err != nil {
				return nil, err
			}

			_, err = tx.ExecContext(ctx, "delete from message_pinned where chat_id = $1 and message_id = $2", event.ChatId, event.MessageId)
			if err != nil {
				return nil, err
			}

			// set pinned
			err = m.setMessagePinned(ctx, tx, event.ChatId, event.MessageId, false)
			if err != nil {
				return nil, err
			}

			if isPromoted {
				promotedMessageId, err = m.tryNominatePreviousToPromote(ctx, tx, event.ChatId)
				if err != nil {
					return nil, err
				}
			}
		}

		var errc error
		pinnedCount, errc = m.GetPinnedMessageCount(ctx, tx, event.ChatId)
		if errc != nil {
			return nil, errc
		}

		return &resDto{
			count:             pinnedCount,
			promotedMessageId: promotedMessageId,
		}, nil
	})
	if errOuter != nil {
		return nil, 0, errOuter
	}
	return res.promotedMessageId, res.count, nil
}

func (m *EnrichingProjection) enrichMessagePinned(ctx context.Context, pinnedMessage *dto.PinnedMessage, chatRegularParticipantCanPinMessage bool, chatIsAdmin bool, messageOwnerUsersMap map[int64]*dto.User) *dto.PinnedMessageDto {
	owner := messageOwnerUsersMap[pinnedMessage.OwnerId]
	if owner == nil {
		owner = getDeletedUser(pinnedMessage.OwnerId)
	}
	res := dto.PinnedMessageDto{
		Id:             pinnedMessage.Id,
		Text:           pinnedMessage.Text,
		ChatId:         pinnedMessage.ChatId,
		OwnerId:        pinnedMessage.OwnerId,
		Owner:          owner,
		PinnedPromoted: pinnedMessage.Promoted,
		CreateDateTime: pinnedMessage.CreateDateTime,
		CanPin:         CanPinMessage(chatRegularParticipantCanPinMessage, chatIsAdmin),
	}

	return &res
}
