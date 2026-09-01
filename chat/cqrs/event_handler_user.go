package cqrs

import (
	"context"
	"fmt"
	"slices"

	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
)

func getChatIds(events *UserChatParticipantAddedBatch) []int64 {
	res := []int64{}
	for _, v := range events.UserChatAddeds {
		res = append(res, v.ChatId)
	}

	un := utils.Unique(res)
	slices.Sort(un)

	return un
}

func (m *EventHandler) OnUserChatViewCreatedBatch(events *UserChatParticipantAddedBatch) (context.Context, error) {
	ctx := events.FirstElementContext

	eventTypeParticipantAdded := dto.EventTypeParticipantAdded

	userIds := []int64{events.UserId}

	chatIds := getChatIds(events)

	correlationIdByChatId := map[int64]*string{}
	for _, v := range events.UserChatAddeds {
		correlationIdByChatId[v.ChatId] = v.CorrelationId
	}

	err := m.commonProjection.OnUserChatViewCreated(ctx, events.UserId, events.UserChatAddeds)
	if err != nil {
		return ctx, err
	}

	eventTypeChatCreated := dto.EventTypeChatCreated
	eventTypeUnreadMessagesChanged := dto.EventTypeHasUnreadMessagesChanged

	m.lgr.DebugContext(ctx, "Sending notification about the chat to participants", "event_type", eventTypeChatCreated, "user_ids", userIds)

	// we don't need to change GetChatsEnriched to additionally process [behalf]userIds because we've already added users in our projection and the projection return all the users
	chatViews, _, err := m.enrichingProjection.GetChatsEnriched(ctx, userIds, int32(len(userIds)), nil, true, false, false, dto.NoSearchString, chatIds, false)
	if err != nil {
		return ctx, err
	}

	var hasUnreadMessages = map[int64]bool{}
	hasUnreadMessages, err = m.commonProjection.GetHasUnreadMessages(ctx, userIds)
	if err != nil {
		return ctx, err
	}

	var firstCorrelationId *string

	for _, cv := range chatViews {
		dt := dto.GlobalUserEvent{
			UserId:           cv.BehalfUserId,
			EventType:        eventTypeChatCreated,
			ChatNotification: &cv,
		}

		correlationId := correlationIdByChatId[cv.Id]
		if firstCorrelationId == nil {
			firstCorrelationId = correlationId
		}

		err = m.rabbitmqOutputEventPublisher.Publish(ctx, correlationId, dt)
		if err != nil {
			m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
		}

		if cv.TetATet {
			err = m.rabbitmqOutputEventPublisher.Publish(ctx, correlationId, dto.GlobalUserEvent{
				UserId:    cv.BehalfUserId,
				EventType: dto.EventTypeChatTetATetUpserted,
				ChatTetATetUpsertedDto: &dto.ChatTetATetUpsertedDto{
					ChatId: cv.Id,
				},
			})
			if err != nil {
				m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
			}
		}
	}

	err = m.rabbitmqOutputEventPublisher.Publish(ctx, firstCorrelationId, dto.GlobalUserEvent{
		UserId:    events.UserId,
		EventType: eventTypeUnreadMessagesChanged,
		HasUnreadMessagesChanged: &dto.HasUnreadMessagesChanged{
			HasUnreadMessages: hasUnreadMessages[events.UserId],
		},
	})
	if err != nil {
		m.lgr.ErrorContext(ctx, "Error during IterateOverParticipantsChatIds", logger.AttributeError, err)
	}

	m.lgr.DebugContext(ctx, "Sending notification about the participants", "event_type", eventTypeParticipantAdded, "user_ids", userIds)

	users, err := m.aaaRestClient.GetUsers(ctx, []int64{events.UserId})
	if err != nil {
		m.lgr.WarnContext(ctx, "unable to get users")
	}

	usersMap := utils.ToMap(users)

	consideredUserAdminByChatIds, err := m.enrichingProjection.cp.getAreAdminsOfChatIds(ctx, m.enrichingProjection.cp.db, events.UserId, chatIds)
	if err != nil {
		return nil, err
	}

	chatsById, err := m.enrichingProjection.cp.GetChatsBasic(ctx, m.enrichingProjection.cp.db, chatIds)
	if err != nil {
		m.lgr.ErrorContext(ctx, "unable to get chats")
		return nil, err
	}

	// send an event participant_added for ChatParticipantsModal.vue to all the participants of the chosen chats
	err = m.commonProjection.IterateOverAllParticipantsByChatIds(ctx, m.db, chatIds, func(participantIdsPortion []*ParticipantWithChatIdWithAdmin) error {
		participantsByChatIds, errInn := m.enrichingProjection.GetParticipantsBehalfOfGivenParticipants(ctx, participantIdsPortion, events.UserId, consideredUserAdminByChatIds, usersMap, chatsById)
		if errInn != nil {
			return errInn
		}

		// for every participant of chat we send an info about the newly added participants
		for _, participant := range participantIdsPortion {
			hisParticipantsView := participantsByChatIds[ChatIdUserId{ChatId: participant.ChatId, UserId: participant.ParticipantId}]
			if hisParticipantsView != nil {
				errInn = m.rabbitmqOutputEventPublisher.Publish(ctx, correlationIdByChatId[participant.ChatId], dto.ChatEvent{
					EventType:    eventTypeParticipantAdded,
					UserId:       participant.ParticipantId,
					ChatId:       participant.ChatId,
					Participants: &[]*dto.UserViewEnrichedDto{hisParticipantsView},
				})
				if errInn != nil {
					m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, errInn)
				}
			} else {
				m.lgr.InfoContext(ctx, "Not found view behalf", logger.AttributeChatId, participant.ChatId, logger.AttributeUserId, participant.ParticipantId)
			}
		}
		return nil
	})
	if err != nil {
		m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
	}

	return ctx, nil
}

