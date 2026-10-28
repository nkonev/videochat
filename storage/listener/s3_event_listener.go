package listener

import (
	"context"
	"errors"
	"fmt"

	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel"
	"nkonev.name/storage/client"
	"nkonev.name/storage/db"
	"nkonev.name/storage/dto"
	"nkonev.name/storage/logger"
	"nkonev.name/storage/services"
	"nkonev.name/storage/utils"
)

type MinioEventsListener func(*amqp.Delivery) error

func CreateS3EventListener(
	lgr *logger.Logger,
	previewService *services.PreviewService,
	eventService *services.EventService,
	client *client.RestClient,
	minioConfig *utils.MinioConfig,
	minioClient *client.InternalMinioClient,
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

		meDto, err := services.DeserializeMetadataFromEvent(minioConfig.Files, strData)
		if err != nil {
			lgr.WithTracing(ctx).Errorf("Error during parsing DeserializeMetadataFromEvent: %v", err)
			return err
		}

		var eventServiceResponse *services.HandleEventResponse
		var previewServiceResponse *services.PreviewResponse
		previewAlreadyExists, err := isPreviewAlreadyExists(ctx, lgr, minioConfig, minioClient, meDto.NormalizedKey)
		if err != nil {
			return err
		}
		var eventForConvertingService = isEventForConvertingService(meDto.EventType, &meDto.MinioEvent, previewAlreadyExists, meDto.IsMessageRecording)
		if isEventForEventService(meDto.EventType) {
			eventServiceResponse = eventService.HandleEvent(ctx, meDto.NormalizedKey, meDto.ChatId, meDto.EventType, &meDto.MinioEvent)
		}
		if isEventForPreviewService(meDto.EventType, previewAlreadyExists, meDto.NormalizedKey) {
			previewServiceResponse = previewService.HandleMinioEvent(ctx, &meDto.MinioEvent, eventForConvertingService)
		}

		var mce *dto.MetadataCache
		// store to db before sending events
		switch meDto.EventType {
		case dto.FILE_CREATED:
			fallthrough
		case dto.FILE_UPDATED:
			mce, err = createdDbEntity(meDto.NormalizedKey, meDto.ChatId, meDto.OwnerId, meDto.CorrelationId, meDto.Timestamp, eventServiceResponse)
			if err != nil {
				lgr.WithTracing(ctx).Errorf("Error during creating db entity: %v", err)
				return err
			}
			err = db.Set(ctx, dba, *mce)
			if err != nil {
				lgr.WithTracing(ctx).Errorf("Error during saving to database: %v", err)
				return err
			}
		case dto.FILE_DELETED:
			mck, err := utils.BuildMetadataCacheId(meDto.NormalizedKey)
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
			return fmt.Errorf("Unknown case %v", meDto.EventType)
		}

		err = client.GetChatParticipantIds(ctx, meDto.ChatId, func(participantIds []int64) error {
			if isEventForEventService(meDto.EventType) {
				eventService.SendToParticipants(ctx, meDto.NormalizedKey, meDto.ChatId, meDto.EventType, participantIds, eventServiceResponse, mce)
			}
			if isEventForPreviewService(meDto.EventType, previewAlreadyExists, meDto.NormalizedKey) {
				previewService.SendToParticipants(ctx, &meDto.MinioEvent, participantIds, previewServiceResponse)
			}
			return nil
		})
		if err != nil {
			lgr.WithTracing(ctx).Errorf("Error during getting participant ids: %v", err)
		}
		// because converting is longer than creating the preview, we do this long job in the end, after sending preview_created event
		if eventForConvertingService {
			convertingService.HandleEvent(ctx, &meDto.MinioEvent)
		}

		return nil
	}
}

func isEventForEventService(eventType dto.EventType) bool {
	if eventType == dto.FILE_CREATED || eventType == dto.FILE_DELETED || eventType == dto.FILE_UPDATED {
		return true
	} else {
		return false
	}
}

func isEventForPreviewService(eventType dto.EventType, previewExists bool, normalizedKey string) bool {
	return eventType == dto.FILE_CREATED && !previewExists && utils.IsPreviewable(normalizedKey)
}

func isEventForConvertingService(eventType dto.EventType, minioEvent *dto.MinioEvent, previewExists, isMessageRecording bool) bool {
	return eventType == dto.FILE_CREATED &&
		isMessageRecording &&
		utils.IsVideo(minioEvent.Key) &&
		!previewExists // prevents the indefinite converting
}

func isPreviewAlreadyExists(ctx context.Context, lgr *logger.Logger, minioConfig *utils.MinioConfig, minioClient *client.InternalMinioClient, normalizedKey string) (bool, error) {
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

	publishedP, correlationIdFromTags, err := eventServiceResponse.GetTags()
	if err != nil {
		return nil, err
	}
	var published bool
	if publishedP != nil {
		published = *publishedP
	}

	if eventServiceResponse.GetMinioEvent().EventName == dto.ObjectCreatedPutTagging {
		correlationId = correlationIdFromTags
	}

	me := eventServiceResponse.GetMinioEvent()
	if me == nil {
		return nil, errors.New("nil MinioEvent")
	}

	return &dto.MetadataCache{
		ChatId:         chatId,
		FileItemUuid:   fileItemUuid,
		Filename:       filename,
		OwnerId:        ownerId,
		CorrelationId:  correlationId,
		Published:      published,
		FileSize:       me.Size,
		CreateDateTime: eventTime,
		EditDateTime:   eventTime,
	}, nil
}
