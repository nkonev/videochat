package cqrs

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"nkonev.name/chat/client"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/preview"
	"nkonev.name/chat/utils"
)

func (m *EventHandler) OnParticipantAdded(ctx context.Context, event *ParticipantsAdded) error {

	eventTypeParticipantAdded := dto.EventTypeParticipantAdded
	ctx, participantAddSpan := m.tr.Start(ctx, fmt.Sprintf("participant.%s", eventTypeParticipantAdded))
	defer participantAddSpan.End()

	adt, err := m.commonProjection.GetChatDataForAuthorization(ctx, m.db, event.AdditionalData.BehalfUserId, event.ChatId)
	if err != nil {
		return err
	}

	if !CanAddParticipant(adt.IsChatAdmin, adt.ChatIsTetATet, event.IsJoining, adt.AvailableToSearch, adt.IsBlog, event.IsChatCreating, adt.IsParticipant, adt.RegularParticipantCanAddParticipants) {
		m.lgr.InfoContext(ctx, "Skipping ParticipantsAdded because there is no authorization to do so", logger.AttributeChatId, event.ChatId, logger.AttributeUserId, event.AdditionalData.BehalfUserId)
		return nil
	}

	// also updateViewableParticipants()
	resp, errp := m.commonProjection.OnParticipantAdded(ctx, event)
	if errp != nil {
		return errp
	}

	if !resp.ChatExists {
		m.lgr.InfoContext(ctx, "Skipping ParticipantsAdded because there is no chat exists. Probably it's protection against ahead creating tet-a-tet", logger.AttributeChatId, event.ChatId, logger.AttributeUserId, event.AdditionalData.BehalfUserId)
		return nil
	}

	userIds := event.GetParticipantIds()

	// send an output event for the users themselves
	for _, userId := range userIds {
		ue := &UserChatParticipantAdded{
			EventTime:     event.AdditionalData.CreatedAt,
			CorrelationId: event.AdditionalData.CorrelationId,
			ChatId:        event.ChatId,
			UserId:        userId,
			TetATet:       adt.ChatIsTetATet,
		}
		err = m.eventBus.Publish(ctx, ue)
		if err != nil {
			return err
		}
	}

	errOuter := m.commonProjection.IterateOverChatParticipantIdsExcepting(ctx, m.db, event.ChatId, nil, func(participantIdsPortion []int64) error {
		// transmit an output event with changed last participants for the existing participants
		for _, participantId := range participantIdsPortion {
			ue := &UserChatEdited{
				ChatId:        event.ChatId,
				UserId:        participantId,
				ChatAction:    ChatActionRefresh,
				EventTime:     event.AdditionalData.CreatedAt,
				CorrelationId: event.AdditionalData.CorrelationId,
			}
			errInn := m.eventBus.Publish(ctx, ue)
			if errInn != nil {
				return errInn
			}
		}
		return nil
	})
	if errOuter != nil {
		return errOuter
	}

	return nil
}

func (m *EventHandler) OnParticipantRemoved(ctx context.Context, event *ParticipantDeleted) error {
	eventTypeParticipantDeleted := dto.EventTypeParticipantDeleted
	ctx, participantSpan := m.tr.Start(ctx, fmt.Sprintf("participant.%s", eventTypeParticipantDeleted))
	defer participantSpan.End()

	adt, err := m.commonProjection.GetChatDataForAuthorization(ctx, m.db, event.AdditionalData.BehalfUserId, event.ChatId)
	if err != nil {
		return err
	}

	isChatRemoving := event.IsChatRemoving

	for _, participantId := range event.ParticipantIds {
		if !CanRemoveParticipant(event.AdditionalData.BehalfUserId, adt.IsChatAdmin, adt.ChatIsTetATet, event.IsLeaving, adt.IsParticipant, participantId, isChatRemoving) {
			m.lgr.InfoContext(ctx, "Skipping ParticipantRemoved because there is no authorization to do so", logger.AttributeChatId, event.ChatId, logger.AttributeUserId, event.AdditionalData.BehalfUserId)
			return nil
		}
	}

	if event.GetParticipantsType == GetParticipantsTypeNormal {
		return m.handleParticipantRemoved(ctx, event.AdditionalData, event.ParticipantIds, event.ChatId, event.AdditionalData.BehalfUserId, event.IsLeaving, isChatRemoving, adt, event.WereRemovedUsersFromAaa)
	} else if event.GetParticipantsType == GetParticipantsTypeAllInChatExcepting { // delete chat
		return m.commonProjection.IterateOverChatParticipantIdsExcepting(ctx, m.db, event.ChatId, event.AllParticipantIdsExcepting, func(participantIdsPortion []int64) error {
			return m.handleParticipantRemoved(ctx, event.AdditionalData, participantIdsPortion, event.ChatId, event.AdditionalData.BehalfUserId, event.IsLeaving, isChatRemoving, adt, event.WereRemovedUsersFromAaa)
		})
	} else {
		return fmt.Errorf("Unknown event.GetParticipantsType = %v", event.GetParticipantsType)
	}
}

func (m *EventHandler) handleParticipantRemoved(ctx context.Context, additionalData *AdditionalData, participantIds []int64, chatId int64, behalfUserId int64, isLeaving bool, isChatRemoving bool, adt dto.ChatAuthorizationData, wereRemovedUsersFromAaa bool) error {
	userIds := participantIds

	eventType := dto.EventTypeChatDeleted

	eventTypeParticipantDeleted := dto.EventTypeParticipantDeleted

	var pseudoUsers = []*dto.UserViewEnrichedDto{}
	for _, participantIdToRemove := range userIds {
		pseudoUsers = append(pseudoUsers, &dto.UserViewEnrichedDto{
			UserWithAdmin: dto.UserWithAdmin{
				User: dto.User{Id: participantIdToRemove},
			},
		})
	}

	if isChatRemoving {
		m.lgr.DebugContext(ctx, "Sending notification about the participant during chat deletion", "event_type", eventTypeParticipantDeleted, "user_ids", userIds)

		// in case chat removing no sense to send removing events to all the users (m x n), so we send it only to the removee's
		for _, participantId := range participantIds {
			errInn := m.rabbitmqOutputEventPublisher.Publish(ctx, additionalData.GetCorrelationId(), dto.ChatEvent{
				EventType:    eventTypeParticipantDeleted,
				UserId:       participantId,
				ChatId:       chatId,
				Participants: &pseudoUsers,
			})
			if errInn != nil {
				m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
			}
		}
	} else {
		m.lgr.DebugContext(ctx, "Sending notification about the participants", "event_type", eventTypeParticipantDeleted, "user_ids", userIds)

		// this is an event for ChatParticipantsModal.vue
		// we send to all the participant an event about removing removees
		err := m.commonProjection.IterateOverChatParticipantIdsExcepting(ctx, m.db, chatId, nil, func(participantIdsPortion []int64) error {
			// for every participant of chat we send an info about the newly added participants
			for _, participantId := range participantIdsPortion {
				errInn := m.rabbitmqOutputEventPublisher.Publish(ctx, additionalData.GetCorrelationId(), dto.ChatEvent{
					EventType:    eventTypeParticipantDeleted,
					UserId:       participantId,
					ChatId:       chatId,
					Participants: &pseudoUsers,
				})
				if errInn != nil {
					m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
				}
			}
			return nil
		})
		if err != nil {
			m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
		}
	}

	m.lgr.DebugContext(ctx, "Sending notification about the chat to participants", "event_type", eventType, "user_ids", userIds)

	isPubliclyAvailable := adt.AvailableToSearch || adt.IsBlog

	// just send an update if it was a participant removing from the publiclyAvailable chat
	if isPubliclyAvailable && !isChatRemoving { // in case isChatRemoving would be a race condition because of OnUserChatEdited() impl, so we disable it
		errOuter := m.commonProjection.IterateOverChatParticipantIdsIncluding(ctx, m.db, chatId, userIds, func(participantIdsPortion []int64) error {
			// here we invoke with "forceNonParticipant"
			for _, participantId := range participantIdsPortion {
				ue := &UserChatEdited{
					ChatId:        chatId,
					UserId:        participantId,
					ChatAction:    ChatActionRedraw,
					EventTime:     additionalData.CreatedAt,
					CorrelationId: additionalData.GetCorrelationId(),
				}
				errInn := m.eventBus.Publish(ctx, ue)
				if errInn != nil {
					return errInn
				}
			}
			return nil
		})
		if errOuter != nil {
			return errOuter
		}
	}

	// also updateViewableParticipants()
	errp := m.commonProjection.OnParticipantRemoved(ctx, userIds, chatId, isChatRemoving)
	if errp != nil {
		return errp
	}

	for _, userId := range userIds {
		errInn := m.eventBus.Publish(ctx, &UserChatParticipantRemoved{
			EventTime:               additionalData.CreatedAt,
			CorrelationId:           additionalData.CorrelationId,
			ChatId:                  chatId,
			UserId:                  userId,
			IsChatPubliclyAvailable: isPubliclyAvailable,
			WereRemovedUsersFromAaa: wereRemovedUsersFromAaa,
			IsChatRemoving:          isChatRemoving,
		})
		if errInn != nil {
			m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
		}
	}

	if !isChatRemoving {
		errOuter := m.commonProjection.IterateOverChatParticipantIdsExcepting(ctx, m.db, chatId, nil, func(participantIdsPortion []int64) error {
			// transmit an output event with changed last participants for the existing participants
			for _, participantId := range participantIdsPortion {
				ue := &UserChatEdited{
					ChatId:        chatId,
					UserId:        participantId,
					ChatAction:    ChatActionRefresh,
					EventTime:     additionalData.CreatedAt,
					CorrelationId: additionalData.CorrelationId,
				}
				errInn := m.eventBus.Publish(ctx, ue)
				if errInn != nil {
					return errInn
				}
			}
			return nil
		})
		if errOuter != nil {
			return errOuter
		}
	}

	return nil
}

