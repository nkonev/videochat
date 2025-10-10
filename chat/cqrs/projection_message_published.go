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

func (m *CommonProjection) isMessagePublished(ctx context.Context, co db.CommonOperations, chatId, messageId int64) (bool, error) {
	var isMessagePublished bool
	err := sqlscan.Get(ctx, co, &isMessagePublished, "select exists (select * from message_published where chat_id = $1 and message_id = $2)", chatId, messageId)
	if err != nil {
		return false, err
	}

	return isMessagePublished, nil
}

func (m *CommonProjection) setMessagePublished(ctx context.Context, tx *db.Tx, chatId, messageId int64, published bool) error {
	_, err := tx.ExecContext(ctx, `update message set published = $3 where chat_id = $1 and id = $2`, chatId, messageId, published)
	if err != nil {
		return err
	}
	return nil
}

func (m *EnrichingProjection) GetPublishedMessagesEnriched(ctx context.Context, chatId, userId, offset int64, size int32) ([]dto.PublishedMessageDto, int64, error) {
	type txRes struct {
		list  []dto.PublishedMessageDto
		count int64
	}
	res, errOuter := db.TransactWithResult(ctx, m.cp.db, func(tx *db.Tx) (*txRes, error) {
		rs := txRes{
			list:  []dto.PublishedMessageDto{},
			count: 0,
		}

		participant, err := m.cp.IsParticipant(ctx, tx, userId, chatId)
		if err != nil {
			return nil, err
		}
		if !participant {
			return nil, NewUnauthorizedError(fmt.Sprintf("user %v is not a participant of chat %v", userId, chatId))
		}

		publishedMessages, err := m.cp.GetPublishedMessages(ctx, tx, chatId, offset, size)
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
		for _, msg := range publishedMessages {
			messageOwners[msg.OwnerId] = struct{}{}
		}

		messageOwnerUsers, err := m.aaaRestClient.GetUsers(ctx, utils.SetMapIdStructToSlice(messageOwners))
		if err != nil {
			m.lgr.WarnContext(ctx, "unable to get users", logger.AttributeError, err)
		}

		messageOwnerUsersMap := utils.ToMap(messageOwnerUsers)

		for _, pm := range publishedMessages {
			publishedEnriched := m.enrichMessagePublished(ctx, &pm, cb.RegularParticipantCanPublishMessage, areAdmins[userId], messageOwnerUsersMap, userId)
			rs.list = append(rs.list, *publishedEnriched)
		}

		cnt, err := m.cp.GetPublishedMessageCount(ctx, tx, chatId)
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

func (m *EnrichingProjection) GetPublishedMessageEnriched(ctx context.Context, co db.CommonOperations, chatId, messageId int64, behalfUserIds []int64, messageOwnerUsersMap map[int64]*dto.User) (map[int64]*dto.PublishedMessageDto, error) {
	published, err := m.cp.GetPublishedMessage(ctx, co, chatId, messageId)
	if err != nil {
		return nil, err
	}
	if published != nil {
		areAdmins, err := m.cp.getAreAdminsOfUserIds(ctx, co, behalfUserIds, chatId)
		if err != nil {
			return nil, err
		}

		cb, err := m.cp.GetChatBasic(ctx, co, chatId)
		if err != nil {
			return nil, err
		}

		resMap := map[int64]*dto.PublishedMessageDto{}

		if cb != nil {
			for _, participantId := range behalfUserIds {
				publishedEnriched := m.enrichMessagePublished(ctx, published, cb.RegularParticipantCanPublishMessage, areAdmins[participantId], messageOwnerUsersMap, participantId)
				resMap[participantId] = publishedEnriched
			}
		} else {
			m.lgr.ErrorContext(ctx, "Chat isn't found", logger.AttributeChatId, chatId)
		}

		return resMap, nil
	} else {
		return nil, nil
	}
}

func (m *EnrichingProjection) GetPublishedMessageForPublic(ctx context.Context, chatId, messageId int64) (*dto.PublishedMessageWrapper, bool, error) {
	cb, err := m.cp.GetChatBasic(ctx, m.cp.db, chatId)
	if err != nil {
		return nil, false, err
	}
	if cb == nil {
		m.lgr.InfoContext(ctx, "Public message isn't found due to no chat", logger.AttributeChatId, chatId, logger.AttributeMessageId, messageId)
		return nil, true, nil
	}

	tetATetParticipantIds := []int64{}
	if cb.TetATet {
		tetATetParticipantIds, err = m.cp.GetParticipantIds(ctx, m.cp.db, chatId, 2, 0)
		if err != nil {
			return nil, false, err
		}
	}

	msgs, _, users, err := m.GetMessagesEnriched(ctx, []int64{}, false, true, nil, chatId, 1, nil, true, false, dto.NoSearchString, []int64{messageId}, tetATetParticipantIds)
	if err != nil {
		return nil, false, err
	}
	if len(msgs) == 0 {
		m.lgr.InfoContext(ctx, "Public message isn't found due to no message", logger.AttributeChatId, chatId, logger.AttributeMessageId, messageId)
		return nil, true, nil
	}
	if len(msgs) > 1 {
		return nil, false, errors.New("Wrong invariant - more than 1 messsage was returned")
	}
	msg := msgs[0]

	msg.Content = PatchStorageUrlToPublic(ctx, m.lgr, m.cfg, msg.Content, chatId, msg.Id)

	userMap := utils.ToMap(users)

	previewTxt := preview.CreateMessagePreviewWithoutLogin(m.stripAllTags, m.cfg.Message.PreviewMaxTextSize, msg.Content)

	aTitle := cb.Title
	if cb.TetATet {
		first := true
		aTitle = ""
		for _, userId := range tetATetParticipantIds {
			if !first {
				aTitle += ", "
			}
			usr := userMap[userId]
			if usr != nil {
				aTitle += usr.Login
			}
			first = false
		}
	}

	return &dto.PublishedMessageWrapper{
		Message: &msg,
		Title:   aTitle,
		Preview: previewTxt,
	}, false, nil
}

const publishedMessageCols = `
		message_id
		,chat_id
		,owner_id
		,create_date_time
		,preview
`

func (m *CommonProjection) GetPublishedMessage(ctx context.Context, co db.CommonOperations, chatId, messageId int64) (*dto.PublishedMessage, error) {
	var pm dto.PublishedMessage
	err := sqlscan.Get(ctx, co, &pm, fmt.Sprintf(`
	select 
		%s
	from message_published 
	where chat_id = $1 and message_id = $2
	`, publishedMessageCols), chatId, messageId)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
	} else if err != nil {
		return nil, err
	}

	return &pm, nil
}

func (m *CommonProjection) GetPublishedMessages(ctx context.Context, co db.CommonOperations, chatId int64, offset int64, size int32) ([]dto.PublishedMessage, error) {
	var pm = []dto.PublishedMessage{}

	err := sqlscan.Select(ctx, co, &pm, fmt.Sprintf(`
	select 
		%s
	from message_published 
	where chat_id = $1 
	order by create_date_time desc
	limit $2 offset $3
	`, publishedMessageCols),
		chatId, size, offset)
	if err != nil {
		return nil, err
	}

	return pm, nil
}

func (m *CommonProjection) GetPublishedMessageCount(ctx context.Context, co db.CommonOperations, chatId int64) (int64, error) {
	var count int64
	err := sqlscan.Get(ctx, co, &count, "select count (*) from message_published where chat_id = $1", chatId)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (m *CommonProjection) createMessagePublishedText(content string) string {
	return preview.CreateMessagePreviewWithoutLogin(m.stripTags, m.cfg.Message.PreviewMaxTextSize, m.stripTags.Sanitize(content))
}

func (m *CommonProjection) OnMessagePublished(ctx context.Context, event *MessagePublished) (int64, error) {
	type resDto struct {
		count int64
	}
	res, errOuter := db.TransactWithResult(ctx, m.db, func(tx *db.Tx) (*resDto, error) {
		var publishCount int64
		if event.Published {
			mb, err := m.GetMessageBasic(ctx, tx, event.ChatId, event.MessageId)
			if err != nil {
				return nil, err
			}

			if mb != nil {
				previewTxt := m.createMessagePublishedText(mb.Content)

				_, err = tx.ExecContext(ctx, `
					insert into message_published (chat_id, message_id, owner_id, create_date_time, update_date_time, preview)
					values ($1, $2, $3, $4, $5, $6)
					on conflict (chat_id, message_id) do update set
					preview = excluded.preview
					,update_date_time = excluded.update_date_time
				`,
					event.ChatId, event.MessageId, mb.OwnerId, event.AdditionalData.CreatedAt, event.AdditionalData.CreatedAt, previewTxt)
				if err != nil {
					return nil, err
				}

				err = m.setMessagePublished(ctx, tx, event.ChatId, event.MessageId, true)
				if err != nil {
					return nil, err
				}
			} else {
				m.lgr.InfoContext(ctx, "Skipping publishing the mesage because it is not exists", logger.AttributeChatId, event.ChatId, logger.AttributeMessageId, event.MessageId)
			}
		} else {
			_, err := tx.ExecContext(ctx, "delete from message_published where chat_id = $1 and message_id = $2", event.ChatId, event.MessageId)
			if err != nil {
				return nil, err
			}

			err = m.setMessagePublished(ctx, tx, event.ChatId, event.MessageId, false)
			if err != nil {
				return nil, err
			}
		}

		var errc error
		publishCount, errc = m.GetPublishedMessageCount(ctx, tx, event.ChatId)
		if errc != nil {
			return nil, errc
		}

		return &resDto{
			count: publishCount,
		}, nil
	})
	if errOuter != nil {
		return 0, errOuter
	}
	return res.count, nil
}

func (m *EnrichingProjection) enrichMessagePublished(ctx context.Context, publishedMessage *dto.PublishedMessage, chatRegularParticipantCanPublishMessage bool, chatIsAdmin bool, messageOwnerUsersMap map[int64]*dto.User, behalfUserId int64) *dto.PublishedMessageDto {
	owner := messageOwnerUsersMap[publishedMessage.OwnerId]
	if owner == nil {
		owner = getDeletedUser(publishedMessage.OwnerId)
	}

	res := dto.PublishedMessageDto{
		Id:             publishedMessage.Id,
		Text:           publishedMessage.Text,
		ChatId:         publishedMessage.ChatId,
		OwnerId:        publishedMessage.OwnerId,
		Owner:          owner,
		CreateDateTime: publishedMessage.CreateDateTime,
		CanPublish:     CanPublishMessage(chatRegularParticipantCanPublishMessage, chatIsAdmin, publishedMessage.OwnerId, behalfUserId),
	}

	return &res
}