func (m *EventHandler) OnUserChatViewUpdated(ctx context.Context, event *UserChatEdited) error {
	eventType := dto.EventTypeChatEdited

	ctx, messageSpan := m.tr.Start(ctx, fmt.Sprintf("chat.%s", eventType))
	defer messageSpan.End()

	userIds := []int64{event.UserId}

	m.lgr.DebugContext(ctx, "Sending notification about the chat to participants", "event_type", eventType, logger.AttributeUserId, event.UserId)

	if event.ChatAction == ChatActionRefresh {
		errp := m.commonProjection.OnChatViewRefreshedForPartitionUser(ctx, event.EventTime, event.UserId, event.ChatId)
		if errp != nil {
			return errp
		}
	} else if event.ChatAction == ChatActionRedraw {
		eventType = dto.EventTypeChatRedraw
	}

	chatViews, _, err := m.enrichingProjection.GetChatsEnriched(ctx, userIds, int32(len(userIds)), nil, true, false, false, dto.NoSearchString, []int64{event.ChatId}, false)
	if err != nil {
		return err
	}

	for _, cv := range chatViews {
		err = m.rabbitmqOutputEventPublisher.Publish(ctx, event.CorrelationId, dto.GlobalUserEvent{
			UserId:           cv.BehalfUserId,
			EventType:        eventType,
			ChatNotification: &cv,
		})
		if err != nil {
			m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
		}
	}
	return nil

}