func (m *EventHandler) OnParticipantChanged(ctx context.Context, event *ParticipantChanged) error {
	eventTypeParticipantChanged := dto.EventTypeParticipantEdited
	ctx, participantAddSpan := m.tr.Start(ctx, fmt.Sprintf("participant.%s", eventTypeParticipantChanged))
	defer participantAddSpan.End()

	adt, err := m.commonProjection.GetChatDataForAuthorization(ctx, m.db, event.AdditionalData.BehalfUserId, event.ChatId)
	if err != nil {
		return err
	}

	if !CanChangeParticipant(event.AdditionalData.BehalfUserId, adt.IsChatAdmin, adt.ChatIsTetATet, event.ParticipantId) {
		m.lgr.InfoContext(ctx, "Skipping ParticipantChanged because there is no authorization to do so", logger.AttributeChatId, event.ChatId, logger.AttributeUserId, event.AdditionalData.BehalfUserId)
		return nil
	}

	userIds := []int64{event.ParticipantId}

	participantsAdminsBefore, err := m.commonProjection.getAreAdminsOfUserIds(ctx, m.db, userIds, event.ChatId)
	if err != nil {
		return err
	}

	errp := m.commonProjection.OnParticipantChanged(ctx, event)
	if errp != nil {
		return errp
	}

	m.lgr.DebugContext(ctx, "Sending notification about the participant", "event_type", eventTypeParticipantChanged, "user_ids", userIds)

	errOuter := m.commonProjection.IterateOverChatParticipantIdsExcepting(ctx, m.db, event.ChatId, nil, func(participantIdsPortion []int64) error {
		participantsByBehalfs, _, errInn := m.enrichingProjection.GetParticipantsEnriched(ctx, participantIdsPortion, event.ChatId, int32(len(userIds)), utils.DefaultOffset, dto.NoSearchString, false, userIds)
		if errInn != nil {
			return errInn
		}

		sortedParticipants := slices.Sorted(maps.Keys(participantsByBehalfs))

		for _, behalfUserId := range sortedParticipants {
			hisParticipantsViews := participantsByBehalfs[behalfUserId]
			errInn = m.rabbitmqOutputEventPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.ChatEvent{
				EventType:    eventTypeParticipantChanged,
				UserId:       behalfUserId,
				ChatId:       event.ChatId,
				Participants: &hisParticipantsViews,
			})
			if errInn != nil {
				m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
			}

			ue := &UserChatEdited{
				ChatId:        event.ChatId,
				UserId:        behalfUserId,
				ChatAction:    ChatActionRefresh,
				EventTime:     event.AdditionalData.CreatedAt,
				CorrelationId: event.AdditionalData.CorrelationId,
			}
			errInn = m.eventBus.Publish(ctx, ue)
			if errInn != nil {
				return errInn
			}
		}

		return nil
	})
	if errOuter != nil {
		m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errOuter)
	}

	changedUserIds := []int64{}
	for _, participantId := range userIds {
		isAdminBefore := participantsAdminsBefore[participantId]
		isAdminAfter := event.NewAdmin
		if isChatAdminInternal(isAdminBefore) != isChatAdminInternal(isAdminAfter) {
			changedUserIds = append(changedUserIds, participantId)
		}
	}
	m.notifyMessagesReloadCommand(ctx, event.ChatId, changedUserIds, event.AdditionalData.GetCorrelationId())

	// ParticipantChanged == changing isAdmin, so CanChangeParticipant(), CanDeleteParticipant() are going to yield the different result so we forcibly refresh his ChatParticipantsModal
	m.notifyParticipantsReloadCommand(ctx, event.ChatId, changedUserIds, event.AdditionalData.GetCorrelationId())

	return nil
}

func (m *EventHandler) OnChatCreated(ctx context.Context, event *ChatCreated) error {
	// we don't check authorization for the chat creation
	err := m.commonProjection.OnChatCreated(ctx, event)
	if err != nil {
		return err
	}

	return nil
}

func (m *EventHandler) OnChatEdited(ctx context.Context, event *ChatEdited) error {
	adt, err := m.commonProjection.GetChatDataForAuthorization(ctx, m.db, event.AdditionalData.BehalfUserId, event.ChatId)
	if err != nil {
		return err
	}

	if !CanEditChat(adt.IsChatAdmin, adt.ChatIsTetATet) {
		m.lgr.InfoContext(ctx, "Skipping OnChatEdited because there is no authorization to do so", logger.AttributeChatId, event.ChatId, logger.AttributeUserId, event.AdditionalData.BehalfUserId)
		return nil
	}

	chatBasicBefore, err := m.commonProjection.GetChatBasic(ctx, m.commonProjection.db, event.ChatId)
	if err != nil {
		return err
	}

	previousBlogAbout, err := m.commonProjection.OnChatEdited(ctx, event)
	if err != nil {
		return err
	}

	errOuter := m.commonProjection.IterateOverChatParticipantIdsExcepting(ctx, m.db, event.ChatId, []int64{}, func(participantIdsPortion []int64) error {
		// if it was another "blog about" and now there is set a new - send an event to that's chat of the previous blog about in order to disable the previous blog about
		if previousBlogAbout != nil {
			for _, participantId := range participantIdsPortion {
				ue := &UserChatEdited{
					ChatId:        *previousBlogAbout,
					UserId:        participantId,
					ChatAction:    ChatActionRefresh,
					EventTime:     event.AdditionalData.CreatedAt,
					CorrelationId: event.AdditionalData.CorrelationId,
				}
				errInn := m.eventBus.Publish(ctx, ue)
				if errInn != nil {
					return errInn
				}
			}
		}

		for _, participantId := range participantIdsPortion {
			ue := &UserChatEdited{
				ChatId:        event.ChatId,
				UserId:        participantId,
				ChatAction:    ChatActionRefresh,
				EventTime:     event.AdditionalData.CreatedAt,
				CorrelationId: event.AdditionalData.CorrelationId,
			}
			errInn := m.eventBus.Publish(ctx, ue)
			if errInn != nil {
				return errInn
			}
		}

		return nil
	})
	if errOuter != nil {
		return errOuter
	}

	chatBasicAfter, err := m.commonProjection.GetChatBasic(ctx, m.commonProjection.db, event.ChatId)
	if err != nil {
		return err
	}

	// if any of message-related fields were changed we need to reload messages on user's side
	if canPublishMessageInternal(chatBasicBefore.RegularParticipantCanPublishMessage) != canPublishMessageInternal(chatBasicAfter.RegularParticipantCanPublishMessage) ||
		canPinMessageInternal(chatBasicBefore.RegularParticipantCanPinMessage) != canPinMessageInternal(chatBasicAfter.RegularParticipantCanPinMessage) ||
		canWriteMessageInternal(chatBasicBefore.RegularParticipantCanWriteMessage) != canWriteMessageInternal(chatBasicAfter.RegularParticipantCanWriteMessage) ||
		isBlogInternal(chatBasicBefore.IsBlog) != isBlogInternal(chatBasicAfter.IsBlog) {

		errOuter := m.commonProjection.IterateOverChatParticipantIdsExcepting(ctx, m.db, event.ChatId, nil, func(participantIdsPortion []int64) error {
			m.notifyMessagesReloadCommand(ctx, event.ChatId, participantIdsPortion, event.AdditionalData.GetCorrelationId())

			return nil
		})
		if errOuter != nil {
			return errOuter
		}

	}

	return nil
}

func (m *EventHandler) OnChatRemoved(ctx context.Context, event *ChatDeleted) error {

	// we don't check authorization here because the participants already were removed

	err := m.commonProjection.OnChatRemoved(ctx, event)
	if err != nil {
		return err
	}

	return nil
}

func (m *EventHandler) notifyMessagesReloadCommand(ctx context.Context, chatId int64, participantIds []int64, correlationId *string) {
	eventType := dto.EventTypeMessagesReload
	ctx, messageSpan := m.tr.Start(ctx, fmt.Sprintf("chat.%s", eventType))
	defer messageSpan.End()

	for _, participantId := range participantIds {
		err := m.rabbitmqOutputEventPublisher.Publish(ctx, correlationId, dto.ChatEvent{
			EventType: eventType,
			UserId:    participantId,
			ChatId:    chatId,
		})
		if err != nil {
			m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
		}
	}

}

func (m *EventHandler) notifyParticipantsReloadCommand(ctx context.Context, chatId int64, participantIds []int64, correlationId *string) {
	eventType := dto.EventTypeParticipantsReload
	ctx, messageSpan := m.tr.Start(ctx, fmt.Sprintf("chat.%s", eventType))
	defer messageSpan.End()

	for _, participantId := range participantIds {
		err := m.rabbitmqOutputEventPublisher.Publish(ctx, correlationId, dto.ChatEvent{
			EventType: eventType,
			UserId:    participantId,
			ChatId:    chatId,
		})
		if err != nil {
			m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
		}
	}
}

func (m *EventHandler) OnBatchMessagesCreated(event *MessageCreatedEventBatch) (context.Context, error) {
	for _, e := range event.MessageCreateds {
		if e.MessageCommoned.ChatId != event.ChatId {
			m.lgr.ErrorContext(event.FirstElementContext, "A mismatch for chatId is detected for the message", logger.AttributeChatId, e.MessageCommoned.ChatId, logger.AttributeMessageId, e.MessageCommoned.Id)
			return nil, nil // we cannot return a single context form a batch
		}
	}

	if len(event.MessageCreateds) > 0 {
		err := m.onMessagesCreated(event.FirstElementContext, event.MessageCreateds, event.ChatId)
		if err != nil {
			return nil, err
		}
	}

	return nil, nil // we cannot return a single context form a batch
}

