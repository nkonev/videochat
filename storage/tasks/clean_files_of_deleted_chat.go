package tasks

import (
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/nkonev/dcron"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"nkonev.name/storage/client"
	"nkonev.name/storage/logger"
	"nkonev.name/storage/s3"
	"nkonev.name/storage/utils"
)

type CleanFilesOfDeletedChatTask struct {
	dcron.Job
}

func CleanFilesOfDeletedChatScheduler(
	lgr *logger.Logger,
	service *CleanFilesOfDeletedChatService,
) *CleanFilesOfDeletedChatTask {
	const key = "cleanFilesOfDeletedChatTask"
	var str = viper.GetString("schedulers." + key + ".cron")
	lgr.Infof("Created CleanFilesOfDeletedChatScheduler with cron %v", str)

	job := dcron.NewJob(key, str, func(ctx context.Context) error {
		service.doJob(ctx)
		return nil
	}, dcron.WithTracing(service.spanStarter, service.spanFinisher))

	return &CleanFilesOfDeletedChatTask{job}
}

type CleanFilesOfDeletedChatService struct {
	minioClient        *s3.InternalMinioClient
	minioBucketsConfig *utils.MinioConfig
	chatClient         *client.RestClient
	tracer             trace.Tracer
	lgr                *logger.Logger
}

func (srv *CleanFilesOfDeletedChatService) doJob(ctx context.Context) {
	srv.processChats(ctx)
}

func (srv *CleanFilesOfDeletedChatService) processChats(c context.Context) {
	filenameChatPrefix := "chat/"

	srv.lgr.WithTracing(c).Infof("Starting cleaning files of deleted chats job")

	// get only top-level chats (no recursive)
	var objectsChats <-chan minio.ObjectInfo = srv.minioClient.ListObjects(c, srv.minioBucketsConfig.Files, minio.ListObjectsOptions{
		Prefix:    filenameChatPrefix,
		Recursive: false,
	})

	chatIds := make([]int64, 0)
	for chatObjInfo := range objectsChats {
		// here in minio 'chat/108/'
		srv.lgr.WithTracing(c).Debugf("Start processing minio key '%v'", chatObjInfo.Key)
		chatId, err := utils.ParseChatId(chatObjInfo.Key)
		if err != nil {
			srv.lgr.WithTracing(c).Errorf("Unable to extract chat id from %v", chatObjInfo.Key)
			continue
		}
		srv.lgr.WithTracing(c).Debugf("Successfully got chatId '%v'", chatId)

		chatIds = append(chatIds, chatId)
		if len(chatIds) >= viper.GetInt("schedulers.cleanFilesOfDeletedChatTask.batchChats") {
			srv.processBatch(c, chatIds)
			chatIds = make([]int64, 0)
		}

	}

	// process leftovers
	if len(chatIds) > 0 {
		srv.processBatch(c, chatIds)
	}

	srv.lgr.WithTracing(c).Infof("End of cleaning files of deleted chats job")
}

func (srv *CleanFilesOfDeletedChatService) processBatch(c context.Context, chatIds []int64) {
	// check chat's existence
	chatsExists, err := srv.chatClient.CheckIsChatExists(c, chatIds)
	if err != nil {
		srv.lgr.WithTracing(c).Errorf("Unable to chech existence of chat id %v", chatIds)
		return
	}

	for _, chatExists := range *chatsExists {
		chatId := chatExists.ChatId
		doesChatExists := chatExists.Exists
		// performing cleanup in minio - getting subfolders (recursively)
		filenameChatFilesPrefix := fmt.Sprintf("chat/%v/", chatId)
		var objectsOfChat <-chan minio.ObjectInfo = srv.minioClient.ListObjects(c, srv.minioBucketsConfig.Files, minio.ListObjectsOptions{
			Prefix:    filenameChatFilesPrefix,
			Recursive: true,
		})
		for objInfo := range objectsOfChat {
			// here in minio 'chat/108/'
			srv.lgr.WithTracing(c).Debugf("Start processing minio key '%v'", objInfo.Key)

			if !doesChatExists {
				srv.lgr.WithTracing(c).Infof("Deleting file(directory) object %v", objInfo.Key)
				err := srv.minioClient.RemoveObject(c, srv.minioBucketsConfig.Files, objInfo.Key, minio.RemoveObjectOptions{})
				if err != nil {
					srv.lgr.WithTracing(c).Errorf("Object file %v has been cleared from minio with error: %v", objInfo.Key, err)
				} else {
					srv.lgr.WithTracing(c).Debugf("Object file %v has been cleared from minio successfully", objInfo.Key)
				}
			} else {
				srv.lgr.WithTracing(c).Debugf("Chat %v is present, skipping", chatId)
			}
		}
	}
}

func (srv *CleanFilesOfDeletedChatService) spanStarter(ctx context.Context) (context.Context, any) {
	return srv.tracer.Start(ctx, "scheduler.cleanFilesOfDeletedChat")
}

func (srv *CleanFilesOfDeletedChatService) spanFinisher(ctx context.Context, span any) {
	span.(trace.Span).End()
}

func NewCleanFilesOfDeletedChatService(lgr *logger.Logger, minioClient *s3.InternalMinioClient, minioBucketsConfig *utils.MinioConfig, chatClient *client.RestClient) *CleanFilesOfDeletedChatService {
	trcr := otel.Tracer("scheduler/clean-files-of-deleted-chat")
	return &CleanFilesOfDeletedChatService{
		lgr:                lgr,
		minioClient:        minioClient,
		minioBucketsConfig: minioBucketsConfig,
		chatClient:         chatClient,
		tracer:             trcr,
	}
}
