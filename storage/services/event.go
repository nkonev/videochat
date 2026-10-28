package services

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"nkonev.name/storage/client"
	"nkonev.name/storage/dto"
	"nkonev.name/storage/logger"
	"nkonev.name/storage/producer"
	"nkonev.name/storage/utils"
)

func NewEventService(lgr *logger.Logger, client *client.RestClient, awsS3 *client.S3Wrapper, minioConfig *utils.MinioConfig, filesService *FilesService, publisher *producer.RabbitFileUploadedPublisher) *EventService {
	return &EventService{
		client:       client,
		awsS3:        awsS3,
		minioConfig:  minioConfig,
		filesService: filesService,
		publisher:    publisher,
		lgr:          lgr,
	}
}

type EventService struct {
	client       *client.RestClient
	awsS3        *client.S3Wrapper
	minioConfig  *utils.MinioConfig
	filesService *FilesService
	publisher    *producer.RabbitFileUploadedPublisher
	lgr          *logger.Logger
}

func (s *EventService) HandleEvent(ctx context.Context, normalizedKey string, chatId int64, eventType dto.EventType, minioEvent *dto.MinioEvent) *HandleEventResponse {
	s.lgr.WithTracing(ctx).Debugf("Got %v %v", normalizedKey, eventType)

	var err error

	var tm map[string]string
	if eventType == dto.FILE_CREATED || eventType == dto.FILE_UPDATED {
		tagging, err := s.awsS3.GetObjectTagging(ctx, &s3.GetObjectTaggingInput{
			Bucket: &s.minioConfig.Files,
			Key:    &normalizedKey,
		})
		if err != nil {
			s.lgr.WithTracing(ctx).Errorf("Error during getting tags %v", err)
			return nil
		}

		tm = ConvertTaggingToMap(tagging.TagSet)
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
		tagging:    tm,
		users:      users,
	}
}

type HandleEventResponse struct {
	minioEvent *dto.MinioEvent
	tagging    map[string]string
	users      map[int64]*dto.User
}

func (r *HandleEventResponse) GetTags() (*bool, *string, error) {
	if r == nil {
		return nil, nil, nil
	}
	published, correlationId, err := DeserializeTags(r.tagging)
	if err != nil {
		return nil, nil, err
	}
	return &published, correlationId, nil
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
				if fileInfo == nil {
					s.lgr.WithTracing(ctx).Errorf("nil file info, skipping")
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
		if mce != nil {
			if mce.CorrelationId != nil {
				correlationId = mce.CorrelationId
			}
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
