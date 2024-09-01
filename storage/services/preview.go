package services

import (
	"bytes"
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/siyouyun-open/imaging"
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

// in case video and _converted.webm this function is invoked twice
// pass 1 - to create preview for both original and _converted
// pass 2 - to do nothing and return zero response in order not to send preview_created events below in SendToParticipants()
func (s *PreviewService) HandleMinioEvent(ctx context.Context, data *dto.MinioEvent, shouldSavePreviewForConverted bool) *PreviewResponse {
	GetLogEntry(ctx).Debugf("Got %v", data)
	normalizedKey := utils.StripBucketName(data.Key, s.minioConfig.Files)

	if utils.IsVideo(normalizedKey) {
		previewKey := utils.SetVideoPreviewExtension(normalizedKey)
		var previewExists = false
		_, err := s.minio.StatObject(ctx, s.minioConfig.FilesPreview, previewKey, minio.StatObjectOptions{})
		previewExists = err == nil // error means there is no file

		var theBytes *[]byte

		if !previewExists { // don't make preview again for converted file
			// pass 1: save byteBuffer for the original
			theBytes, err = s.createVideoPreview(ctx, normalizedKey)
			if err != nil {
				GetLogEntry(ctx).Errorf("Error during creating video preview %v for %v", err, normalizedKey)
				return nil
			}

			byteBuffer := bytes.NewBuffer(*theBytes)
			err = s.saveVideoPreviewToMinio(ctx, normalizedKey, byteBuffer)
			if err != nil {
				GetLogEntry(ctx).Errorf("Error during storing video thumbnail %v for %v", err, normalizedKey)
				return nil
			}
		} else {
			// pass 2
			// if we already have preview then this is minio_created event for _converted.webm, we just exit becasue we created both previews in the previous pass
			return nil
		}

		// pass 1: save byteBuffer as of keyOfConverted - as of (yet not existed _converted.webm) _converted.jpg
		if shouldSavePreviewForConverted && theBytes != nil {
			keyOfConverted := utils.GetKeyForConverted(normalizedKey)
			byteBuffer := bytes.NewBuffer(*theBytes)
			err = s.saveVideoPreviewToMinio(ctx, keyOfConverted, byteBuffer)
			if err != nil {
				GetLogEntry(ctx).Errorf("Error during storing video thumbnail %v for %v", err, normalizedKey)
				return nil
			}

			return &PreviewResponse{ // next we send preview_created event for converted
				normalizedKey: keyOfConverted,
			}
		}

		// pass 1: normal, non-convertable video
		return &PreviewResponse{
			normalizedKey: normalizedKey,
		}
	} else {
		s.CreatePreview(ctx, normalizedKey)
		return &PreviewResponse {
			normalizedKey: normalizedKey,
		}
	}
}

type PreviewResponse struct {
	normalizedKey string
}

func (s *PreviewService) SendToParticipants(ctx context.Context, data *dto.MinioEvent, participantIds []int64, response *PreviewResponse) {
	if response != nil {
		if pu, err := s.getFileUploadedEvent(ctx, response.normalizedKey, data.ChatId, data.CorrelationId); err == nil {
			for _, participantId := range participantIds {
				err = s.rabbitFileUploadedPublisher.Publish(ctx, participantId, data.ChatId, pu)
				if err != nil {
					GetLogEntry(ctx).Errorf("Error during ending: %v", err)
					continue
				}
			}
		} else {
			GetLogEntry(ctx).Errorf("Error during constructing uploaded event %v for %v", err, response.normalizedKey)
		}
	}
}

func (s *PreviewService) CreatePreview(ctx context.Context, normalizedKey string) {
	if utils.IsImage(normalizedKey) {
		object, err := s.minio.GetObject(ctx, s.minioConfig.Files, normalizedKey, minio.GetObjectOptions{})
		if err != nil {
			GetLogEntry(ctx).Errorf("Error during getting image file %v for %v", err, normalizedKey)
			return
		}
		byteBuffer, err := s.resizeImageToJpg(ctx, object)
		if err != nil {
			GetLogEntry(ctx).Errorf("Error during resizing image %v for %v", err, normalizedKey)
			return
		}

		newKey := utils.SetImagePreviewExtension(normalizedKey)

		var objectSize int64 = int64(byteBuffer.Len())
		_, err = s.minio.PutObject(ctx, s.minioConfig.FilesPreview, newKey, byteBuffer, objectSize, minio.PutObjectOptions{ContentType: "image/jpg", UserMetadata: SerializeOriginalKeyToMetadata(normalizedKey)})
		if err != nil {
			GetLogEntry(ctx).Errorf("Error during storing image thumbnail %v for %v", err, normalizedKey)
			return
		}
	} else if utils.IsVideo(normalizedKey) {
		theBytes, err := s.createVideoPreview(ctx, normalizedKey)
		if err != nil {
			GetLogEntry(ctx).Errorf("Error during creating video preview %v for %v", err, normalizedKey)
			return
		}

		byteBuffer := bytes.NewBuffer(*theBytes)

		err = s.saveVideoPreviewToMinio(ctx, normalizedKey, byteBuffer)
		if err != nil {
			GetLogEntry(ctx).Errorf("Error during storing video thumbnail %v for %v", err, normalizedKey)
			return
		}
	}
	return
}

func (s *PreviewService) resizeImageToJpg(ctx context.Context, reader io.Reader) (*bytes.Buffer, error) {
	srcImage, _, err := image.Decode(reader)
	if err != nil {
		GetLogEntry(ctx).Errorf("Error during decoding image: %v", err)
		return nil, err
	}
	dstImage := imaging.Resize(srcImage, 0, 360, imaging.Lanczos)
	byteBuffer := new(bytes.Buffer)
	err = jpeg.Encode(byteBuffer, dstImage, nil)
	if err != nil {
		GetLogEntry(ctx).Errorf("Error during encoding image: %v", err)
		return nil, err
	}
	return byteBuffer, nil
}

func (s *PreviewService) createVideoPreview(ctx context.Context, normalizedKey string) (*[]byte, error) {
	d, _ := time.ParseDuration("10m")
	presignedUrl, err := s.minio.PresignedGetObject(ctx, s.minioConfig.Files, normalizedKey, d, url.Values{})
	if err != nil {
		GetLogEntry(ctx).Errorf("Error during getting presigned url for %v", normalizedKey)
		return nil, err
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
		GetLogEntry(ctx).Errorf("Error during creating thumbnail for key %v: %v: %v", normalizedKey, fmt.Sprint(err), stderr.String())
		return nil, err
	}

	reader := bytes.NewReader(out.Bytes())

	byteBuffer, err := s.resizeImageToJpg(ctx, reader)
	if err != nil {
		GetLogEntry(ctx).Errorf("Error during resizing image %v for %v", err, normalizedKey)
		return nil, err
	}
	theBytes := byteBuffer.Bytes()
	return &theBytes, nil
}

func (s *PreviewService) saveVideoPreviewToMinio(ctx context.Context, normalizedKey string, byteBuffer *bytes.Buffer) error {
	newKey := utils.SetVideoPreviewExtension(normalizedKey)

	var objectSize int64 = int64(byteBuffer.Len())

	_, err := s.minio.PutObject(ctx, s.minioConfig.FilesPreview, newKey, byteBuffer, objectSize, minio.PutObjectOptions{ContentType: "image/jpg", UserMetadata: SerializeOriginalKeyToMetadata(normalizedKey)})
	return err
}

func (s *PreviewService) getFileUploadedEvent(ctx context.Context, normalizedKey string, chatId int64, correlationId *string) (*dto.PreviewCreatedEvent, error) {
	downloadUrl, err := s.filesService.GetConstantDownloadUrl(normalizedKey)
	if err != nil {
		GetLogEntry(ctx).Errorf("Error during getting url: %v", err)
		return nil, err
	}
	var previewUrl *string = s.filesService.GetPreviewUrlSmart(ctx, normalizedKey)
	var aType = GetType(normalizedKey)

	return &dto.PreviewCreatedEvent{
		Id:            normalizedKey,
		Url:           downloadUrl,
		PreviewUrl:    previewUrl,
		Type:          aType,
		CorrelationId: correlationId,
	}, nil
}