func (m *EventHandler) OnUserChatViewRemoved(ctx context.Context, event *UserChatParticipantRemoved) error {
	eventType := dto.EventTypeChatDeleted
	eventTypeUnreadMessagesChanged := dto.EventTypeHasUnreadMessagesChanged

	err := m.commonProjection.OnParticipantRemovedSingle(ctx, event.UserId, event.ChatId, event.WereRemovedUsersFromAaa)
	if err != nil {
		return err
	}

	var hasUnreadMessages = map[int64]bool{}
	hasUnreadMessages, err = m.commonProjection.GetHasUnreadMessages(ctx, []int64{event.UserId})
	if err != nil {
		return err
	}

	if !event.IsChatPubliclyAvailable || event.IsChatRemoving {
		err = m.rabbitmqOutputEventPublisher.Publish(ctx, event.CorrelationId, dto.GlobalUserEvent{
			UserId:         event.UserId,
			EventType:      eventType,
			ChatDeletedDto: &dto.ChatDeletedDto{Id: event.ChatId},
		})
		if err != nil {
			m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
		}
	}

	err = m.rabbitmqOutputEventPublisher.Publish(ctx, event.CorrelationId, dto.GlobalUserEvent{
		UserId:    event.UserId,
		EventType: eventTypeUnreadMessagesChanged,
		HasUnreadMessagesChanged: &dto.HasUnreadMessagesChanged{
			HasUnreadMessages: hasUnreadMessages[event.UserId],
		},
	})
	if err != nil {
		m.lgr.ErrorContext(ctx, "Error during IterateOverParticipantsChatIds", logger.AttributeError, err)
	}

	return nil
}

func (m *EventHandler) OnUserUnreadMessageReaded(ctx context.Context, event *UserMessageReaded) error {
	userIds := []int64{event.AdditionalData.BehalfUserId}

	eventTypeUnreadMessagesChanged := dto.EventTypeHasUnreadMessagesChanged
	eventTypeChatUnreadMessagesChanged := dto.EventTypeChatUnreadMessagesChanged

	ctx, messageSpan := m.tr.Start(ctx, fmt.Sprintf("message.%s", eventTypeChatUnreadMessagesChanged))
	defer messageSpan.End()

	if event.ReadMessagesAction == ReadMessagesActionOneMessage || event.ReadMessagesAction == ReadMessagesActionAllMessagesInOneChat {
		adt, err := m.commonProjection.GetMessageDataForAuthorization(ctx, m.db, event.AdditionalData.BehalfUserId, event.ChatId, event.MessageId)
		if err != nil {
			return err
		}

		if !CanReadMessage(adt.IsParticipant) {
			m.lgr.InfoContext(ctx, "Skipping OnUnreadMessageReaded because there is no authorization to do so", logger.AttributeChatId, event.ChatId, logger.AttributeUserId, event.AdditionalData.BehalfUserId)
			return nil
		}
	}

	err := m.commonProjection.OnUserUnreadMessageReaded(ctx, event, func(updatedChatsPortion []dto.ChatUserViewBasic) {
		if event.ReadMessagesAction != ReadMessagesActionAllChats {
			m.lgr.ErrorContext(ctx, "wrong invariant: a logical error in commonProjection.OnUnreadMessageReaded")
			return
		}

		for _, cvb := range updatedChatsPortion {
			err := m.rabbitmqOutputEventPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.GlobalUserEvent{
				UserId:    event.AdditionalData.BehalfUserId,
				EventType: eventTypeChatUnreadMessagesChanged,
				UnreadMessagesNotification: &dto.ChatUnreadMessageChanged{
					ChatId:             cvb.ChatId,
					UnreadMessages:     cvb.UnreadMessages,
					LastUpdateDateTime: cvb.UpdateDateTime,
				},
			})
			if err != nil {
				m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
			}
		}
	})
	if err != nil {
		return err
	}

	var hasUnreadMessages = map[int64]bool{}
	hasUnreadMessages, err = m.commonProjection.GetHasUnreadMessages(ctx, userIds)
	if err != nil {
		return err
	}

	if event.ReadMessagesAction == ReadMessagesActionOneMessage || event.ReadMessagesAction == ReadMessagesActionAllMessagesInOneChat {
		// not.NotifyAboutUnreadMessage(ctx, chatId, participantId, unreadMessagesByUserId[participantId], lastUpdated)
		cvb, err := m.commonProjection.GetChatUserViewBasic(ctx, m.db, event.ChatId, event.AdditionalData.BehalfUserId)
		if err != nil {
			m.lgr.ErrorContext(ctx, "Error during getting chat UserViewBasic", logger.AttributeError, err)
		} else {
			err = m.rabbitmqOutputEventPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.GlobalUserEvent{
				UserId:    event.AdditionalData.BehalfUserId,
				EventType: eventTypeChatUnreadMessagesChanged,
				UnreadMessagesNotification: &dto.ChatUnreadMessageChanged{
					ChatId:             cvb.ChatId,
					UnreadMessages:     cvb.UnreadMessages,
					LastUpdateDateTime: cvb.UpdateDateTime,
				},
			})
			if err != nil {
				m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
			}
		}
	} else if event.ReadMessagesAction == ReadMessagesActionAllChats {
		// nothing, see the callback of commonProjection.OnUnreadMessageReaded()
	} else {
		return fmt.Errorf("Unknown action: %T", event.ReadMessagesAction)
	}

	err = m.rabbitmqOutputEventPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.GlobalUserEvent{
		UserId:    event.AdditionalData.BehalfUserId,
		EventType: eventTypeUnreadMessagesChanged,
		HasUnreadMessagesChanged: &dto.HasUnreadMessagesChanged{
			HasUnreadMessages: hasUnreadMessages[event.AdditionalData.BehalfUserId],
		},
	})
	if err != nil {
		m.lgr.ErrorContext(ctx, "Error during IterateOverParticipantsChatIds", logger.AttributeError, err)
	}

	return nil
}

