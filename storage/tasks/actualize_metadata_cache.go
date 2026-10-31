package tasks

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/nkonev/dcron"
	redisLock "github.com/nkonev/dcron/plugin/lock/redis"
	otelTrace "github.com/nkonev/dcron/plugin/trace/otel"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"nkonev.name/storage/client"
	"nkonev.name/storage/db"
	"nkonev.name/storage/dto"
	"nkonev.name/storage/logger"
	"nkonev.name/storage/services"
	"nkonev.name/storage/utils"
)

type ActualizeMetadataCacheTask struct {
	dcron.Job
}

func ActualizeMetadataCacheScheduler(
	lgr *logger.Logger,
	service *ActualizeMetadataCacheService,
) *ActualizeMetadataCacheTask {
	const key = "actualizeMetadataCacheTask"
	var str = viper.GetString("schedulers." + key + ".cron")
	lgr.Infof("Created ActualizeMetadataCacheScheduler with cron %v", str)

	job := dcron.NewJob(key, str, func(ctx context.Context) error {
		service.doJob(ctx)
		return nil
	},
		otelTrace.WithTracing(service.tracer, "scheduler.ActualizeMetadataCache"),
		redisLock.WithLockTTL(viper.GetDuration("schedulers."+key+".expiration")),
	)

	return &ActualizeMetadataCacheTask{job}
}

type ActualizeMetadataCacheService struct {
	awsS3              *client.S3Wrapper
	minioBucketsConfig *utils.MinioConfig
	dba                *db.DB
	tracer             trace.Tracer
	lgr                *logger.Logger
}

func (srv *ActualizeMetadataCacheService) doJob(ctx context.Context) {
	filenameChatPrefix := "chat/"
	srv.processFiles(ctx, filenameChatPrefix)
}

func (srv *ActualizeMetadataCacheService) processFiles(c context.Context, filenameChatPrefix string) {
	srv.lgr.WithTracing(c).Infof("Starting actualize metadata cache job")

	maxFilesInBatch := viper.GetInt("schedulers.actualizeMetadataCacheTask.batch")

	// create metadata cache for files if need
	srv.lgr.WithTracing(c).Infof("Checking for missing metadata cache items")
	var continuationToken *string
	for {
		lst, err := srv.awsS3.ListObjectsV2WithMetadataAndTags(c, &s3.ListObjectsV2Input{
			Bucket:            &srv.minioBucketsConfig.Files,
			ContinuationToken: continuationToken,
			Prefix:            &filenameChatPrefix,
			MaxKeys:           new(int32(maxFilesInBatch)),
		})
		if err != nil {
			srv.lgr.WithTracing(c).Errorf("Error list objects for actualizing metadata cache: %v", err)
			break
		}
		continuationToken = lst.NextContinuationToken

		for i, fileOjInfo := range lst.Contents {
			if fileOjInfo.Key == nil {
				srv.lgr.WithTracing(c).Error("Nil key for idx '%v'", i)
				continue
			}

			aKey := *fileOjInfo.Key

			if fileOjInfo.Size == nil {
				srv.lgr.WithTracing(c).Error("Nil Size for key '%v'", aKey)
				continue
			}

			// here in minio 'chat/108/'

			srv.lgr.WithTracing(c).Debugf("Start processing minio key '%v'", aKey)

			chatId, err := utils.ParseChatId(aKey)
			if err != nil {
				srv.lgr.WithTracing(c).Errorf("Unable to parse chatId for %v: %v", aKey, err)
				continue
			}
			fileItemUuid, err := utils.ParseFileItemUuid(aKey)
			if err != nil {
				srv.lgr.WithTracing(c).Errorf("Unable to parse fileItemUuid for %v: %v", aKey, err)
				continue
			}
			filename, err := utils.ParseFileName(aKey)
			if err != nil {
				srv.lgr.WithTracing(c).Errorf("Unable to parse filename for %v: %v", aKey, err)
				continue
			}

			mcid := dto.MetadataCacheId{
				ChatId:       chatId,
				FileItemUuid: fileItemUuid,
				Filename:     filename,
			}
			metadataCache, err := db.Get(c, srv.dba, mcid, nil)
			if err != nil {
				srv.lgr.WithTracing(c).Errorf("Unable to check existence for %v: %v", mcid.String(), err)
				continue
			}

			if metadataCache == nil {
				srv.lgr.WithTracing(c).Infof("Create metadata cache item for missing %v", aKey)

				// use try* function for the case when there is no metadata (files were copied on the disk, so metadata wasn't preserved)
				ownerId, correlationId, timestamp, err := srv.tryGetMetadata(c, fileOjInfo)
				if err != nil {
					srv.lgr.WithTracing(c).Errorf("Unable to get metadata for %v: %v", aKey, err)

					continue
				}

				var correlationIdPtr *string
				if len(correlationId) > 0 {
					correlationIdPtr = &correlationId
				}

				eventTime := utils.GetEventTimeFromTimestamp(timestamp)

				// use try* function for the case when there is no metadata (files were copied on the disk, so metadata wasn't preserved)
				published, correlationIdFromTags := srv.tryGetTags(c, fileOjInfo)

				if correlationIdFromTags != nil {
					correlationIdPtr = correlationIdFromTags
				}

				err = db.Set(c, srv.dba, dto.MetadataCache{
					ChatId:         chatId,
					FileItemUuid:   fileItemUuid,
					Filename:       filename,
					OwnerId:        ownerId,
					CorrelationId:  correlationIdPtr,
					Published:      published,
					FileSize:       *fileOjInfo.Size,
					CreateDateTime: eventTime,
					EditDateTime:   eventTime,
				})
				if err != nil {
					srv.lgr.WithTracing(c).Errorf("Unable to create metadata cache for %v: %v", mcid.String(), err)
					continue
				}
			}
			// we ignore the case when metadataCache differs from fileOjInfo
			// because the checking would require get tags of each fileOjInfo
			// it can be a performance slowdown
			// tags -> published is the only thing which can differ, it will be synced when user press "[un]publish" button
		}

		if len(lst.Contents) < maxFilesInBatch {
			break
		}
	}
	srv.lgr.WithTracing(c).Infof("Checking for missing metadata cache items finished")

	// remove metadata cache of removed files
	srv.lgr.WithTracing(c).Infof("Checking for excess metadata cache items")
	// TODO rewrite with NewListPaginationKeyset()
	offset := 0
	for {
		metadatas, err := db.GetList(c, srv.dba, dto.NoChatId, dto.NoFileItemUuid, nil, db.NewListPaginationOffset(utils.DefaultSize, offset), true)
		if err != nil {
			srv.lgr.WithTracing(c).Errorf("Error during paginate: %v", err)
			return
		}

		wasDeletions := false
		for _, metadata := range metadatas {
			aKey := utils.BuildNormalizedKey(&metadata)
			srv.lgr.WithTracing(c).Debugf("Start processing minio key '%v'", aKey)
			// TODO not very optimal
			exists, _, err := srv.awsS3.FileExists(c, srv.minioBucketsConfig.Files, aKey)
			if err != nil {
				srv.lgr.WithTracing(c).Errorf("Unable to get exists for %v: %v", aKey, err)
				continue
			}

			if !exists {
				srv.lgr.WithTracing(c).Infof("Will remove metadata cache for %v", aKey)
				mcid := dto.MetadataCacheId{
					ChatId:       metadata.ChatId,
					FileItemUuid: metadata.FileItemUuid,
					Filename:     metadata.Filename,
				}
				err := db.Remove(c, srv.dba, mcid)
				if err != nil {
					srv.lgr.WithTracing(c).Errorf("Error during removing metadata cache for %v: %v", mcid.String(), err)
					continue
				}
				wasDeletions = true
			}
		}

		if !wasDeletions {
			offset += utils.DefaultSize
		}
		if len(metadatas) < utils.DefaultSize {
			break
		}
	}
	srv.lgr.WithTracing(c).Infof("Checking for excess metadata cache items finished")

	srv.lgr.WithTracing(c).Infof("End of actualize metadata cache job")
}

