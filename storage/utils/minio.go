package utils

import (
	"context"
	"errors"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/s3"
	"strings"
)

func ensureBucket(minioClient *s3.InternalMinioClient, bucketName, location string) error {
	// Check to see if we already own this bucket (which happens if you run this twice)
	exists, err := minioClient.BucketExists(context.Background(), bucketName)
	if err == nil && exists {
		Logger.Infof("Bucket '%s' already present", bucketName)
		return nil
	} else if err != nil {
		return err
	} else {
		if err := minioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{
			Region:        location,
			ObjectLocking: false,
		}); err != nil {
			Logger.Errorf("Error during creating bucket '%s'", bucketName)
			return err
		} else {
			Logger.Infof("Successfully created bucket '%s'", bucketName)
			return nil
		}
	}
}

func EnsureAndGetUserAvatarBucket(minioClient *s3.InternalMinioClient) (string, error) {
	bucketName := viper.GetString("minio.bucket.userAvatar")
	bucketLocation := viper.GetString("minio.location")
	err := ensureBucket(minioClient, bucketName, bucketLocation)
	return bucketName, err
}

func EnsureAndGetChatAvatarBucket(minioClient *s3.InternalMinioClient) (string, error) {
	bucketName := viper.GetString("minio.bucket.chatAvatar")
	bucketLocation := viper.GetString("minio.location")
	err := ensureBucket(minioClient, bucketName, bucketLocation)
	return bucketName, err
}

func EnsureAndGetFilesBucket(minioClient *s3.InternalMinioClient) (string, error) {
	bucketName := viper.GetString("minio.bucket.files")
	bucketLocation := viper.GetString("minio.location")
	err := ensureBucket(minioClient, bucketName, bucketLocation)
	return bucketName, err
}

func EnsureAndGetFilesPreviewBucket(minioClient *s3.InternalMinioClient) (string, error) {
	bucketName := viper.GetString("minio.bucket.filesPreview")
	bucketLocation := viper.GetString("minio.location")
	err := ensureBucket(minioClient, bucketName, bucketLocation)
	return bucketName, err
}

type MinioConfig struct {
	UserAvatar, ChatAvatar, Files, FilesPreview string
}

// https://min.io/docs/minio/linux/reference/minio-mc/mc-event-add.html#mc-event-supported-events

const ObjectCreated = "s3:ObjectCreated"
const ObjectRemoved = "s3:ObjectRemoved"

const ObjectCreatedCompleteMultipartUpload = ObjectCreated + ":CompleteMultipartUpload"

const ObjectRemovedDelete = ObjectRemoved + ":Delete"

const ObjectCreatedPutTagging = ObjectCreated + ":PutTagging"
const ObjectCreatedPut = ObjectCreated + ":Put"

func SetVideoPreviewExtension(key string) string {
	return SetExtension(key, "jpg")
}

func SetImagePreviewExtension(key string) string {
	return SetExtension(key, "jpg")
}

const FileParam = "file"
const TimeParam = "time"
const MessageIdParam = "messageId"

func ParseChatId(minioKey string) (int64, error) {
	// "chat/116/0W007Z2P0CRT2G4E1X0DCWB0DK/561ae246-7eff-45a6-a480-2b2be254c768.jpg"
	split := strings.Split(minioKey, "/")
	if len(split) >= 2 {
		str := split[1]
		return ParseInt64(str)
	}
	return 0, errors.New("Unable to parse chat id")
}

func ParseFileItemUuid(minioKey string) (string, error) {
	// "chat/116/0W007Z2P0CRT2G4E1X0DCWB0DK/561ae246-7eff-45a6-a480-2b2be254c768.jpg"
	split := strings.Split(minioKey, "/")
	if len(split) >= 3 {
		str := split[2]
		return str, nil
	}
	return "", errors.New("Unable to parse file id")
}

func StripBucketName(minioKey string, bucketName string) string {
	// "files/chat/116/0W007Z2P0CRT2G4E1X0DCWB0DK/561ae246-7eff-45a6-a480-2b2be254c768.jpg"
	toStrip := bucketName + "/"
	return strings.ReplaceAll(minioKey, toStrip, "")
}