func (m *EventHandler) OnUserChatPinned(ctx context.Context, event *UserChatPinned) error {
	// we don't check authorization here because all the participants can pin chat (their chat_user_view)
	err := m.commonProjection.OnChatPinned(ctx, event)
	if err != nil {
		return err
	}

	userIds := []int64{event.AdditionalData.BehalfUserId}

	chatViews, _, err := m.enrichingProjection.GetChatsEnriched(ctx, userIds, int32(len(userIds)), nil, true, false, false, dto.NoSearchString, []int64{event.ChatId}, false)
	if err != nil {
		return err
	}

	for _, cv := range chatViews {
		dt := dto.GlobalUserEvent{
			UserId:           cv.BehalfUserId,
			EventType:        dto.EventTypeChatEdited,
			ChatNotification: &cv,
		}
		err = m.rabbitmqOutputEventPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dt)
		if err != nil {
			m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
		}
	}

	return nil
}

func (m *EventHandler) OnUserChatNotificationSettingsSetted(ctx context.Context, event *UserChatNotificationSettingsSetted) error {
	// we don't check authorization here because all the participants can change their notification setting
	err := m.commonProjection.OnChatNotificationSettingsSetted(ctx, event)
	if err != nil {
		return err
	}

	d := dto.ChatNotificationSettingsChanged{
		ChatId:                   event.ChatId,
		ConsiderMessagesAsUnread: event.Setted,
	}

	err = m.rabbitmqOutputEventPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.GlobalUserEvent{
		UserId:                          event.AdditionalData.BehalfUserId,
		EventType:                       dto.EventTypeChatNotificationSettingsChanged,
		ChatNotificationSettingsChanged: &d,
	})
	if err != nil {
		m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
	}

	hasUnreadMessages, err := m.commonProjection.GetHasUnreadMessages(ctx, []int64{event.AdditionalData.BehalfUserId})
	if err != nil {
		return err
	}

	err = m.rabbitmqOutputEventPublisher.Publish(ctx, event.AdditionalData.GetCorrelationId(), dto.GlobalUserEvent{
		UserId:    event.AdditionalData.BehalfUserId,
		EventType: dto.EventTypeHasUnreadMessagesChanged,
		HasUnreadMessagesChanged: &dto.HasUnreadMessagesChanged{
			HasUnreadMessages: hasUnreadMessages[event.AdditionalData.BehalfUserId],
		},
	})
	if err != nil {
		m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
	}

	return nil
}

