package redis

import (
	"context"
	"github.com/ehsaniara/gointerlock"
	redisV8 "github.com/go-redis/redis/v8"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
	"nkonev.name/storage/client"
	"nkonev.name/storage/logger"
	"nkonev.name/storage/utils"
)

type CleanFilesOfDeletedChatTask struct {
	*gointerlock.GoInterval
}

func CleanFilesOfDeletedChatScheduler(
	redisConnector *redisV8.Client,
	service *CleanFilesOfDeletedChatService,
) *CleanFilesOfDeletedChatTask {
	var interv = viper.GetDuration("minio.cleaner.files.interval")
	logger.Logger.Infof("Created CleanFilesOfDeletedChatScheduler with interval %v", interv)
	return &CleanFilesOfDeletedChatTask{&gointerlock.GoInterval{
		Name:           "deletedChatFilesCleaner",
		Interval:       interv,
		Arg:            service.doJob,
		RedisConnector: redisConnector,
	}}
}

type CleanFilesOfDeletedChatService struct {
	minioClient        *minio.Client
	minioBucketsConfig *utils.MinioConfig
	chatClient         *client.RestClient
}

func (srv *CleanFilesOfDeletedChatService) doJob() {
	ct := context.Background()
	filenameChatPrefix := "chat/"
	srv.processChats(filenameChatPrefix, ct)
}

func (srv *CleanFilesOfDeletedChatService) processChats(filenameChatPrefix string, c context.Context) {
	logger.Logger.Infof("Starting cleaning files of deleted chats job")
	var objects <-chan minio.ObjectInfo = srv.minioClient.ListObjects(c, srv.minioBucketsConfig.Files, minio.ListObjectsOptions{
		Prefix:    filenameChatPrefix,
		Recursive: true,
	})

	for objInfo := range objects {
		// here in minio 'chat/108/'
		logger.Logger.Debugf("Start processing minio key '%v'", objInfo.Key)
		chatId, err := utils.ParseChatId(objInfo.Key)
		if err != nil {
			logger.Logger.Errorf("Unable to extract chat id from %v", objInfo.Key)
			continue
		}
		logger.Logger.Debugf("Successfully get chatId '%v'", chatId)

		exists, err := srv.chatClient.CheckIsChatExists(chatId, c)
		if err != nil {
			logger.Logger.Errorf("Unable to chech existence of chat id from %v", objInfo.Key)
			continue
		}
		if !exists {
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
	logger.Logger.Infof("End of processChats job")
}

func NewCleanFilesOfDeletedChatService(minioClient *minio.Client, minioBucketsConfig *utils.MinioConfig, chatClient *client.RestClient) *CleanFilesOfDeletedChatService {
	return &CleanFilesOfDeletedChatService{
		minioClient:        minioClient,
		minioBucketsConfig: minioBucketsConfig,
		chatClient:         chatClient,
	}
}
