package tasks

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/nkonev/dcron"
	redisLock "github.com/nkonev/dcron/plugin/lock/redis"
	otelTrace "github.com/nkonev/dcron/plugin/trace/otel"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"nkonev.name/storage/client"
	"nkonev.name/storage/logger"
	"nkonev.name/storage/utils"
)

type CleanFilesOfDeletedChatTask struct {
	dcron.Job
}

func CleanFilesOfDeletedChatScheduler(
	lgr *logger.Logger,
	service *CleanFilesOfDeletedChatService,
) *CleanFilesOfDeletedChatTask {
	const key = "cleanFilesOfDeletedChatTask"
	var str = viper.GetString("schedulers." + key + ".cron")
	lgr.Infof("Created CleanFilesOfDeletedChatScheduler with cron %v", str)

	job := dcron.NewJob(key, str, func(ctx context.Context) error {
		service.doJob(ctx)
		return nil
	},
		otelTrace.WithTracing(service.tracer, "scheduler.cleanFilesOfDeletedChat"),
		redisLock.WithLockTTL(viper.GetDuration("schedulers."+key+".expiration")),
	)

	return &CleanFilesOfDeletedChatTask{job}
}

type CleanFilesOfDeletedChatService struct {
	awsS3              *client.S3Wrapper
	minioBucketsConfig *utils.MinioConfig
	chatClient         *client.RestClient
	tracer             trace.Tracer
	lgr                *logger.Logger
	count              int64
}

func (srv *CleanFilesOfDeletedChatService) doJob(ctx context.Context) {
	srv.processChats(ctx)
}

func (srv *CleanFilesOfDeletedChatService) processChats(c context.Context) {
	filenameChatPrefix := "chat/"

	srv.lgr.WithTracing(c).Infof("Starting cleaning files of deleted chats job")

	chatCount, err := srv.chatClient.CountChats(c)
	if err != nil {
		srv.lgr.WithTracing(c).Errorf("Got error getting chat count: %v", err)
		return
	}

	if chatCount < srv.count {
		srv.lgr.WithTracing(c).Infof("There is %v chats in chat which is lower than %v threshold to start cleaning FilesOfDeletedChat", chatCount, srv.count)
		return
	}

	maxFilesInBatch := viper.GetInt("schedulers.cleanFilesOfDeletedChatTask.batch")

	var continuationToken *string
	for {
		lst, err := srv.awsS3.ListObjectsV2(c, &s3.ListObjectsV2Input{
			Bucket:            &srv.minioBucketsConfig.Files,
			Prefix:            &filenameChatPrefix,
			ContinuationToken: continuationToken,
			MaxKeys:           new(int32(maxFilesInBatch)),
		})
		if err != nil {
			srv.lgr.WithTracing(c).Errorf("Error list objects for cleaning files of deleted chats: %v", err)
			break
		}
		continuationToken = lst.NextContinuationToken

		chatIds := make([]int64, 0)

		for _, chatObjInfo := range lst.Contents {
			srv.lgr.WithTracing(c).Debugf("Start processing minio key '%v'", *chatObjInfo.Key)
			chatId, err := utils.ParseChatId(*chatObjInfo.Key)
			if err != nil {
				srv.lgr.WithTracing(c).Errorf("Unable to extract chat id from %v", *chatObjInfo.Key)
				continue
			}

			chatIds = append(chatIds, chatId)
		}

		uniqueChatIds := utils.Unique(chatIds)

		chatsExists, err := srv.chatClient.CheckIsChatExists(c, uniqueChatIds)
		if err != nil {
			srv.lgr.WithTracing(c).Errorf("Unable to check existence of chat ids %v: %v", chatIds, err)
			return
		}

		chatExistsMap := map[int64]bool{}
		for _, v := range *chatsExists {
			chatExistsMap[v.ChatId] = v.Exists
		}

		objsToDelete := []types.ObjectIdentifier{}

		for _, chatObjInfo := range lst.Contents {
			srv.lgr.WithTracing(c).Debugf("Start processing minio key '%v'", *chatObjInfo.Key)
			chatId, err := utils.ParseChatId(*chatObjInfo.Key)
			if err != nil {
				srv.lgr.WithTracing(c).Errorf("Unable to extract chat id from %v: %v", *chatObjInfo.Key, err)
				continue
			}

			if !chatExistsMap[chatId] {
				srv.lgr.WithTracing(c).Infof("Going to delete file object %v", *chatObjInfo.Key)
				objsToDelete = append(objsToDelete, types.ObjectIdentifier{
					Key: chatObjInfo.Key,
				})
			}
		}

		if len(objsToDelete) > 0 {
			_, err = srv.awsS3.DeleteObjects(c, &s3.DeleteObjectsInput{
				Bucket: &srv.minioBucketsConfig.Files,
				Delete: &types.Delete{
					Objects: objsToDelete,
				},
			})
			if err != nil {
				srv.lgr.WithTracing(c).Errorf("Error during deleting objects: %v", err)
				return
			}
		}

		if len(lst.Contents) < maxFilesInBatch {
			break
		}
	}

	srv.lgr.WithTracing(c).Infof("End of cleaning files of deleted chats job")
}

func NewCleanFilesOfDeletedChatService(lgr *logger.Logger, awsS3 *client.S3Wrapper, minioBucketsConfig *utils.MinioConfig, chatClient *client.RestClient) *CleanFilesOfDeletedChatService {
	trcr := otel.Tracer("scheduler/clean-files-of-deleted-chat")
	return &CleanFilesOfDeletedChatService{
		lgr:                lgr,
		awsS3:              awsS3,
		minioBucketsConfig: minioBucketsConfig,
		chatClient:         chatClient,
		tracer:             trcr,
		count:              viper.GetInt64("schedulers.cleanFilesOfDeletedChatTask.chatCount"),
	}
}
