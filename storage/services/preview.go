package services

import (
	"bytes"
	"context"
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/minio/minio-go/v7"
	"image"
	"image/jpeg"
	"io"
	"net/url"
	"nkonev.name/storage/dto"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/producer"
	"nkonev.name/storage/s3"
	"nkonev.name/storage/utils"
	"os/exec"
	"strings"
	"time"
)

type PreviewService struct {
	minio                       *s3.InternalMinioClient
	minioConfig                 *utils.MinioConfig
	rabbitFileUploadedPublisher *producer.RabbitFileUploadedPublisher
	filesService                *FilesService
}

func NewPreviewService(minio *s3.InternalMinioClient, minioConfig *utils.MinioConfig, rabbitFileUploadedPublisher *producer.RabbitFileUploadedPublisher, filesService *FilesService) *PreviewService {
	return &PreviewService{
		minio:                       minio,
		minioConfig:                 minioConfig,
		rabbitFileUploadedPublisher: rabbitFileUploadedPublisher,
		filesService:                filesService,
	}
}

func (s PreviewService) HandleMinioEvent(data *dto.MinioEvent, ctx context.Context) {
	Logger.Debugf("Got %v", data)
	normalizedKey := utils.StripBucketName(data.Key, s.minioConfig.Files)
	if strings.HasPrefix(data.EventName, utils.ObjectCreated) {
		s.CreatePreview(normalizedKey, ctx)

		if pu, err := s.getFileUploadedEvent(normalizedKey, data.ChatId, data.CorrelationId, ctx); err == nil {
			s.rabbitFileUploadedPublisher.Publish(data.OwnerId, data.ChatId, pu, ctx)
		} else {
			Logger.Errorf("Error during constructing uploaded event %v for %v", err, normalizedKey)
		}
	}
}

func (s PreviewService) CreatePreview(normalizedKey string, ctx context.Context) {
	if utils.IsImage(normalizedKey) {
		object, err := s.minio.GetObject(ctx, s.minioConfig.Files, normalizedKey, minio.GetObjectOptions{})
		if err != nil {
			Logger.Errorf("Error during getting image file %v for %v", err, normalizedKey)
			return
		}
		byteBuffer, err := s.resizeImageToJpg(object)
		if err != nil {
			Logger.Errorf("Error during resizing image %v for %v", err, normalizedKey)
			return
		}

		newKey := utils.SetImagePreviewExtension(normalizedKey)

		var objectSize int64 = int64(byteBuffer.Len())
		_, err = s.minio.PutObject(ctx, s.minioConfig.FilesPreview, newKey, byteBuffer, objectSize, minio.PutObjectOptions{ContentType: "image/jpg", UserMetadata: SerializeOriginalKeyToMetadata(normalizedKey)})
		if err != nil {
			Logger.Errorf("Error during storing thumbnail %v for %v", err, normalizedKey)
			return
		}
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
		var out bytes.Buffer
		var stderr bytes.Buffer
		ffCmd.Stdout = &out
		ffCmd.Stderr = &stderr
		err = ffCmd.Run()
		if err != nil {
			Logger.Errorf("Error during creating thumbnail for key %v: %v: %v", normalizedKey, fmt.Sprint(err), stderr.String())
			return
		}
		newKey := utils.SetVideoPreviewExtension(normalizedKey)

		reader := bytes.NewReader(out.Bytes())

		byteBuffer, err := s.resizeImageToJpg(reader)
		if err != nil {
			Logger.Errorf("Error during resizing image %v for %v", err, normalizedKey)
			return
		}

		var objectSize int64 = int64(byteBuffer.Len())

		_, err = s.minio.PutObject(ctx, s.minioConfig.FilesPreview, newKey, byteBuffer, objectSize, minio.PutObjectOptions{ContentType: "image/jpg", UserMetadata: SerializeOriginalKeyToMetadata(normalizedKey)})
		if err != nil {
			Logger.Errorf("Error during storing thumbnail %v for %v", err, normalizedKey)
			return
		}
	}
	return
}

func (s PreviewService) resizeImageToJpg(reader io.Reader) (*bytes.Buffer, error) {
	srcImage, _, err := image.Decode(reader)
	if err != nil {
		Logger.Errorf("Error during decoding image: %v", err)
		return nil, err
	}
	dstImage := imaging.Resize(srcImage, 400, 300, imaging.Lanczos)
	byteBuffer := new(bytes.Buffer)
	err = jpeg.Encode(byteBuffer, dstImage, nil)
	if err != nil {
		Logger.Errorf("Error during encoding image: %v", err)
		return nil, err
	}
	return byteBuffer, nil
}

func (s PreviewService) getFileUploadedEvent(normalizedKey string, chatId int64, correlationId string, ctx context.Context) (*dto.FileUploadedEvent, error) {
	_, downloadUrl, err := s.filesService.GetChatPrivateUrl(normalizedKey, chatId)
	if err != nil {
		GetLogEntry(ctx).Errorf("Error during getting url: %v", err)
		return nil, err
	}
	var previewUrl *string = s.filesService.GetPreviewUrlSmart(downloadUrl)
	var aType = GetType(downloadUrl)

	return &dto.FileUploadedEvent{
		Id:            normalizedKey,
		Url:           downloadUrl,
		PreviewUrl:    previewUrl,
		Type:          aType,
		CorrelationId: correlationId,
	}, nil
}