func (srv *ActualizeMetadataCacheService) tryGetTags(ctx context.Context, fileOjInfo dto.Object) (bool, *string) {
	published, correlationId, err := services.DeserializeTags(fileOjInfo.UserTags)
	if err != nil {
		srv.lgr.WithTracing(ctx).Errorf("Unable to deserialize tags for %v: %v", *fileOjInfo.Key, err)
		return false, correlationId
	}

	return published, correlationId
}

func (srv *ActualizeMetadataCacheService) tryGetMetadata(ctx context.Context, fileOjInfo dto.Object) (int64, string, int64, error) {
	// metadata can be absent because of poor backup
	ownerId, correlationId, timestamp, err := services.DeserializeMetadataFromMap(fileOjInfo.UserMetadata)
	if err != nil {
		fallbackUserId := viper.GetInt64("schedulers.actualizeMetadataCacheTask.fallbackOwnerId")

		if fallbackUserId > 0 {
			srv.lgr.WithTracing(ctx).Debugf("Unable to get metadata for %v: %v, going to fallback to user %v", *fileOjInfo.Key, err, fallbackUserId)
			ownerId = fallbackUserId
		} else {
			srv.lgr.WithTracing(ctx).Debugf("The fallbackOwnerId isn't configured, returning error")

			return 0, "", 0, err
		}
	}

	return ownerId, correlationId, timestamp, nil
}

func NewActualizeMetadataCacheService(lgr *logger.Logger, awsS3 *client.S3Wrapper, minioBucketsConfig *utils.MinioConfig, dba *db.DB) *ActualizeMetadataCacheService {
	trcr := otel.Tracer("scheduler/actualize-metadata-cache")
	return &ActualizeMetadataCacheService{
		awsS3:              awsS3,
		lgr:                lgr,
		minioBucketsConfig: minioBucketsConfig,
		dba:                dba,
		tracer:             trcr,
	}
}