type MemoizedGetUserOnline struct {
	reqCtx                   context.Context
	reqParticipantIdsPortion []int64
	aaaRestClient            client.AaaRestClient

	calculated bool

	cachedRes []*dto.UserOnline
	cachedErr error
}

func (p *MemoizedGetUserOnline) GetValues() ([]*dto.UserOnline, error) {
	if p.calculated {
		return p.cachedRes, p.cachedErr
	} else {
		p.cachedRes, p.cachedErr = p.aaaRestClient.GetOnlines(p.reqCtx, p.reqParticipantIdsPortion)
		p.calculated = true
		return p.cachedRes, p.cachedErr
	}
}

func (m *EventHandler) getMemoizedUserOnlines(ctx context.Context, participantIdsPortion []int64, aaaRestClient client.AaaRestClient) *MemoizedGetUserOnline {
	return &MemoizedGetUserOnline{
		reqCtx:                   ctx,
		reqParticipantIdsPortion: participantIdsPortion,
		aaaRestClient:            aaaRestClient,
	}
}

func (m *EventHandler) convertMessageCreatedToUser(messageEvents []MessageCreated) []UserMessageCreated {
	res := make([]UserMessageCreated, 0, len(messageEvents))
	for _, event := range messageEvents {
		res = append(res, UserMessageCreated{
			Id:             event.MessageCommoned.Id,
			ChatId:         event.MessageCommoned.ChatId,
			AdditionalData: event.AdditionalData,
		})
	}
	return res
}

func (m *EventHandler) onMessagesCreated(ctx context.Context, events []MessageCreated, chatId int64) error {
	eventType := dto.EventTypeMessageCreated

	ctx, messageSpan := m.tr.Start(ctx, fmt.Sprintf("message.%s", eventType))
	defer messageSpan.End()

	behalfUserIds := []int64{}
	for _, msg := range events {
		behalfUserIds = append(behalfUserIds, msg.AdditionalData.BehalfUserId)
	}

	adts, err := m.commonProjection.GetMessageDataForAuthorizationMessageCreatedBatch(ctx, m.db, behalfUserIds, chatId)
	if err != nil {
		return err
	}

	authorizedMessageEvents := []MessageCreated{}
	authorizedMessageEventsByMessageId := map[int64]MessageCreated{}

	for _, msg := range events {
		adt := adts[msg.AdditionalData.BehalfUserId]
		if !CanWriteMessage(adt.IsParticipant, adt.IsChatAdmin, adt.ChatCanWriteMessage) {
			m.lgr.InfoContext(ctx, "Skipping MessageCreated (part of OnBatchMessagesCreated) because there is no authorization to do so", logger.AttributeChatId, chatId, logger.AttributeUserId, msg.AdditionalData.BehalfUserId, logger.AttributeMessageId, msg.MessageCommoned.Id)
		} else {
			authorizedMessageEvents = append(authorizedMessageEvents, msg)
			authorizedMessageEventsByMessageId[msg.MessageCommoned.Id] = msg
		}
	}

	errOuter := db.Transact(ctx, m.db, func(tx *db.Tx) error {
		err = m.commonProjection.OnMessageCreatedBatch(ctx, tx, authorizedMessageEvents)
		if err != nil {
			return err
		}

		err = m.commonProjection.updateParticipantMessageReadIdBatch(ctx, tx, chatId, convertToMessageOwners(authorizedMessageEvents))
		if err != nil {
			return err
		}

		err = m.commonProjection.setLastMessage(ctx, tx, chatId)
		if err != nil {
			return err
		}

		return nil
	})
	if errOuter != nil {
		return errOuter
	}

	m.lgr.DebugContext(ctx, "Sending notification about the message to participants", "event_type", eventType)

	chatNotificationTitle, err := m.commonProjection.getChatNameForNotification(ctx, m.db, chatId)
	if err != nil {
		m.lgr.WarnContext(ctx, "Unable to get chatNotificationTitle", logger.AttributeChatId, chatId, logger.AttributeError, err)
		// nothing
	}

	authorizedBehalfUserIds := []int64{}
	authorizedMessageIds := []int64{}
	for _, event := range authorizedMessageEvents {
		authorizedBehalfUserIds = append(authorizedBehalfUserIds, event.AdditionalData.BehalfUserId)
		authorizedMessageIds = append(authorizedMessageIds, event.MessageCommoned.Id)
	}

	tetATetOpposites := map[int64]*int64{} // map userId:oppositeParticipantId
	tetATetOpposites, err = m.enrichingProjection.getTetATetOpposites(ctx, m.db, chatId, authorizedBehalfUserIds)
	if err != nil {
		m.lgr.WarnContext(ctx, "Unable to get opposites", logger.AttributeChatId, chatId, logger.AttributeError, err)
	}

	cin, err := m.enrichingProjection.getChatInfoForMessageNotification(ctx, m.db, chatId)
	if err != nil {
		return err
	}

	var additionalUserIdToFetch []int64 = []int64{}

	for _, event := range authorizedMessageEvents {
		additionalUserIdToFetch = append(additionalUserIdToFetch, event.AdditionalData.BehalfUserId)

		adt := adts[event.AdditionalData.BehalfUserId]

		var oppositeTetATetUserId *int64
		if adt.ChatIsTetATet {
			oppositeTetATetUserId = tetATetOpposites[event.AdditionalData.BehalfUserId]
			if oppositeTetATetUserId == nil {
				m.lgr.DebugContext(ctx, "single tet-a-tet", logger.AttributeChatId, event.MessageCommoned.ChatId)
			} else {
				additionalUserIdToFetch = append(additionalUserIdToFetch, *oppositeTetATetUserId)
			}
		}
	}

	if len(authorizedMessageEvents) > 0 {
		errOuter0 := m.commonProjection.IterateOverChatParticipantIdsExcepting(ctx, m.db, chatId, nil, func(participantIdsPortion []int64) error {
			userOnlines := m.getMemoizedUserOnlines(ctx, participantIdsPortion, m.aaaRestClient)

			messageViews, _, allPortionUsers, errInn := m.enrichingProjection.GetMessagesEnriched(ctx, participantIdsPortion, false, false, nil, chatId, dto.NoSize, nil, true, false, dto.NoSearchString, authorizedMessageIds, additionalUserIdToFetch)
			if errInn != nil {
				return errInn
			}
			allPortionUsersMap := utils.ToMap(allPortionUsers)

			inPortionMessageIds := []int64{}
			for _, messageView := range messageViews {
				inPortionMessageIds = append(inPortionMessageIds, messageView.Id)
			}
			inPortionMessageIds = slices.Compact(inPortionMessageIds)

			// send new message for participants portion
			for _, messageView := range messageViews {
				if event, ok := authorizedMessageEventsByMessageId[messageView.Id]; !ok {
					m.lgr.ErrorContext(ctx, "unable to find message", logger.AttributeChatId, chatId, logger.AttributeMessageId, messageView.Id)
				} else {
					_, _, _, newWithoutAnyHtml, _ := m.getNotificationData(ctx, event.MessageCommoned.Content, event.MessageCommoned.Embed)

					adt := adts[event.AdditionalData.BehalfUserId]

					var oppositeTetATetUserId *int64
					if adt.ChatIsTetATet {
						oppositeTetATetUserId = tetATetOpposites[event.AdditionalData.BehalfUserId]
					}

					cinp := m.enrichingProjection.patchChatInfoForMessageNotification(ctx, cin, allPortionUsersMap, oppositeTetATetUserId)

					// frontend event to add the message on the web page
					errInn = m.rabbitmqOutputEventPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.ChatEvent{
						EventType:           eventType,
						UserId:              messageView.BehalfUserId,
						ChatId:              event.MessageCommoned.ChatId,
						MessageNotification: &messageView,
					})
					if errInn != nil {
						m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
					}

					// notification about the new message (red dot)
					if messageView.BehalfUserId != event.AdditionalData.BehalfUserId { // skip myself
						if owner, ok := allPortionUsersMap[messageView.OwnerId]; !ok {
							m.lgr.InfoContext(ctx, "Message owner isn't found", logger.AttributeUserId, messageView.OwnerId)
						} else {
							errInn = m.rabbitmqOutputEventPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.GlobalUserEvent{
								UserId:    messageView.BehalfUserId,
								EventType: dto.EventTypeMessageBrowserNotificationAdd,
								BrowserNotification: &dto.BrowserNotification{
									ChatId:      messageView.ChatId,
									ChatName:    cinp.ChatName,
									ChatAvatar:  cinp.ChatAvatar,
									MessageId:   messageView.Id,
									MessageText: newWithoutAnyHtml,
									OwnerId:     owner.Id,
									OwnerLogin:  owner.Login,
								},
							})
							if errInn != nil {
								m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
							}
						}
					}
				}
			}

			// send mentions and replies for participants portion
			for _, messageId := range inPortionMessageIds {
				if event, ok := authorizedMessageEventsByMessageId[messageId]; !ok {
					m.lgr.ErrorContext(ctx, "unable to find message", logger.AttributeChatId, chatId, logger.AttributeMessageId, messageId)
				} else {
					// all these ids (and newRepliedUserId) should be joined with participantIdsPortion in order to avoid duplication oth the subsequent iterations
					newMentionedUserIds, newHasHere, newHasAll, newWithoutAnyHtml, newRepliedUserId := m.getNotificationData(ctx, event.MessageCommoned.Content, event.MessageCommoned.Embed)
					// per this MessageCreated in the current chat participants portion
					newToSendMentions := m.prepareMentionParticipantIds(ctx, newHasAll, newHasHere, newMentionedUserIds, userOnlines, participantIdsPortion)

					// for cache purposes, kinda optimization
					var behalfUserDto *dto.User
					behalfUserDto = allPortionUsersMap[event.AdditionalData.BehalfUserId]

					if behalfUserDto == nil {
						m.lgr.InfoContext(ctx, "Unable to get behalf user for mention notification", logger.AttributeUserId, event.AdditionalData.BehalfUserId)
					} else {
						for _, participantId := range newToSendMentions {
							if participantId == event.AdditionalData.BehalfUserId {
								continue // skip myself
							}

							errInn = m.rabbitmqNotificationEventsPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.NotificationEvent{
								EventType: dto.EventTypeMentionAdded,
								UserId:    participantId,
								ChatId:    event.MessageCommoned.ChatId,
								MentionNotification: &dto.MentionNotification{
									Id:   event.MessageCommoned.Id,
									Text: newWithoutAnyHtml,
								},
								ByUserId:  behalfUserDto.Id,
								ByLogin:   behalfUserDto.Login,
								ByAvatar:  behalfUserDto.Avatar,
								ChatTitle: chatNotificationTitle,
							})
							if errInn != nil {
								m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
							}
						}
					}

					if newRepliedUserId != nil { // per this MessageCreated in the current chat participants portion
						if behalfUserDto == nil {
							m.lgr.InfoContext(ctx, "Unable to get behalf user for reply notification", logger.AttributeUserId, event.AdditionalData.BehalfUserId)
						} else {
							if *newRepliedUserId != event.AdditionalData.BehalfUserId && slices.Contains(participantIdsPortion, *newRepliedUserId) { // skip myself and don't duplicate
								err = m.rabbitmqNotificationEventsPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.NotificationEvent{
									EventType: dto.EventTypeReplyAdded,
									UserId:    *newRepliedUserId,
									ChatId:    event.MessageCommoned.ChatId,
									ReplyNotification: &dto.ReplyDto{
										MessageId:        event.MessageCommoned.Id,
										ChatId:           event.MessageCommoned.ChatId,
										ReplyableMessage: newWithoutAnyHtml,
									},
									ByUserId:  behalfUserDto.Id,
									ByLogin:   behalfUserDto.Login,
									ByAvatar:  behalfUserDto.Avatar,
									ChatTitle: chatNotificationTitle,
								})
								if err != nil {
									m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
								}
							}
						}
					}
				}
			}

			for _, userId := range participantIdsPortion {
				ui := &UserMessagesCreatedEvent{
					ChatId:          chatId,
					UserId:          userId,
					MessageCreateds: m.convertMessageCreatedToUser(authorizedMessageEvents),
				}

				errInner := m.eventBus.Publish(ctx, ui)
				if errInner != nil {
					return errInner
				}

				// transmit an output event with changed last message for the existing participants (should be after UserMessagesCreatedEvent, because it depends on first)
				for _, participantId := range participantIdsPortion {
					ue := &UserChatEdited{
						ChatId:        chatId,
						UserId:        participantId,
						ChatAction:    ChatActionRefresh,
						EventTime:     authorizedMessageEvents[0].AdditionalData.CreatedAt,
						CorrelationId: authorizedMessageEvents[0].AdditionalData.CorrelationId,
					}
					errInn = m.eventBus.Publish(ctx, ue)
					if errInn != nil {
						return errInn
					}
				}
			}

			return nil
		})
		if errOuter0 != nil {
			return errOuter0
		}
	}

	return nil
}

