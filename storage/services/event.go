package services

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/tags"
	"nkonev.name/storage/client"
	"nkonev.name/storage/dto"
	"nkonev.name/storage/logger"
	"nkonev.name/storage/producer"
	"nkonev.name/storage/s3"
	"nkonev.name/storage/utils"
	"time"
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

func (s *EventService) HandleEvent(ctx context.Context, normalizedKey string, chatId int64, eventType utils.EventType) *HandleEventResponse {
	s.lgr.WithTracing(ctx).Debugf("Got %v %v", normalizedKey, eventType)

	var err error

	var objectInfo minio.ObjectInfo
	var tagging *tags.Tags
	if eventType == utils.FILE_CREATED || eventType == utils.FILE_UPDATED {
		objectInfo, err = s.minio.StatObject(ctx, s.minioConfig.Files, normalizedKey, minio.StatObjectOptions{})
		if err != nil {
			s.lgr.WithTracing(ctx).Errorf("Error during stat %v", err)
			return nil
		}

		tagging, err = s.minio.GetObjectTagging(ctx, s.minioConfig.Files, normalizedKey, minio.GetObjectTaggingOptions{})
		if err != nil {
			s.lgr.WithTracing(ctx).Errorf("Error during getting tags %v", err)
			return nil
		}
	}

	var users map[int64]*dto.User = map[int64]*dto.User{}
	var fileOwnerId int64
	if eventType == utils.FILE_CREATED || eventType == utils.FILE_UPDATED {
		_, fileOwnerId, _, _, err = DeserializeMetadata(objectInfo.UserMetadata, false)
		if err != nil {
			s.lgr.WithTracing(ctx).Errorf("Error get metadata: %v", err)
			return nil
		}

		var participantIdSet = map[int64]bool{}
		participantIdSet[fileOwnerId] = true
		users = GetUsersRemotelyOrEmpty(s.lgr, participantIdSet, s.client, ctx)
	}
	return &HandleEventResponse{
		objectInfo:  &objectInfo,
		tagging:     tagging,
		users:       users,
		fileOwnerId: fileOwnerId,
	}
}

type HandleEventResponse struct {
	objectInfo  *minio.ObjectInfo
	tagging     *tags.Tags
	users       map[int64]*dto.User
	fileOwnerId int64
}

func (s *EventService) SendToParticipants(ctx context.Context, normalizedKey string, chatId int64, eventType utils.EventType, participantIds []int64, response *HandleEventResponse) {
	if response != nil {
		// iterate over chat participants
		for _, participantId := range participantIds {
			var fileInfo *dto.FileInfoDto
			var err error
			if eventType == utils.FILE_CREATED || eventType == utils.FILE_UPDATED {
				if response.objectInfo != nil {
					userId := &participantId
					fileInfo, err = s.filesService.GetFileInfo(ctx, false, utils.ChatIdNonExistent, utils.MessageIdNonExistent, userId, *response.objectInfo, response.tagging, false)
					if err != nil {
						s.lgr.WithTracing(ctx).Errorf("Error get file info: %v, skipping", err)
						continue
					}
					fileInfo.Owner = response.users[response.fileOwnerId]
				} else {
					s.lgr.WithTracing(ctx).Errorf("Missed objectInfo")
					continue
				}
			} else if eventType == utils.FILE_DELETED {
				fileInfo = &dto.FileInfoDto{
					Id:           normalizedKey,
					LastModified: time.Now().UTC(),
				}
			}
			s.publisher.PublishFileEvent(ctx, participantId, chatId, &dto.WrappedFileInfoDto{
				FileInfoDto: fileInfo,
			}, eventType)
		}
	}
}
