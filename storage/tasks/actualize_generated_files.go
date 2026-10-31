package tasks

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/nkonev/dcron"
	redisLock "github.com/nkonev/dcron/plugin/lock/redis"
	otelTrace "github.com/nkonev/dcron/plugin/trace/otel"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"nkonev.name/storage/client"
	"nkonev.name/storage/logger"
	"nkonev.name/storage/services"
	"nkonev.name/storage/utils"
)

type ActualizeGeneratedFilesTask struct {
	dcron.Job
}

func ActualizeGeneratedFilesScheduler(
	lgr *logger.Logger,
	service *ActualizeGeneratedFilesService,
) *ActualizeGeneratedFilesTask {
	const key = "actualizeGeneratedFilesTask"
	var str = viper.GetString("schedulers." + key + ".cron")
	lgr.Infof("Created ActualizeGeneratedFilesScheduler with cron %v", str)

	job := dcron.NewJob(key, str, func(ctx context.Context) error {
		service.doJob(ctx)
		return nil
	},
		otelTrace.WithTracing(service.tracer, "scheduler.ActualizeGeneratedFiles"),
		redisLock.WithLockTTL(viper.GetDuration("schedulers."+key+".expiration")),
	)

	return &ActualizeGeneratedFilesTask{job}
}

type ActualizeGeneratedFilesService struct {
	awsS3              *client.S3Wrapper
	minioBucketsConfig *utils.MinioConfig
	previewService     *services.PreviewService
	tracer             trace.Tracer
	redisInfoService   *services.RedisInfoService
	convertingService  *services.ConvertingService
	lgr                *logger.Logger
}

func (srv *ActualizeGeneratedFilesService) doJob(ctx context.Context) {
	filenameChatPrefix := "chat/"
	srv.processFiles(ctx, filenameChatPrefix)
}

