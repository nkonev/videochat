package tasks

import (
	"context"
	"io"
	"time"

	smithymiddleware "github.com/aws/smithy-go/middleware"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/minio/minio-go/v7"
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
	minioClient        *client.InternalMinioClient
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

// rawResponseKey is the accessor key used to store and access the
// raw response within the response metadata.
type rawResponseKey struct{}

// AddRawResponseString middleware adds raw response on to the metadata
type AddRawResponseString struct{}

// ID the identifier for the ClientRequestID
func (m *AddRawResponseString) ID() string {
	return "AddRawResponseStringToMetadata"
}

// HandleDeserialize adds raw response on the middleware metadata
func (m AddRawResponseString) HandleDeserialize(ctx context.Context, in smithymiddleware.DeserializeInput, next smithymiddleware.DeserializeHandler) (
	out smithymiddleware.DeserializeOutput, metadata smithymiddleware.Metadata, err error,
) {
	out, metadata, err = next.HandleDeserialize(ctx, in)
	metadata.Set(rawResponseKey{}, out.RawResponse)
	return out, metadata, err
}

// AddRawResponseToMetadata adds middleware to the middleware stack that
// store raw response on to the metadata.
func AddRawResponseToMetadata(stack *smithymiddleware.Stack) error {
	return stack.Deserialize.Add(&AddRawResponseString{}, smithymiddleware.Before)
}

// GetRawResponse returns raw response set on metadata
func GetRawResponse(metadata smithymiddleware.Metadata) interface{} {
	return metadata.Get(rawResponseKey{})
}

func (srv *ActualizeGeneratedFilesService) processFiles(c context.Context, filenameChatPrefix string) {
	srv.lgr.WithTracing(c).Infof("Starting actualize generated files job")

	maxFilesInBatch := viper.GetInt("schedulers.actualizeGeneratedFilesTask.batch")

	// create preview for files if need
	// and create _converted.webm
	srv.lgr.WithTracing(c).Infof("Checking for missing previews and converted")
	var continuationToken *string
	for {
		lst, err := srv.awsS3.ListObjectsV2(c, &s3.ListObjectsV2Input{
			Bucket:            &srv.minioBucketsConfig.Files,
			Prefix:            &filenameChatPrefix,
			ContinuationToken: continuationToken,
		}, func(o *s3.Options) {
			o.APIOptions = append(o.APIOptions, AddRawResponseToMetadata)
		})
		if err != nil {
			srv.lgr.WithTracing(c).Errorf("Error list objects for checking for generated files")
			break
		}
		rr := GetRawResponse(lst.ResultMetadata).(*smithyhttp.Response)
		defer rr.Body.Close()
		bb, err := io.ReadAll(rr.Body)
		bbs := string(bb)
		println(bbs)

		continuationToken = lst.NextContinuationToken

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
				if !convertedExists && utils.IsVideo(aKey) && !utils.IsConverted(aKey) && !isConverting {
					// get metadata
					mdt, err := srv.awsS3.HeadObject(c, &s3.HeadObjectInput{
						Bucket: &srv.minioBucketsConfig.Files,
						Key:    &aKey,
					})
					if err != nil {
						srv.lgr.WithTracing(c).Errorf("Unable to get metadata of %v: %v", aKey, err)
						continue
					}

					// produce _converted.webm for message recording
					isMessageRecording := utils.GetBoolean(mdt.Metadata[services.MessageRecordingKey()])
					if isMessageRecording {
						srv.lgr.WithTracing(c).Infof("Create missed converted for %v", aKey)
						srv.convertingService.Convert(c, aKey)
					}
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

			if len(lst.Contents) < maxFilesInBatch {
				break
			}
		}
	}
	srv.lgr.WithTracing(c).Infof("Checking for missing previews and converted finished")
	continuationToken = nil

	// remove previews of removed files
	srv.lgr.WithTracing(c).Infof("Checking for excess previews")
	var previewObjects <-chan minio.ObjectInfo = srv.minioClient.ListObjects(c, srv.minioBucketsConfig.FilesPreview, minio.ListObjectsOptions{
		Prefix:       filenameChatPrefix,
		Recursive:    true,
		WithMetadata: true,
	})
	for previewOjInfo := range previewObjects {
		srv.lgr.WithTracing(c).Debugf("Start processing minio key '%v'", previewOjInfo.Key)
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
			srv.lgr.WithTracing(c).Infof("Key %v is not found, deciding whether to remove the preview %v", originalKey, previewOjInfo.Key)
			if utils.IsConverted(originalKey) && previewOjInfo.LastModified.Add(maxConvertingDuration).After(time.Now().UTC()) {
				srv.lgr.WithTracing(c).Infof("Age of the converted preview %v for key %v is lesser than %v, skipping deletion", previewOjInfo.Key, originalKey, maxConvertingDuration)
				continue
			} else {
				srv.lgr.WithTracing(c).Infof("Will remove preview for %v", originalKey)
				err := srv.minioClient.RemoveObject(c, srv.minioBucketsConfig.FilesPreview, previewOjInfo.Key, minio.RemoveObjectOptions{})
				if err != nil {
					srv.lgr.WithTracing(c).Errorf("Error during removing preview key %v", err)
					continue
				}
			}
		}
	}
	srv.lgr.WithTracing(c).Infof("Checking for excess previews finished")

	srv.lgr.WithTracing(c).Infof("End of generated files job")
}

func NewActualizeGeneratedFilesService(awsS3 *client.S3Wrapper, lgr *logger.Logger, minioClient *client.InternalMinioClient, minioBucketsConfig *utils.MinioConfig, previewService *services.PreviewService, redisInfoService *services.RedisInfoService, convertingService *services.ConvertingService) *ActualizeGeneratedFilesService {
	trcr := otel.Tracer("scheduler/actualize-generated-files")
	return &ActualizeGeneratedFilesService{
		awsS3:              awsS3,
		lgr:                lgr,
		minioClient:        minioClient,
		minioBucketsConfig: minioBucketsConfig,
		previewService:     previewService,
		tracer:             trcr,
		redisInfoService:   redisInfoService,
		convertingService:  convertingService,
	}
}
