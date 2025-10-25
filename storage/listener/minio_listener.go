package listener

import (
	"context"
	"errors"
	"fmt"
	"github.com/streadway/amqp"
	"github.com/tidwall/gjson"
	"go.opentelemetry.io/otel"
	"nkonev.name/storage/client"
	"nkonev.name/storage/db"
	"nkonev.name/storage/dto"
	"nkonev.name/storage/logger"
	"nkonev.name/storage/s3"
	"nkonev.name/storage/services"
	"nkonev.name/storage/utils"
)

type MinioEventsListener func(*amqp.Delivery) error

func CreateMinioEventsListener(
	lgr *logger.Logger,
	previewService *services.PreviewService,
	eventService *services.EventService,
	client *client.RestClient,
	minioConfig *utils.MinioConfig,
	minioClient *s3.InternalMinioClient,
	convertingService *services.ConvertingService,
	dba *db.DB,
) MinioEventsListener {
	tr := otel.Tracer("amqp/listener")

	return func(msg *amqp.Delivery) error {
		ctx, span := tr.Start(context.Background(), "storage.minio.listener")
		defer span.End()

		bytesData := msg.Body
		strData := string(bytesData)
		lgr.WithTracing(ctx).Debugf("Received %v", strData)

		eventName := gjson.Get(strData, "EventName").String()
		key := gjson.Get(strData, "Key").String()
		userMetadata := gjson.Get(strData, "Records.0.s3.object.userMetadata") // empty in case delete
		maybeChatId := userMetadata.Get(services.ChatIdKey(true)).Int()
		ownerId := userMetadata.Get(services.OwnerIdKey(true)).Int()
		correlationIdStr := userMetadata.Get(services.CorrelationIdKey(true)).String()
		isConferenceRecording := userMetadata.Get(services.ConferenceRecordingKey(true)).Bool()
		isMessageRecording := userMetadata.Get(services.MessageRecordingKey(true)).Bool()
		var correlationId *string
		if correlationIdStr != "" {
			correlationId = &correlationIdStr
		}
		timestamp := userMetadata.Get(services.TimestampKey(true)).Int() // unix milli in UTC

		var minioEvent = &dto.MinioEvent{
			EventName:     eventName,
			Key:           key,
			ChatId:        maybeChatId,
			OwnerId:       ownerId,
			CorrelationId: correlationId,
		}

		normalizedKey := utils.StripBucketName(key, minioConfig.Files)
		workingChatId, err := utils.ParseChatId(normalizedKey)
		if err != nil {
			lgr.WithTracing(ctx).Errorf("Error during parsing chatId: %v", err)
			return err
		}
		eventType, err := utils.GetEventType(eventName, isConferenceRecording || isMessageRecording)
		if err != nil {
			lgr.WithTracing(ctx).Errorf("Logical error during getting event type: %v. It can be caused by new event that is not parsed correctly", err)
			return err
		}

		var eventServiceResponse *services.HandleEventResponse
		var previewServiceResponse *services.PreviewResponse
		previewAlreadyExists, err := isPreviewAlreadyExists(ctx, lgr, minioConfig, minioClient, normalizedKey)
		if err != nil {
			return err
		}
		var eventForConvertingService = isEventForConvertingService(eventType, minioEvent, previewAlreadyExists, isMessageRecording)
		if isEventForEventService(eventType) {
			eventServiceResponse = eventService.HandleEvent(ctx, normalizedKey, workingChatId, eventType)
		}
		if isEventForPreviewService(eventType, previewAlreadyExists, normalizedKey) {
			previewServiceResponse = previewService.HandleMinioEvent(ctx, minioEvent, eventForConvertingService)
		}

		var mce *dto.MetadataCache
		// store to db before sending events
		switch eventType {
		case utils.FILE_CREATED:
			fallthrough
		case utils.FILE_UPDATED:
			mce, err = createdDbEntity(normalizedKey, workingChatId, ownerId, correlationId, timestamp, eventServiceResponse)
			if err != nil {
				lgr.WithTracing(ctx).Errorf("Error during creating db entity: %v", err)
				return err
			}
			err = db.Set(ctx, dba, *mce)
			if err != nil {
				lgr.WithTracing(ctx).Errorf("Error during saving to database: %v", err)
				return err
			}
		case utils.FILE_DELETED:
			mck, err := createDbKey(normalizedKey, workingChatId)
			if err != nil {
				lgr.WithTracing(ctx).Errorf("Error during creating db key: %v", err)
				return err
			}
			err = db.Remove(ctx, dba, *mck)
			if err != nil {
				lgr.WithTracing(ctx).Errorf("Error during removing from database: %v", err)
				return err
			}
		default:
			return fmt.Errorf("Unknown case %v", eventType)
		}

		err = client.GetChatParticipantIds(ctx, workingChatId, func(participantIds []int64) error {
			if isEventForEventService(eventType) {
				eventService.SendToParticipants(ctx, normalizedKey, workingChatId, eventType, participantIds, eventServiceResponse, mce)
			}
			if isEventForPreviewService(eventType, previewAlreadyExists, normalizedKey) {
				previewService.SendToParticipants(ctx, minioEvent, participantIds, previewServiceResponse)
			}
			return nil
		})
		if err != nil {
			lgr.WithTracing(ctx).Errorf("Error during getting participant ids: %v", err)
		}
		// because converting is longer than creating the preview, we do this long job in the end, after sending preview_created event
		if eventForConvertingService {
			convertingService.HandleEvent(ctx, minioEvent)
		}

		return nil
	}
}

