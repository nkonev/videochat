package services
//
//import (
//	"context"
//	"nkonev.name/video/client"
//	"nkonev.name/video/dto"
//	. "nkonev.name/video/logger"
//	"nkonev.name/video/producer"
//)
//
//type ChatInvitationService struct {
//	rabbitMqInvitePublisher *producer.RabbitInvitePublisher
//	chatClient              *client.RestClient
//}
//
//func NewChatInvitationService(rabbitMqInvitePublisher *producer.RabbitInvitePublisher, chatClient *client.RestClient) *ChatInvitationService {
//	return &ChatInvitationService{
//		rabbitMqInvitePublisher: rabbitMqInvitePublisher,
//		chatClient: chatClient,
//	}
//}
//
//
//func (srv *ChatInvitationService) SendInvitationsWithStatuses(ctx context.Context, chatId, ownerId int64, statuses map[int64]string) {
//	if len(statuses) == 0 {
//		return
//	}
//
//	var userIdsToDial []int64 = make([]int64, 0)
//	for userId, _ := range statuses {
//		userIdsToDial = append(userIdsToDial, userId)
//	}
//
//	inviteNames, err := srv.chatClient.GetChatNameForInvite(chatId, ownerId, userIdsToDial, ctx)
//	if err != nil {
//		GetLogEntry(ctx).Error(err, "Failed during getting chat invite names")
//		return
//	}
//
//	// this is sending call invitations to all the ivitees
//	for _, chatInviteName := range inviteNames {
//		status := statuses[chatInviteName.UserId]
//
//		invitation := dto.VideoCallInvitation{
//			ChatId:   chatId,
//			ChatName: chatInviteName.Name,
//			Status:   status,
//		}
//
//		err = srv.rabbitMqInvitePublisher.Publish(&invitation, chatInviteName.UserId)
//		if err != nil {
//			GetLogEntry(ctx).Error(err, "Error during sending VideoInviteDto")
//		}
//	}
//}
