package redis

import (
	"context"
	"errors"
	"fmt"
	"github.com/ehsaniara/gointerlock"
	redisV8 "github.com/go-redis/redis/v8"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
	"nkonev.name/storage/logger"
	"nkonev.name/storage/utils"
	"strings"
)

type Tuple struct {
	minioKey string
	filename string
	exists   bool
}

func embedFilesJobFactory(minioClient *minio.Client, minioBucketsConfig *utils.MinioConfig) func() {
	return func() {
		filenameChatPrefix := fmt.Sprintf("chat/")
		var maxMinioKeysInBatch = viper.GetInt("minio.cleaner.embedded.maxKeys")
		logger.Logger.Infof("Starting cleaning embed files job with max minio keys limit = %v", maxMinioKeysInBatch)
		var objects <-chan minio.ObjectInfo = minioClient.ListObjects(context.Background(), minioBucketsConfig.Embedded, minio.ListObjectsOptions{
			WithMetadata: true,
			Prefix:       filenameChatPrefix,
			Recursive:    true,
		})

		var chatsWithFiles map[int64][]Tuple = make(map[int64][]Tuple)

		var i = 0
		// TODO is it ok to perform potentially long operations inside processing the channel ?
		for objInfo := range objects {
			i++

			// here in minio 'chat/108/b4c03030-e054-49b5-b63c-78808b4bdeff.png'
			logger.Logger.Infof("Processing object '%v'", objInfo.Key)
			// in chat <p><img src="/api/storage/108/embed/b4c03030-e054-49b5-b63c-78808b4bdeff.png" style="width: 600px; height: 480px;"></p>
			chatId, err := extractChatId(objInfo.Key)
			if err != nil {
				logger.Logger.Errorf("Unable to extract chat id from %v", objInfo.Key)
				continue
			}

			if _, ok := chatsWithFiles[chatId]; !ok {
				logger.Logger.Infof("Creating tuple array for chat id %v", chatId)
				chatsWithFiles[chatId] = []Tuple{} // create tuple array if need
			}
			filename, err := extractFileName(objInfo.Key)
			if err != nil {
				logger.Logger.Errorf("Unable to extract filename from %v", objInfo.Key)
				continue
			}
			chatsWithFiles[chatId] = append(chatsWithFiles[chatId], Tuple{filename: filename, exists: false, minioKey: objInfo.Key})

			if i >= maxMinioKeysInBatch {
				i = 0
				processChunk(chatsWithFiles, minioClient, minioBucketsConfig)
				chatsWithFiles = map[int64][]Tuple{}
			}
		}
		processChunk(chatsWithFiles, minioClient, minioBucketsConfig)
		chatsWithFiles = map[int64][]Tuple{}

		logger.Logger.Infof("End of cleaning job")
	}
}

func processChunk(chatsWithFiles map[int64][]Tuple, minioClient *minio.Client, minioBucketsConfig *utils.MinioConfig) {
	chatsWithFilesResponse, err := invokeChat(chatsWithFiles)
	if err != nil {
		logger.Logger.Errorf("Error during asking chat %v", err)
		return
	}
	for keyChatId, valuePairs := range chatsWithFilesResponse {
		logger.Logger.Infof("Processing chat id %v files", keyChatId)
		for _, valuePair := range valuePairs {
			if !valuePair.exists {
				err := minioClient.RemoveObject(context.Background(), minioBucketsConfig.Embedded, valuePair.minioKey, minio.RemoveObjectOptions{})
				if err != nil {
					logger.Logger.Errorf("Object %v has been cleared from minio with error: %v", valuePair.minioKey, err)
				} else {
					logger.Logger.Infof("Object %v has been cleared from minio successfully", valuePair.minioKey)
				}
			}
		}
		logger.Logger.Infof("Completed processing chat id files", keyChatId)
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

func invokeChat(input map[int64][]Tuple) (map[int64][]Tuple, error) {
	return nil, errors.New("Not implemented") // TODO
}

func CleanDeletedImagesFromMessageBody(
	redisConnector *redisV8.Client,
	minioClient *minio.Client,
	minioBucketsConfig *utils.MinioConfig,
) *gointerlock.GoInterval {
	var interv = viper.GetDuration("minio.cleaner.embedded.interval")
	return &gointerlock.GoInterval{
		Name:           "embedFilesCleaner",
		Interval:       interv,
		Arg:            embedFilesJobFactory(minioClient, minioBucketsConfig),
		RedisConnector: redisConnector,
	}
}
