package services

import (
	"bytes"
	"context"
	"github.com/disintegration/imaging"
	"github.com/minio/minio-go/v7"
	"image"
	"image/jpeg"
	"io"
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
			_, err = s.minio.PutObject(ctx, s.minioConfig.FilesPreview, newKey, byteBuffer, objectSize, minio.PutObjectOptions{ContentType: "image/jpg"})
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
			output, err := ffCmd.Output()
			if err != nil {
				Logger.Errorf("Error during creating thumbnail %v for %v", err, normalizedKey)
				return
			}
			newKey := utils.SetVideoPreviewExtension(normalizedKey)

			reader := bytes.NewReader(output)

			byteBuffer, err := s.resizeImageToJpg(reader)
			if err != nil {
				Logger.Errorf("Error during resizing image %v for %v", err, normalizedKey)
				return
			}

			var objectSize int64 = int64(byteBuffer.Len())

			_, err = s.minio.PutObject(ctx, s.minioConfig.FilesPreview, newKey, byteBuffer, objectSize, minio.PutObjectOptions{ContentType: "image/jpg"})
			if err != nil {
				Logger.Errorf("Error during storing thumbnail %v for %v", err, normalizedKey)
				return
			}
		}
	} else if strings.HasPrefix(data.EventName, utils.ObjectRemoved) {
		// TODO remove the preview
	}
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
