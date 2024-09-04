package services

import (
	"bytes"
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
	"net/url"
	"nkonev.name/storage/dto"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/s3"
	"nkonev.name/storage/utils"
	"os/exec"
)

type ConvertingService struct {
	minio                       *s3.InternalMinioClient
	minioConfig                 *utils.MinioConfig
	redisInfoService *RedisInfoService
}

func NewConvertingService(minio *s3.InternalMinioClient, minioConfig *utils.MinioConfig, redisInfoService *RedisInfoService) *ConvertingService {
	return &ConvertingService{
		minio:       minio,
		minioConfig: minioConfig,
		redisInfoService: redisInfoService,
	}
}

func (s *ConvertingService) HandleEvent(ctx context.Context, event *dto.MinioEvent) {
	normalizedKey := utils.StripBucketName(event.Key, s.minioConfig.Files)
	s.Convert(ctx, normalizedKey)
}

func (s *ConvertingService) Convert(ctx context.Context, normalizedKey string) {
	convertedKey := utils.GetKeyForConverted(normalizedKey)

	s.redisInfoService.SetOriginalConverting(ctx, normalizedKey)
	defer s.redisInfoService.RemoveOriginalConverting(ctx, normalizedKey)

	s.redisInfoService.SetConvertedConverting(ctx, convertedKey)
	defer s.redisInfoService.RemoveConvertedConverting(ctx, convertedKey)

	GetLogEntry(ctx).Infof("Converting %v to %v to the common compatible format", normalizedKey, convertedKey)

	// run ffmpeg
	d := viper.GetDuration("converting.presignedDuration")
	presignedGetUrl, err := s.minio.PresignedGetObject(ctx, s.minioConfig.Files, normalizedKey, d, url.Values{})
	if err != nil {
		GetLogEntry(ctx).Errorf("Error during getting presigned get url for %v", normalizedKey)
		return
	}
	stringPresingedGetUrl := presignedGetUrl.String()

	// copy the tag messageRecording=true in order to correct work utils.GetEventType in minio_listener in pass 2
	// also set the owner
	tags := url.Values{}
	tags.Set("aaa", "bbb")
	presignedPutUrl, err := s.minio.Presign(ctx, "PUT", s.minioConfig.Files, convertedKey, d, tags)
	if err != nil {
		GetLogEntry(ctx).Errorf("Error during getting presigned put url for %v", normalizedKey)
		return
	}
	stringPresingedPutUrl := presignedPutUrl.String()

	// https://superuser.com/questions/424015/what-bunch-of-ffmpeg-scripts-do-i-need-to-get-html5-compatible-video-for-everyb/424024#424024
	ffCmd := exec.Command(viper.GetString("converting.ffmpegPath"),
		"-i", stringPresingedGetUrl,
		"-c:v", "libvpx-vp9",
		"-c:a", "libopus",
		"-f", "webm",
		// for debug purposes you can just send it to a file
		// search "Publish contents of your desktop directly to a WebDAV server every second" on https://www.ffmpeg.org/ffmpeg-all.html
		"-protocol_opts",
		"method=PUT",
		stringPresingedPutUrl,
	)
	// getting real error msg : https://stackoverflow.com/questions/18159704/how-to-debug-exit-status-1-error-when-running-exec-command-in-golang
	var out bytes.Buffer
	var stderr bytes.Buffer
	ffCmd.Stdout = &out
	ffCmd.Stderr = &stderr
	err = ffCmd.Run()
	if err != nil {
		GetLogEntry(ctx).Errorf("Error during converting for key %v: %v: stderr: %v, stdout: %v", normalizedKey, fmt.Sprint(err), stderr.String(), out.String())
		return
	}

	GetLogEntry(ctx).Infof("Converted %v to %v", normalizedKey, convertedKey)

	if viper.GetBool("converting.removeOriginal") {
		GetLogEntry(ctx).Infof("Going to remove original from minio %v", normalizedKey)
		err = s.minio.RemoveObject(ctx, s.minioConfig.Files, normalizedKey, minio.RemoveObjectOptions{})
		if err != nil {
			GetLogEntry(ctx).Errorf("Error during removing original from minio %v: %v", normalizedKey, err)
			return
		}
	}
}
