package services

import (
	"context"
	"nkonev.name/video/client"
	"nkonev.name/video/dto"
	. "nkonev.name/video/logger"
	"nkonev.name/video/producer"
)

type ChatInvitationService struct {
	rabbitMqInvitePublisher *producer.RabbitInvitePublisher
	chatClient              *client.RestClient
	redisService            *DialRedisRepository
}

func NewChatInvitationService(rabbitMqInvitePublisher *producer.RabbitInvitePublisher, chatClient *client.RestClient, redisService *DialRedisRepository) *ChatInvitationService {
	return &ChatInvitationService{
		rabbitMqInvitePublisher: rabbitMqInvitePublisher,
		chatClient: chatClient,
		redisService: redisService,
	}
}


func (srv *ChatInvitationService) SendInvitationsWithStatuses(ctx context.Context, chatId, ownerId int64, statuses map[int64]string) {
	var userIdsToDial []int64 = make([]int64, 0)
	for userId, _ := range statuses {
		userIdsToDial = append(userIdsToDial, userId)
	}

	inviteNames, err := srv.chatClient.GetChatNameForInvite(chatId, ownerId, userIdsToDial, ctx)
	if err != nil {
		GetLogEntry(ctx).Error(err, "Failed during getting chat invite names")
		return
	}

	// this is sending call invitations to all the ivitees
	for _, chatInviteName := range inviteNames {
		status, err := srv.redisService.GetUserCallStatus(ctx, chatInviteName.UserId)
		if err != nil {
			GetLogEntry(ctx).Errorf("Error during getting callStatus of %v: %v", chatInviteName.UserId, err)
			continue
		}

		if status == CallStatusNotFound {
			GetLogEntry(ctx).Warnf("Call status isn't found for user %v", chatInviteName.UserId)
			continue
		}

		invitation := dto.VideoCallInvitation{
			ChatId:   chatId,
			ChatName: chatInviteName.Name,
			Status:   status,
		}

		err = srv.rabbitMqInvitePublisher.Publish(&invitation, chatInviteName.UserId)
		if err != nil {
			GetLogEntry(ctx).Error(err, "Error during sending VideoInviteDto")
		}
	}
}
