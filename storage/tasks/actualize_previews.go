package tasks

import (
	"context"
	"github.com/ehsaniara/gointerlock"
	redisV8 "github.com/go-redis/redis/v8"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"nkonev.name/storage/logger"
	"nkonev.name/storage/s3"
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
	var interv = viper.GetDuration("schedulers.actualizePreviewsTask.interval")
	logger.Logger.Infof("Created ActualizePreviewsScheduler with interval %v", interv)
	return &ActualizePreviewsTask{&gointerlock.GoInterval{
		Name:           "actualizePreviewsTask",
		Interval:       interv,
		Arg:            service.doJob,
		RedisConnector: redisConnector,
	}}
}

type ActualizePreviewsService struct {
	minioClient        *s3.InternalMinioClient
	minioBucketsConfig *utils.MinioConfig
	previewService     *services.PreviewService
	tracer             trace.Tracer
}

func (srv *ActualizePreviewsService) doJob() {
	ctx, span := srv.tracer.Start(context.Background(), "scheduler.ActualizePreviews")
	defer span.End()
	filenameChatPrefix := "chat/"
	srv.processFiles(ctx, filenameChatPrefix)
}

func (srv *ActualizePreviewsService) processFiles(c context.Context, filenameChatPrefix string) {
	logger.Logger.Infof("Starting actualize previews job")

	// create preview for files if need
	logger.Logger.Infof("Checking for missing previews")
	var fileObjects <-chan minio.ObjectInfo = srv.minioClient.ListObjects(c, srv.minioBucketsConfig.Files, minio.ListObjectsOptions{
		Prefix:    filenameChatPrefix,
		Recursive: true,
	})
	for fileOjInfo := range fileObjects {
		// here in minio 'chat/108/'
		logger.Logger.Debugf("Start processing minio key '%v'", fileOjInfo.Key)
		if utils.IsVideo(fileOjInfo.Key) {
			previewToCheck := utils.SetVideoPreviewExtension(fileOjInfo.Key)
			_, err := srv.minioClient.StatObject(c, srv.minioBucketsConfig.FilesPreview, previewToCheck, minio.StatObjectOptions{})
			if err != nil {
				logger.Logger.Infof("Create preview for missing %v", fileOjInfo.Key)
				srv.previewService.CreatePreview(c, fileOjInfo.Key)
			}
		} else if utils.IsImage(fileOjInfo.Key) {
			previewToCheck := utils.SetImagePreviewExtension(fileOjInfo.Key)
			_, err := srv.minioClient.StatObject(c, srv.minioBucketsConfig.FilesPreview, previewToCheck, minio.StatObjectOptions{})
			if err != nil {
				logger.Logger.Infof("Create preview for missing %v", fileOjInfo.Key)
				srv.previewService.CreatePreview(c, fileOjInfo.Key)
			}
		}

	}
	logger.Logger.Infof("Checking for missing previews finished")

	// remove previews of removed files
	logger.Logger.Infof("Checking for excess previews")
	var previewObjects <-chan minio.ObjectInfo = srv.minioClient.ListObjects(c, srv.minioBucketsConfig.FilesPreview, minio.ListObjectsOptions{
		Prefix:       filenameChatPrefix,
		Recursive:    true,
		WithMetadata: true,
	})
	for previewOjInfo := range previewObjects {
		logger.Logger.Debugf("Start processing minio key '%v'", previewOjInfo.Key)
		originalKey, err := services.GetOriginalKeyFromMetadata(previewOjInfo.UserMetadata, true)
		if err != nil {
			logger.Logger.Errorf("Error during getting original key %v", err)
			continue
		}
		_, err = srv.minioClient.StatObject(c, srv.minioBucketsConfig.Files, originalKey, minio.StatObjectOptions{})
		if err != nil {
			logger.Logger.Infof("Will remove preview for %v", originalKey)
			err := srv.minioClient.RemoveObject(c, srv.minioBucketsConfig.FilesPreview, previewOjInfo.Key, minio.RemoveObjectOptions{})
			if err != nil {
				logger.Logger.Errorf("Error during removing preview key %v", err)
				continue
			}
		}
	}
	logger.Logger.Infof("Checking for excess previews finished")

	logger.Logger.Infof("End of actualize previews job")
}

func NewActualizePreviewsService(minioClient *s3.InternalMinioClient, minioBucketsConfig *utils.MinioConfig, previewService *services.PreviewService) *ActualizePreviewsService {
	trcr := otel.Tracer("scheduler/clean-files-of-deleted-chat")
	return &ActualizePreviewsService{
		minioClient:        minioClient,
		minioBucketsConfig: minioBucketsConfig,
		previewService:     previewService,
		tracer:             trcr,
	}
}
