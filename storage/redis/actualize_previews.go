package redis

/*
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

type ActualizePreviewsTask struct {
	*gointerlock.GoInterval
}

func ActualizePreviewsScheduler(
	redisConnector *redisV8.Client,
	service *ActualizePreviewsService,
) *ActualizePreviewsTask {
	var interv = viper.GetDuration("minio.cleaner.previews.interval")
	logger.Logger.Infof("Created ActualizePreviewsScheduler with interval %v", interv)
	return &ActualizePreviewsTask{&gointerlock.GoInterval{
		Name:           "deletePreviewsOfDeletedFilesCleaner",
		Interval:       interv,
		Arg:            service.doJob,
		RedisConnector: redisConnector,
	}}
}

type ActualizePreviewsService struct {
	minioClient        *minio.Client
	minioBucketsConfig *utils.MinioConfig
	chatClient         *client.RestClient
}

func (srv *ActualizePreviewsService) doJob() {
	ct := context.Background()
	filenameChatPrefix := "chat/"
	srv.processFiles(filenameChatPrefix, ct)
}

func (srv *ActualizePreviewsService) processFiles(filenameChatPrefix string, c context.Context) {
	logger.Logger.Infof("Starting cleaning files of deleted chats job")
	var objects <-chan minio.ObjectInfo = srv.minioClient.ListObjects(c, srv.minioBucketsConfig.Files, minio.ListObjectsOptions{
		Prefix:    filenameChatPrefix,
		Recursive: true,
	})

	for objInfo := range objects {
		// here in minio 'chat/108/'
		logger.Logger.Infof("Start processing minio key '%v'", objInfo.Key)
		objInfo.Key
	}
	logger.Logger.Infof("End of processFiles job")
}

func NewActualizePreviewsService(minioClient *minio.Client, minioBucketsConfig *utils.MinioConfig, chatClient *client.RestClient) *ActualizePreviewsService {
	return &ActualizePreviewsService{
		minioClient:        minioClient,
		minioBucketsConfig: minioBucketsConfig,
		chatClient:         chatClient,
	}
}
*/
