package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/tags"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net/url"
	"nkonev.name/storage/client"
	"nkonev.name/storage/dto"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/s3"
	"nkonev.name/storage/utils"
	"strings"
	"time"
)

type FilesService struct {
	minio       *s3.InternalMinioClient
	restClient  *client.RestClient
	minioConfig *utils.MinioConfig
	lgr         *log.Logger
}

func NewFilesService(
	lgr *log.Logger,
	minio *s3.InternalMinioClient,
	chatClient *client.RestClient,
	minioConfig *utils.MinioConfig,
) *FilesService {
	return &FilesService{
		minio:       minio,
		restClient:  chatClient,
		minioConfig: minioConfig,
		lgr:         lgr,
	}
}

func (h *FilesService) GetListFilesInFileItem(
	c context.Context,
	behalfUserId int64,
	bucket, filenameChatPrefix string,
	chatId int64,
	filter func(*minio.ObjectInfo) bool,
	requestOwners bool,
	size, offset int,
) ([]*dto.FileInfoDto, int, error) {
	var objects <-chan minio.ObjectInfo = h.minio.ListObjects(c, bucket, minio.ListObjectsOptions{
		WithMetadata: true,
		Prefix:       filenameChatPrefix,
		Recursive:    true,
	})

	var list []*dto.FileInfoDto = make([]*dto.FileInfoDto, 0)
	var offsetCounter = 0
	var respCounter = 0

	for objInfo := range objects {
		GetLogEntry(c, h.lgr).Debugf("Object '%v'", objInfo.Key)
		if (filter != nil && filter(&objInfo)) || filter == nil {
			if offsetCounter >= offset {

				if respCounter < size {
					tagging, err := h.minio.GetObjectTagging(c, bucket, objInfo.Key, minio.GetObjectTaggingOptions{})
					if err != nil {
						GetLogEntry(c, h.lgr).Errorf("Error during getting tags %v", err)
						continue
					}

					info, err := h.GetFileInfo(c, behalfUserId, objInfo, chatId, tagging, true)
					if err != nil {
						GetLogEntry(c, h.lgr).Errorf("Error get file info: %v, skipping", err)
						continue
					}

					list = append(list, info)
					respCounter++
				}
			}
			offsetCounter++
		}
	}

	if requestOwners {
		var participantIdSet = map[int64]bool{}
		for _, fileDto := range list {
			participantIdSet[fileDto.OwnerId] = true
		}
		var users = GetUsersRemotelyOrEmpty(h.lgr, participantIdSet, h.restClient, c)
		for _, fileDto := range list {
			user := users[fileDto.OwnerId]
			if user != nil {
				fileDto.Owner = user
			}
		}
	}

	return list, offsetCounter, nil
}

type SimpleFileItem struct {
	Id           string    `json:"id"`
	Filename     string    `json:"filename"`
	LastModified time.Time `json:"time"`
}

type GroupedByFileItemUuid struct {
	FileItemUuid string           `json:"fileItemUuid"`
	Files        []SimpleFileItem `json:"files"`
}

