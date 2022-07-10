package redis

import (
	"context"
	"errors"
	"fmt"
	"github.com/ehsaniara/gointerlock"
	redisV8 "github.com/go-redis/redis/v8"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
	"nkonev.name/storage/client"
	"nkonev.name/storage/logger"
	"nkonev.name/storage/utils"
	"strings"
	"time"
)

type DeleteMissedInChatFilesService struct {
	minioClient        *minio.Client
	minioBucketsConfig *utils.MinioConfig
	chatClient         *client.RestClient
}

func NewDeleteMissedInChatFilesService(minioClient *minio.Client, minioBucketsConfig *utils.MinioConfig, chatClient *client.RestClient) *DeleteMissedInChatFilesService {
	return &DeleteMissedInChatFilesService{
		minioClient:        minioClient,
		minioBucketsConfig: minioBucketsConfig,
		chatClient:         chatClient,
	}
}

func (srv *DeleteMissedInChatFilesService) doJob() {
	filenameChatPrefix := fmt.Sprintf("chat/")
	ct := context.Background()
	srv.processEmbeddedFiles(filenameChatPrefix, ct)

	logger.Logger.Infof("End of cleaning embedded files job")
}

func (srv *DeleteMissedInChatFilesService) processEmbeddedFiles(filenameChatPrefix string, c context.Context) {
	var maxMinioKeysInBatch = viper.GetInt("minio.cleaner.embedded.maxKeys")
	logger.Logger.Infof("Starting cleaning embed files job with max minio keys limit = %v", maxMinioKeysInBatch)
	var objects <-chan minio.ObjectInfo = srv.minioClient.ListObjects(c, srv.minioBucketsConfig.Embedded, minio.ListObjectsOptions{
		WithMetadata: true,
		Prefix:       filenameChatPrefix,
		Recursive:    true,
	})

	var chatsWithFiles map[int64][]utils.Tuple = make(map[int64][]utils.Tuple)

	var i = 0
	// TODO is it ok to perform potentially long operations inside processing the channel ?
	for objInfo := range objects {
		// here in minio 'chat/108/b4c03030-e054-49b5-b63c-78808b4bdeff.png'
		logger.Logger.Infof("Start processing minio key '%v'", objInfo.Key)
		// in chat <p><img src="/api/storage/108/embed/b4c03030-e054-49b5-b63c-78808b4bdeff.png" style="width: 600px; height: 480px;"></p>
		chatId, err := extractChatId(objInfo.Key)
		if err != nil {
			logger.Logger.Errorf("Unable to extract chat id from %v", objInfo.Key)
			continue
		}

		if _, ok := chatsWithFiles[chatId]; !ok {
			logger.Logger.Debugf("Creating tuple array for chat id %v", chatId)
			chatsWithFiles[chatId] = []utils.Tuple{} // create tuple array if need
		}
		filename, err := extractFileName(objInfo.Key)
		if err != nil {
			logger.Logger.Errorf("Unable to extract filename from %v", objInfo.Key)
			continue
		}
		if time.Now().UTC().Sub(objInfo.LastModified) < viper.GetDuration("minio.cleaner.embedded.threshold") {
			logger.Logger.Infof("Minio object %v is too young to be cleared", objInfo.Key)
			continue
		}
		i++
		chatsWithFiles[chatId] = append(chatsWithFiles[chatId], utils.Tuple{Filename: filename, Exists: true, MinioKey: objInfo.Key})

		if i >= maxMinioKeysInBatch {
			i = 0
			srv.processChunk(chatsWithFiles, c)
			chatsWithFiles = map[int64][]utils.Tuple{}
		}
	}
	srv.processChunk(chatsWithFiles, c)
	chatsWithFiles = map[int64][]utils.Tuple{}
	logger.Logger.Infof("End of processEmbeddedFiles job")
}

func (srv *DeleteMissedInChatFilesService) processChunk(chatsWithFiles map[int64][]utils.Tuple, c context.Context) {
	chatsWithFilesResponse, err := srv.invokeChat(chatsWithFiles, c)
	if err != nil {
		logger.Logger.Errorf("Error during asking chat %v, skipping", err)
		return
	}
	for keyChatId, valuePairs := range chatsWithFilesResponse {
		for _, valuePair := range valuePairs {
			logger.Logger.Infof("Processing responded chat id %v file %v", keyChatId, valuePair.MinioKey)
			if !valuePair.Exists {
				logger.Logger.Infof("Deleting embedded file object %v", valuePair.MinioKey)
				err := srv.minioClient.RemoveObject(c, srv.minioBucketsConfig.Embedded, valuePair.MinioKey, minio.RemoveObjectOptions{})
				if err != nil {
					logger.Logger.Errorf("Object embedded file %v has been cleared from minio with error: %v", valuePair.MinioKey, err)
				} else {
					logger.Logger.Debugf("Object embedded file %v has been cleared from minio successfully", valuePair.MinioKey)
				}
			} else {
				logger.Logger.Infof("Responded chat id %v files file %v is present", keyChatId, valuePair.MinioKey)
			}
		}
		logger.Logger.Debugf("Completed processing chat id %v files", keyChatId)
	}
}

func extractChatId(minioKey string) (int64, error) {
	split := strings.Split(minioKey, "/")
	if len(split) < 2 {
		return 0, errors.New("Minio key is too short")
	}
	return utils.ParseInt64(split[1])
}

func extractFileName(minioKey string) (string, error) {
	split := strings.Split(minioKey, "/")
	if len(split) < 3 {
		return "", errors.New("Minio key is too short")
	}

	return split[2], nil
}

func (srv *DeleteMissedInChatFilesService) invokeChat(input map[int64][]utils.Tuple, c context.Context) (map[int64][]utils.Tuple, error) {
	return srv.chatClient.CheckFilesInChat(input, c)
}

type CleanEmbeddedFilesTask struct {
	*gointerlock.GoInterval
}

func DeleteMissedInChatFilesScheduler(
	redisConnector *redisV8.Client,
	service *DeleteMissedInChatFilesService,
) *CleanEmbeddedFilesTask {
	var interv = viper.GetDuration("minio.cleaner.embedded.interval")
	logger.Logger.Infof("Created DeleteMissedInChatFilesScheduler with interval %v", interv)
	return &CleanEmbeddedFilesTask{&gointerlock.GoInterval{
		Name:           "embeddedFilesCleaner",
		Interval:       interv,
		Arg:            service.doJob,
		RedisConnector: redisConnector,
	}}
}
