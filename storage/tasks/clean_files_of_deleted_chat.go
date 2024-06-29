package tasks

import (
	"context"
	"fmt"
	"github.com/ehsaniara/gointerlock"
	redisV8 "github.com/go-redis/redis/v8"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"nkonev.name/storage/client"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/s3"
	"nkonev.name/storage/utils"
)

type CleanFilesOfDeletedChatTask struct {
	*gointerlock.GoInterval
}

func CleanFilesOfDeletedChatScheduler(
	redisConnector *redisV8.Client,
	service *CleanFilesOfDeletedChatService,
) *CleanFilesOfDeletedChatTask {
	var interv = viper.GetDuration("schedulers.cleanFilesOfDeletedChatTask.interval")
	Logger.Infof("Created CleanFilesOfDeletedChatScheduler with interval %v", interv)
	return &CleanFilesOfDeletedChatTask{&gointerlock.GoInterval{
		Name:           "cleanFilesOfDeletedChatTask",
		Interval:       interv,
		Arg:            service.doJob,
		RedisConnector: redisConnector,
	}}
}

type CleanFilesOfDeletedChatService struct {
	minioClient        *s3.InternalMinioClient
	minioBucketsConfig *utils.MinioConfig
	chatClient         *client.RestClient
	tracer             trace.Tracer
}

func (srv *CleanFilesOfDeletedChatService) doJob() {
	ctx, span := srv.tracer.Start(context.Background(), "scheduler.cleanFilesOfDeletedChat")
	defer span.End()
	srv.processChats(ctx)
}

func (srv *CleanFilesOfDeletedChatService) processChats(c context.Context) {
	filenameChatPrefix := "chat/"

	GetLogEntry(c).Infof("Starting cleaning files of deleted chats job")

	// get only top-level chats (no recursive)
	var objectsChats <-chan minio.ObjectInfo = srv.minioClient.ListObjects(c, srv.minioBucketsConfig.Files, minio.ListObjectsOptions{
		Prefix:    filenameChatPrefix,
		Recursive: false,
	})

	chatIds := make([]int64, 0)
	for chatObjInfo := range objectsChats {
		// here in minio 'chat/108/'
		GetLogEntry(c).Debugf("Start processing minio key '%v'", chatObjInfo.Key)
		chatId, err := utils.ParseChatId(chatObjInfo.Key)
		if err != nil {
			GetLogEntry(c).Errorf("Unable to extract chat id from %v", chatObjInfo.Key)
			continue
		}
		GetLogEntry(c).Debugf("Successfully got chatId '%v'", chatId)

		chatIds = append(chatIds, chatId)
		if len(chatIds) >= viper.GetInt("schedulers.cleanFilesOfDeletedChatTask.batchChats") {
			srv.processBatch(c, chatIds)
			chatIds = make([]int64, 0)
		}

	}

	// process leftovers
	if len(chatIds) > 0 {
		srv.processBatch(c, chatIds)
	}

	GetLogEntry(c).Infof("End of cleaning files of deleted chats job")
}

func(srv *CleanFilesOfDeletedChatService) processBatch(c context.Context, chatIds []int64) {
	// check chat's existence
	chatsExists, err := srv.chatClient.CheckIsChatExists(c, chatIds)
	if err != nil {
		GetLogEntry(c).Errorf("Unable to chech existence of chat id %v", chatIds)
		return
	}

	for _, chatExists := range *chatsExists {
		chatId := chatExists.ChatId
		doesChatExists := chatExists.Exists
		// performing cleanup in minio - getting subfolders (recursively)
		filenameChatFilesPrefix := fmt.Sprintf("chat/%v/", chatId)
		var objectsOfChat <-chan minio.ObjectInfo = srv.minioClient.ListObjects(c, srv.minioBucketsConfig.Files, minio.ListObjectsOptions{
			Prefix:    filenameChatFilesPrefix,
			Recursive: true,
		})
		for objInfo := range objectsOfChat {
			// here in minio 'chat/108/'
			GetLogEntry(c).Debugf("Start processing minio key '%v'", objInfo.Key)

			if !doesChatExists {
				GetLogEntry(c).Infof("Deleting file(directory) object %v", objInfo.Key)
				err := srv.minioClient.RemoveObject(c, srv.minioBucketsConfig.Files, objInfo.Key, minio.RemoveObjectOptions{})
				if err != nil {
					GetLogEntry(c).Errorf("Object file %v has been cleared from minio with error: %v", objInfo.Key, err)
				} else {
					GetLogEntry(c).Debugf("Object file %v has been cleared from minio successfully", objInfo.Key)
				}
			} else {
				GetLogEntry(c).Debugf("Chat %v is present, skipping", chatId)
			}
		}
	}
}

func NewCleanFilesOfDeletedChatService(minioClient *s3.InternalMinioClient, minioBucketsConfig *utils.MinioConfig, chatClient *client.RestClient) *CleanFilesOfDeletedChatService {
	trcr := otel.Tracer("scheduler/clean-files-of-deleted-chat")
	return &CleanFilesOfDeletedChatService{
		minioClient:        minioClient,
		minioBucketsConfig: minioBucketsConfig,
		chatClient:         chatClient,
		tracer:             trcr,
	}
}