func convertToMessageOwners(events []MessageCreated) []MessageOwner {
	res := make([]MessageOwner, 0, len(events))
	for _, v := range events {
		res = append(res, MessageOwner{
			MessageId: v.MessageCommoned.Id,
			OwnerId:   v.AdditionalData.BehalfUserId,
			Time:      v.AdditionalData.CreatedAt,
		})
	}
	return res
}

func (m *EventHandler) prepareMentionParticipantIds(ctx context.Context, newHasAll, newHasHere bool, newMentionedUserIds []int64, userOnlinesMem *MemoizedGetUserOnline, participantIdsPortion []int64) []int64 {
	newToSendMentions := []int64{}

	newMentionedUserIdsMap := utils.SliceToSetMapIdStruct(newMentionedUserIds)

	// see also cqrs/projection_message.go :: parseMentionUserIdsFromMessageHtml()
	if newHasAll {
		newToSendMentions = append(newToSendMentions, participantIdsPortion...)
	} else if newHasHere {
		userOnlines, err := userOnlinesMem.GetValues() // get online for opposite user
		if err != nil {
			m.lgr.WarnContext(ctx, "Unable to get online for", "user_ids", participantIdsPortion, logger.AttributeError, err)
			// nothing
		}

		for _, uo := range userOnlines {
			newToSendMentions = append(newToSendMentions, uo.Id)
		}
	} else {
		for _, pi := range participantIdsPortion {
			if _, ok := newMentionedUserIdsMap[pi]; ok {
				newToSendMentions = append(newToSendMentions, pi)
			}
		}
	}

	return newToSendMentions
}

