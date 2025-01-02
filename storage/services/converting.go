package services

import (
	"bytes"
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
	"io/ioutil"
	"net/url"
	"nkonev.name/storage/dto"
	"nkonev.name/storage/logger"
	"nkonev.name/storage/s3"
	"nkonev.name/storage/utils"
	"os"
	"os/exec"
	"syscall"
)

type ConvertingService struct {
	minio            *s3.InternalMinioClient
	minioConfig      *utils.MinioConfig
	tempDirPrefix    string
	redisInfoService *RedisInfoService
	lgr              *logger.Logger
}

func NewConvertingService(lgr *logger.Logger, minio *s3.InternalMinioClient, minioConfig *utils.MinioConfig, redisInfoService *RedisInfoService) *ConvertingService {
	tempDirPrefix := viper.GetString("converting.tempDir")
	lgr.Infof("Ensuring temp root dir for the converting videos using ffmpeg: %v", tempDirPrefix)
	os.MkdirAll(tempDirPrefix, os.ModePerm)

	return &ConvertingService{
		minio:            minio,
		minioConfig:      minioConfig,
		tempDirPrefix:    tempDirPrefix,
		redisInfoService: redisInfoService,
		lgr:              lgr,
	}
}

func (s *ConvertingService) HandleEvent(ctx context.Context, event *dto.MinioEvent) {
	normalizedKey := utils.StripBucketName(event.Key, s.minioConfig.Files)
	s.Convert(ctx, normalizedKey)
}

func (s *ConvertingService) Convert(ctx context.Context, normalizedKey string) {
	fileName := utils.GetFilename(normalizedKey)
	convertedKey := utils.GetKeyForConverted(normalizedKey)

	s.redisInfoService.SetOriginalConverting(ctx, normalizedKey)
	defer s.redisInfoService.RemoveOriginalConverting(ctx, normalizedKey)

	s.redisInfoService.SetConvertedConverting(ctx, convertedKey)
	defer s.redisInfoService.RemoveConvertedConverting(ctx, convertedKey)

	s.lgr.WithTracing(ctx).Infof("Converting %v to %v to the common compatible format", normalizedKey, convertedKey)

	// create temp dir
	fileWoExt := utils.RemoveExtension(fileName)
	dir, err := ioutil.TempDir(s.tempDirPrefix, fileWoExt+"__")
	if err != nil {
		s.lgr.WithTracing(ctx).Errorf("error during create temp dir for the converting videos using ffmpeg: %v", err)
		return
	}
	defer os.RemoveAll(dir)

	filePath := dir + string(os.PathSeparator) + fileName
	pathOfConvertedFile := utils.GetKeyForConverted(filePath)

	// run ffmpeg
	d := viper.GetDuration("converting.presignedDuration")
	presignedUrl, err := s.minio.PresignedGetObject(ctx, s.minioConfig.Files, normalizedKey, d, url.Values{})
	if err != nil {
		s.lgr.WithTracing(ctx).Errorf("Error during getting presigned url for %v", normalizedKey)
		return
	}
	stringPresingedUrl := presignedUrl.String()
	ffCmd := exec.Command(viper.GetString("converting.ffmpegPath"),
		"-i", stringPresingedUrl,
		// convert to format, which can be played in both Firefox and Chrome
		"-c:v", "libvpx-vp9",
		"-c:a", "libopus",
		pathOfConvertedFile,
	)
	// https://medium.com/@ganeshmaharaj/clean-exit-of-golangs-exec-command-897832ac3fa5
	ffCmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
	}

	// getting real error msg : https://stackoverflow.com/questions/18159704/how-to-debug-exit-status-1-error-when-running-exec-command-in-golang
	var out bytes.Buffer
	var stderr bytes.Buffer
	ffCmd.Stdout = &out
	ffCmd.Stderr = &stderr
	err = ffCmd.Run()
	if err != nil {
		s.lgr.WithTracing(ctx).Errorf("Error during converting for key %v: %v: stderr: %v, stdout: %v", normalizedKey, fmt.Sprint(err), stderr.String(), out.String())
		return
	}

	// copy the tag messageRecording=true in order to correct work utils.GetEventType in minio_listener in pass 2
	objectInfo, err := s.minio.StatObject(ctx, s.minioConfig.Files, normalizedKey, minio.StatObjectOptions{})
	if err != nil {
		s.lgr.WithTracing(ctx).Errorf("Error during stat for key %v: %v", normalizedKey, err)
		return
	}
	// put recording_123_converted.webm to minio
	_, err = s.minio.FPutObject(ctx, s.minioConfig.Files, convertedKey, pathOfConvertedFile, minio.PutObjectOptions{ContentType: utils.ConvertedContentType, UserMetadata: objectInfo.UserMetadata})
	if err != nil {
		s.lgr.WithTracing(ctx).Errorf("Error during storing to minio %v: %v", pathOfConvertedFile, err)
		return
	}

	s.lgr.WithTracing(ctx).Infof("Converted %v to %v", normalizedKey, pathOfConvertedFile)
	// defer removes recording_123_converted.webm from the temporary directory

	if viper.GetBool("converting.removeOriginal") {
		s.lgr.WithTracing(ctx).Infof("Going to remove original from minio %v", normalizedKey)
		err = s.minio.RemoveObject(ctx, s.minioConfig.Files, normalizedKey, minio.RemoveObjectOptions{})
		if err != nil {
			s.lgr.WithTracing(ctx).Errorf("Error during removing original from minio %v: %v", normalizedKey, err)
			return
		}
	}
}
