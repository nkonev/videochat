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
	"strings"
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

func (s *EventService) HandleEvent(participantIds []int64, normalizedKey string, chatId int64, eventName string, ctx context.Context) {
	GetLogEntry(ctx).Debugf("Got %v %v", normalizedKey, eventName)

	var objectInfo minio.ObjectInfo
	var err error
	var tagging *tags.Tags
	if strings.HasPrefix(eventName, utils.ObjectCreated) {
		objectInfo, err = s.minio.StatObject(ctx, s.minioConfig.Files, normalizedKey, minio.StatObjectOptions{})
		if err != nil {
			GetLogEntry(ctx).Errorf("Error during stat %v", err)
			return
		}

		tagging, err = s.minio.GetObjectTagging(ctx, s.minioConfig.Files, normalizedKey, minio.GetObjectTaggingOptions{})
		if err != nil {
			GetLogEntry(ctx).Errorf("Error during getting tags %v", err)
			return
		}
	}

	var filenameChatPrefix string = fmt.Sprintf("chat/%v/", chatId)
	count, err := s.filesService.GetCount(ctx, filenameChatPrefix)
	if err != nil {
		GetLogEntry(ctx).Errorf("Error during getting count %v", err)
		return
	}

	var users map[int64]*dto.User = map[int64]*dto.User{}
	var fileOwnerId int64
	if strings.HasPrefix(eventName, utils.ObjectCreated) {
		_, fileOwnerId, _, err = DeserializeMetadata(objectInfo.UserMetadata, false)
		if err != nil {
			Logger.Errorf("Error get metadata: %v", err)
			return
		}

		var participantIdSet = map[int64]bool{}
		participantIdSet[fileOwnerId] = true
		users = GetUsersRemotelyOrEmpty(participantIdSet, s.client, ctx)
	}

	// iterate over chat participants
	for _, participantId := range participantIds {
		var created bool
		var fileInfo *dto.FileInfoDto
		if strings.HasPrefix(eventName, utils.ObjectCreated) {
			created = true
			fileInfo, err = s.filesService.GetFileInfo(participantId, objectInfo, chatId, tagging, false)
			if err != nil {
				GetLogEntry(ctx).Errorf("Error get file info: %v, skipping", err)
				continue
			}
			fileInfo.Owner = users[fileOwnerId]
		} else if strings.HasPrefix(eventName, utils.ObjectRemoved) {
			created = false
			fileInfo = &dto.FileInfoDto{
				Id:           normalizedKey,
				LastModified: time.Now(),
			}
		}
		s.publisher.PublishFileEvent(participantId, chatId, &dto.WrappedFileInfoDto{
			FileInfoDto: fileInfo,
			Count:       int64(count),
		}, created, ctx)
	}
}