func (m *EventHandler) OnMessageEdited(ctx context.Context, event *MessageEdited) error {
	eventType := dto.EventTypeMessageEdited

	ctx, messageSpan := m.tr.Start(ctx, fmt.Sprintf("message.%s", eventType))
	defer messageSpan.End()

	adt, err := m.commonProjection.GetMessageDataForAuthorization(ctx, m.db, event.AdditionalData.BehalfUserId, event.MessageCommoned.ChatId, event.MessageCommoned.Id)
	if err != nil {
		return err
	}

	canWriteMessage := CanWriteMessage(adt.IsParticipant, adt.IsChatAdmin, adt.ChatCanWriteMessage)

	if event.IsEmbedSync {
		if !CanSyncEmbedMessage(event.AdditionalData.BehalfUserId, adt.MessageOwnerId, adt.HasEmbedMessage, canWriteMessage) {
			m.lgr.InfoContext(ctx, "Skipping OnMessageEdited because there is no authorization to do so (sync)", logger.AttributeChatId, event.MessageCommoned.ChatId, logger.AttributeUserId, event.AdditionalData.BehalfUserId)
			return nil
		}
	} else {
		if !CanEditMessage(event.AdditionalData.BehalfUserId, adt.MessageOwnerId, adt.HasEmbedMessage, adt.EmbedMessageTypeSafe, canWriteMessage) {
			m.lgr.InfoContext(ctx, "Skipping OnMessageEdited because there is no authorization to do so (edit)", logger.AttributeChatId, event.MessageCommoned.ChatId, logger.AttributeUserId, event.AdditionalData.BehalfUserId)
			return nil
		}
	}

	messageBasicOld, err := m.commonProjection.GetMessageWithEmbed(ctx, m.db, event.MessageCommoned.ChatId, event.MessageCommoned.Id)
	if err != nil {
		return err
	}

	oldMentionedUserIds, oldHasHere, oldHasAll, _, oldRepliedUserId := m.getNotificationData(ctx, messageBasicOld.GetContentOrEmpty(), messageBasicOld.GetEmbed())
	oldMentionedUserIdsMap := utils.SliceToSetMapIdStruct(oldMentionedUserIds)

	var isLastMessage bool

	res, errOuter := db.TransactWithResult(ctx, m.db, func(tx *db.Tx) (*MessageEditDto, error) {
		resDto, errInn := m.commonProjection.OnMessageEdited(ctx, tx, event)
		if errInn != nil {
			return nil, errInn
		}

		lastMessageId, errInn := m.commonProjection.GetLastMessageId(ctx, tx, event.MessageCommoned.ChatId)
		if errInn != nil {
			return nil, errInn
		}

		isLastMessage = event.MessageCommoned.Id == lastMessageId

		if isLastMessage {
			errInn = m.commonProjection.setLastMessage(ctx, tx, event.MessageCommoned.ChatId)
			if errInn != nil {
				return nil, errInn
			}
		}

		return resDto, nil
	})

	var isPinned, isPublished bool
	var pinnedCount, publishedCount int64

	if res != nil {
		isPinned = res.isPinned
		isPublished = res.isPublished
		pinnedCount = res.pinnedCount
		publishedCount = res.publishedCount
	}

	m.lgr.DebugContext(ctx, "Sending notification about the message to participants", "event_type", eventType, logger.AttributeUserId, event.AdditionalData.BehalfUserId)

	chatNotificationTitle, err := m.commonProjection.getChatNameForNotification(ctx, m.db, event.MessageCommoned.ChatId)
	if err != nil {
		m.lgr.WarnContext(ctx, "Unable to get chatNotificationTitle", logger.AttributeChatId, event.MessageCommoned.ChatId, logger.AttributeError, err)
		// nothing
	}

	newMentionedUserIds, newHasHere, newHasAll, newWithoutAnyHtml, newRepliedUserId := m.getNotificationData(ctx, event.MessageCommoned.Content, event.MessageCommoned.Embed)
	newMentionedUserIdsMap := utils.SliceToSetMapIdStruct(newMentionedUserIds)

	// for cache purposes, kinda optimization
	var behalfUserDto *dto.User

	addedMentionedUserIds := []int64{}
	removedMentionedUserIds := []int64{}

	var addedRepliedUserId *int64
	var removedRepliedUserId *int64

	for newUserId := range newMentionedUserIdsMap {
		if _, ok := oldMentionedUserIdsMap[newUserId]; !ok {
			addedMentionedUserIds = append(addedMentionedUserIds, newUserId)
		}
	}

	for oldUserId := range oldMentionedUserIdsMap {
		if _, ok := newMentionedUserIdsMap[oldUserId]; !ok {
			removedMentionedUserIds = append(removedMentionedUserIds, oldUserId)
		}
	}

	if newRepliedUserId != nil && oldRepliedUserId != nil {
		if *newRepliedUserId != *oldRepliedUserId {
			addedRepliedUserId = newRepliedUserId
			removedRepliedUserId = oldRepliedUserId
		}
	} else if newRepliedUserId != nil {
		addedRepliedUserId = newRepliedUserId
	} else if oldRepliedUserId != nil {
		removedRepliedUserId = oldRepliedUserId
	}

	addedHasAll := !oldHasAll && newHasAll
	addedHasHere := !oldHasHere && newHasHere

	removedHasAll := oldHasAll && !newHasAll
	removedHasHere := oldHasHere && !newHasHere

	var additionalUserIdToFetch []int64 = []int64{event.AdditionalData.BehalfUserId}

	errOuter = m.commonProjection.IterateOverChatParticipantIdsExcepting(ctx, m.db, event.MessageCommoned.ChatId, nil, func(participantIdsPortion []int64) error {

		userOnlines := m.getMemoizedUserOnlines(ctx, participantIdsPortion, m.aaaRestClient)

		messageViews, _, allPortionUsers, errInn := m.enrichingProjection.GetMessagesEnriched(ctx, participantIdsPortion, false, false, nil, event.MessageCommoned.ChatId, int32(len(participantIdsPortion)), nil, true, false, dto.NoSearchString, []int64{event.MessageCommoned.Id}, additionalUserIdToFetch)
		if errInn != nil {
			return errInn
		}

		allPortionUsersMap := utils.ToMap(allPortionUsers)
		behalfUserDto = allPortionUsersMap[event.AdditionalData.BehalfUserId]

		var pinnedEnricheds = map[int64]*dto.PinnedMessageDto{}
		var publishedEnricheds = map[int64]*dto.PublishedMessageDto{}
		if isPinned {
			// allPortionUsersMap contains message owner
			pinnedEnricheds, errInn = m.enrichingProjection.GetPinnedMessageEnriched(ctx, m.db, event.MessageCommoned.ChatId, event.MessageCommoned.Id, participantIdsPortion, allPortionUsersMap)
			if errInn != nil {
				return errInn
			}
		}

		if isPublished {
			// allPortionUsersMap contains message owner
			publishedEnricheds, errInn = m.enrichingProjection.GetPublishedMessageEnriched(ctx, m.db, event.MessageCommoned.ChatId, event.MessageCommoned.Id, participantIdsPortion, allPortionUsersMap)
			if errInn != nil {
				return errInn
			}
		}

		for _, messageView := range messageViews {
			errInn = m.rabbitmqOutputEventPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.ChatEvent{
				EventType:           eventType,
				UserId:              messageView.BehalfUserId,
				ChatId:              event.MessageCommoned.ChatId,
				MessageNotification: &messageView,
			})
			if errInn != nil {
				m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
			}

			if isPinned {
				// almost the same as sendPromotePinned() but
				// the differense is that this particular method just sends pinneds, not pinned promoteds
				// and here is different event - dto.EventTypePinnedMessageEdit
				pinnedEnriched := pinnedEnricheds[messageView.BehalfUserId]
				if pinnedEnriched != nil {
					errInn = m.rabbitmqOutputEventPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.ChatEvent{
						EventType: dto.EventTypePinnedMessageEdit,
						PromoteMessageNotification: &dto.PinnedMessageEvent{
							Message:    *pinnedEnriched,
							TotalCount: pinnedCount,
						},
						UserId: messageView.BehalfUserId,
						ChatId: event.MessageCommoned.ChatId,
					})
					if errInn != nil {
						m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
					}
				} else {
					m.lgr.WarnContext(ctx, "Pinned enriched isn't found", logger.AttributeUserId, messageView.BehalfUserId)
				}
			}

			if isPublished {
				publishedEnriched := publishedEnricheds[messageView.BehalfUserId]
				if publishedEnriched != nil {
					errInn = m.rabbitmqOutputEventPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.ChatEvent{
						EventType: dto.EventTypePublishedMessageEdit,
						PublishedMessageNotification: &dto.PublishedMessageEvent{
							Message:    *publishedEnriched,
							TotalCount: publishedCount,
						},
						UserId: messageView.BehalfUserId,
						ChatId: event.MessageCommoned.ChatId,
					})
					if errInn != nil {
						m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
					}
				} else {
					m.lgr.WarnContext(ctx, "Published enriched isn't found", logger.AttributeUserId, messageView.BehalfUserId)
				}
			}
		}

		addedToSendMentions := m.prepareMentionParticipantIds(ctx, addedHasAll, addedHasHere, addedMentionedUserIds, userOnlines, participantIdsPortion)
		removedToSendMentions := m.prepareMentionParticipantIds(ctx, removedHasAll, removedHasHere, removedMentionedUserIds, userOnlines, participantIdsPortion)

		if behalfUserDto == nil {
			m.lgr.InfoContext(ctx, "Unable to get behalf user for mention notification", logger.AttributeUserId, event.AdditionalData.BehalfUserId)
		} else {

			// add notification
			for _, participantId := range addedToSendMentions {
				if participantId == event.AdditionalData.BehalfUserId {
					continue // skip myself
				}

				errInn = m.rabbitmqNotificationEventsPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.NotificationEvent{
					EventType: dto.EventTypeMentionAdded,
					UserId:    participantId,
					ChatId:    event.MessageCommoned.ChatId,
					MentionNotification: &dto.MentionNotification{
						Id:   event.MessageCommoned.Id,
						Text: newWithoutAnyHtml,
					},
					ByUserId:  behalfUserDto.Id,
					ByLogin:   behalfUserDto.Login,
					ByAvatar:  behalfUserDto.Avatar,
					ChatTitle: chatNotificationTitle,
				})
				if errInn != nil {
					m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
				}
			}

			// remove notification
			for _, participantId := range removedToSendMentions {
				if participantId == event.AdditionalData.BehalfUserId {
					continue // skip myself
				}

				errInn = m.rabbitmqNotificationEventsPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.NotificationEvent{
					EventType: dto.EventTypeMentionDeleted,
					UserId:    participantId,
					ChatId:    event.MessageCommoned.ChatId,
					MentionNotification: &dto.MentionNotification{
						Id: event.MessageCommoned.Id,
					},
				})
				if errInn != nil {
					m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
				}
			}
		}

		// transmit an output event with changed last message for the existing participants
		if isLastMessage {
			for _, participantId := range participantIdsPortion {
				ue := &UserChatEdited{
					ChatId:        event.MessageCommoned.ChatId,
					UserId:        participantId,
					ChatAction:    ChatActionRefresh,
					EventTime:     event.AdditionalData.CreatedAt,
					CorrelationId: event.AdditionalData.CorrelationId,
				}
				errInn = m.eventBus.Publish(ctx, ue)
				if errInn != nil {
					return errInn
				}
			}
		}

		return nil
	})
	if errOuter != nil {
		return errOuter
	}

	if addedRepliedUserId != nil {
		if behalfUserDto == nil {
			m.lgr.InfoContext(ctx, "Unable to get behalf user for reply notification", logger.AttributeUserId, event.AdditionalData.BehalfUserId)
		} else {
			if *addedRepliedUserId != event.AdditionalData.BehalfUserId { // skip myself
				err = m.rabbitmqNotificationEventsPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.NotificationEvent{
					EventType: dto.EventTypeReplyAdded,
					UserId:    *addedRepliedUserId,
					ChatId:    event.MessageCommoned.ChatId,
					ReplyNotification: &dto.ReplyDto{
						MessageId:        event.MessageCommoned.Id,
						ChatId:           event.MessageCommoned.ChatId,
						ReplyableMessage: newWithoutAnyHtml,
					},
					ByUserId:  behalfUserDto.Id,
					ByLogin:   behalfUserDto.Login,
					ByAvatar:  behalfUserDto.Avatar,
					ChatTitle: chatNotificationTitle,
				})
				if err != nil {
					m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
				}
			}
		}
	}

	if removedRepliedUserId != nil {
		if behalfUserDto == nil {
			m.lgr.InfoContext(ctx, "Unable to get behalf user for reply notification", logger.AttributeUserId, event.AdditionalData.BehalfUserId)
		} else {

			err = m.rabbitmqNotificationEventsPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.NotificationEvent{
				EventType: dto.EventTypeReplyDeleted,
				UserId:    *removedRepliedUserId,
				ChatId:    event.MessageCommoned.ChatId,
				ReplyNotification: &dto.ReplyDto{
					MessageId: event.MessageCommoned.Id,
					ChatId:    event.MessageCommoned.ChatId,
				},
			})
			if err != nil {
				m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
			}
		}
	}

	return nil
}

func (m *EventHandler) getNotificationData(ctx context.Context, messageHtml string, em dto.Embeddable) ([]int64, bool, bool, string, *int64) {
	newWithoutSourceTags := m.stripSourceContent.Sanitize(messageHtml)
	newMentionedUserIds, newHasHere, newHasAll := m.enrichingProjection.parseMentionUserIdsFromMessageHtml(ctx, newWithoutSourceTags)

	newWithoutAnyHtml := m.stripAllTags.Sanitize(newWithoutSourceTags)
	newWithoutAnyHtml = preview.CreateMessagePreviewWithoutLogin(m.stripAllTags, m.cfg.Message.PreviewMaxTextSize, newWithoutAnyHtml)

	var repliedUserId *int64
	if em != nil {
		if reply, ok := em.(*dto.EmbedReply); ok {
			repliedUserId = &reply.OwnerId
		}
	}

	return newMentionedUserIds, newHasHere, newHasAll, newWithoutAnyHtml, repliedUserId
}