func (srv *ActualizeGeneratedFilesService) processFiles(c context.Context, filenameChatPrefix string) {
	srv.lgr.WithTracing(c).Infof("Starting actualize generated files job")

	maxFilesInBatch := viper.GetInt("schedulers.actualizeGeneratedFilesTask.batch")

	// create preview for files if need
	// and create _converted.webm
	srv.lgr.WithTracing(c).Infof("Checking for missing previews and converted")
	var continuationToken1 *string
	for {
		lst, err := srv.awsS3.ListObjectsV2WithMetadataAndTags(c, &s3.ListObjectsV2Input{
			Bucket:            &srv.minioBucketsConfig.Files,
			Prefix:            &filenameChatPrefix,
			ContinuationToken: continuationToken1,
			MaxKeys:           new(int32(maxFilesInBatch)),
		})
		if err != nil {
			srv.lgr.WithTracing(c).Errorf("Error list objects for checking for generated files")
			break
		}
		continuationToken1 = lst.NextContinuationToken

		for _, fileOjInfo := range lst.Contents {
			// here in minio 'chat/108/'
			aKey := *fileOjInfo.Key
			srv.lgr.WithTracing(c).Debugf("Start processing minio key '%v'", aKey)
			if utils.IsVideo(aKey) {
				// preview
				previewToCheck := utils.SetVideoPreviewExtension(aKey)

				previewExists, _, err := srv.awsS3.FileExists(c, srv.minioBucketsConfig.FilesPreview, previewToCheck)
				if err != nil {
					srv.lgr.WithTracing(c).Errorf("Unable to check existence for %v: %v", previewToCheck, err)
					continue
				}
				if !previewExists {
					srv.lgr.WithTracing(c).Infof("Create missed preview %v for %v", previewToCheck, aKey)
					srv.previewService.CreatePreview(c, aKey)
				}

				isMessageRecording := utils.GetBoolean(fileOjInfo.UserMetadata[services.MessageRecordingKey()])

				isConverting, err := srv.redisInfoService.GetOriginalConverting(c, aKey)
				if err != nil {
					srv.lgr.WithTracing(c).Errorf("Unable to isConverting for key %v from redis: %v", aKey, err)
					continue
				}

				keyOfConverted := utils.GetKeyForConverted(aKey)
				convertedExists, _, err := srv.awsS3.FileExists(c, srv.minioBucketsConfig.Files, keyOfConverted)
				if err != nil {
					srv.lgr.WithTracing(c).Errorf("Unable to check existence for %v: %v", keyOfConverted, err)
					continue
				}

				if !convertedExists && utils.IsVideo(aKey) && isMessageRecording && !utils.IsConverted(aKey) && !isConverting {
					// produce _converted.webm for message recording
					srv.lgr.WithTracing(c).Infof("Create missed converted for %v", aKey)
					srv.convertingService.Convert(c, aKey)
				}
			} else if utils.IsImage(aKey) {
				previewToCheck := utils.SetImagePreviewExtension(aKey)
				exists, _, err := srv.awsS3.FileExists(c, srv.minioBucketsConfig.FilesPreview, previewToCheck)
				if err != nil {
					srv.lgr.WithTracing(c).Errorf("Unable to check existence for %v: %v", previewToCheck, err)
					continue
				}
				if !exists {
					srv.lgr.WithTracing(c).Infof("Create preview for missing %v", aKey)
					srv.previewService.CreatePreview(c, aKey)
				}
			}
		}

		if len(lst.Contents) < maxFilesInBatch {
			break
		}
	}
	srv.lgr.WithTracing(c).Infof("Checking for missing previews and converted finished")

	// remove previews of removed files
	var continuationToken2 *string
	srv.lgr.WithTracing(c).Infof("Checking for excess previews")
	for {
		lst, err := srv.awsS3.ListObjectsV2WithMetadataAndTags(c, &s3.ListObjectsV2Input{
			Bucket:            &srv.minioBucketsConfig.FilesPreview,
			Prefix:            &filenameChatPrefix,
			ContinuationToken: continuationToken2,
			MaxKeys:           new(int32(maxFilesInBatch)),
		})
		if err != nil {
			srv.lgr.WithTracing(c).Errorf("Error list objects for checking for generated files")
			break
		}
		continuationToken2 = lst.NextContinuationToken

		for _, previewOjInfo := range lst.Contents {
			aKey := *previewOjInfo.Key
			srv.lgr.WithTracing(c).Debugf("Start processing minio key '%v'", aKey)
			originalKey, err := services.GetOriginalKeyFromMetadata(previewOjInfo.UserMetadata)
			if err != nil {
				srv.lgr.WithTracing(c).Debugf("There is no original key, it's an old entry, skipping")
				continue
			}
			exists, _, err := srv.awsS3.FileExists(c, srv.minioBucketsConfig.Files, originalKey)
			if err != nil {
				srv.lgr.WithTracing(c).Errorf("Unable to get exists for %v: %v", originalKey, err)
				continue
			}

			maxConvertingDuration := viper.GetDuration("converting.maxDuration")
			if !exists {
				srv.lgr.WithTracing(c).Infof("Key %v is not found, deciding whether to remove the preview %v", originalKey, aKey)
				if utils.IsConverted(originalKey) && previewOjInfo.LastModified.Add(maxConvertingDuration).After(time.Now().UTC()) {
					srv.lgr.WithTracing(c).Infof("Age of the converted preview %v for key %v is lesser than %v, skipping deletion", aKey, originalKey, maxConvertingDuration)
					continue
				} else {
					srv.lgr.WithTracing(c).Infof("Will remove preview for %v", originalKey)
					_, err = srv.awsS3.DeleteObject(c, &s3.DeleteObjectInput{
						Bucket: &srv.minioBucketsConfig.FilesPreview,
						Key:    &aKey,
					})
					if err != nil {
						srv.lgr.WithTracing(c).Errorf("Error during removing preview key %v", err)
						continue
					}
				}
			}
		}

		if len(lst.Contents) < maxFilesInBatch {
			break
		}
	}
	srv.lgr.WithTracing(c).Infof("Checking for excess previews finished")

	srv.lgr.WithTracing(c).Infof("End of generated files job")
}

func NewActualizeGeneratedFilesService(awsS3 *client.S3Wrapper, lgr *logger.Logger, minioBucketsConfig *utils.MinioConfig, previewService *services.PreviewService, redisInfoService *services.RedisInfoService, convertingService *services.ConvertingService) *ActualizeGeneratedFilesService {
	trcr := otel.Tracer("scheduler/actualize-generated-files")
	return &ActualizeGeneratedFilesService{
		awsS3:              awsS3,
		lgr:                lgr,
		minioBucketsConfig: minioBucketsConfig,
		previewService:     previewService,
		tracer:             trcr,
		redisInfoService:   redisInfoService,
		convertingService:  convertingService,
	}
}