func (h *FilesService) GetListFilesItemUuids(
	c context.Context,
	bucket, filenameChatPrefix string,
	size, offset int,
) ([]*GroupedByFileItemUuid, int, error) {
	var objects <-chan minio.ObjectInfo = h.minio.ListObjects(c, bucket, minio.ListObjectsOptions{
		WithMetadata: true,
		Prefix:       filenameChatPrefix,
		Recursive:    true,
	})

	var list []*GroupedByFileItemUuid = make([]*GroupedByFileItemUuid, 0)
	var counter = 0
	var lastItemUuid = ""

	var files = []SimpleFileItem{}
	for m := range objects {
		itemUuid, err := utils.ParseFileItemUuid(m.Key)
		if err != nil {
			GetLogEntry(c, h.lgr).Errorf("Unable for %v to get fileItemUuid '%v'", m.Key, err)
			continue
		}

		itemIdHasChanged := itemUuid != lastItemUuid
		if itemIdHasChanged {
			counter++
		}
		lastLastItemId := lastItemUuid
		lastItemUuid = itemUuid

		if counter >= offset {
			if len(list) < size {
				if itemIdHasChanged {
					if len(files) > 0 {
						list = append(list, &GroupedByFileItemUuid{lastLastItemId, files}) // process from previous iteration
					}

					// prepare for current iteration
					files = []SimpleFileItem{}
				}

				files = append(files, SimpleFileItem{
					Id:           m.Key,
					Filename:     ReadFilename(m.Key),
					LastModified: m.LastModified,
				})
			}
		}
	}

	// process leftovers
	if len(files) > 0 && len(list) < size && lastItemUuid != "" {
		list = append(list, &GroupedByFileItemUuid{lastItemUuid, files})
	}

	return list, counter, nil
}

func (h *FilesService) GetCount(ctx context.Context, filenameChatPrefix string) (int, error) {
	var objects <-chan minio.ObjectInfo = h.minio.ListObjects(ctx, h.minioConfig.Files, minio.ListObjectsOptions{
		Prefix:    filenameChatPrefix,
		Recursive: true,
	})

	var count int = 0
	for objInfo := range objects {
		GetLogEntry(ctx, h.lgr).Debugf("Object '%v'", objInfo.Key)
		count++
	}
	return count, nil
}

func (h *FilesService) GetTemporaryDownloadUrl(ctx context.Context, aKey string) (string, time.Duration, error) {
	ttl := viper.GetDuration("minio.publicDownloadTtl")

	u, err := h.minio.PresignedGetObject(ctx, h.minioConfig.Files, aKey, ttl, url.Values{})
	if err != nil {
		return "", time.Second, err
	}

	downloadUrl, err := ChangeMinioUrl(u)
	if err != nil {
		return "", time.Second, err
	}

	return downloadUrl, ttl, nil
}

func (h *FilesService) GetConstantDownloadUrl(aKey string) (string, error) {
	downloadUrl, err := url.Parse(utils.UrlStorageGetFile)
	if err != nil {
		return "", err
	}

	query := downloadUrl.Query()
	query.Add(utils.FileParam, aKey)
	downloadUrl.RawQuery = query.Encode()

	downloadUrlStr := fmt.Sprintf("%v", downloadUrl)
	return downloadUrlStr, nil
}

func ChangeMinioUrl(url *url.URL) (string, error) {
	publicUrlPrefix := viper.GetString("minio.publicUrlPrefix")
	parsed, err := url.Parse(publicUrlPrefix)
	if err != nil {
		return "", err
	}

	url.Path = parsed.Path + url.Path
	url.Host = ""
	url.Scheme = ""

	stringV := url.String()

	return stringV, nil
}

func (h *FilesService) GetPublicUrl(public bool, fileName string) (*string, error) {
	if !public {
		return nil, nil
	}

	downloadUrl, err := url.Parse(utils.UrlStorageGetFilePublicExternal)
	if err != nil {
		return nil, err
	}

	query := downloadUrl.Query()
	query.Add(utils.FileParam, fileName)
	downloadUrl.RawQuery = query.Encode()
	str := downloadUrl.String()
	return &str, nil
}

func (h *FilesService) GetAnonymousUrl(fileName string, overrideChatId, overrideMessageId int64) (string, error) {
	downloadUrl, err := url.Parse(utils.UrlStorageGetFilePublicExternal)
	if err != nil {
		return "", err
	}

	query := downloadUrl.Query()
	query.Add(utils.FileParam, fileName)
	query.Add(utils.OverrideChatId, utils.Int64ToString(overrideChatId))
	query.Add(utils.OverrideMessageId, utils.Int64ToString(overrideMessageId))
	downloadUrl.RawQuery = query.Encode()
	str := downloadUrl.String()
	return str, nil
}

