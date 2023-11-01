package tasks

import (
	"context"
	"fmt"
	"github.com/ehsaniara/gointerlock"
	redisV8 "github.com/go-redis/redis/v8"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
	"nkonev.name/storage/client"
	"nkonev.name/storage/logger"
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
	logger.Logger.Infof("Created CleanFilesOfDeletedChatScheduler with interval %v", interv)
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
}

func (srv *CleanFilesOfDeletedChatService) doJob() {
	ct := context.Background()
	srv.processChats(ct)
}

func (srv *CleanFilesOfDeletedChatService) processChats(c context.Context) {
	filenameChatPrefix := "chat/"

	logger.Logger.Infof("Starting cleaning files of deleted chats job")

	// get only top-level chats (no recursive)
	var objectsChats <-chan minio.ObjectInfo = srv.minioClient.ListObjects(c, srv.minioBucketsConfig.Files, minio.ListObjectsOptions{
		Prefix:    filenameChatPrefix,
		Recursive: false,
	})

	for chatObjInfo := range objectsChats {
		// here in minio 'chat/108/'
		logger.Logger.Debugf("Start processing minio key '%v'", chatObjInfo.Key)
		chatId, err := utils.ParseChatId(chatObjInfo.Key)
		if err != nil {
			logger.Logger.Errorf("Unable to extract chat id from %v", chatObjInfo.Key)
			continue
		}
		logger.Logger.Debugf("Successfully got chatId '%v'", chatId)


		// check chat's existence
		chatExists, err := srv.chatClient.CheckIsChatExists(chatId, c)
		if err != nil {
			logger.Logger.Errorf("Unable to chech existence of chat id %v", chatId)
			continue
		}


		// performing cleanup in minio - getting subfolders (recursively)
		filenameChatFilesPrefix := fmt.Sprintf("chat/%v/", chatId)
		var objectsOfChat <-chan minio.ObjectInfo = srv.minioClient.ListObjects(c, srv.minioBucketsConfig.Files, minio.ListObjectsOptions{
			Prefix:    filenameChatFilesPrefix,
			Recursive: true,
		})
		for objInfo := range objectsOfChat {
			// here in minio 'chat/108/'
			logger.Logger.Debugf("Start processing minio key '%v'", objInfo.Key)

			if !chatExists {
				logger.Logger.Infof("Deleting file(directory) object %v", objInfo.Key)
				err := srv.minioClient.RemoveObject(c, srv.minioBucketsConfig.Files, objInfo.Key, minio.RemoveObjectOptions{})
				if err != nil {
					logger.Logger.Errorf("Object file %v has been cleared from minio with error: %v", objInfo.Key, err)
				} else {
					logger.Logger.Debugf("Object file %v has been cleared from minio successfully", objInfo.Key)
				}
			} else {
				logger.Logger.Debugf("Chat %v is present, skipping", chatId)
			}
		}

	}


	logger.Logger.Infof("End of cleaning files of deleted chats job")
}

func NewCleanFilesOfDeletedChatService(minioClient *s3.InternalMinioClient, minioBucketsConfig *utils.MinioConfig, chatClient *client.RestClient) *CleanFilesOfDeletedChatService {
	return &CleanFilesOfDeletedChatService{
		minioClient:        minioClient,
		minioBucketsConfig: minioBucketsConfig,
		chatClient:         chatClient,
	}
}
