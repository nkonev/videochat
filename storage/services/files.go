package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/tags"
	"github.com/spf13/viper"
	"net/url"
	"nkonev.name/storage/client"
	"nkonev.name/storage/db"
	"nkonev.name/storage/dto"
	"nkonev.name/storage/logger"
	"nkonev.name/storage/s3"
	"nkonev.name/storage/utils"
	"strings"
	"time"
)

type FilesService struct {
	minio       *s3.InternalMinioClient
	restClient  *client.RestClient
	minioConfig *utils.MinioConfig
	dba         *db.DB
	lgr         *logger.Logger
}

func NewFilesService(
	lgr *logger.Logger,
	minio *s3.InternalMinioClient,
	chatClient *client.RestClient,
	dba *db.DB,
	minioConfig *utils.MinioConfig,
) *FilesService {
	return &FilesService{
		minio:       minio,
		restClient:  chatClient,
		minioConfig: minioConfig,
		dba:         dba,
		lgr:         lgr,
	}
}

func (h *FilesService) GetListFilesInFileItem(
	c context.Context,
	public bool,
	overrideChatId, overrideMessageId int64,
	behalfUserId *int64,
	bucket string,
	chatId int64,
	fileItemUuid string, // can be empty string
	filterObj db.Filter,
	requestOwners bool,
	size, offset int,
) ([]*dto.FileInfoDto, int64, error) {
	if !public && behalfUserId == nil {
		return nil, 0, errors.New("wrong invariant")
	}

	metadatas, err := db.GetList(c, h.dba, chatId, fileItemUuid, filterObj, size, offset)
	if err != nil {
		h.lgr.WithTracing(c).Errorf("Error during getting list, userId = %v, chatId = %v: %v", behalfUserId, chatId, err)
		return []*dto.FileInfoDto{}, 0, err
	}

	var count int64
	count, err = db.GetCount(c, h.dba, chatId, fileItemUuid, filterObj)
	if err != nil {
		h.lgr.WithTracing(c).Errorf("Error during getting count %v", err)
		return []*dto.FileInfoDto{}, 0, err
	}

	var list []*dto.FileInfoDto = make([]*dto.FileInfoDto, 0)

	for _, mce := range metadatas {
		info, err := h.GetFileInfo(c, public, overrideChatId, overrideMessageId, behalfUserId, &mce)
		if err != nil {
			h.lgr.WithTracing(c).Errorf("Error get file info: %v, skipping", err)
			continue
		}
		list = append(list, info)
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

	return list, count, nil
}

func (h *FilesService) GetListFilesItemUuids(
	c context.Context,
	chatId int64,
	size, offset int,
) ([]db.GroupedByFileItemUuid, int, error) {
	datas, err := db.GetListFilesItemUuids(c, h.dba, chatId, size, offset)
	if err != nil {
		h.lgr.WithTracing(c).Errorf("Unable to get GroupedByFileItemUuid: %v", err)
		return nil, 0, err
	}
	count, err := db.GetCountFilesItemUuids(c, h.dba, chatId)
	if err != nil {
		h.lgr.WithTracing(c).Errorf("Unable to get count GroupedByFileItemUuid: %v", err)
		return nil, 0, err
	}

	return datas, int(count), nil
}

func (h *FilesService) GetTemporaryDownloadUrl(ctx context.Context, aKey string) (string, time.Duration, error) {
	ttl := viper.GetDuration("minio.presignDownloadTtl")

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
	externalS3UrlPrefix := viper.GetString("minio.externalS3UrlPrefix")
	parsed, err := url.Parse(externalS3UrlPrefix)
	if err != nil {
		return "", err
	}

	url.Path = parsed.Path + url.Path
	url.Host = ""
	url.Scheme = ""

	stringV := url.String()

	return stringV, nil
}

func (h *FilesService) GetPublishedUrl(public bool, fileName string) (*string, error) {
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

func (h *FilesService) GetFileInfo(c context.Context, public bool, overrideChatId, overrideMessageId int64, behalfUserId *int64, mce *dto.MetadataCache) (*dto.FileInfoDto, error) {
	if !public && behalfUserId == nil {
		return nil, errors.New("wrong invariant")
	}

	if mce == nil {
		return nil, errors.New("nil MetadataCache")
	}

	fileOwnerId := mce.OwnerId
	theCorrelationId := mce.CorrelationId

	filename := mce.Filename

	published := mce.Published

	aKey := utils.BuildNormalizedKey(mce)

	publishedUrl, err := h.GetPublishedUrl(published, aKey)
	if err != nil {
		h.lgr.WithTracing(c).Errorf("Error get published url: %v", err)
		return nil, err
	}

	itemUuid := mce.FileItemUuid

	var downloadUrl string
	var previewUrl *string

	var canDelete, canEdit, canShare bool

	downloadUrltmp, err := h.GetConstantDownloadUrl(aKey)
	if err != nil {
		h.lgr.WithTracing(c).Errorf("Error during getting downlad url %v", err)
		return nil, err
	}

	previewUrltmp := h.GetPreviewUrlSmart(c, aKey)

	if !public {
		// normal flow
		canDelete = fileOwnerId == *behalfUserId
		canEdit = fileOwnerId == *behalfUserId && utils.IsPlainText(aKey)
		canShare = fileOwnerId == *behalfUserId

		downloadUrl = downloadUrltmp
		previewUrl = previewUrltmp
	} else {
		// public microservice flow - user clicks on FileListModal
		// it's safe becasue we already checked the access before
		downloadUrl, err = makeUrlPublic(downloadUrltmp, "", overrideChatId, overrideMessageId)
		if err != nil {
			h.lgr.WithTracing(c).Errorf("Error during getting downlad url %v", err)
			return nil, err
		}

		if previewUrltmp != nil {
			previewUrlpublic, err := makeUrlPublic(*previewUrltmp, utils.UrlStorageEmbedPreview, overrideChatId, overrideMessageId)
			if err != nil {
				h.lgr.WithTracing(c).Errorf("Error during getting downlad url %v", err)
				return nil, err
			}
			previewUrl = &previewUrlpublic
		}
	}

	var aType = GetType(aKey)

	info := &dto.FileInfoDto{
		Id:             aKey,
		Filename:       filename,
		Url:            downloadUrl,
		Size:           mce.FileSize,
		CanDelete:      canDelete,
		CanEdit:        canEdit,
		CanShare:       canShare,
		LastModified:   mce.EditDateTime,
		OwnerId:        fileOwnerId,
		PublishedUrl:   publishedUrl,
		PreviewUrl:     previewUrl,
		CanPlayAsVideo: utils.IsVideo(aKey),
		CanShowAsImage: utils.IsImage(aKey),
		CanPlayAsAudio: utils.IsAudio(aKey),
		FileItemUuid:   itemUuid,
		CorrelationId:  theCorrelationId,
		Previewable:    utils.IsPreviewable(aKey),
		Type:           aType,
	}
	return info, nil
}

// prepares url to use in public microservice
// in case getting file list
// see also chat/handlers/blog.go :: makeUrlPublic
func makeUrlPublic(src string, additionalSegment string, overrideChatId, overrideMessageId int64) (string, error) {
	if strings.HasPrefix(src, "/images/covers/") { // don't touch built-in default urls (used for video-by-link, audio)
		return src, nil
	}

	// we add time in order not to cache the video itself
	parsed, err := url.Parse(src)
	if err != nil {
		return "", err
	}

	parsed.Path = utils.UrlStoragePublicGetFile + additionalSegment

	query := parsed.Query()

	query.Set(utils.OverrideMessageId, utils.Int64ToString(overrideMessageId))
	query.Set(utils.OverrideChatId, utils.Int64ToString(overrideChatId))

	parsed.RawQuery = query.Encode()

	newurl := parsed.String()
	return newurl, nil
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

func GetTypeExtensions(requestedMediaType string) []string {
	switch requestedMediaType {
	case Media_video:
		return viper.GetStringSlice("types.video")
	case Media_image:
		return viper.GetStringSlice("types.image")
	case Media_audio:
		return viper.GetStringSlice("types.audio")
	default:
		return []string{}
	}
}

func GetPreviewableExtensions() []string {
	res := []string{}
	res = append(res, viper.GetStringSlice("types.video")...)
	res = append(res, viper.GetStringSlice("types.image")...)
	return res
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
		h.lgr.WithTracing(c).Errorf("Unknown requested type %v", requestedMediaType)
	}

	return previewUrl
}

const publishedKey = "published"

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

func DeserializeMetadata(userMetadata minio.StringMap, hasAmzPrefix bool) (int64, int64, string, error) {
	var prefix = ""
	if hasAmzPrefix {
		prefix = xAmzMetaPrefix
	}

	ownerIdString, ok := userMetadata[prefix+strings.Title(ownerIdKey)]
	if !ok {
		return 0, 0, "", errors.New("Unable to get owner id")
	}
	ownerId, err := utils.ParseInt64(ownerIdString)
	if err != nil {
		return 0, 0, "", err
	}

	chatIdString, ok := userMetadata[prefix+strings.Title(chatIdKey)]
	if !ok {
		return 0, 0, "", errors.New("Unable to get chat id")
	}
	chatId, err := utils.ParseInt64(chatIdString)
	if err != nil {
		return 0, 0, "", err
	}
	correlationId := userMetadata[prefix+strings.Title(correlationIdKey)]

	return chatId, ownerId, correlationId, nil
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

func SerializeTags(published bool) map[string]string {
	var userTags = map[string]string{}
	userTags[publishedKey] = fmt.Sprintf("%v", published)
	return userTags
}

func DeserializeTags(tagging *tags.Tags) (bool, error) {
	if tagging == nil {
		return false, nil
	}

	var tagsMap map[string]string = tagging.ToMap()
	publishedString, ok := tagsMap[publishedKey]
	if !ok {
		return false, nil
	}
	return utils.ParseBoolean(publishedString)
}

func GetUsersRemotelyOrEmpty(lgr *logger.Logger, userIdSet map[int64]bool, restClient *client.RestClient, c context.Context) map[int64]*dto.User {
	if remoteUsers, err := getUsersRemotely(lgr, userIdSet, restClient, c); err != nil {
		lgr.WithTracing(c).Warn("Error during getting users from aaa")
		return map[int64]*dto.User{}
	} else {
		return remoteUsers
	}
}

func getUsersRemotely(lgr *logger.Logger, userIdSet map[int64]bool, restClient *client.RestClient, c context.Context) (map[int64]*dto.User, error) {
	var userIds = utils.SetToArray(userIdSet)
	length := len(userIds)
	lgr.WithTracing(c).Infof("Requested user length is %v", length)
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
