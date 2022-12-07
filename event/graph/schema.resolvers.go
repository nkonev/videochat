package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"errors"
	"time"

	"github.com/montag451/go-eventbus"
	"nkonev.name/event/auth"
	"nkonev.name/event/dto"
	"nkonev.name/event/graph/generated"
	"nkonev.name/event/graph/model"
	"nkonev.name/event/logger"
	"nkonev.name/event/utils"
)

// Ping is the resolver for the ping field.
func (r *queryResolver) Ping(ctx context.Context) (*bool, error) {
	res := true
	return &res, nil
}

// ChatEvents is the resolver for the chatEvents field.
func (r *subscriptionResolver) ChatEvents(ctx context.Context, chatID int64) (<-chan *model.ChatEvent, error) {
	authResult, ok := ctx.Value(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		return nil, errors.New("Unable to get auth context")
	}

	hasAccess, err := r.HttpClient.CheckAccess(authResult.UserId, chatID, ctx)
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during checking participant user %v, chat %v", authResult.UserId, chatID)
		return nil, err
	}
	if !hasAccess {
		logger.GetLogEntry(ctx).Infof("User %v is not participant of chat %v", authResult.UserId, chatID)
		return nil, errors.New("Unauthorized")
	}
	logger.GetLogEntry(ctx).Infof("Subscribing to chatEvents channel as user %v", authResult.UserId)

	var cam = make(chan *model.ChatEvent)
	subscribeHandler, err := r.Bus.Subscribe(dto.CHAT_EVENTS, func(event eventbus.Event, t time.Time) {
		switch typedEvent := event.(type) {
		case dto.ChatEvent:
			if isReceiverOfEvent(typedEvent.UserId, authResult) && typedEvent.ChatId == chatID {
				cam <- convertToChatEvent(&typedEvent)
			}
			break
		default:
			logger.GetLogEntry(ctx).Debugf("Skipping %v as is no mapping here for this type, user %v, chat %v", typedEvent, authResult.UserId, chatID)
		}
	})
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during creating eventbus subscription user %v, chat %v", authResult.UserId, chatID)
		return nil, err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.GetLogEntry(ctx).Infof("Closing chatEvents channel for user %v", authResult.UserId)
				err := r.Bus.Unsubscribe(subscribeHandler)
				if err != nil {
					logger.GetLogEntry(ctx).Errorf("Error during unsubscribing from bus in chatEvents channel for user %v", authResult.UserId)
				}
				close(cam)
				return
			}
		}
	}()

	return cam, nil
}

// GlobalEvents is the resolver for the globalEvents field.
func (r *subscriptionResolver) GlobalEvents(ctx context.Context) (<-chan *model.GlobalEvent, error) {
	authResult, ok := ctx.Value(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		return nil, errors.New("Unable to get auth context")
	}
	logger.GetLogEntry(ctx).Infof("Subscribing to globalEvents channel as user %v", authResult.UserId)

	var cam = make(chan *model.GlobalEvent)
	subscribeHandler, err := r.Bus.Subscribe(dto.GLOBAL_EVENTS, func(event eventbus.Event, t time.Time) {
		switch typedEvent := event.(type) {
		case dto.GlobalEvent:
			if isReceiverOfEvent(typedEvent.UserId, authResult) {
				cam <- convertToGlobalEvent(&typedEvent)
			}
			break
		default:
			logger.GetLogEntry(ctx).Debugf("Skipping %v as is no mapping here for this type, user %v", typedEvent, authResult.UserId)
		}
	})
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during creating eventbus subscription user %v", authResult.UserId)
		return nil, err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.GetLogEntry(ctx).Infof("Closing globalEvents channel for user %v", authResult.UserId)
				err := r.Bus.Unsubscribe(subscribeHandler)
				if err != nil {
					logger.GetLogEntry(ctx).Errorf("Error during unsubscribing from bus in globalEvents channel for user %v", authResult.UserId)
				}
				close(cam)
				return
			}
		}
	}()

	return cam, nil
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

// Subscription returns generated.SubscriptionResolver implementation.
func (r *Resolver) Subscription() generated.SubscriptionResolver { return &subscriptionResolver{r} }

