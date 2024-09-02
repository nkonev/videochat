package tasks

import (
	"context"
	"github.com/ehsaniara/gointerlock"
	redisV8 "github.com/go-redis/redis/v8"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/s3"
	"nkonev.name/storage/services"
	"nkonev.name/storage/utils"
	"time"
)

type ActualizeGeneratedFilesTask struct {
	*gointerlock.GoInterval
}

func ActualizeGeneratedFilesScheduler(
	redisConnector *redisV8.Client,
	service *ActualizeGeneratedFilesService,
) *ActualizeGeneratedFilesTask {
	var interv = viper.GetDuration("schedulers.actualizeGeneratedFilesTask.interval")
	Logger.Infof("Created ActualizeGeneratedFilesScheduler with interval %v", interv)
	return &ActualizeGeneratedFilesTask{&gointerlock.GoInterval{
		Name:           "actualizeGeneratedFilesTask",
		Interval:       interv,
		Arg:            service.doJob,
		RedisConnector: redisConnector,
	}}
}

type ActualizeGeneratedFilesService struct {
	minioClient        *s3.InternalMinioClient
	minioBucketsConfig *utils.MinioConfig
	previewService     *services.PreviewService
	tracer             trace.Tracer
	redisInfoService   *services.RedisInfoService
	convertingService  *services.ConvertingService
}

func (srv *ActualizeGeneratedFilesService) doJob() {
	ctx, span := srv.tracer.Start(context.Background(), "scheduler.ActualizeGeneratedFiles")
	defer span.End()
	filenameChatPrefix := "chat/"
	srv.processFiles(ctx, filenameChatPrefix)
}

func (srv *ActualizeGeneratedFilesService) processFiles(c context.Context, filenameChatPrefix string) {
	GetLogEntry(c).Infof("Starting actualize generated files job")

	// create preview for files if need
	// and create _converted.webm
	GetLogEntry(c).Infof("Checking for missing previews and converted")
	var fileObjects <-chan minio.ObjectInfo = srv.minioClient.ListObjects(c, srv.minioBucketsConfig.Files, minio.ListObjectsOptions{
		Prefix:    filenameChatPrefix,
		Recursive: true,
		WithMetadata: true,
	})
	for fileOjInfo := range fileObjects {
		// here in minio 'chat/108/'
		GetLogEntry(c).Debugf("Start processing minio key '%v'", fileOjInfo.Key)
		if utils.IsVideo(fileOjInfo.Key) {
			// preview
			previewToCheck := utils.SetVideoPreviewExtension(fileOjInfo.Key)
			_, err := srv.minioClient.StatObject(c, srv.minioBucketsConfig.FilesPreview, previewToCheck, minio.StatObjectOptions{})
			if err != nil {
				GetLogEntry(c).Infof("Create missed preview %v for %v", previewToCheck, fileOjInfo.Key)
				srv.previewService.CreatePreview(c, fileOjInfo.Key)
			}

			// _converted.webm
			_, _, _, isMessageRecording, err := services.DeserializeMetadata(fileOjInfo.UserMetadata, true)
			if err != nil {
				GetLogEntry(c).Errorf("Unable to convert metadata for key %v: %v", fileOjInfo.Key, err)
				continue
			}
			isConverting, err := srv.redisInfoService.GetConverting(c, fileOjInfo.Key)
			if err != nil {
				GetLogEntry(c).Errorf("Unable to isConverting for key %v from redis: %v", fileOjInfo.Key, err)
				continue
			}
			_, err = srv.minioClient.StatObject(c, srv.minioBucketsConfig.Files, utils.GetKeyForConverted(fileOjInfo.Key), minio.StatObjectOptions{})
			if err != nil && (isMessageRecording != nil && *isMessageRecording) && !utils.IsConverted(fileOjInfo.Key) && !isConverting {
				GetLogEntry(c).Infof("Create missed converted for %v", fileOjInfo.Key)
				srv.convertingService.Convert(c, fileOjInfo.Key)
			}
		} else if utils.IsImage(fileOjInfo.Key) {
			previewToCheck := utils.SetImagePreviewExtension(fileOjInfo.Key)
			_, err := srv.minioClient.StatObject(c, srv.minioBucketsConfig.FilesPreview, previewToCheck, minio.StatObjectOptions{})
			if err != nil {
				GetLogEntry(c).Infof("Create preview for missing %v", fileOjInfo.Key)
				srv.previewService.CreatePreview(c, fileOjInfo.Key)
			}
		}

	}
	GetLogEntry(c).Infof("Checking for missing previews and converted finished")

	// remove previews of removed files
	GetLogEntry(c).Infof("Checking for excess previews")
	var previewObjects <-chan minio.ObjectInfo = srv.minioClient.ListObjects(c, srv.minioBucketsConfig.FilesPreview, minio.ListObjectsOptions{
		Prefix:       filenameChatPrefix,
		Recursive:    true,
		WithMetadata: true,
	})
	for previewOjInfo := range previewObjects {
		GetLogEntry(c).Debugf("Start processing minio key '%v'", previewOjInfo.Key)
		originalKey, err := services.GetOriginalKeyFromMetadata(previewOjInfo.UserMetadata, true)
		if err != nil {
			GetLogEntry(c).Errorf("Error during getting original key %v", err)
			continue
		}
		_, err = srv.minioClient.StatObject(c, srv.minioBucketsConfig.Files, originalKey, minio.StatObjectOptions{})
		maxConvertingDuration := viper.GetDuration("converting.maxDuration")
		if err != nil {
			GetLogEntry(c).Infof("Key %v is not found, deciding whether to remove the preview %v", originalKey, previewOjInfo.Key)
			if utils.IsConverted(originalKey) && previewOjInfo.LastModified.Add(maxConvertingDuration).After(time.Now().UTC()) {
				GetLogEntry(c).Infof("Age of the converted preview %v for key %v is lesser than %v, skipping deletion", previewOjInfo.Key, originalKey, maxConvertingDuration)
				continue
			} else {
				GetLogEntry(c).Infof("Will remove preview for %v", originalKey)
				err := srv.minioClient.RemoveObject(c, srv.minioBucketsConfig.FilesPreview, previewOjInfo.Key, minio.RemoveObjectOptions{})
				if err != nil {
					GetLogEntry(c).Errorf("Error during removing preview key %v", err)
					continue
				}
			}
		}
	}
	GetLogEntry(c).Infof("Checking for excess previews finished")

	GetLogEntry(c).Infof("End of generated files job")
}

func NewActualizeGeneratedFilesService(minioClient *s3.InternalMinioClient, minioBucketsConfig *utils.MinioConfig, previewService *services.PreviewService, redisInfoService *services.RedisInfoService, convertingService *services.ConvertingService) *ActualizeGeneratedFilesService {
	trcr := otel.Tracer("scheduler/actualize-generated-files")
	return &ActualizeGeneratedFilesService{
		minioClient:        minioClient,
		minioBucketsConfig: minioBucketsConfig,
		previewService:     previewService,
		tracer:             trcr,
		redisInfoService:   redisInfoService,
		convertingService:  convertingService,
	}
}
