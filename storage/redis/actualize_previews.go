package redis

import (
	"context"
	"github.com/ehsaniara/gointerlock"
	redisV8 "github.com/go-redis/redis/v8"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
	"nkonev.name/storage/logger"
	"nkonev.name/storage/services"
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
	previewService     *services.PreviewService
}

func (srv *ActualizePreviewsService) doJob() {
	ct := context.Background()
	filenameChatPrefix := "chat/"
	srv.processFiles(filenameChatPrefix, ct)
}

func (srv *ActualizePreviewsService) processFiles(filenameChatPrefix string, c context.Context) {
	logger.Logger.Infof("Starting actualize previews job")
	var fileObjects <-chan minio.ObjectInfo = srv.minioClient.ListObjects(c, srv.minioBucketsConfig.Files, minio.ListObjectsOptions{
		Prefix:    filenameChatPrefix,
		Recursive: true,
	})

	logger.Logger.Infof("Checking for missing previews")
	// create preview for files if need
	for fileOjInfo := range fileObjects {
		// here in minio 'chat/108/'
		logger.Logger.Infof("Start processing minio key '%v'", fileOjInfo.Key)
		if utils.IsVideo(fileOjInfo.Key) {
			previewToCheck := utils.SetVideoPreviewExtension(fileOjInfo.Key)
			_, err := srv.minioClient.StatObject(c, srv.minioBucketsConfig.FilesPreview, previewToCheck, minio.StatObjectOptions{})
			if err != nil {
				logger.Logger.Infof("Create preview for missing %v", fileOjInfo.Key)
				srv.previewService.CreatePreview(fileOjInfo.Key, c)
			}
		} else if utils.IsImage(fileOjInfo.Key) {
			previewToCheck := utils.SetImagePreviewExtension(fileOjInfo.Key)
			_, err := srv.minioClient.StatObject(c, srv.minioBucketsConfig.FilesPreview, previewToCheck, minio.StatObjectOptions{})
			if err != nil {
				logger.Logger.Infof("Create preview for missing %v", fileOjInfo.Key)
				srv.previewService.CreatePreview(fileOjInfo.Key, c)
			}
		}

	}
	logger.Logger.Infof("Checking for missing previews finished")

	logger.Logger.Infof("End of actualize previews job")
}

func NewActualizePreviewsService(minioClient *minio.Client, minioBucketsConfig *utils.MinioConfig, previewService *services.PreviewService) *ActualizePreviewsService {
	return &ActualizePreviewsService{
		minioClient:        minioClient,
		minioBucketsConfig: minioBucketsConfig,
		previewService:     previewService,
	}
}