func (m *EventHandler) OnMessageRemoved(ctx context.Context, event *MessageDeleted) error {
	eventType := dto.EventTypeMessageDeleted

	ctx, messageSpan := m.tr.Start(ctx, fmt.Sprintf("message.%s", eventType))
	defer messageSpan.End()

	adt, err := m.commonProjection.GetMessageDataForAuthorization(ctx, m.db, event.AdditionalData.BehalfUserId, event.ChatId, event.MessageId)
	if err != nil {
		return err
	}

	canWriteMessage := CanWriteMessage(adt.IsParticipant, adt.IsChatAdmin, adt.ChatCanWriteMessage)

	if !CanDeleteMessage(event.AdditionalData.BehalfUserId, adt.MessageOwnerId, canWriteMessage) {
		m.lgr.InfoContext(ctx, "Skipping OnMessageRemoved because there is no authorization to do so", logger.AttributeChatId, event.ChatId, logger.AttributeUserId, event.AdditionalData.BehalfUserId)
		return nil
	}

	messageBasic, err := m.commonProjection.GetMessageBasic(ctx, m.db, event.ChatId, event.MessageId)
	if err != nil {
		return err
	}

	reactions, err := m.commonProjection.GetReactionsOnMessage(ctx, m.db, event.ChatId, event.MessageId)
	if err != nil {
		return err
	}

	var isLastMessage bool

	res, errOuter := db.TransactWithResult(ctx, m.db, func(tx *db.Tx) (*MessageRemovedDto, error) {
		lastMessageId, errInn := m.commonProjection.GetLastMessageId(ctx, tx, event.ChatId)
		if errInn != nil {
			return nil, errInn
		}

		isLastMessage = event.MessageId == lastMessageId

		messageRemovedDto, errInn := m.commonProjection.OnMessageRemoved(ctx, tx, event)
		if errInn != nil {
			return nil, errInn
		}

		if isLastMessage {
			errInn = m.commonProjection.setLastMessage(ctx, tx, event.ChatId)
			if errInn != nil {
				return nil, errInn
			}
		}

		return messageRemovedDto, nil
	})
	if errOuter != nil {
		return errOuter
	}

	wasPinned := res.wasMessagePinned
	wasPublished := res.wasMessagePublished
	pinnedCount := res.pinnedCount
	publishedCount := res.publishedCount
	promotedMessageId := res.promotedMessageId

	m.lgr.DebugContext(ctx, "Sending notification about the message to participants", "event_type", eventType, logger.AttributeUserId, event.AdditionalData.BehalfUserId)

	errOuter = m.commonProjection.IterateOverChatParticipantIdsExcepting(ctx, m.db, event.ChatId, nil, func(participantIdsPortion []int64) error {
		for _, userId := range participantIdsPortion {
			ui := &UserMessageDeletedEvent{
				ChatId:        event.ChatId,
				UserId:        userId,
				MessageId:     event.MessageId,
				CorrelationId: event.AdditionalData.GetCorrelationId(),
			}

			errInner := m.eventBus.Publish(ctx, ui)
			if errInner != nil {
				return errInner
			}
		}

		for _, participantId := range participantIdsPortion {
			errInn := m.rabbitmqOutputEventPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.ChatEvent{
				EventType: eventType,
				UserId:    participantId,
				ChatId:    event.ChatId,
				MessageDeletedNotification: &dto.MessageDeletedDto{
					Id:     event.MessageId,
					ChatId: event.ChatId,
				},
			})
			if errInn != nil {
				m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
			}

			err = m.rabbitmqNotificationEventsPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.NotificationEvent{
				EventType: dto.EventTypeMentionDeleted,
				UserId:    participantId,
				ChatId:    event.ChatId,
				MentionNotification: &dto.MentionNotification{
					Id: event.MessageId,
				},
			})
			if err != nil {
				m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
			}

			err = m.rabbitmqNotificationEventsPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.NotificationEvent{
				EventType: dto.EventTypeReplyDeleted,
				UserId:    participantId,
				ChatId:    event.ChatId,
				ReplyNotification: &dto.ReplyDto{
					MessageId: event.MessageId,
					ChatId:    event.ChatId,
				},
			})
			if err != nil {
				m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
			}

			for _, reaction := range reactions {
				var messageOwnerId = messageBasic.GetOwnerId()
				if messageOwnerId == dto.NoOwner || messageOwnerId == dto.NoId {
					m.lgr.InfoContext(ctx, "Unable to get message owner for reaction notification")
				} else {
					err = m.rabbitmqNotificationEventsPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.NotificationEvent{
						EventType: dto.EventTypeReactionDeleted,
						ReactionEvent: &dto.ReactionEvent{
							Reaction:  reaction,
							MessageId: event.MessageId,
						},
						UserId: messageOwnerId,
						ChatId: event.ChatId,
					})
					if err != nil {
						m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
					}
				}
			}

			// send unpromote/unpin
			if wasPinned {
				m.sendUnpromotePinned(ctx, participantIdsPortion, pinnedCount, event.ChatId, event.MessageId, event.AdditionalData.CreatedAt, event.AdditionalData.GetCorrelationId())
			}

			// send unpublish
			if wasPublished {
				m.sendUnpublish(ctx, participantIdsPortion, publishedCount, event.ChatId, event.MessageId, event.AdditionalData.CreatedAt, event.AdditionalData.GetCorrelationId())
			}

			errInn = m.rabbitmqOutputEventPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.GlobalUserEvent{
				UserId:    participantId,
				EventType: dto.EventTypeMessageBrowserNotificationDelete,
				BrowserNotification: &dto.BrowserNotification{
					ChatId:    event.ChatId,
					MessageId: event.MessageId,
					OwnerId:   dto.NonExistentUser,
				},
			})
			if errInn != nil {
				m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
			}
		}

		for _, userId := range participantIdsPortion {
			// transmit an output event with changed last message for the existing participants (should be after UserMessageDeletedEvent, because it depends on first)
			if isLastMessage {
				ue := &UserChatEdited{
					ChatId:        event.ChatId,
					UserId:        userId,
					ChatAction:    ChatActionRefresh,
					EventTime:     event.AdditionalData.CreatedAt,
					CorrelationId: event.AdditionalData.CorrelationId,
				}
				errInn := m.eventBus.Publish(ctx, ue)
				if errInn != nil {
					return errInn
				}
			}
		}

		return nil
	})
	if errOuter != nil {
		return errOuter
	}

	if promotedMessageId != nil {
		errOuter = m.sendPromotePinned(ctx, event.ChatId, *promotedMessageId, pinnedCount, event.AdditionalData.GetCorrelationId())
		if errOuter != nil {
			return errOuter
		}
	}

	return nil
}

func (m *EventHandler) OnMessagePinned(ctx context.Context, event *MessagePinned) error {
	var eventTypeV string
	if event.Pinned {
		eventTypeV = dto.EventTypePinnedMessagePromote
	} else {
		eventTypeV = dto.EventTypePinnedMessageUnpromote
	}

	eventTypeMessageEdit := dto.EventTypeMessageEdited

	ctx, messageSpan := m.tr.Start(ctx, fmt.Sprintf("chat.%s", eventTypeV))
	defer messageSpan.End()

	adt, err := m.commonProjection.GetMessageDataForAuthorization(ctx, m.db, event.AdditionalData.BehalfUserId, event.ChatId, event.MessageId)
	if err != nil {
		return err
	}

	if !CanPinMessage(adt.ChatCanPinMessage, adt.IsChatAdmin) {
		m.lgr.InfoContext(ctx, "Skipping OnMessagePinned because there is no authorization to do so", logger.AttributeChatId, event.ChatId, logger.AttributeUserId, event.AdditionalData.BehalfUserId)
		return nil
	}

	promotedMessageId, pinnedCount, err := m.commonProjection.OnMessagePinned(ctx, event)
	if err != nil {
		return err
	}

	// send unpromote/unpin
	if !event.Pinned {
		errOuter := m.commonProjection.IterateOverChatParticipantIdsExcepting(ctx, m.db, event.ChatId, nil, func(participantIdsPortion []int64) error {
			messageViews, _, _, errInn := m.enrichingProjection.GetMessagesEnriched(ctx, participantIdsPortion, false, false, nil, event.ChatId, int32(len(participantIdsPortion)), nil, true, false, dto.NoSearchString, []int64{event.MessageId}, nil)
			if errInn != nil {
				return errInn
			}

			m.sendUnpromotePinned(ctx, participantIdsPortion, pinnedCount, event.ChatId, event.MessageId, event.AdditionalData.CreatedAt, event.AdditionalData.GetCorrelationId())

			for _, messageView := range messageViews {
				errInn = m.rabbitmqOutputEventPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.ChatEvent{
					EventType:           eventTypeMessageEdit,
					UserId:              messageView.BehalfUserId,
					ChatId:              event.ChatId,
					MessageNotification: &messageView,
				})
				if errInn != nil {
					m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
				}
			}

			return nil
		})
		if errOuter != nil {
			return errOuter
		}
	}

	// send promote current message event or send promote previous pinned message event
	if promotedMessageId != nil {
		errOuter := m.sendPromotePinned(ctx, event.ChatId, *promotedMessageId, pinnedCount, event.AdditionalData.GetCorrelationId())
		if errOuter != nil {
			return errOuter
		}
	}

	return nil
}

func (m *EventHandler) sendUnpublish(ctx context.Context, participantIdsPortion []int64, publishedCount int64, chatId, messageId int64, createdAt time.Time, correlationId *string) {
	for _, participantId := range participantIdsPortion {
		errInn := m.rabbitmqOutputEventPublisher.Publish(ctx, correlationId, dto.ChatEvent{
			EventType: dto.EventTypePublishedMessageRemove,
			PublishedMessageNotification: &dto.PublishedMessageEvent{
				Message: dto.PublishedMessageDto{
					Id:             messageId,
					ChatId:         chatId,
					CreateDateTime: createdAt, // to pass thru graphql
				},
				TotalCount: publishedCount,
			},
			UserId: participantId,
			ChatId: chatId,
		})
		if errInn != nil {
			m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
		}
	}
}

