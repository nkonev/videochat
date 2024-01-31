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
		defer func() {
			if err := recover(); err != nil {
				logger.GetLogEntry(ctx).Errorf("In processing ChatEvents panic recovered: %v", err)
			}
		}()

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
	subscribeHandler, err := r.Bus.Subscribe(dto.GLOBAL_USER_EVENTS, func(event eventbus.Event, t time.Time) {
		defer func() {
			if err := recover(); err != nil {
				logger.GetLogEntry(ctx).Errorf("In processing GlobalEvents panic recovered: %v", err)
			}
		}()

		switch typedEvent := event.(type) {
		case dto.GlobalUserEvent:
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

// UserStatusEvents is the resolver for the userStatusEvents field.
func (r *subscriptionResolver) UserStatusEvents(ctx context.Context, userIds []int64) (<-chan []*model.UserStatusEvent, error) {
	// user online
	authResult, ok := ctx.Value(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		return nil, errors.New("Unable to get auth context")
	}
	logger.GetLogEntry(ctx).Infof("Subscribing to UserOnline channel as user %v", authResult.UserId)

	var cam = make(chan []*model.UserStatusEvent)

	subscribeHandlerUserOnline, err := r.Bus.Subscribe(dto.USER_ONLINE, func(event eventbus.Event, t time.Time) {
		defer func() {
			if err := recover(); err != nil {
				logger.GetLogEntry(ctx).Errorf("In processing UserOnline panic recovered: %v", err)
			}
		}()

		switch typedEvent := event.(type) {
		case dto.ArrayUserOnline:
			var batch = []*model.UserStatusEvent{}
			for _, userOnline := range typedEvent {
				if utils.Contains(userIds, userOnline.UserId) {
					batch = append(batch, convertToUserOnline(userOnline))
				}
			}
			if len(batch) > 0 {
				cam <- batch
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

	subscribeHandlerVideoCallStatus, err := r.Bus.Subscribe(dto.GENERAL, func(event eventbus.Event, t time.Time) {
		defer func() {
			if err := recover(); err != nil {
				logger.GetLogEntry(ctx).Errorf("In processing UserVideoStatus panic recovered: %v", err)
			}
		}()

		switch typedEvent := event.(type) {
		case dto.GeneralEvent:
			var videoCallUsersCallStatusChangedEvent = typedEvent.VideoCallUsersCallStatusChangedEvent
			if videoCallUsersCallStatusChangedEvent != nil {
				var batch = []*model.UserStatusEvent{}
				for _, userCallStatus := range videoCallUsersCallStatusChangedEvent.Users {
					if utils.Contains(userIds, userCallStatus.UserId) {
						batch = append(batch, convertToUserCallStatusChanged(typedEvent, userCallStatus))
					}
				}
				if len(batch) > 0 {
					cam <- batch
				}
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
				logger.GetLogEntry(ctx).Infof("Closing UserOnline channel for user %v", authResult.UserId)
				err := r.Bus.Unsubscribe(subscribeHandlerUserOnline)
				if err != nil {
					logger.GetLogEntry(ctx).Errorf("Error during unsubscribing from bus in UserOnline channel for user %v", authResult.UserId)
				}

				logger.GetLogEntry(ctx).Infof("Closing UserVideoStatus channel for user %v", authResult.UserId)
				err = r.Bus.Unsubscribe(subscribeHandlerVideoCallStatus)
				if err != nil {
					logger.GetLogEntry(ctx).Errorf("Error during unsubscribing from bus in UserVideoStatus channel for user %v", authResult.UserId)
				}

				close(cam)
				return
			}
		}
	}()

	r.HttpClient.AskForUserOnline(userIds, ctx)

	return cam, nil
}

// UserAccountEvents is the resolver for the userAccountEvents field.
func (r *subscriptionResolver) UserAccountEvents(ctx context.Context, userIds []int64) (<-chan *model.UserAccountEvent, error) {
	// user online
	authResult, ok := ctx.Value(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		return nil, errors.New("Unable to get auth context")
	}
	logger.GetLogEntry(ctx).Infof("Subscribing to UserAccount channel as user %v", authResult.UserId)

	var cam = make(chan *model.UserAccountEvent)

	subscribeHandler, err := r.Bus.Subscribe(dto.AAA, func(event eventbus.Event, t time.Time) {
		defer func() {
			if err := recover(); err != nil {
				logger.GetLogEntry(ctx).Errorf("In processing UserAccount panic recovered: %v", err)
			}
		}()

		switch typedEvent := event.(type) {
		case dto.UserAccountEventGroup:
			// prepare dto and send it to channel for myself (if utils.Contains()). Remove this id from userIds and go ahead
			if authResult.UserId == typedEvent.UserId && utils.Contains(userIds, typedEvent.UserId) {
				var anEvent = convertUserAccountEventExtended(typedEvent.EventType, typedEvent.ForMyself)
				if anEvent != nil {
					cam <- anEvent
				}
				break
			}
			// if I'm an admin then prepare dto with admin's fields
			if utils.ContainsString(authResult.Roles, "ROLE_ADMIN") && utils.Contains(userIds, typedEvent.UserId) {
				var anEvent = convertUserAccountEventExtended(typedEvent.EventType, typedEvent.ForRoleAdmin)
				if anEvent != nil {
					cam <- anEvent
				}
				break
			}
			// else if I'm un user then prepare dto with user's fields
			if utils.ContainsString(authResult.Roles, "ROLE_USER") && utils.Contains(userIds, typedEvent.UserId) {
				var anEvent = convertUserAccountEvent(typedEvent.EventType, typedEvent.ForRoleUser)
				if anEvent != nil {
					cam <- anEvent
				}
				break
			}
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
				logger.GetLogEntry(ctx).Infof("Closing UserAccount channel for user %v", authResult.UserId)
				err := r.Bus.Unsubscribe(subscribeHandler)
				if err != nil {
					logger.GetLogEntry(ctx).Errorf("Error during unsubscribing from bus in UserAccount channel for user %v", authResult.UserId)
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
func convertUserAccountEvent(eventType string, aDto *dto.UserAccount) *model.UserAccountEvent {
	if aDto != nil {
		ret := model.UserAccountEvent{}
		ret.EventType = eventType
		ret.UserAccountEvent = &model.UserAccountDto{
			ID:                aDto.Id,
			Login:             aDto.Login,
			Avatar:            aDto.Avatar,
			AvatarBig:         aDto.AvatarBig,
			ShortInfo:         aDto.ShortInfo,
			LastLoginDateTime: aDto.LastLoginDateTime,
			Oauth2Identifiers: convertOauth2Identifiers(aDto.Oauth2Identifiers),
		}
		return &ret
	}
	return nil
}
func convertUserAccountEventExtended(eventType string, aDto *dto.UserAccountExtended) *model.UserAccountEvent {
	if aDto != nil {
		ret := model.UserAccountEvent{}
		ret.EventType = eventType

		ret.UserAccountEvent = &model.UserAccountExtendedDto{
			ID:                aDto.Id,
			Login:             aDto.Login,
			Avatar:            aDto.Avatar,
			AvatarBig:         aDto.AvatarBig,
			ShortInfo:         aDto.ShortInfo,
			LastLoginDateTime: aDto.LastLoginDateTime,
			Oauth2Identifiers: convertOauth2Identifiers(aDto.Oauth2Identifiers),
			AdditionalData: &model.DataDto{
				Enabled:   aDto.AdditionalData.Enabled,
				Expired:   aDto.AdditionalData.Expired,
				Locked:    aDto.AdditionalData.Locked,
				Confirmed: aDto.AdditionalData.Confirmed,
				Roles:     aDto.AdditionalData.Roles,
			},
			CanLock:       aDto.CanLock,
			CanDelete:     aDto.CanDelete,
			CanChangeRole: aDto.CanChangeRole,
		}
		return &ret
	}
	return nil
}
func convertOauth2Identifiers(identifiers *dto.Oauth2Identifiers) *model.OAuth2Identifiers {
	if identifiers == nil {
		return nil
	}
	return &model.OAuth2Identifiers{
		FacebookID:  identifiers.FacebookId,
		VkontakteID: identifiers.VkontakteId,
		GoogleID:    identifiers.GoogleId,
		KeycloakID:  identifiers.KeycloakId,
	}
}
func convertToUserCallStatusChanged(event dto.GeneralEvent, u dto.VideoCallUserCallStatusChangedDto) *model.UserStatusEvent {
	return &model.UserStatusEvent{
		EventType: event.EventType,
		UserID:    u.UserId,
		IsInVideo: &u.IsInVideo,
	}
}
func convertToUserOnline(userOnline dto.UserOnline) *model.UserStatusEvent {
	return &model.UserStatusEvent{
		EventType: "user_online",
		UserID:    userOnline.UserId,
		Online:    &userOnline.Online,
	}
}
func convertToChatEvent(e *dto.ChatEvent) *model.ChatEvent {
	var result = &model.ChatEvent{
		EventType: e.EventType,
	}
	messageDto := e.MessageNotification
	if messageDto != nil {
		result.MessageEvent = convertDisplayMessageDto(messageDto)
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

	fileUploadedEvent := e.PreviewCreatedEvent
	if fileUploadedEvent != nil {
		result.PreviewCreatedEvent = &model.PreviewCreatedEvent{
			ID:            fileUploadedEvent.Id,
			URL:           fileUploadedEvent.Url,
			PreviewURL:    fileUploadedEvent.PreviewUrl,
			AType:         fileUploadedEvent.Type,
			CorrelationID: &fileUploadedEvent.CorrelationId,
		}
	}

	participants := e.Participants
	if participants != nil {
		result.ParticipantsEvent = convertParticipantsWithAdmin(*participants)
	}

	promotePinnedMessageEvent := e.PromoteMessageNotification
	if promotePinnedMessageEvent != nil {
		result.PromoteMessageEvent = convertPinnedMessageEvent(promotePinnedMessageEvent)
	}

	fileEvent := e.FileEvent
	if fileEvent != nil {
		result.FileEvent = &model.WrappedFileInfoDto{
			FileInfoDto: &model.FileInfoDto{
				ID:             fileEvent.FileInfoDto.Id,
				Filename:       fileEvent.FileInfoDto.Filename,
				URL:            fileEvent.FileInfoDto.Url,
				PublicURL:      fileEvent.FileInfoDto.PublicUrl,
				PreviewURL:     fileEvent.FileInfoDto.PreviewUrl,
				Size:           fileEvent.FileInfoDto.Size,
				CanDelete:      fileEvent.FileInfoDto.CanDelete,
				CanEdit:        fileEvent.FileInfoDto.CanEdit,
				CanShare:       fileEvent.FileInfoDto.CanShare,
				LastModified:   fileEvent.FileInfoDto.LastModified,
				OwnerID:        fileEvent.FileInfoDto.OwnerId,
				Owner:          convertParticipant(fileEvent.FileInfoDto.Owner),
				CanPlayAsVideo: fileEvent.FileInfoDto.CanPlayAsVideo,
				CanShowAsImage: fileEvent.FileInfoDto.CanShowAsImage,
				CanPlayAsAudio: fileEvent.FileInfoDto.CanPlayAsAudio,
				FileItemUUID:   &fileEvent.FileInfoDto.FileItemUuid,
			},
			Count: fileEvent.Count,
		}
	}

	reactionChangedEvent := e.ReactionChangedEvent
	if reactionChangedEvent != nil {
		result.ReactionChangedEvent = &model.ReactionChangedEvent{
			MessageID: reactionChangedEvent.MessageId,
			Reaction: &model.Reaction{
				Count:    reactionChangedEvent.Reaction.Count,
				Reaction: reactionChangedEvent.Reaction.Reaction,
			},
		}
	}

	return result
}
func convertDisplayMessageDto(messageDto *dto.DisplayMessageDto) *model.DisplayMessageDto {
	var result = &model.DisplayMessageDto{ // dto.DisplayMessageDto
		ID:             messageDto.Id,
		Text:           messageDto.Text,
		ChatID:         messageDto.ChatId,
		OwnerID:        messageDto.OwnerId,
		CreateDateTime: messageDto.CreateDateTime,
		EditDateTime:   messageDto.EditDateTime.Ptr(),
		Owner:          convertParticipant(messageDto.Owner),
		CanEdit:        messageDto.CanEdit,
		CanDelete:      messageDto.CanDelete,
		FileItemUUID:   messageDto.FileItemUuid,
		Pinned:         messageDto.Pinned,
		BlogPost:       messageDto.BlogPost,
		PinnedPromoted: messageDto.PinnedPromoted,
	}
	embedMessageDto := messageDto.EmbedMessage
	if embedMessageDto != nil {
		result.EmbedMessage = &model.EmbedMessageResponse{
			ID:            embedMessageDto.Id,
			ChatID:        embedMessageDto.ChatId,
			ChatName:      embedMessageDto.ChatName,
			Text:          embedMessageDto.Text,
			Owner:         convertParticipant(embedMessageDto.Owner),
			EmbedType:     embedMessageDto.EmbedType,
			IsParticipant: embedMessageDto.IsParticipant,
		}
	}
	reactions := messageDto.Reactions
	if reactions != nil {
		result.Reactions = convertReactions(reactions)
	}
	return result
}
func convertReactions(reactions []dto.Reaction) []*model.Reaction {
	ret := make([]*model.Reaction, 0)
	for _, r := range reactions {
		rr := r
		ret = append(ret, convertReaction(&rr))
	}
	return ret
}
func convertReaction(r *dto.Reaction) *model.Reaction {
	return &model.Reaction{
		Count:    r.Count,
		Reaction: r.Reaction,
	}
}
func convertPinnedMessageEvent(e *dto.PinnedMessageEvent) *model.PinnedMessageEvent {
	return &model.PinnedMessageEvent{
		Message:    convertDisplayMessageDto(&e.Message),
		TotalCount: e.TotalCount,
	}
}
func convertToGlobalEvent(e *dto.GlobalUserEvent) *model.GlobalEvent {
	//eventType string, chatDtoWithAdmin *dto.ChatDtoWithAdmin
	var ret = &model.GlobalEvent{
		EventType: e.EventType,
	}
	chatDtoWithAdmin := e.ChatNotification
	if chatDtoWithAdmin != nil {
		ret.ChatEvent = &model.ChatDto{ // dto.ChatDtoWithAdmin
			ID:                  chatDtoWithAdmin.Id,
			Name:                chatDtoWithAdmin.Name,
			Avatar:              chatDtoWithAdmin.Avatar.Ptr(),
			AvatarBig:           chatDtoWithAdmin.AvatarBig.Ptr(),
			ShortInfo:           chatDtoWithAdmin.ShortInfo.Ptr(),
			LastUpdateDateTime:  chatDtoWithAdmin.LastUpdateDateTime,
			ParticipantIds:      chatDtoWithAdmin.ParticipantIds,
			CanEdit:             chatDtoWithAdmin.CanEdit.Ptr(),
			CanDelete:           chatDtoWithAdmin.CanDelete.Ptr(),
			CanLeave:            chatDtoWithAdmin.CanLeave.Ptr(),
			UnreadMessages:      chatDtoWithAdmin.UnreadMessages,
			CanBroadcast:        chatDtoWithAdmin.CanBroadcast,
			CanVideoKick:        chatDtoWithAdmin.CanVideoKick,
			CanAudioMute:        chatDtoWithAdmin.CanAudioMute,
			CanChangeChatAdmins: chatDtoWithAdmin.CanChangeChatAdmins,
			TetATet:             chatDtoWithAdmin.IsTetATet,
			ParticipantsCount:   chatDtoWithAdmin.ParticipantsCount,
			Participants:        convertParticipantsWithAdmin(chatDtoWithAdmin.Participants),
			CanResend:           chatDtoWithAdmin.CanResend,
			AvailableToSearch:   chatDtoWithAdmin.AvailableToSearch,
			IsResultFromSearch:  chatDtoWithAdmin.IsResultFromSearch.Ptr(),
			Pinned:              chatDtoWithAdmin.Pinned,
			Blog:                chatDtoWithAdmin.Blog,
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
		ret.ParticipantEvent = &model.Participant{
			ID:        userProfileDto.Id,
			Login:     userProfileDto.Login,
			Avatar:    userProfileDto.Avatar.Ptr(),
			ShortInfo: userProfileDto.ShortInfo.Ptr(),
		}
	}

	videoUserCountEvent := e.VideoCallUserCountEvent
	if videoUserCountEvent != nil {
		ret.VideoUserCountChangedEvent = &model.VideoUserCountChangedDto{
			UsersCount: videoUserCountEvent.UsersCount,
			ChatID:     videoUserCountEvent.ChatId,
		}
	}

	videoCallScreenShareChangedEvent := e.VideoCallScreenShareChangedDto
	if videoCallScreenShareChangedEvent != nil {
		ret.VideoCallScreenShareChangedDto = &model.VideoCallScreenShareChangedDto{
			ChatID:          videoCallScreenShareChangedEvent.ChatId,
			HasScreenShares: videoCallScreenShareChangedEvent.HasScreenShares,
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
			Status:   videoChatInvite.Status,
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
			ChatID:             unreadMessages.ChatId,
			UnreadMessages:     unreadMessages.UnreadMessages,
			LastUpdateDateTime: unreadMessages.LastUpdateDateTime,
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
		ret.NotificationEvent = &model.WrapperNotificationDto{
			TotalCount: userNotification.TotalCount,
			NotificationDto: &model.NotificationDto{
				ID:               userNotification.NotificationDto.Id,
				ChatID:           userNotification.NotificationDto.ChatId,
				MessageID:        userNotification.NotificationDto.MessageId,
				NotificationType: userNotification.NotificationDto.NotificationType,
				Description:      userNotification.NotificationDto.Description,
				CreateDateTime:   userNotification.NotificationDto.CreateDateTime,
				ByUserID:         userNotification.NotificationDto.ByUserId,
				ByLogin:          userNotification.NotificationDto.ByLogin,
				ChatTitle:        userNotification.NotificationDto.ChatTitle,
			},
		}
	}

	return ret
}
func convertParticipant(owner *dto.User) *model.Participant {
	if owner == nil {
		return nil
	}
	return &model.Participant{
		ID:        owner.Id,
		Login:     owner.Login,
		Avatar:    owner.Avatar.Ptr(),
		ShortInfo: owner.ShortInfo.Ptr(),
	}
}
func convertParticipants(participants []*dto.User) []*model.Participant {
	if participants == nil {
		return nil
	}
	usrs := []*model.Participant{}
	for _, user := range participants {
		usrs = append(usrs, convertParticipant(user))
	}
	return usrs
}
func convertParticipantWithAdmin(owner *dto.UserWithAdmin) *model.ParticipantWithAdmin {
	if owner == nil {
		return nil
	}
	return &model.ParticipantWithAdmin{
		ID:        owner.Id,
		Login:     owner.Login,
		Avatar:    owner.Avatar.Ptr(),
		Admin:     owner.Admin,
		ShortInfo: owner.ShortInfo.Ptr(),
	}
}
func convertParticipantsWithAdmin(participants []*dto.UserWithAdmin) []*model.ParticipantWithAdmin {
	if participants == nil {
		return nil
	}
	usrs := []*model.ParticipantWithAdmin{}
	for _, user := range participants {
		usrs = append(usrs, convertParticipantWithAdmin(user))
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
