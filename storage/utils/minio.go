package utils

import (
	"errors"
	"fmt"
	"strings"

	"nkonev.name/storage/dto"
)

type MinioConfig struct {
	UserAvatar, ChatAvatar, Files, FilesPreview string
}

func SetVideoPreviewExtension(key string) string {
	return SetExtension(key, "jpg")
}

func SetImagePreviewExtension(key string) string {
	return SetExtension(key, "jpg")
}

const FileParam = "file"
const TimeParam = "time"
const OverrideMessageId = "overrideMessageId"
const OverrideChatId = "overrideChatId"

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

func ParseFileName(minioKey string) (string, error) {
	// "chat/116/0W007Z2P0CRT2G4E1X0DCWB0DK/561ae246-7eff-45a6-a480-2b2be254c768.jpg"
	split := strings.Split(minioKey, "/")
	if len(split) == 4 {
		str := split[3]
		return str, nil
	}
	return "", errors.New("Unable to parse file name")
}

func StripBucketName(minioKey string, bucketName string) string {
	// "files/chat/116/0W007Z2P0CRT2G4E1X0DCWB0DK/561ae246-7eff-45a6-a480-2b2be254c768.jpg"
	toStrip := bucketName + "/"
	return strings.ReplaceAll(minioKey, toStrip, "")
}

// normalized means without bucket name
func BuildNormalizedKey(mce *dto.MetadataCache) string {
	return fmt.Sprintf("chat/%v/%v/%s", mce.ChatId, mce.FileItemUuid, mce.Filename)
}

func BuildMetadataCacheId(key string) (*dto.MetadataCacheId, error) {
	chatId, err := ParseChatId(key)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse chatId for %v: %v", key, err)
	}
	fileItemUuid, err := ParseFileItemUuid(key)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse fileItemUuid for %v: %v", key, err)
	}
	filename, err := ParseFileName(key)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse filename for %v: %v", key, err)
	}

	mcid := dto.MetadataCacheId{
		ChatId:       chatId,
		FileItemUuid: fileItemUuid,
		Filename:     filename,
	}

	return &mcid, nil
}