func (m *EventHandler) sendUnpromotePinned(ctx context.Context, participantIdsPortion []int64, pinnedCount int64, chatId, messageId int64, createdAt time.Time, correlationId *string) {
	for _, participantId := range participantIdsPortion {
		errInn := m.rabbitmqOutputEventPublisher.Publish(ctx, correlationId, dto.ChatEvent{
			EventType: dto.EventTypePinnedMessageUnpromote,
			PromoteMessageNotification: &dto.PinnedMessageEvent{
				Message: dto.PinnedMessageDto{
					Id:             messageId,
					ChatId:         chatId,
					CreateDateTime: createdAt, // to pass thru graphql
				},
				TotalCount: pinnedCount,
			},
			UserId: participantId,
			ChatId: chatId,
		})
		if errInn != nil {
			m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
		}
	}
}

func (m *EventHandler) sendPromotePinned(ctx context.Context, chatId, promotedMessageId, pinnedCount int64, correlationId *string) error {
	eventTypeMessageEdit := dto.EventTypeMessageEdited

	errOuter := m.commonProjection.IterateOverChatParticipantIdsExcepting(ctx, m.db, chatId, nil, func(participantIdsPortion []int64) error {
		messageViews, _, allPortionUsers, errInn := m.enrichingProjection.GetMessagesEnriched(ctx, participantIdsPortion, false, false, nil, chatId, int32(len(participantIdsPortion)), nil, true, false, dto.NoSearchString, []int64{promotedMessageId}, nil)
		if errInn != nil {
			return errInn
		}

		// allPortionUsersMap contains message owner
		allPortionUsersMap := utils.ToMap(allPortionUsers)

		enrichedsPinnedPromoted, errInn := m.enrichingProjection.GetPinnedMessageEnriched(ctx, m.db, chatId, promotedMessageId, participantIdsPortion, allPortionUsersMap)
		if errInn != nil {
			return errInn
		}

		for _, participantId := range participantIdsPortion {
			promotedMessage := enrichedsPinnedPromoted[participantId]
			if promotedMessage != nil {
				errInn = m.rabbitmqOutputEventPublisher.Publish(ctx, correlationId, dto.ChatEvent{
					EventType: dto.EventTypePinnedMessagePromote,
					PromoteMessageNotification: &dto.PinnedMessageEvent{
						Message:    *promotedMessage,
						TotalCount: pinnedCount,
					},
					UserId: participantId,
					ChatId: chatId,
				})
				if errInn != nil {
					m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
				}
			} else {
				m.lgr.WarnContext(ctx, "Pinned promoted isn't found for the participant", logger.AttributeUserId, participantId)
			}
		}

		for _, messageView := range messageViews {
			errInn = m.rabbitmqOutputEventPublisher.Publish(ctx, correlationId, dto.ChatEvent{
				EventType:           eventTypeMessageEdit,
				UserId:              messageView.BehalfUserId,
				ChatId:              chatId,
				MessageNotification: &messageView,
			})
			if errInn != nil {
				m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
			}
		}

		return nil
	})
	if errOuter != nil {
		return errOuter
	}

	return nil
}

func (m *EventHandler) OnMessagePublished(ctx context.Context, event *MessagePublished) error {
	var eventTypeV string
	if event.Published {
		eventTypeV = dto.EventTypePublishedMessageAdd
	} else {
		eventTypeV = dto.EventTypePublishedMessageRemove
	}

	eventTypeMessageEdit := dto.EventTypeMessageEdited

	ctx, messageSpan := m.tr.Start(ctx, fmt.Sprintf("chat.%s", eventTypeV))
	defer messageSpan.End()

	adt, err := m.commonProjection.GetMessageDataForAuthorization(ctx, m.db, event.AdditionalData.BehalfUserId, event.ChatId, event.MessageId)
	if err != nil {
		return err
	}

	if !CanPublishMessage(adt.ChatCanPublishMessage, adt.IsChatAdmin, adt.MessageOwnerId, event.AdditionalData.BehalfUserId) {
		m.lgr.InfoContext(ctx, "Skipping OnMessagePublished because there is no authorization to do so", logger.AttributeChatId, event.ChatId, logger.AttributeUserId, event.AdditionalData.BehalfUserId)
		return nil
	}

	publishedCount, err := m.commonProjection.OnMessagePublished(ctx, event)
	if err != nil {
		return err
	}

	// send unpublish
	if !event.Published {
		errOuter := m.commonProjection.IterateOverChatParticipantIdsExcepting(ctx, m.db, event.ChatId, nil, func(participantIdsPortion []int64) error {
			messageViews, _, _, errInn := m.enrichingProjection.GetMessagesEnriched(ctx, participantIdsPortion, false, false, nil, event.ChatId, int32(len(participantIdsPortion)), nil, true, false, dto.NoSearchString, []int64{event.MessageId}, nil)
			if errInn != nil {
				return errInn
			}

			m.sendUnpublish(ctx, participantIdsPortion, publishedCount, event.ChatId, event.MessageId, event.AdditionalData.CreatedAt, event.AdditionalData.GetCorrelationId())

			for _, messageView := range messageViews {
				errInn = m.rabbitmqOutputEventPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.ChatEvent{
					EventType:           eventTypeMessageEdit,
					UserId:              messageView.BehalfUserId,
					ChatId:              event.ChatId,
					MessageNotification: &messageView,
				})
				if errInn != nil {
					m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
				}
			}

			return nil
		})
		if errOuter != nil {
			return errOuter
		}
	} else {
		// send publish
		errOuter := m.sendPublish(ctx, event.ChatId, event.MessageId, publishedCount, event.AdditionalData.GetCorrelationId())
		if errOuter != nil {
			return errOuter
		}
	}

	return nil
}

func (m *EventHandler) sendPublish(ctx context.Context, chatId, messageId, publishedCount int64, correlationId *string) error {
	eventTypeMessageEdit := dto.EventTypeMessageEdited

	errOuter := m.commonProjection.IterateOverChatParticipantIdsExcepting(ctx, m.db, chatId, nil, func(participantIdsPortion []int64) error {
		messageViews, _, allPortionUsers, errInn := m.enrichingProjection.GetMessagesEnriched(ctx, participantIdsPortion, false, false, nil, chatId, int32(len(participantIdsPortion)), nil, true, false, dto.NoSearchString, []int64{messageId}, nil)
		if errInn != nil {
			return errInn
		}

		// allPortionUsersMap contains message owner
		allPortionUsersMap := utils.ToMap(allPortionUsers)

		enrichedsPublished, errInn := m.enrichingProjection.GetPublishedMessageEnriched(ctx, m.db, chatId, messageId, participantIdsPortion, allPortionUsersMap)
		if errInn != nil {
			return errInn
		}

		for _, participantId := range participantIdsPortion {
			publishedMessage := enrichedsPublished[participantId]
			if publishedMessage != nil {
				errInn = m.rabbitmqOutputEventPublisher.Publish(ctx, correlationId, dto.ChatEvent{
					EventType: dto.EventTypePublishedMessageAdd,
					PublishedMessageNotification: &dto.PublishedMessageEvent{
						Message:    *publishedMessage,
						TotalCount: publishedCount,
					},
					UserId: participantId,
					ChatId: chatId,
				})
				if errInn != nil {
					m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
				}
			} else {
				m.lgr.WarnContext(ctx, "Published isn't found for the participant", logger.AttributeUserId, participantId)
			}
		}

		for _, messageView := range messageViews {
			errInn = m.rabbitmqOutputEventPublisher.Publish(ctx, correlationId, dto.ChatEvent{
				EventType:           eventTypeMessageEdit,
				UserId:              messageView.BehalfUserId,
				ChatId:              chatId,
				MessageNotification: &messageView,
			})
			if errInn != nil {
				m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
			}
		}

		return nil
	})
	if errOuter != nil {
		return errOuter
	}

	return nil
}

func (m *EventHandler) OnMessageBlogPostMade(ctx context.Context, event *MessageBlogPostMade) error {
	adt, err := m.commonProjection.GetMessageDataForAuthorization(ctx, m.db, event.AdditionalData.BehalfUserId, event.ChatId, event.MessageId)
	if err != nil {
		return err
	}

	if !CanMakeMessageBlogPost(adt.IsChatAdmin, adt.ChatIsTetATet, adt.IsMessageBlogPost, adt.IsBlog, true) {
		m.lgr.InfoContext(ctx, "Skipping OnMessageBlogPostMade because there is no authorization to do so", logger.AttributeChatId, event.ChatId, logger.AttributeUserId, event.AdditionalData.BehalfUserId)
		return nil
	}

	currentBlogPost, err := m.commonProjection.GetCurrentBlogPostMessage(ctx, m.db, event.ChatId)
	if err != nil {
		return err
	}

	err = m.commonProjection.OnMessageBlogPostMade(ctx, event)
	if err != nil {
		return err
	}

	eventType := dto.EventTypeMessageEdited

	if currentBlogPost != nil { // here, after OnMessageBlogPostMade() ex. blog post message is no more blog post
		m.lgr.DebugContext(ctx, "Sending notification about the message is no more blog post to participants", "event_type", eventType, logger.AttributeUserId, event.AdditionalData.BehalfUserId)

		errOuter := m.commonProjection.IterateOverChatParticipantIdsExcepting(ctx, m.db, event.ChatId, nil, func(participantIdsPortion []int64) error {
			messageViews, _, _, errInn := m.enrichingProjection.GetMessagesEnriched(ctx, participantIdsPortion, false, false, nil, event.ChatId, int32(len(participantIdsPortion)), nil, true, false, dto.NoSearchString, []int64{*currentBlogPost}, nil)
			if errInn != nil {
				return errInn
			}

			for _, messageView := range messageViews {
				errInn = m.rabbitmqOutputEventPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.ChatEvent{
					EventType:           eventType,
					UserId:              messageView.BehalfUserId,
					ChatId:              event.ChatId,
					MessageNotification: &messageView,
				})
				if errInn != nil {
					m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
				}
			}

			return nil
		})
		if errOuter != nil {
			return errOuter
		}
	}

	m.lgr.DebugContext(ctx, "Sending notification about the message become blog post to participants", "event_type", eventType, logger.AttributeUserId, event.AdditionalData.BehalfUserId)

	errOuter := m.commonProjection.IterateOverChatParticipantIdsExcepting(ctx, m.db, event.ChatId, nil, func(participantIdsPortion []int64) error {
		messageViews, _, _, errInn := m.enrichingProjection.GetMessagesEnriched(ctx, participantIdsPortion, false, false, nil, event.ChatId, int32(len(participantIdsPortion)), nil, true, false, dto.NoSearchString, []int64{event.MessageId}, nil)
		if errInn != nil {
			return errInn
		}

		for _, messageView := range messageViews {
			errInn = m.rabbitmqOutputEventPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.ChatEvent{
				EventType:           eventType,
				UserId:              messageView.BehalfUserId,
				ChatId:              event.ChatId,
				MessageNotification: &messageView,
			})
			if errInn != nil {
				m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
			}
		}

		return nil
	})
	if errOuter != nil {
		return errOuter
	}
	return nil
}