type queryResolver struct{ *Resolver }
type subscriptionResolver struct{ *Resolver }

// !!! WARNING !!!
// The code below was going to be deleted when updating resolvers. It has been copied here so you have
// one last chance to move it out of harms way if you want. There are two reasons this happens:
//   - When renaming or deleting a resolver the old code will be put in here. You can safely delete
//     it when you're done.
//   - You have helper methods in this file. Move them out to keep these resolver files clean.
func convertToChatEvent(e *dto.ChatEvent) *model.ChatEvent {
	var result = &model.ChatEvent{
		EventType: e.EventType,
	}
	notificationDto := e.MessageNotification
	if notificationDto != nil {
		result.MessageEvent = &model.DisplayMessageDto{ // dto.DisplayMessageDto
			ID:             notificationDto.Id,
			Text:           notificationDto.Text,
			ChatID:         notificationDto.ChatId,
			OwnerID:        notificationDto.OwnerId,
			CreateDateTime: notificationDto.CreateDateTime,
			EditDateTime:   notificationDto.EditDateTime.Ptr(),
			Owner:          convertUser(notificationDto.Owner),
			CanEdit:        notificationDto.CanEdit,
			CanDelete:      notificationDto.CanDelete,
			FileItemUUID:   notificationDto.FileItemUuid,
		}
	}

	messageDeleted := e.MessageDeletedNotification
	if messageDeleted != nil {
		result.MessageDeletedEvent = &model.MessageDeletedDto{
			ID:     messageDeleted.Id,
			ChatID: messageDeleted.ChatId,
		}
	}

	userTypingEvent := e.UserTypingNotification
	if userTypingEvent != nil {
		result.UserTypingEvent = &model.UserTypingDto{
			Login:         userTypingEvent.Login,
			ParticipantID: userTypingEvent.ParticipantId,
		}
	}
	messageBroadcast := e.MessageBroadcastNotification
	if messageBroadcast != nil {
		result.MessageBroadcastEvent = &model.MessageBroadcastNotification{
			Login:  messageBroadcast.Login,
			UserID: messageBroadcast.UserId,
			Text:   messageBroadcast.Text,
		}
	}
	return result
}
func convertToGlobalEvent(e *dto.GlobalEvent) *model.GlobalEvent {
	//eventType string, chatDtoWithAdmin *dto.ChatDtoWithAdmin
	var ret = &model.GlobalEvent{
		EventType: e.EventType,
	}
	chatDtoWithAdmin := e.ChatNotification
	if chatDtoWithAdmin != nil {
		ret.ChatEvent = &model.ChatDto{ // dto.ChatDtoWithAdmin
			ID:                       chatDtoWithAdmin.Id,
			Name:                     chatDtoWithAdmin.Name,
			Avatar:                   chatDtoWithAdmin.Avatar.Ptr(),
			AvatarBig:                chatDtoWithAdmin.AvatarBig.Ptr(),
			LastUpdateDateTime:       chatDtoWithAdmin.LastUpdateDateTime,
			ParticipantIds:           chatDtoWithAdmin.ParticipantIds,
			CanEdit:                  chatDtoWithAdmin.CanEdit.Ptr(),
			CanDelete:                chatDtoWithAdmin.CanDelete.Ptr(),
			CanLeave:                 chatDtoWithAdmin.CanLeave.Ptr(),
			UnreadMessages:           chatDtoWithAdmin.UnreadMessages,
			CanBroadcast:             chatDtoWithAdmin.CanBroadcast,
			CanVideoKick:             chatDtoWithAdmin.CanVideoKick,
			CanAudioMute:             chatDtoWithAdmin.CanAudioMute,
			CanChangeChatAdmins:      chatDtoWithAdmin.CanChangeChatAdmins,
			TetATet:                  chatDtoWithAdmin.IsTetATet,
			ParticipantsCount:        chatDtoWithAdmin.ParticipantsCount,
			ChangingParticipantsPage: chatDtoWithAdmin.ChangingParticipantsPage,
			Participants:             convertUsers(chatDtoWithAdmin.Participants),
		}
	}

	chatDeleted := e.ChatDeletedDto
	if chatDeleted != nil {
		ret.ChatDeletedEvent = &model.ChatDeletedDto{
			ID: chatDeleted.Id,
		}
	}

	userProfileDto := e.UserProfileNotification
	if userProfileDto != nil {
		ret.UserEvent = &model.User{
			ID:     userProfileDto.Id,
			Login:  userProfileDto.Login,
			Avatar: userProfileDto.Avatar.Ptr(),
		}
	}

	videoUserCountEvent := e.VideoCallUserCountEvent
	if videoUserCountEvent != nil {
		ret.VideoUserCountChangedEvent = &model.VideoUserCountChangedDto{
			UsersCount: videoUserCountEvent.UsersCount,
			ChatID:     videoUserCountEvent.ChatId,
		}
	}

	videoRecordingEvent := e.VideoCallRecordingEvent
	if videoRecordingEvent != nil {
		ret.VideoRecordingChangedEvent = &model.VideoRecordingChangedDto{
			RecordInProgress: videoRecordingEvent.RecordInProgress,
			ChatID:           videoRecordingEvent.ChatId,
		}
	}

	videoChatInvite := e.VideoChatInvitation
	if videoChatInvite != nil {
		ret.VideoCallInvitation = &model.VideoCallInvitationDto{
			ChatID:   videoChatInvite.ChatId,
			ChatName: videoChatInvite.ChatName,
		}
	}

	videoDial := e.VideoParticipantDialEvent
	if videoDial != nil {
		ret.VideoParticipantDialEvent = &model.VideoDialChanges{
			ChatID: videoDial.ChatId,
			Dials:  convertDials(videoDial.Dials),
		}
	}

	unreadMessages := e.UnreadMessagesNotification
	if unreadMessages != nil {
		ret.UnreadMessagesNotification = &model.ChatUnreadMessageChanged{
			ChatID:         unreadMessages.ChatId,
			UnreadMessages: unreadMessages.UnreadMessages,
		}
	}

	allUnreadMessages := e.AllUnreadMessagesNotification
	if allUnreadMessages != nil {
		ret.AllUnreadMessagesNotification = &model.AllUnreadMessages{
			AllUnreadMessages: allUnreadMessages.MessagesCount,
		}
	}

	userNotification := e.UserNotificationEvent
	if userNotification != nil {
		ret.NotificationEvent = &model.NotificationDto{
			ID:               userNotification.Id,
			ChatID:           userNotification.ChatId,
			MessageID:        userNotification.MessageId,
			NotificationType: userNotification.NotificationType,
			Description:      userNotification.Description,
			CreateDateTime:   userNotification.CreateDateTime,
		}
	}

	return ret
}
func convertUser(owner *dto.User) *model.User {
	if owner == nil {
		return nil
	}
	return &model.User{
		ID:     owner.Id,
		Login:  owner.Login,
		Avatar: owner.Avatar.Ptr(),
	}
}
func convertUserWithAdmin(owner *dto.UserWithAdmin) *model.UserWithAdmin {
	if owner == nil {
		return nil
	}
	return &model.UserWithAdmin{
		ID:     owner.Id,
		Login:  owner.Login,
		Avatar: owner.Avatar.Ptr(),
		Admin:  owner.Admin,
	}
}
func convertUsers(participants []*dto.UserWithAdmin) []*model.UserWithAdmin {
	if participants == nil {
		return nil
	}
	usrs := []*model.UserWithAdmin{}
	for _, user := range participants {
		usrs = append(usrs, convertUserWithAdmin(user))
	}
	return usrs
}
func convertDials(dials []*dto.VideoDialChanged) []*model.VideoDialChanged {
	if dials == nil {
		return nil
	}
	dls := []*model.VideoDialChanged{}
	for _, dl := range dials {
		dls = append(dls, convertDial(dl))
	}
	return dls
}
func convertDial(dl *dto.VideoDialChanged) *model.VideoDialChanged {
	if dl == nil {
		return nil
	}
	return &model.VideoDialChanged{
		UserID: dl.UserId,
		Status: dl.Status,
	}
}
func isReceiverOfEvent(userId int64, authResult *auth.AuthResult) bool {
	return userId == authResult.UserId
}