func isEventForEventService(eventType utils.EventType) bool {
	if eventType == utils.FILE_CREATED || eventType == utils.FILE_DELETED || eventType == utils.FILE_UPDATED {
		return true
	} else {
		return false
	}
}

func isEventForPreviewService(eventType utils.EventType, previewExists bool, normalizedKey string) bool {
	return eventType == utils.FILE_CREATED && !previewExists && utils.IsPreviewable(normalizedKey)
}

func isEventForConvertingService(eventType utils.EventType, minioEvent *dto.MinioEvent, previewExists, isMessageRecording bool) bool {
	return eventType == utils.FILE_CREATED &&
		isMessageRecording &&
		utils.IsVideo(minioEvent.Key) &&
		!previewExists // prevents the indefinite converting
}

func isPreviewAlreadyExists(ctx context.Context, lgr *logger.Logger, minioConfig *utils.MinioConfig, minioClient *s3.InternalMinioClient, normalizedKey string) (bool, error) {
	previewKey := utils.SetVideoPreviewExtension(normalizedKey)
	exists, _, err := minioClient.FileExists(ctx, minioConfig.FilesPreview, previewKey)
	if err != nil {
		lgr.WithTracing(ctx).Errorf("Error during checking existence for %v: %v", previewKey, err)
	}
	return exists, err
}

func createdDbEntity(normalizedKey string, chatId, ownerId int64, correlationId *string, timestamp int64, eventServiceResponse *services.HandleEventResponse) (*dto.MetadataCache, error) {
	fileItemUuid, err := utils.ParseFileItemUuid(normalizedKey)
	if err != nil {
		return nil, err
	}

	filename, err := utils.ParseFileName(normalizedKey)
	if err != nil {
		return nil, err
	}

	eventTime := utils.GetEventTimeFromTimestamp(timestamp)

	publishedP, err := eventServiceResponse.GetTags()
	if err != nil {
		return nil, err
	}
	var published bool
	if publishedP != nil {
		published = *publishedP
	}

	objInfo := eventServiceResponse.GetObjectInfo()
	if objInfo == nil {
		return nil, errors.New("nil ObjectInfo")
	}

	return &dto.MetadataCache{
		ChatId:         chatId,
		FileItemUuid:   fileItemUuid,
		Filename:       filename,
		OwnerId:        ownerId,
		CorrelationId:  correlationId,
		Published:      published,
		FileSize:       objInfo.Size,
		CreateDateTime: eventTime,
		EditDateTime:   eventTime,
	}, nil
}

func createDbKey(normalizedKey string, chatId int64) (*dto.MetadataCacheId, error) {
	fileItemUuid, err := utils.ParseFileItemUuid(normalizedKey)
	if err != nil {
		return nil, err
	}

	filename, err := utils.ParseFileName(normalizedKey)
	if err != nil {
		return nil, err
	}

	return &dto.MetadataCacheId{
		ChatId:       chatId,
		FileItemUuid: fileItemUuid,
		Filename:     filename,
	}, nil
}