func (m *EventHandler) OnMessageReactionCreated(ctx context.Context, event *MessageReactionCreated) error {
	return m.onMessageReactionFlipped(ctx, event.AdditionalData, event.Metadata, event.MessageReactionCommoned, true)
}

func (m *EventHandler) OnMessageReactionRemoved(ctx context.Context, event *MessageReactionRemoved) error {
	return m.onMessageReactionFlipped(ctx, event.AdditionalData, event.Metadata, event.MessageReactionCommoned, false)
}

func (m *EventHandler) onMessageReactionFlipped(ctx context.Context, additionalData *AdditionalData, metadata *Metadata, mrc MessageReactionCommoned, created bool) error {
	ctx, messageSpan := m.tr.Start(ctx, fmt.Sprintf("message.reaction"))
	defer messageSpan.End()

	adt, err := m.commonProjection.GetChatDataForAuthorization(ctx, m.db, additionalData.BehalfUserId, mrc.ChatId)
	if err != nil {
		return err
	}

	if !CanReactOnMessage(adt.ChatCanReactOnMessage, adt.IsParticipant) {
		m.lgr.InfoContext(ctx, "Skipping OnMessageReactionCreated because there is no authorization to do so", logger.AttributeChatId, mrc.ChatId, logger.AttributeUserId, additionalData.BehalfUserId)
		return nil
	}

	var wasAdded bool
	if created {
		wasAdded, err = m.commonProjection.OnMessageReactionCreated(ctx, additionalData, metadata, mrc.ChatId, mrc.MessageId, mrc.Reaction)
		if err != nil {
			return err
		}
	} else {
		wasAdded, err = m.commonProjection.OnMessageReactionDeleted(ctx, additionalData, metadata, mrc.ChatId, mrc.MessageId, mrc.Reaction)
		if err != nil {
			return err
		}
	}

	messageBasic, err := m.commonProjection.GetMessageBasic(ctx, m.db, mrc.ChatId, mrc.MessageId)
	if err != nil {
		return err
	}

	chatNotificationTitle, err := m.commonProjection.getChatNameForNotification(ctx, m.db, mrc.ChatId)
	if err != nil {
		m.lgr.WarnContext(ctx, "Unable to get chatNotificationTitle", logger.AttributeChatId, mrc.ChatId, logger.AttributeError, err)
		// nothing
	}

	var behalfUserDto *dto.User

	reaction, err := m.commonProjection.GetReaction(ctx, m.db, mrc.ChatId, mrc.MessageId, mrc.Reaction)
	if err != nil {
		m.lgr.ErrorContext(ctx, "Error during IterateOverReactionParticipantsIds", logger.AttributeError, err)
		return nil
	}

	var wasChanged bool
	if reaction.Count > 0 {
		wasChanged = true // false means removed
	}

	toQueryUserIds := []int64{}
	toQueryUserIds = append(toQueryUserIds, additionalData.BehalfUserId)
	toQueryUserIds = append(toQueryUserIds, reaction.UserIds...)

	users, err := m.aaaRestClient.GetUsers(ctx, toQueryUserIds)
	if err != nil {
		m.lgr.WarnContext(ctx, "unable to get users")
	}
	reactionUserMap := utils.ToMap(users)
	behalfUserDto = reactionUserMap[additionalData.BehalfUserId]

	reactionUsers := make([]*dto.User, 0)
	for _, userId := range reaction.UserIds {
		user := reactionUserMap[userId]
		if user != nil {
			reactionUsers = append(reactionUsers, user)
		} else {
			reactionUsers = append(reactionUsers, getDeletedUser(userId)) // fallback
		}
	}

	var eventType string
	if wasChanged {
		eventType = dto.EventTypeReactionChanged
	} else {
		eventType = dto.EventTypeReactionRemoved
	}

	aReaction := dto.Reaction{
		Count:    reaction.Count,
		Reaction: reaction.Reaction,
		Users:    reactionUsers,
	}

	reactionChangedEvent := dto.ReactionChangedEvent{
		MessageId: mrc.MessageId,
		Reaction:  aReaction,
	}

	errOuter := m.commonProjection.IterateOverChatParticipantIdsExcepting(ctx, m.db, mrc.ChatId, []int64{}, func(participantIds []int64) error {
		for _, participantId := range participantIds {
			errInner := m.rabbitmqOutputEventPublisher.Publish(ctx, additionalData.GetCorrelationId(), dto.ChatEvent{
				EventType:            eventType,
				ReactionChangedEvent: &reactionChangedEvent,
				UserId:               participantId,
				ChatId:               mrc.ChatId,
			})
			if errInner != nil {
				m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInner)
			}
		}
		return nil
	})
	if errOuter != nil {
		return errOuter
	}

	var reactionEventType string
	if wasAdded {
		reactionEventType = dto.EventTypeReactionAdded
		re := dto.ReactionEvent{
			UserId:    additionalData.BehalfUserId,
			Reaction:  mrc.Reaction,
			MessageId: mrc.MessageId,
		}

		var messageOwnerId = messageBasic.GetOwnerId()
		if messageOwnerId == dto.NoOwner || messageOwnerId == dto.NoId {
			m.lgr.InfoContext(ctx, "Unable to get message owner for reaction notification")
		} else {
			if behalfUserDto == nil {
				m.lgr.InfoContext(ctx, "Unable to get behalf user for reply notification", logger.AttributeUserId, additionalData.BehalfUserId)
			} else {
				if messageOwnerId != additionalData.BehalfUserId { // skip myself
					err = m.rabbitmqNotificationEventsPublisher.Publish(ctx, additionalData.GetCorrelationId(), dto.NotificationEvent{
						EventType:     reactionEventType,
						ReactionEvent: &re,
						UserId:        messageOwnerId,
						ChatId:        mrc.ChatId,
						ByUserId:      behalfUserDto.Id,
						ByLogin:       behalfUserDto.Login,
						ByAvatar:      behalfUserDto.Avatar,
						ChatTitle:     chatNotificationTitle,
					})
					if err != nil {
						m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
					}
				}
			}
		}
	} else {
		reactionEventType = dto.EventTypeReactionDeleted
		re := dto.ReactionEvent{
			Reaction:  mrc.Reaction,
			MessageId: mrc.MessageId,
		}

		var messageOwnerId = messageBasic.GetOwnerId()
		if messageOwnerId == dto.NoOwner || messageOwnerId == dto.NoId {
			m.lgr.InfoContext(ctx, "Unable to get message owner for reaction notification")
		} else {
			if behalfUserDto == nil {
				m.lgr.InfoContext(ctx, "Unable to get behalf user for reply notification", logger.AttributeUserId, additionalData.BehalfUserId)
			} else {
				err = m.rabbitmqNotificationEventsPublisher.Publish(ctx, additionalData.GetCorrelationId(), dto.NotificationEvent{
					EventType:     reactionEventType,
					ReactionEvent: &re,
					UserId:        messageOwnerId,
					ChatId:        mrc.ChatId,
					ByUserId:      behalfUserDto.Id,
					ByLogin:       behalfUserDto.Login,
					ByAvatar:      behalfUserDto.Avatar,
					ChatTitle:     chatNotificationTitle,
				})
				if err != nil {
					m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
				}
			}
		}
	}

	return nil
}

func (m *EventHandler) OnChatPinned(ctx context.Context, event *ChatPinned) error {
	return m.eventBus.Publish(ctx, &UserChatPinned{
		AdditionalData: event.AdditionalData,
		ChatId:         event.ChatId,
		Pinned:         event.Pinned,
	})
}

func (m *EventHandler) OnChatNotificationSettingsSetted(ctx context.Context, event *ChatNotificationSettingsSetted) error {
	return m.eventBus.Publish(ctx, &UserChatNotificationSettingsSetted{
		AdditionalData: event.AdditionalData,
		ChatId:         event.ChatId,
		Setted:         event.Setted,
	})
}

func (m *EventHandler) OnUnreadMessageReaded(ctx context.Context, event *MessageReaded) error {
	err := m.commonProjection.OnChatUnreadMessageReaded(ctx, event)
	if err != nil {
		return err
	}
	return m.eventBus.Publish(ctx, &UserMessageReaded{
		AdditionalData:     event.AdditionalData,
		ChatId:             event.ChatId,
		MessageId:          event.MessageId,
		ReadMessagesAction: event.ReadMessagesAction,
	})
}

func (m *EventHandler) OnTechnicalProjectionsTruncated(ctx context.Context, event *ProjectionsTruncated) error {
	return m.commonProjection.OnTechnicalProjectionsTruncated(ctx, event)
}

func (m *EventHandler) OnTechnicalAbandonedChatRemoved(ctx context.Context, event *TechnicalAbandonedChatRemoved) error {
	return m.commonProjection.OnTechnicalAbandonedChatRemoved(ctx, event)
}
