package services

import (
	"bytes"
	"context"
	"github.com/minio/minio-go/v7"
	"net/url"
	"nkonev.name/storage/dto"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/utils"
	"os/exec"
	"strings"
	"time"
)

type PreviewService struct {
	minio       *minio.Client
	minioConfig *utils.MinioConfig
}

func NewPreviewService(minio *minio.Client, minioConfig *utils.MinioConfig) *PreviewService {
	return &PreviewService{
		minio:       minio,
		minioConfig: minioConfig,
	}
}

func (s PreviewService) HandleMinioEvent(data *dto.MinioEvent) {
	Logger.Debugf("Got %v", data)
	ctx := context.Background()
	normalizedKey := utils.StripBucketName(data.Key, s.minioConfig.Files)
	if strings.HasPrefix(data.EventName, utils.ObjectCreated) {
		if utils.IsImage(normalizedKey) {
			// TODO image preview
		} else if utils.IsVideo(normalizedKey) {
			d, _ := time.ParseDuration("10m")
			presignedUrl, err := s.minio.PresignedGetObject(ctx, s.minioConfig.Files, normalizedKey, d, url.Values{})
			if err != nil {
				Logger.Errorf("Error during getting presigned url for %v", normalizedKey)
				return
			}
			stringPresingedUrl := presignedUrl.String()

			ffCmd := exec.Command("ffmpeg",
				"-i", stringPresingedUrl, "-vf", "thumbnail", "-frames:v", "1",
				"-c:v", "png", "-f", "rawvideo", "-an", "-")

			// getting real error msg : https://stackoverflow.com/questions/18159704/how-to-debug-exit-status-1-error-when-running-exec-command-in-golang
			output, err := ffCmd.Output()
			if err != nil {
				Logger.Errorf("Error during creating thumbnail %v for %v", err, normalizedKey)
				return
			}
			newKey := utils.FilesIdToFilesPreviewId(normalizedKey, s.minioConfig)
			newKey = utils.SetVideoPreviewExtension(newKey)

			var objectSize int64 = int64(len(output))
			reader := bytes.NewReader(output)
			_, err = s.minio.PutObject(ctx, s.minioConfig.FilesPreview, newKey, reader, objectSize, minio.PutObjectOptions{ContentType: "image/png"})
			if err != nil {
				Logger.Errorf("Error during storing thumbnail %v for %v", err, normalizedKey)
				return
			}
		}
	} else if strings.HasPrefix(data.EventName, utils.ObjectRemoved) {
		// TODO remove the preview
	}
}
