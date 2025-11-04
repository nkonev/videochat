package graph

import (
	"context"
	"nkonev.name/event/auth"
	"nkonev.name/event/dto"
	"nkonev.name/event/graph/model"
	"nkonev.name/event/utils"
)

func filter(userFromBus int64, userIdsFilter []int64) bool {
	if len(userIdsFilter) == 0 {
		return true
	}
	return utils.Contains(userIdsFilter, userFromBus)
}
func (sr *subscriptionResolver) prepareUserAccountEvent(ctx context.Context, myUserId int64, eventType string, user *dto.UserAccountEvent) *model.UserAccountEvent {
	if user == nil {
		sr.Lgr.WithTracing(ctx).Errorf("Logical mistake")
		return nil
	}

	extended, err := sr.HttpClient.GetUserExtended(ctx, user.Id, myUserId)
	if err != nil {
		sr.Lgr.WithTracing(ctx).Errorf("error during getting user extended: %v", err)
		return nil
	}

	ret := model.UserAccountEvent{}
	ret.EventType = eventType
	ret.UserAccountEvent = convertUserAccountExtended(myUserId, user, extended)
	return &ret
}
func convertUserAccountDeletedEvent(eventType string, userId int64) *model.UserAccountEvent {
	ret := model.UserAccountEvent{}
	ret.UserAccountEvent = &model.UserDeletedDto{ID: userId}
	ret.EventType = eventType
	return &ret
}
func convertUserAccountExtended(myUserId int64, user *dto.UserAccountEvent, aDto *dto.UserAccountExtended) *model.UserAccountExtendedDto {
	userAccountEvent := &model.UserAccountExtendedDto{
		ID:                aDto.Id,
		Login:             aDto.Login,
		Avatar:            aDto.Avatar,
		AvatarBig:         aDto.AvatarBig,
		ShortInfo:         aDto.ShortInfo,
		LastSeenDateTime:  aDto.LastSeenDateTime,
		Oauth2Identifiers: convertOauth2Identifiers(aDto.Oauth2Identifiers),
		CanLock:           aDto.CanLock,
		CanEnable:         aDto.CanEnable,
		CanDelete:         aDto.CanDelete,
		CanChangeRole:     aDto.CanChangeRole,
		CanConfirm:        aDto.CanConfirm,
		LoginColor:        aDto.LoginColor,
		CanRemoveSessions: aDto.CanRemoveSessions,
		Ldap:              aDto.Ldap,
		CanSetPassword:    aDto.CanSetPassword, // can forcibly set somebody's password

		CanChangeSelfLogin:    aDto.CanChangeSelfLogin,
		CanChangeSelfEmail:    aDto.CanChangeSelfEmail,
		CanChangeSelfPassword: aDto.CanChangeSelfPassword,
	}
	if myUserId == aDto.Id {
		userAccountEvent.Email = user.Email
		userAccountEvent.AwaitingForConfirmEmailChange = &user.AwaitingForConfirmEmailChange
	}

	if user.AdditionalData != nil {
		userAccountEvent.AdditionalData = convertAdditionalData(user.AdditionalData)
	}

	return userAccountEvent
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
		EventType:        "user_online",
		UserID:           userOnline.UserId,
		Online:           &userOnline.Online,
		LastSeenDateTime: userOnline.LastSeenDateTime,
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
			FileItemUUID:  fileUploadedEvent.FileItemUuid,
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

	publishedMessageEvent := e.PublishedMessageNotification
	if publishedMessageEvent != nil {
		result.PublishedMessageEvent = convertPublishedMessageEvent(publishedMessageEvent)
	}

	fileEvent := e.FileEvent
	if fileEvent != nil {
		result.FileEvent = &model.WrappedFileInfoDto{
			FileInfoDto: &model.FileInfoDto{
				ID:             fileEvent.FileInfoDto.Id,
				Filename:       fileEvent.FileInfoDto.Filename,
				URL:            fileEvent.FileInfoDto.Url,
				PublishedURL:   fileEvent.FileInfoDto.PublishedUrl,
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
				FileItemUUID:   fileEvent.FileInfoDto.FileItemUuid,
				CorrelationID:  fileEvent.FileInfoDto.CorrelationId,
				Previewable:    fileEvent.FileInfoDto.Previewable,
				AType:          fileEvent.FileInfoDto.Type,
			},
		}
	}

	reactionChangedEvent := e.ReactionChangedEvent
	if reactionChangedEvent != nil {
		result.ReactionChangedEvent = &model.ReactionChangedEvent{
			MessageID: reactionChangedEvent.MessageId,
			Reaction:  convertReaction(&reactionChangedEvent.Reaction),
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
		EditDateTime:   messageDto.EditDateTime,
		Owner:          convertParticipant(messageDto.Owner),
		CanEdit:        messageDto.CanEdit,
		CanDelete:      messageDto.CanDelete,
		FileItemUUID:   messageDto.FileItemUuid,
		Pinned:         messageDto.Pinned,
		BlogPost:       messageDto.BlogPost,
		PinnedPromoted: messageDto.PinnedPromoted,
		Published:      messageDto.Published,
		CanPublish:     messageDto.CanPublish,
		CanPin:         messageDto.CanPin,
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
		Users:    convertParticipants(r.Users),
	}
}
func convertPinnedMessageEvent(e *dto.PinnedMessageEvent) *model.PinnedMessageEvent {
	return &model.PinnedMessageEvent{
		Message: convertPinnedMessageDto(&e.Message),
		Count:   e.TotalCount,
	}
}
func convertPublishedMessageEvent(e *dto.PublishedMessageEvent) *model.PublishedMessageEvent {
	return &model.PublishedMessageEvent{
		Message: convertPublishedMessageDto(&e.Message),
		Count:   e.TotalCount,
	}
}
func convertPublishedMessageDto(e *dto.PublishedMessageDto) *model.PublishedMessageDto {
	return &model.PublishedMessageDto{
		ID:             e.Id,
		Text:           e.Text,
		ChatID:         e.ChatId,
		OwnerID:        e.OwnerId,
		Owner:          convertParticipant(e.Owner),
		CanPublish:     e.CanPublish,
		CreateDateTime: e.CreateDateTime,
	}
}
func convertPinnedMessageDto(e *dto.PinnedMessageDto) *model.PinnedMessageDto {
	return &model.PinnedMessageDto{
		ID:             e.Id,
		Text:           e.Text,
		ChatID:         e.ChatId,
		OwnerID:        e.OwnerId,
		Owner:          convertParticipant(e.Owner),
		PinnedPromoted: e.PinnedPromoted,
		CreateDateTime: e.CreateDateTime,
		CanPin:         e.CanPin,
	}
}
func convertToGlobalEvent(e *dto.GlobalUserEvent) *model.GlobalEvent {
	//eventType string, chatDtoWithAdmin *dto.ChatDtoWithAdmin
	var ret = &model.GlobalEvent{
		EventType: e.EventType,
	}
	chatEvent := e.ChatNotification
	if chatEvent != nil {
		ret.ChatEvent = &model.ChatDto{
			ID:                                  chatEvent.Id,
			Name:                                chatEvent.Name,
			Avatar:                              chatEvent.Avatar,
			AvatarBig:                           chatEvent.AvatarBig,
			ShortInfo:                           chatEvent.ShortInfo,
			LastUpdateDateTime:                  chatEvent.LastUpdateDateTime,
			ParticipantIds:                      chatEvent.ParticipantIds,
			CanEdit:                             chatEvent.CanEdit,
			CanDelete:                           chatEvent.CanDelete,
			CanLeave:                            chatEvent.CanLeave,
			UnreadMessages:                      chatEvent.UnreadMessages,
			CanBroadcast:                        chatEvent.CanBroadcast,
			CanVideoKick:                        chatEvent.CanVideoKick,
			CanAudioMute:                        chatEvent.CanAudioMute,
			CanChangeChatAdmins:                 chatEvent.CanChangeChatAdmins,
			TetATet:                             chatEvent.IsTetATet,
			ParticipantsCount:                   chatEvent.ParticipantsCount,
			Participants:                        convertParticipants(chatEvent.Participants),
			CanResend:                           chatEvent.CanResend,
			AvailableToSearch:                   chatEvent.AvailableToSearch,
			IsResultFromSearch:                  chatEvent.IsResultFromSearch,
			Pinned:                              chatEvent.Pinned,
			Blog:                                chatEvent.Blog,
			LoginColor:                          chatEvent.LoginColor,
			RegularParticipantCanPublishMessage: chatEvent.RegularParticipantCanPublishMessage,
			LastSeenDateTime:                    chatEvent.LastSeenDateTime,
			RegularParticipantCanPinMessage:     chatEvent.RegularParticipantCanPinMessage,
			BlogAbout:                           chatEvent.BlogAbout,
			RegularParticipantCanWriteMessage:   chatEvent.RegularParticipantCanWriteMessage,
			CanWriteMessage:                     chatEvent.CanWriteMessage,
			LastMessagePreview:                  chatEvent.LastMessagePreview,
			CanReact:                            chatEvent.CanReact,
		}

		if chatEvent.AdditionalData != nil {
			ret.ChatEvent.AdditionalData = convertAdditionalData(chatEvent.AdditionalData)
		}
	}

	chatDeleted := e.ChatDeletedDto
	if chatDeleted != nil {
		ret.ChatDeletedEvent = &model.ChatDeletedDto{
			ID: chatDeleted.Id,
		}
	}

	userProfileDto := e.CoChattedParticipantNotification
	if userProfileDto != nil {
		ret.CoChattedParticipantEvent = convertParticipant(userProfileDto)
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
			Avatar:   videoChatInvite.Avatar,
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
			Count: userNotification.TotalCount,
			NotificationDto: &model.NotificationDto{
				ID:               userNotification.NotificationDto.Id,
				ChatID:           userNotification.NotificationDto.ChatId,
				MessageID:        userNotification.NotificationDto.MessageId,
				NotificationType: userNotification.NotificationDto.NotificationType,
				Description:      userNotification.NotificationDto.Description,
				CreateDateTime:   userNotification.NotificationDto.CreateDateTime,
				ByUserID:         userNotification.NotificationDto.ByUserId,
				ByLogin:          userNotification.NotificationDto.ByLogin,
				ByAvatar:         userNotification.NotificationDto.ByAvatar,
				ChatTitle:        userNotification.NotificationDto.ChatTitle,
			},
		}
	}

	hasUnreadMessagesChanged := e.HasUnreadMessagesChanged
	if hasUnreadMessagesChanged != nil {
		ret.HasUnreadMessagesChanged = &model.HasUnreadMessagesChangedEvent{
			HasUnreadMessages: hasUnreadMessagesChanged.HasUnreadMessages,
		}
	}

	browserNotification := e.BrowserNotification
	if browserNotification != nil {
		ret.BrowserNotification = &model.BrowserNotification{
			ChatID:      browserNotification.ChatId,
			ChatName:    browserNotification.ChatName,
			ChatAvatar:  browserNotification.ChatAvatar,
			MessageID:   browserNotification.MessageId,
			MessageText: browserNotification.MessageText,
			OwnerID:     browserNotification.OwnerId,
			OwnerLogin:  browserNotification.OwnerLogin,
		}
	}

	userTypingEvent := e.UserTypingNotification
	if userTypingEvent != nil {
		ret.UserTypingEvent = &model.UserTypingDto{
			Login:         userTypingEvent.Login,
			ParticipantID: userTypingEvent.ParticipantId,
			ChatID:        userTypingEvent.ChatId,
		}
	}

	return ret
}
func convertToUserSessionsKilledEvent(aDto *dto.UserSessionsKilledEvent) *model.GlobalEvent {
	var ret = &model.GlobalEvent{
		EventType:   aDto.EventType,
		ForceLogout: &model.ForceLogoutEvent{ReasonType: aDto.ReasonType},
	}

	return ret
}
func convertParticipant(owner *dto.User) *model.Participant {
	if owner == nil {
		return nil
	}
	p := model.Participant{
		ID:         owner.Id,
		Login:      owner.Login,
		Avatar:     owner.Avatar,
		ShortInfo:  owner.ShortInfo,
		LoginColor: owner.LoginColor,
	}

	if owner.AdditionalData != nil {
		p.AdditionalData = convertAdditionalData(owner.AdditionalData)
	}

	return &p
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
	p := model.ParticipantWithAdmin{
		ID:         owner.Id,
		Login:      owner.Login,
		Avatar:     owner.Avatar,
		Admin:      owner.Admin,
		ShortInfo:  owner.ShortInfo,
		LoginColor: owner.LoginColor,
	}

	if owner.AdditionalData != nil {
		p.AdditionalData = convertAdditionalData(owner.AdditionalData)
	}

	return &p
}
func convertAdditionalData(ad *dto.AdditionalData) *model.AdditionalData {
	if ad == nil {
		return nil
	}
	return &model.AdditionalData{
		Enabled:   ad.Enabled,
		Expired:   ad.Expired,
		Locked:    ad.Locked,
		Confirmed: ad.Confirmed,
		Roles:     ad.Roles,
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
