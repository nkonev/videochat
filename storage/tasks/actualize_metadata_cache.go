package tasks

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/nkonev/dcron"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"nkonev.name/storage/db"
	"nkonev.name/storage/dto"
	"nkonev.name/storage/logger"
	"nkonev.name/storage/s3"
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
		service.doJob()
		return nil
	})

	return &ActualizeMetadataCacheTask{job}
}

type ActualizeMetadataCacheService struct {
	minioClient        *s3.InternalMinioClient
	minioBucketsConfig *utils.MinioConfig
	dba                *db.DB
	tracer             trace.Tracer
	lgr                *logger.Logger
}

func (srv *ActualizeMetadataCacheService) doJob() {
	ctx, span := srv.tracer.Start(context.Background(), "scheduler.ActualizeMetadataCache")
	defer span.End()
	filenameChatPrefix := "chat/"
	srv.processFiles(ctx, filenameChatPrefix)
}

func (srv *ActualizeMetadataCacheService) processFiles(c context.Context, filenameChatPrefix string) {
	srv.lgr.WithTracing(c).Infof("Starting actualize actualize metadata cache job")

	// create metadata cache for files if need
	srv.lgr.WithTracing(c).Infof("Checking for missing metadata cache items")
	var initialFileObjects <-chan minio.ObjectInfo = srv.minioClient.ListObjects(c, srv.minioBucketsConfig.Files, minio.ListObjectsOptions{
		Prefix:       filenameChatPrefix,
		Recursive:    true,
		WithMetadata: true,
	})
	for fileOjInfo := range initialFileObjects {
		// here in minio 'chat/108/'
		srv.lgr.WithTracing(c).Debugf("Start processing minio key '%v'", fileOjInfo.Key)

		chatId, err := utils.ParseChatId(fileOjInfo.Key)
		if err != nil {
			srv.lgr.WithTracing(c).Errorf("Unable to parse chatId for %v: %v", fileOjInfo.Key, err)
			continue
		}
		fileItemUuid, err := utils.ParseFileItemUuid(fileOjInfo.Key)
		if err != nil {
			srv.lgr.WithTracing(c).Errorf("Unable to parse fileItemUuid for %v: %v", fileOjInfo.Key, err)
			continue
		}
		filename, err := utils.ParseFileName(fileOjInfo.Key)
		if err != nil {
			srv.lgr.WithTracing(c).Errorf("Unable to parse filename for %v: %v", fileOjInfo.Key, err)
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
			srv.lgr.WithTracing(c).Infof("Create metadata cache item for missing %v", fileOjInfo.Key)

			_, ownerId, correlationId, err := services.DeserializeMetadata(fileOjInfo.UserMetadata, true)
			if err != nil {
				srv.lgr.WithTracing(c).Errorf("Unable to get metadata for %v: %v", fileOjInfo.Key, err)
				continue
			}

			var correlationIdPtr *string
			if len(correlationId) > 0 {
				correlationIdPtr = &correlationId
			}

			tags, err := srv.minioClient.GetObjectTagging(c, srv.minioBucketsConfig.Files, fileOjInfo.Key, minio.GetObjectTaggingOptions{})
			if err != nil {
				srv.lgr.WithTracing(c).Errorf("Unable to get tags for %v: %v", fileOjInfo.Key, err)
				continue
			}

			published, err := services.DeserializeTags(tags)
			if err != nil {
				srv.lgr.WithTracing(c).Errorf("Unable to deserialize tags for %v: %v", fileOjInfo.Key, err)
				continue
			}

			err = db.Set(c, srv.dba, dto.MetadataCache{
				ChatId:         chatId,
				FileItemUuid:   fileItemUuid,
				Filename:       filename,
				OwnerId:        ownerId,
				CorrelationId:  correlationIdPtr,
				Published:      published,
				FileSize:       fileOjInfo.Size,
				CreateDateTime: fileOjInfo.LastModified,
				EditDateTime:   fileOjInfo.LastModified,
			})
			if err != nil {
				srv.lgr.WithTracing(c).Errorf("Unable to create metadata cache for %v: %v", mcid.String(), err)
				continue
			}
		}
	}
	srv.lgr.WithTracing(c).Infof("Checking for missing metadata cache items finished")

	// remove metadata cache of removed files
	srv.lgr.WithTracing(c).Infof("Checking for excess metadata cache items")
	offset := 0
	for {
		metadatas, err := db.GetList(c, srv.dba, dto.NoChatId, dto.NoFileItemUuid, nil, utils.DefaultSize, offset)
		if err != nil {
			srv.lgr.WithTracing(c).Errorf("Error during paginate: %v", err)
			return
		}
		for _, metadata := range metadatas {
			aKey := utils.BuildNormalizedKey(&metadata)
			srv.lgr.WithTracing(c).Debugf("Start processing minio key '%v'", aKey)
			exists, _, err := srv.minioClient.FileExists(c, srv.minioBucketsConfig.Files, aKey)
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
			}
		}

		offset += utils.DefaultSize
		if len(metadatas) < utils.DefaultSize {
			break
		}
	}
	srv.lgr.WithTracing(c).Infof("Checking for excess metadata cache items finished")

	srv.lgr.WithTracing(c).Infof("End of actualize metadata cache job")
}

func NewActualizeMetadataCacheService(lgr *logger.Logger, minioClient *s3.InternalMinioClient, minioBucketsConfig *utils.MinioConfig, dba *db.DB) *ActualizeMetadataCacheService {
	trcr := otel.Tracer("scheduler/actualize-metadata-cache")
	return &ActualizeMetadataCacheService{
		lgr:                lgr,
		minioClient:        minioClient,
		minioBucketsConfig: minioBucketsConfig,
		dba:                dba,
		tracer:             trcr,
	}
}
