package listener

import (
	"context"
	"github.com/streadway/amqp"
	"github.com/tidwall/gjson"
	"nkonev.name/storage/client"
	"nkonev.name/storage/dto"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/services"
	"nkonev.name/storage/utils"
)

type MinioEventsListener func(*amqp.Delivery) error

func CreateMinioEventsListener(previewService *services.PreviewService, eventService *services.EventService, client *client.RestClient, minioConfig *utils.MinioConfig) MinioEventsListener {
	return func(msg *amqp.Delivery) error {
		bytesData := msg.Body
		strData := string(bytesData)
		Logger.Debugf("Received %v", strData)

		eventName := gjson.Get(strData, "EventName").String()
		key := gjson.Get(strData, "Key").String()
		result := gjson.Get(strData, "Records.0.s3.object.userMetadata") // empty in case delete
		chatId := result.Get(services.ChatIdKey(true)).Int()
		ownerId := result.Get(services.OwnerIdKey(true)).Int()
		correlationIdStr := result.Get(services.CorrelationIdKey(true)).String()
		var correlationId *string
		if correlationIdStr != "" {
			correlationId = &correlationIdStr
		}
		var minioEvent = &dto.MinioEvent{
			EventName:     eventName,
			Key:           key,
			ChatId:        chatId,
			OwnerId:       ownerId,
			CorrelationId: correlationId,
		}
		ctx := context.Background()

		normalizedKey := utils.StripBucketName(key, minioConfig.Files)
		workingChatId, err := utils.ParseChatId(normalizedKey)
		if err != nil {
			GetLogEntry(ctx).Errorf("Error during parsing chatId: %v", err)
			return err
		}
		participantIds, err := client.GetChatParticipantIds(workingChatId, ctx)
		if err != nil {
			GetLogEntry(ctx).Errorf("Error during getting participant ids: %v", err)
			return err
		}

		eventType, err := utils.GetEventType(eventName)
		if err != nil {
			GetLogEntry(ctx).Errorf("Logical error during getting event type: %v. It can be caused by new event that is not parsed correctly", err)
			return err
		}

		if eventType == utils.FILE_CREATED || eventType == utils.FILE_DELETED || eventType == utils.FILE_UPDATED {
			eventService.HandleEvent(participantIds, normalizedKey, workingChatId, eventType, ctx)
		}
		if eventType == utils.FILE_CREATED {
			previewService.HandleMinioEvent(participantIds, minioEvent, ctx)
		}
		return nil
	}
}