func (h *FilesService) GetAnonymousPreviewUrl(c context.Context, fileName string, chatId, messageId int64) (*string, error) {
	anUrl := h.getPreviewUrlSmart(c, fileName, utils.UrlBasePublicPreview, &chatId, &messageId)
	return anUrl, nil
}

func (h *FilesService) GetFileInfo(c context.Context, behalfUserId int64, objInfo minio.ObjectInfo, chatId int64, tagging *tags.Tags, hasAmzPrefix bool) (*dto.FileInfoDto, error) {
	previewUrl := h.GetPreviewUrlSmart(c, objInfo.Key)

	metadata := objInfo.UserMetadata

	_, fileOwnerId, correlationId, _, err := DeserializeMetadata(metadata, hasAmzPrefix)
	if err != nil {
		GetLogEntry(c, h.lgr).Errorf("Error get metadata: %v", err)
		return nil, err
	}

	filename := ReadFilename(objInfo.Key)

	public, err := DeserializeTags(tagging)
	if err != nil {
		GetLogEntry(c, h.lgr).Errorf("Error get tags: %v", err)
		return nil, err
	}

	publicUrl, err := h.GetPublicUrl(public, objInfo.Key)
	if err != nil {
		GetLogEntry(c, h.lgr).Errorf("Error get public url: %v", err)
		return nil, err
	}

	downloadUrl, err := h.GetConstantDownloadUrl(objInfo.Key)
	if err != nil {
		GetLogEntry(c, h.lgr).Errorf("Error during getting downlad url %v", err)
		return nil, err
	}

	itemUuid, err := utils.ParseFileItemUuid(objInfo.Key)
	if err != nil {
		GetLogEntry(c, h.lgr).Errorf("Unable for %v to get fileItemUuid '%v'", objInfo.Key, err)
	}
	var theCorrelationId *string
	if len(correlationId) > 0 {
		theCorrelationId = &correlationId
	}
	info := &dto.FileInfoDto{
		Id:             objInfo.Key,
		Filename:       filename,
		Url:            downloadUrl,
		Size:           objInfo.Size,
		CanDelete:      fileOwnerId == behalfUserId,
		CanEdit:        fileOwnerId == behalfUserId && utils.IsPlainText(objInfo.Key),
		CanShare:       fileOwnerId == behalfUserId,
		LastModified:   objInfo.LastModified,
		OwnerId:        fileOwnerId,
		PublicUrl:      publicUrl,
		PreviewUrl:     previewUrl,
		CanPlayAsVideo: utils.IsVideo(objInfo.Key),
		CanShowAsImage: utils.IsImage(objInfo.Key),
		CanPlayAsAudio: utils.IsAudio(objInfo.Key),
		FileItemUuid:   itemUuid,
		CorrelationId:  theCorrelationId,
	}
	return info, nil
}

const Media_image = "image"
const Media_video = "video"
const Media_audio = "audio"

func (h *FilesService) GetPreviewUrlSmart(c context.Context, aKey string) *string {
	return h.getPreviewUrlSmart(c, aKey, utils.UrlBasePreview, nil, nil)
}

func (h *FilesService) getPreviewUrlSmart(c context.Context, aKey string, urlBase string, overrideChatId, overrideMessageId *int64) *string {
	recognizedType := ""
	if utils.IsVideo(aKey) {
		recognizedType = Media_video
		return h.getPreviewUrl(c, aKey, recognizedType, urlBase, overrideChatId, overrideMessageId)
	} else if utils.IsImage(aKey) {
		recognizedType = Media_image
		return h.getPreviewUrl(c, aKey, recognizedType, urlBase, overrideChatId, overrideMessageId)
	}
	return nil
}

func GetType(itemUrl string) *string {
	var recognizedType string = ""
	if utils.IsVideo(itemUrl) {
		recognizedType = Media_video
	} else if utils.IsImage(itemUrl) {
		recognizedType = Media_image
	} else if utils.IsAudio(itemUrl) {
		recognizedType = Media_audio
	}

	if recognizedType != "" {
		return &recognizedType
	} else {
		return nil
	}
}

