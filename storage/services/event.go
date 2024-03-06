package services

import (
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/tags"
	"nkonev.name/storage/client"
	"nkonev.name/storage/dto"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/producer"
	"nkonev.name/storage/s3"
	"nkonev.name/storage/utils"
	"time"
)

func NewEventService(client *client.RestClient, minio *s3.InternalMinioClient, minioConfig *utils.MinioConfig, filesService *FilesService, publisher *producer.RabbitFileUploadedPublisher) *EventService {
	return &EventService{
		client:       client,
		minio:        minio,
		minioConfig:  minioConfig,
		filesService: filesService,
		publisher:    publisher,
	}
}

type EventService struct {
	client       *client.RestClient
	minio        *s3.InternalMinioClient
	minioConfig  *utils.MinioConfig
	filesService *FilesService
	publisher    *producer.RabbitFileUploadedPublisher
}

func (s *EventService) HandleEvent(normalizedKey string, chatId int64, eventType utils.EventType, ctx context.Context) (*HandleEventResponse, error) {
	GetLogEntry(ctx).Debugf("Got %v %v", normalizedKey, eventType)

	var err error

	var objectInfo minio.ObjectInfo
	var tagging *tags.Tags
	if eventType == utils.FILE_CREATED || eventType == utils.FILE_UPDATED {
		objectInfo, err = s.minio.StatObject(ctx, s.minioConfig.Files, normalizedKey, minio.StatObjectOptions{})
		if err != nil {
			GetLogEntry(ctx).Errorf("Error during stat %v", err)
			return nil, err
		}

		tagging, err = s.minio.GetObjectTagging(ctx, s.minioConfig.Files, normalizedKey, minio.GetObjectTaggingOptions{})
		if err != nil {
			GetLogEntry(ctx).Errorf("Error during getting tags %v", err)
			return nil, err
		}
	}

	var filenameChatPrefix string = fmt.Sprintf("chat/%v/", chatId)
	count, err := s.filesService.GetCount(ctx, filenameChatPrefix)
	if err != nil {
		GetLogEntry(ctx).Errorf("Error during getting count %v", err)
		return nil, err
	}

	var users map[int64]*dto.User = map[int64]*dto.User{}
	var fileOwnerId int64
	if eventType == utils.FILE_CREATED || eventType == utils.FILE_UPDATED {
		_, fileOwnerId, _, err = DeserializeMetadata(objectInfo.UserMetadata, false)
		if err != nil {
			Logger.Errorf("Error get metadata: %v", err)
			return nil, err
		}

		var participantIdSet = map[int64]bool{}
		participantIdSet[fileOwnerId] = true
		users = GetUsersRemotelyOrEmpty(participantIdSet, s.client, ctx)
	}
	return &HandleEventResponse{
		objectInfo:  &objectInfo,
		tagging:     tagging,
		count:       count,
		users:       users,
		fileOwnerId: fileOwnerId,
	}, nil
}

type HandleEventResponse struct {
	objectInfo *minio.ObjectInfo
	tagging *tags.Tags
	count int
	users map[int64]*dto.User
	fileOwnerId int64
}

func (s *EventService) SendToParticipants(normalizedKey string, chatId int64, eventType utils.EventType, participantIds []int64, response *HandleEventResponse, ctx context.Context) {
	// iterate over chat participants
	for _, participantId := range participantIds {
		var fileInfo *dto.FileInfoDto
		var err error
		if eventType == utils.FILE_CREATED || eventType == utils.FILE_UPDATED {
			if response.objectInfo != nil {
				fileInfo, err = s.filesService.GetFileInfo(ctx, participantId, *response.objectInfo, chatId, response.tagging, false)
				if err != nil {
					GetLogEntry(ctx).Errorf("Error get file info: %v, skipping", err)
					continue
				}
				fileInfo.Owner = response.users[response.fileOwnerId]
			} else {
				GetLogEntry(ctx).Errorf("Missed objectInfo")
				continue
			}
		} else if eventType == utils.FILE_DELETED {
			fileInfo = &dto.FileInfoDto{
				Id:           normalizedKey,
				LastModified: time.Now(),
			}
		}
		s.publisher.PublishFileEvent(participantId, chatId, &dto.WrappedFileInfoDto{
			FileInfoDto: fileInfo,
			Count:       int64(response.count),
		}, eventType, ctx)
	}

}
