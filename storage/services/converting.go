package services

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/viper"
	"nkonev.name/storage/client"
	"nkonev.name/storage/dto"
	"nkonev.name/storage/logger"
	"nkonev.name/storage/utils"
)

type ConvertingService struct {
	awsS3            *client.S3Wrapper
	minioConfig      *utils.MinioConfig
	tempDirPrefix    string
	redisInfoService *RedisInfoService
	lgr              *logger.Logger
}

func NewConvertingService(lgr *logger.Logger, awsS3 *client.S3Wrapper, minioConfig *utils.MinioConfig, redisInfoService *RedisInfoService) *ConvertingService {
	tempDirPrefix := viper.GetString("converting.tempDir")
	lgr.Infof("Ensuring temp root dir for the converting videos using ffmpeg: %v", tempDirPrefix)
	os.MkdirAll(tempDirPrefix, os.ModePerm)

	return &ConvertingService{
		awsS3:            awsS3,
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

	pres, err := s.awsS3.PresignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.minioConfig.Files,
		Key:    &normalizedKey,
	}, s3.WithPresignExpires(d))
	if err != nil {
		s.lgr.WithTracing(ctx).Errorf("Error during getting presigned url for %v", normalizedKey)
		return
	}

	u, err := url.Parse(pres.URL)
	if err != nil {
		s.lgr.WithTracing(ctx).Errorf("Error during parsing presigned put url %v", err)
		return
	}

	stringPresingedUrl := u.String()
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
	objectInfo, err := s.awsS3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &s.minioConfig.Files,
		Key:    &normalizedKey,
	})
	if err != nil {
		s.lgr.WithTracing(ctx).Errorf("Error during stat for key %v: %v", normalizedKey, err)
		return
	}

	convertedFile, err := os.Open(pathOfConvertedFile)
	if err != nil {
		s.lgr.WithTracing(ctx).Errorf("Error during opening file '%v': %v", pathOfConvertedFile, err)
		return
	}
	defer convertedFile.Close()

	// put recording_123_converted.webm to minio
	_, err = s.awsS3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s.minioConfig.Files,
		Key:         &convertedKey,
		Body:        convertedFile,
		ContentType: new(utils.ConvertedContentType),
		Metadata:    objectInfo.Metadata,
	})
	if err != nil {
		s.lgr.WithTracing(ctx).Errorf("Error during storing to minio %v: %v", pathOfConvertedFile, err)
		return
	}

	s.lgr.WithTracing(ctx).Infof("Converted %v to %v", normalizedKey, pathOfConvertedFile)
	// defer removes recording_123_converted.webm from the temporary directory

	if viper.GetBool("converting.removeOriginal") {
		s.lgr.WithTracing(ctx).Infof("Going to remove original from minio %v", normalizedKey)
		_, err = s.awsS3.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: &s.minioConfig.Files,
			Key:    &normalizedKey,
		})
		if err != nil {
			s.lgr.WithTracing(ctx).Errorf("Error during removing original from minio %v: %v", normalizedKey, err)
			return
		}
	}
}
