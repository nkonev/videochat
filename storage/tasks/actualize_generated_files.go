package tasks

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/minio/minio-go/v7"
	"github.com/nkonev/dcron"
	redisLock "github.com/nkonev/dcron/plugin/lock/redis"
	otelTrace "github.com/nkonev/dcron/plugin/trace/otel"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"nkonev.name/storage/client"
	"nkonev.name/storage/dto"
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

type RequestUserMetadata struct {
}

func AddRequestUserMetadataMiddleware(stack *middleware.Stack) error {
	return stack.Build.Add(&RequestUserMetadata{}, middleware.After)
}

func (m *RequestUserMetadata) ID() string { return "RequestUserMetadata" }

func (m *RequestUserMetadata) HandleBuild(
	ctx context.Context, in middleware.BuildInput, next middleware.BuildHandler,
) (
	out middleware.BuildOutput, metadata middleware.Metadata, err error,
) {
	req, ok := in.Request.(*smithyhttp.Request)
	if !ok {
		return out, metadata, fmt.Errorf("unknown request type %T", req)
	}

	ok = dto.RequestUserMetadataFromContext(ctx)
	if ok {
		q := req.URL.Query()
		q.Set("metadata", "true")
		req.URL.RawQuery = q.Encode()
	}

	return next.HandleBuild(ctx, in)
}

func AddCacheBodyToStringMiddleware(stack *middleware.Stack) error {
	// /home/nkonev/go/pkg/mod/github.com/aws/aws-sdk-go-v2/service/s3@v1.114.5/internal/customizations/handle_200_error.go
	return stack.Deserialize.Insert(&CacheBodyToString{}, "OperationDeserializer", middleware.After)
}

type CacheBodyToString struct{}

func (*CacheBodyToString) ID() string {
	return "CacheBodyToString"
}

type CacheBodyToStringKeyType string

const CacheBodyToStringKey CacheBodyToStringKeyType = "CacheBodyToString"

func SetCachedResponseToMetadata(metadata middleware.Metadata, v string) {
	metadata.Set(CacheBodyToStringKey, v)
}

func GetCachedResponse(metadata middleware.Metadata) (string, error) {
	v := metadata.Get(CacheBodyToStringKey)
	switch typed := v.(type) {
	case string:
		return typed, nil
	default:
		return "", fmt.Errorf("unknown type: %T", typed)
	}
}

func (m *CacheBodyToString) HandleDeserialize(
	ctx context.Context, input middleware.DeserializeInput, next middleware.DeserializeHandler,
) (
	output middleware.DeserializeOutput, metadata middleware.Metadata, err error,
) {
	out, metadata, err := next.HandleDeserialize(ctx, input)
	if err != nil {
		return out, metadata, fmt.Errorf("error next.HandleDeserialize %w", err)
	}

	ok := dto.CacheResponseToStringFromContext(ctx)
	if ok {
		resp, ok := out.RawResponse.(*smithyhttp.Response)
		if !ok {
			return out, metadata, fmt.Errorf("unknown response type %T", resp)
		}
		origBody := resp.Body
		defer origBody.Close()

		bb, err := io.ReadAll(origBody)
		if err != nil {
			return out, metadata, fmt.Errorf("error reading response %w", err)
		}

		strbb := string(bb)

		SetCachedResponseToMetadata(metadata, strbb)

		resp.Body = io.NopCloser(bytes.NewReader(bb))
	}

	return out, metadata, err
}

func (srv *ActualizeGeneratedFilesService) processFiles(c context.Context, filenameChatPrefix string) {
	srv.lgr.WithTracing(c).Infof("Starting actualize generated files job")

	maxFilesInBatch := viper.GetInt("schedulers.actualizeGeneratedFilesTask.batch")

	// create preview for files if need
	// and create _converted.webm
	srv.lgr.WithTracing(c).Infof("Checking for missing previews and converted")
	var continuationToken *string
	for {
		cr := dto.ContextWithRequestUserMetadata(c)
		crr := dto.ContextWithCacheResponseToString(cr)
		lst, err := srv.awsS3.ListObjectsV2(crr, &s3.ListObjectsV2Input{
			Bucket:            &srv.minioBucketsConfig.Files,
			Prefix:            &filenameChatPrefix,
			ContinuationToken: continuationToken,
			MaxKeys:           new(int32(maxFilesInBatch)),
		}, func(o *s3.Options) {
			// see all middlewares here /home/nkonev/go/pkg/mod/github.com/aws/smithy-go@v1.28.6/middleware/stack.go :: Stack
			o.APIOptions = append(o.APIOptions, AddRequestUserMetadataMiddleware, AddCacheBodyToStringMiddleware)
		})
		if err != nil {
			srv.lgr.WithTracing(c).Errorf("Error list objects for checking for generated files")
			break
		}
		//rr :=awsmiddleware.GetRawResponse(lst.ResultMetadata).(*smithyhttp.Response)
		// defer rr.Body.Close()
		// bb, err := io.ReadAll(rr.Body)
		// bbs := string(bb)
		// println(bbs)
		rr, err := GetCachedResponse(lst.ResultMetadata)
		if err != nil {
			srv.lgr.WithTracing(c).Errorf("Error list objects for checking for generated files")
			break
		}
		println(rr)

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
				// TODO rustfs restore isMessageRecording here
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
		}

		if len(lst.Contents) < maxFilesInBatch {
			break
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