func (m *EventHandler) OnUserMessagesCreated(ctx context.Context, event *UserMessagesCreated) error {
	eventTypeChatUnreadMessagesChanged := dto.EventTypeChatUnreadMessagesChanged

	if len(event.MessageCreateds) == 0 {
		m.lgr.InfoContext(ctx, "Zero MessageCreateds", logger.AttributeChatId, event.ChatId, logger.AttributeUserId, event.UserId)
		return nil
	}

	err := m.commonProjection.OnUserMessagesCreated(ctx, m.db, event)
	if err != nil {
		return err
	}

	cvb, err := m.commonProjection.GetChatUserViewBasic(ctx, m.db, event.ChatId, event.UserId)
	if err != nil {
		m.lgr.ErrorContext(ctx, "Error during getting chat UserViewBasic", logger.AttributeError, err)
	} else {
		err = m.rabbitmqOutputEventPublisher.Publish(ctx, event.MessageCreateds[0].AdditionalData.GetCorrelationId(), dto.GlobalUserEvent{
			UserId:    event.UserId,
			EventType: eventTypeChatUnreadMessagesChanged,
			UnreadMessagesNotification: &dto.ChatUnreadMessageChanged{
				ChatId:             cvb.ChatId,
				UnreadMessages:     cvb.UnreadMessages,
				LastUpdateDateTime: cvb.UpdateDateTime,
			},
		})
		if err != nil {
			m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
		}
	}

	hasUnreadMessages, err := m.commonProjection.GetHasUnreadMessages(ctx, []int64{event.UserId})
	if err != nil {
		return err
	}

	err = m.rabbitmqOutputEventPublisher.Publish(ctx, event.MessageCreateds[0].AdditionalData.GetCorrelationId(), dto.GlobalUserEvent{
		UserId:    event.UserId,
		EventType: dto.EventTypeHasUnreadMessagesChanged,
		HasUnreadMessagesChanged: &dto.HasUnreadMessagesChanged{
			HasUnreadMessages: hasUnreadMessages[event.UserId],
		},
	})
	if err != nil {
		m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
	}

	return nil
}

func (m *EventHandler) OnUserMessagesDeleted(ctx context.Context, event *UserMessageDeleted) error {
	eventTypeChatUnreadMessagesChanged := dto.EventTypeChatUnreadMessagesChanged

	err := m.commonProjection.OnUserMessageDeleted(ctx, m.db, event)
	if err != nil {
		return err
	}

	cvb, err := m.commonProjection.GetChatUserViewBasic(ctx, m.db, event.ChatId, event.UserId)
	if err != nil {
		m.lgr.ErrorContext(ctx, "Error during getting chat UserViewBasic", logger.AttributeError, err)
	} else {
		err = m.rabbitmqOutputEventPublisher.Publish(ctx, event.CorrelationId, dto.GlobalUserEvent{
			UserId:    event.UserId,
			EventType: eventTypeChatUnreadMessagesChanged,
			UnreadMessagesNotification: &dto.ChatUnreadMessageChanged{
				ChatId:             cvb.ChatId,
				UnreadMessages:     cvb.UnreadMessages,
				LastUpdateDateTime: cvb.UpdateDateTime,
			},
		})
		if err != nil {
			m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
		}
	}

	hasUnreadMessages, err := m.commonProjection.GetHasUnreadMessages(ctx, []int64{event.UserId})
	if err != nil {
		return err
	}

	err = m.rabbitmqOutputEventPublisher.Publish(ctx, event.CorrelationId, dto.GlobalUserEvent{
		UserId:    event.UserId,
		EventType: dto.EventTypeHasUnreadMessagesChanged,
		HasUnreadMessagesChanged: &dto.HasUnreadMessagesChanged{
			HasUnreadMessages: hasUnreadMessages[event.UserId],
		},
	})
	if err != nil {
		m.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
	}

	return nil

}
