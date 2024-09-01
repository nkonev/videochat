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
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/s3"
	"nkonev.name/storage/utils"
	"os"
	"os/exec"
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

func (s *ConvertingService) HandleEvent(ctx context.Context, event *dto.MinioEvent) {
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

	//// download the original recording_123.webm to the tmp dir (configurable)
	filePath := dir + fileName
	pathOfConvertedFile := utils.GetKeyForConverted(filePath)

	// run ffmpeg
	d := viper.GetDuration("converting.presignedDuration")
	presignedUrl, err := s.minio.PresignedGetObject(ctx, s.minioConfig.Files, normalizedKey, d, url.Values{})
	if err != nil {
		GetLogEntry(ctx).Errorf("Error during getting presigned url for %v", normalizedKey)
		return
	}
	stringPresingedUrl := presignedUrl.String()
	ffCmd := exec.Command(viper.GetString("converting.ffmpegPath"),
		"-i", stringPresingedUrl,
		"-c:v", "libvpx-vp9",
		"-c:a", "libopus",
		pathOfConvertedFile,
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

	// set tag recording=true in order to correct work utils.GetEventType in minio_listener
	// put recording_123_converted.webm to minio
	convertedKey := utils.GetKeyForConverted(normalizedKey)
	objectInfo, err := s.minio.StatObject(ctx, s.minioConfig.Files, normalizedKey, minio.StatObjectOptions{})

	_, err = s.minio.FPutObject(ctx, s.minioConfig.Files, convertedKey, pathOfConvertedFile, minio.PutObjectOptions{ContentType: utils.ConvertedContentType, UserMetadata: objectInfo.UserMetadata})
	if err != nil {
		GetLogEntry(ctx).Errorf("Error during storing to minio %v: %v", pathOfConvertedFile, err)
		return
	}
	// defer - rm recording_123_converted.webm from the temporary directory
}
