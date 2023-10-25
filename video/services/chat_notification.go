package services

import (
	"context"
	"nkonev.name/video/client"
	"nkonev.name/video/dto"
	. "nkonev.name/video/logger"
	"nkonev.name/video/producer"
	"nkonev.name/video/utils"
)

type NotificationService struct {
	rabbitMqUserCountPublisher *producer.RabbitUserCountPublisher
	rabbitMqRecordPublisher    *producer.RabbitRecordingPublisher
	rabbitMqScreenSharePublisher *producer.RabbitScreenSharePublisher
	rabbitUserIdsPublisher *producer.RabbitUserIdsPublisher
	restClient                 *client.RestClient
}

func NewNotificationService(producer *producer.RabbitUserCountPublisher, restClient *client.RestClient, rabbitMqRecordPublisher *producer.RabbitRecordingPublisher, rabbitMqScreenSharePublisher *producer.RabbitScreenSharePublisher, rabbitUserIdsPublisher *producer.RabbitUserIdsPublisher) *NotificationService {
	return &NotificationService{
		rabbitMqUserCountPublisher: producer,
		rabbitMqScreenSharePublisher: rabbitMqScreenSharePublisher,
		rabbitMqRecordPublisher:    rabbitMqRecordPublisher,
		rabbitUserIdsPublisher: rabbitUserIdsPublisher,
		restClient:                 restClient,
	}
}

// sends notification about video users, which is showed
// as a small handset in ChatList
// and as a number of video users in badge near call button
func (h *NotificationService) NotifyVideoUserCountChanged(participantIds []int64, chatId, usersCount int64, ctx context.Context) error {
	Logger.Debugf("Notifying video call chat_id=%v", chatId)

	var chatNotifyDto = dto.VideoCallUserCountChangedDto{
		UsersCount: usersCount,
		ChatId:     chatId,
	}

	return h.rabbitMqUserCountPublisher.Publish(participantIds, &chatNotifyDto, ctx)
}

func (h *NotificationService) NotifyAboutUsersVideoStatusChanged(participantIds, videoParticipants []int64, ctx context.Context) error {
	Logger.Debugf("Notifying about user ids %v", participantIds)

	var dtos = make([]dto.VideoCallUserCallStatusChangedDto, 0)

	for _, chatParticipant := range participantIds {
		var isInVideoCall = false
		if utils.Contains(videoParticipants, chatParticipant) {
			isInVideoCall = true
		}
		var aDto = dto.VideoCallUserCallStatusChangedDto{
			UserId:    chatParticipant,
			IsInVideo:     isInVideoCall,
		}
		dtos = append(dtos, aDto)
	}

	return h.rabbitUserIdsPublisher.Publish(&dto.VideoCallUsersCallStatusChangedDto{Users: dtos}, ctx)
}

func (h *NotificationService) NotifyVideoScreenShareChanged(participantIds []int64, chatId int64, hasScreenShares bool, ctx context.Context) error {
	Logger.Debugf("Notifying video call chat_id=%v", chatId)

	var chatNotifyDto = dto.VideoCallScreenShareChangedDto{
		HasScreenShares: hasScreenShares,
		ChatId:     chatId,
	}

	return h.rabbitMqScreenSharePublisher.Publish(participantIds, &chatNotifyDto, ctx)
}


func (h *NotificationService) NotifyRecordingChanged(chatId int64, recordInProgress bool, ctx context.Context) error {
	Logger.Debugf("Notifying video call chat_id=%v", chatId)

	var chatNotifyDto = dto.VideoCallRecordingChangedDto{
		RecordInProgress: recordInProgress,
		ChatId:           chatId,
	}

	participantIds, err := h.restClient.GetChatParticipantIds(chatId, ctx)
	if err != nil {
		Logger.Error(err, "Failed during getting chat participantIds")
		return err
	}

	return h.rabbitMqRecordPublisher.Publish(participantIds, &chatNotifyDto, ctx)
}