func (h *FilesService) getPreviewUrl(c context.Context, aKey string, requestedMediaType string, urlBase string, overrideChatId, overrideMessageId *int64) *string {
	var previewUrl *string = nil

	respUrl := url.URL{}
	respUrl.Path = urlBase
	previewMinioKey := ""
	if requestedMediaType == Media_video {
		previewMinioKey = utils.SetVideoPreviewExtension(aKey)
	} else if requestedMediaType == Media_image {
		previewMinioKey = utils.SetImagePreviewExtension(aKey)
	}
	if previewMinioKey != "" {
		query := respUrl.Query()
		query.Set(utils.FileParam, previewMinioKey)

		exists, obj, _ := h.minio.FileExists(c, h.minioConfig.FilesPreview, previewMinioKey)
		if exists {
			// if preview file presents we do set time. it is need to distinguish on front. it's required to update early requested file item without preview
			query.Set(utils.TimeParam, utils.Int64ToString(obj.LastModified.Unix()))
		}

		if overrideChatId != nil {
			query.Add(utils.OverrideChatId, utils.Int64ToString(*overrideChatId))
		}
		if overrideMessageId != nil {
			query.Add(utils.OverrideMessageId, utils.Int64ToString(*overrideMessageId))
		}

		respUrl.RawQuery = query.Encode()

		tmp := respUrl.String()
		previewUrl = &tmp
	} else {
		GetLogEntry(c, h.lgr).Errorf("Unknown requested type %v", requestedMediaType)
	}

	return previewUrl
}

const publicKey = "public"

const ownerIdKey = "ownerid"
const chatIdKey = "chatid"
const correlationIdKey = "correlationid"

const conferenceRecordingKey = "confrecording"
const messageRecordingKey = "msgrecording"

const originalKey = "originalkey"

func SerializeMetadataSimple(userId int64, chatId int64, correlationId *string, isConferenceRecording *bool, isUserMessageRecording *bool) map[string]string {
	var userMetadata = map[string]string{}
	userMetadata[ownerIdKey] = utils.Int64ToString(userId)
	userMetadata[chatIdKey] = utils.Int64ToString(chatId)
	if correlationId != nil {
		userMetadata[correlationIdKey] = *correlationId
	}
	if isConferenceRecording != nil {
		userMetadata[conferenceRecordingKey] = utils.BooleanToString(*isConferenceRecording)
	}
	if isUserMessageRecording != nil {
		userMetadata[messageRecordingKey] = utils.BooleanToString(*isUserMessageRecording)
	}
	return userMetadata
}

const xAmzMetaPrefix = "X-Amz-Meta-"

func DeserializeMetadata(userMetadata minio.StringMap, hasAmzPrefix bool) (int64, int64, string, *bool, error) {
	var prefix = ""
	if hasAmzPrefix {
		prefix = xAmzMetaPrefix
	}

	ownerIdString, ok := userMetadata[prefix+strings.Title(ownerIdKey)]
	if !ok {
		return 0, 0, "", nil, errors.New("Unable to get owner id")
	}
	ownerId, err := utils.ParseInt64(ownerIdString)
	if err != nil {
		return 0, 0, "", nil, err
	}

	chatIdString, ok := userMetadata[prefix+strings.Title(chatIdKey)]
	if !ok {
		return 0, 0, "", nil, errors.New("Unable to get chat id")
	}
	chatId, err := utils.ParseInt64(chatIdString)
	if err != nil {
		return 0, 0, "", nil, err
	}
	correlationId := userMetadata[prefix+strings.Title(correlationIdKey)]

	var isMessageRecording *bool
	isMessageRecordingString, ok := userMetadata[prefix+strings.Title(messageRecordingKey)]
	if ok {
		isMessageRecordingVar, err := utils.ParseBoolean(isMessageRecordingString)
		if err != nil {
			return 0, 0, "", nil, err
		}
		isMessageRecording = &isMessageRecordingVar
	}

	return chatId, ownerId, correlationId, isMessageRecording, nil
}

