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
		dcron.WithOverlappingBehaviour(dcron.OverlappingBehaviourKeepOldRunning),
	)

	return &ActualizeMetadataCacheTask{job}
}

type ActualizeMetadataCacheService struct {
	awsS3              *client.S3Wrapper
	minioBucketsConfig *utils.BucketsConfig
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

		toSave := []dto.MetadataCache{}

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

			srv.lgr.WithTracing(c).Debugf("Start processing minio key '%v' for save", aKey)

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

			updateTime := time.Now().UTC()

			toSave = append(toSave, dto.MetadataCache{
				ChatId:         chatId,
				FileItemUuid:   fileItemUuid,
				Filename:       filename,
				OwnerId:        ownerId,
				CorrelationId:  correlationIdPtr,
				Published:      published,
				FileSize:       *fileOjInfo.Size,
				CreateDateTime: eventTime,
				EditDateTime:   updateTime,
			})
		}

		err = db.SetList(c, srv.dba, toSave)
		if err != nil {
			srv.lgr.WithTracing(c).Errorf("Unable to save metadata cache: %v", err)
			break
		}

		if len(lst.Contents) < maxFilesInBatch {
			break
		}
	}
	srv.lgr.WithTracing(c).Infof("Checking for missing metadata cache items finished")

	// remove metadata cache of removed files
	srv.lgr.WithTracing(c).Infof("Checking for excess metadata cache items")

	ks := db.NewListPaginationKeyset(dto.NoFileItemUuid, dto.NoCreateDateTime, dto.NoFileName, false, maxFilesInBatch)

	for {
		metadatas, err := db.GetList(c, srv.dba, dto.NoChatId, dto.NoFileItemUuid, nil, ks, false)
		if err != nil {
			srv.lgr.WithTracing(c).Errorf("Error during paginate: %v", err)
			return
		}

		toRemove := []dto.MetadataCacheId{}
		for _, metadata := range metadatas {
			aKey := utils.BuildNormalizedKey(&metadata)
			srv.lgr.WithTracing(c).Debugf("Start processing minio key '%v' for remove", aKey)
			// TODO not very optimal
			exists, _, err := srv.awsS3.FileExists(c, srv.minioBucketsConfig.Files, aKey)
			if err != nil {
				srv.lgr.WithTracing(c).Errorf("Unable to get exists for %v: %v", aKey, err)
				continue
			}

			if !exists {
				srv.lgr.WithTracing(c).Infof("Going to remove metadata cache for %v", aKey)
				mcid := dto.MetadataCacheId{
					ChatId:       metadata.ChatId,
					FileItemUuid: metadata.FileItemUuid,
					Filename:     metadata.Filename,
				}
				toRemove = append(toRemove, mcid)
			}
		}

		err = db.RemoveList(c, srv.dba, toRemove)
		if err != nil {
			srv.lgr.WithTracing(c).Errorf("Error during removing metadata cache: %v", err)
			return
		}

		if len(metadatas) < maxFilesInBatch {
			break
		} else {
			if len(metadatas) >= 1 {
				last := metadatas[len(metadatas)-1]
				ks = db.NewListPaginationKeyset(last.FileItemUuid, &last.CreateDateTime, last.Filename, false, maxFilesInBatch)
			} else {
				srv.lgr.WithTracing(c).Warn("unexpected case")
				break
			}
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

func NewActualizeMetadataCacheService(lgr *logger.Logger, awsS3 *client.S3Wrapper, minioBucketsConfig *utils.BucketsConfig, dba *db.DB) *ActualizeMetadataCacheService {
	trcr := otel.Tracer("scheduler/actualize-metadata-cache")
	return &ActualizeMetadataCacheService{
		awsS3:              awsS3,
		lgr:                lgr,
		minioBucketsConfig: minioBucketsConfig,
		dba:                dba,
		tracer:             trcr,
	}
}
