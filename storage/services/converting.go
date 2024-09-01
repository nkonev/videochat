package services

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
	"io/ioutil"
	"nkonev.name/storage/dto"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/s3"
	"nkonev.name/storage/utils"
	"os"
)

type ConvertingService struct {
	minio                       *s3.InternalMinioClient
	minioConfig                 *utils.MinioConfig
	tempDirPrefix               string
}

func NewConvertingService(minio *s3.InternalMinioClient, minioConfig *utils.MinioConfig) *ConvertingService {
	tempDirPrefix := viper.GetString("converting.tempDir")
	Logger.Infof("Ensuring temp root dir for the converting videos using ffmpeg: %v", tempDirPrefix)
	os.MkdirAll(tempDirPrefix, os.ModePerm)

	return &ConvertingService{
		minio:       minio,
		minioConfig: minioConfig,
		tempDirPrefix: tempDirPrefix,
	}
}

func (s *ConvertingService) HandleConvertedEvent(ctx context.Context, event *dto.MinioEvent) {
	normalizedKey := utils.StripBucketName(event.Key, s.minioConfig.Files)
	fileName := utils.GetFilename(normalizedKey)

	// create temp dir
	fileWoExt := utils.RemoveExtension(fileName)
	dir, err := ioutil.TempDir(s.tempDirPrefix, fileWoExt+"__")
	if err != nil {
		GetLogEntry(ctx).Errorf("error during create temp dir for the converting videos using ffmpeg: %v", err)
		return
	}
	defer os.RemoveAll(dir)

	// download the original recording_123.webm to the tmp dir (configurable)
	filePath := dir + fileName
	err = s.minio.FGetObject(ctx, s.minioConfig.Files, normalizedKey, filePath, minio.GetObjectOptions{})
	if err != nil {
		GetLogEntry(ctx).Errorf("error during downloading video file from minio: %v", err)
		return
	}
	// run ffmpeg
	// set tag recording=true in order to correct work utils.GetEventType in minio_listener
	// put recording_123_converted.webm to minio
	// rm recording_123_converted.webm from the temporary directory
}
