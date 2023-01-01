package listener

import (
	"context"
	"github.com/streadway/amqp"
	"github.com/tidwall/gjson"
	"nkonev.name/storage/dto"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/services"
)

type MinioEventsListener func(*amqp.Delivery) error

func CreateMinioEventsListener(previewService *services.PreviewService) MinioEventsListener {
	return func(msg *amqp.Delivery) error {
		bytesData := msg.Body
		strData := string(bytesData)
		Logger.Debugf("Received %v", strData)

		eventName := gjson.Get(strData, "EventName").String()
		key := gjson.Get(strData, "Key").String()
		result := gjson.Get(strData, "Records.0.s3.object.userMetadata")
		chatId := result.Get(services.ChatIdKey(true)).Int()
		ownerId := result.Get(services.OwnerIdKey(true)).Int()
		var minioEvent = &dto.MinioEvent{
			EventName: eventName,
			Key:       key,
			ChatId:    chatId,
			OwnerId:   ownerId,
		}
		ctx := context.Background()

		previewService.HandleMinioEvent(minioEvent, ctx)

		return nil
	}
}
