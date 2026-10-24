package services

import (
	"context"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/tags"
	"nkonev.name/storage/client"
	"nkonev.name/storage/dto"
	"nkonev.name/storage/logger"
	"nkonev.name/storage/producer"
	"nkonev.name/storage/s3"
	"nkonev.name/storage/utils"
)

func NewEventService(lgr *logger.Logger, client *client.RestClient, minio *s3.InternalMinioClient, minioConfig *utils.MinioConfig, filesService *FilesService, publisher *producer.RabbitFileUploadedPublisher) *EventService {
	return &EventService{
		client:       client,
		minio:        minio,
		minioConfig:  minioConfig,
		filesService: filesService,
		publisher:    publisher,
		lgr:          lgr,
	}
}

type EventService struct {
	client       *client.RestClient
	minio        *s3.InternalMinioClient
	minioConfig  *utils.MinioConfig
	filesService *FilesService
	publisher    *producer.RabbitFileUploadedPublisher
	lgr          *logger.Logger
}

func (s *EventService) HandleEvent(ctx context.Context, normalizedKey string, chatId int64, eventType dto.EventType, minioEvent *dto.MinioEvent) *HandleEventResponse {
	s.lgr.WithTracing(ctx).Debugf("Got %v %v", normalizedKey, eventType)

	var err error

	var tagging *tags.Tags
	if eventType == dto.FILE_CREATED || eventType == dto.FILE_UPDATED {
		tagging, err = s.minio.GetObjectTagging(ctx, s.minioConfig.Files, normalizedKey, minio.GetObjectTaggingOptions{})
		if err != nil {
			s.lgr.WithTracing(ctx).Errorf("Error during getting tags %v", err)
			return nil
		}
	}

	var users map[int64]*dto.User = map[int64]*dto.User{}
	var fileOwnerId int64
	if eventType == dto.FILE_CREATED || eventType == dto.FILE_UPDATED {
		if minioEvent == nil {
			s.lgr.WithTracing(ctx).Errorf("nil minioEvent %v", err)
			return nil
		}
		var participantIdSet = map[int64]bool{}
		participantIdSet[fileOwnerId] = true
		users = GetUsersRemotelyOrEmpty(s.lgr, participantIdSet, s.client, ctx)
	}
	return &HandleEventResponse{
		minioEvent: minioEvent,
		tagging:    tagging,
		users:      users,
	}
}

type HandleEventResponse struct {
	minioEvent *dto.MinioEvent
	tagging    *tags.Tags
	users      map[int64]*dto.User
}

func (r *HandleEventResponse) GetTags() (*bool, error) {
	if r == nil {
		return nil, nil
	}
	published, err := DeserializeTags(r.tagging)
	if err != nil {
		return nil, err
	}
	return &published, nil
}

func (r *HandleEventResponse) GetMinioEvent() *dto.MinioEvent {
	if r == nil {
		return nil
	}
	return r.minioEvent
}

func (s *EventService) SendToParticipants(ctx context.Context, normalizedKey string, chatId int64, eventType dto.EventType, participantIds []int64, response *HandleEventResponse, mce *dto.MetadataCache) {
	now := time.Now().UTC()

	// iterate over chat participants
	for _, participantId := range participantIds {
		var fileInfo *dto.FileInfoDto
		var err error
		if eventType == dto.FILE_CREATED || eventType == dto.FILE_UPDATED {
			if mce != nil {
				userId := &participantId
				fileInfo, err = s.filesService.GetFileInfo(ctx, false, utils.ChatIdNonExistent, utils.MessageIdNonExistent, userId, mce)
				if err != nil {
					s.lgr.WithTracing(ctx).Errorf("Error get file info: %v, skipping", err)
					continue
				}
				if response.minioEvent == nil {
					s.lgr.WithTracing(ctx).Errorf("nil minioEvent, skipping", err)
					continue
				}
				fileInfo.Owner = response.users[response.minioEvent.OwnerId]
			} else {
				s.lgr.WithTracing(ctx).Errorf("Missed objectInfo for %v", normalizedKey)
				continue
			}
		} else if eventType == dto.FILE_DELETED {
			fileItemUuid, err := utils.ParseFileItemUuid(normalizedKey)
			if err != nil {
				s.lgr.WithTracing(ctx).Errorf("Error parsing fileItemUuid for %v: %v", normalizedKey, err)
				return
			}

			fileInfo = &dto.FileInfoDto{
				Id:             normalizedKey,
				FileItemUuid:   fileItemUuid,
				LastModified:   now,
				CreateDateTime: now,
			}
		}
		var correlationId *string
		if mce != nil && mce.CorrelationId != nil {
			correlationId = mce.CorrelationId
		}
		err = s.publisher.PublishFileEvent(ctx, participantId, chatId, &dto.WrappedFileInfoDto{
			FileInfoDto: fileInfo,
		}, eventType, correlationId)
		if err != nil {
			s.lgr.WithTracing(ctx).Errorf("Error sending event for %v: %v", normalizedKey, err)
			return
		}
	}
}