func GetKey(filename string, chatFileItemUuid string, chatId int64) string {
	return fmt.Sprintf("chat/%v/%v/%v", chatId, chatFileItemUuid, filename)
}

func ReadFilename(key string) string {
	arr := strings.Split(key, "/")
	return arr[len(arr)-1]
}

func SerializeOriginalKeyToMetadata(originalKeyParam string) map[string]string {
	var userMetadata = map[string]string{}
	userMetadata[originalKey] = originalKeyParam
	return userMetadata
}

func GetOriginalKeyFromMetadata(userMetadata minio.StringMap, hasAmzPrefix bool) (string, error) {
	var prefix = ""
	if hasAmzPrefix {
		prefix = xAmzMetaPrefix
	}

	originalKeyParam, ok := userMetadata[prefix+strings.Title(originalKey)]
	if !ok {
		return "", errors.New("Unable to get originalKey")
	}
	return originalKeyParam, nil
}

func ChatIdKey(hasAmzPrefix bool) string {
	var prefix = ""
	if hasAmzPrefix {
		prefix = xAmzMetaPrefix
	}
	return prefix + strings.Title(chatIdKey)
}

func OwnerIdKey(hasAmzPrefix bool) string {
	var prefix = ""
	if hasAmzPrefix {
		prefix = xAmzMetaPrefix
	}
	return prefix + strings.Title(ownerIdKey)
}

func CorrelationIdKey(hasAmzPrefix bool) string {
	var prefix = ""
	if hasAmzPrefix {
		prefix = xAmzMetaPrefix
	}
	return prefix + strings.Title(correlationIdKey)
}

func ConferenceRecordingKey(hasAmzPrefix bool) string {
	var prefix = ""
	if hasAmzPrefix {
		prefix = xAmzMetaPrefix
	}
	return prefix + strings.Title(conferenceRecordingKey)
}

func MessageRecordingKey(hasAmzPrefix bool) string {
	var prefix = ""
	if hasAmzPrefix {
		prefix = xAmzMetaPrefix
	}
	return prefix + strings.Title(messageRecordingKey)
}

func SerializeTags(public bool) map[string]string {
	var userTags = map[string]string{}
	userTags[publicKey] = fmt.Sprintf("%v", public)
	return userTags
}

func DeserializeTags(tagging *tags.Tags) (bool, error) {
	if tagging == nil {
		return false, nil
	}

	var tagsMap map[string]string = tagging.ToMap()
	publicString, ok := tagsMap[publicKey]
	if !ok {
		return false, nil
	}
	return utils.ParseBoolean(publicString)
}

func GetUsersRemotelyOrEmpty(lgr *log.Logger, userIdSet map[int64]bool, restClient *client.RestClient, c context.Context) map[int64]*dto.User {
	if remoteUsers, err := getUsersRemotely(lgr, userIdSet, restClient, c); err != nil {
		GetLogEntry(c, lgr).Warn("Error during getting users from aaa")
		return map[int64]*dto.User{}
	} else {
		return remoteUsers
	}
}

func getUsersRemotely(lgr *log.Logger, userIdSet map[int64]bool, restClient *client.RestClient, c context.Context) (map[int64]*dto.User, error) {
	var userIds = utils.SetToArray(userIdSet)
	length := len(userIds)
	GetLogEntry(c, lgr).Infof("Requested user length is %v", length)
	if length == 0 {
		return map[int64]*dto.User{}, nil
	}
	users, err := restClient.GetUsers(c, userIds)
	if err != nil {
		return nil, err
	}
	var ownersObjects = map[int64]*dto.User{}
	for _, u := range users {
		ownersObjects[u.Id] = u
	}
	return ownersObjects, nil
}
