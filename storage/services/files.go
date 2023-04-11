package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/tags"
	"github.com/spf13/viper"
	"net/url"
	"nkonev.name/storage/client"
	"nkonev.name/storage/dto"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/s3"
	"nkonev.name/storage/utils"
	"sort"
	"strings"
)

type FilesService struct {
	minio       *s3.InternalMinioClient
	restClient  *client.RestClient
	minioConfig *utils.MinioConfig
}

func NewFilesService(
	minio *s3.InternalMinioClient,
	chatClient *client.RestClient,
	minioConfig *utils.MinioConfig,
) *FilesService {
	return &FilesService{
		minio:       minio,
		restClient:  chatClient,
		minioConfig: minioConfig,
	}
}

func (h *FilesService) GetListFilesInFileItem(
	behalfUserId int64,
	bucket, filenameChatPrefix string,
	chatId int64,
	c context.Context,
	filter func(*minio.ObjectInfo) bool,
	requestOwners bool,
	originalFilename bool,
	size, offset int,
) ([]*dto.FileInfoDto, int, error) {
	var objects <-chan minio.ObjectInfo = h.minio.ListObjects(context.Background(), bucket, minio.ListObjectsOptions{
		WithMetadata: true,
		Prefix:       filenameChatPrefix,
		Recursive:    true,
	})

	var intermediateList []minio.ObjectInfo = make([]minio.ObjectInfo, 0)
	for objInfo := range objects {
		GetLogEntry(c).Debugf("Object '%v'", objInfo.Key)
		if filter != nil && filter(&objInfo) {
			intermediateList = append(intermediateList, objInfo)
		} else if filter == nil {
			intermediateList = append(intermediateList, objInfo)
		}
	}
	sort.SliceStable(intermediateList, func(i, j int) bool {
		return intermediateList[i].LastModified.Unix() > intermediateList[j].LastModified.Unix()
	})

	count := len(intermediateList)

	var list []*dto.FileInfoDto = make([]*dto.FileInfoDto, 0)
	var counter = 0
	var respCounter = 0

	for _, objInfo := range intermediateList {

		if counter >= offset {
			tagging, err := h.minio.GetObjectTagging(c, bucket, objInfo.Key, minio.GetObjectTaggingOptions{})
			if err != nil {
				GetLogEntry(c).Errorf("Error during getting tags %v", err)
				return nil, 0, err
			}

			info, err := h.GetFileInfo(behalfUserId, objInfo, chatId, tagging, true, originalFilename)
			if err != nil {
				GetLogEntry(c).Errorf("Error get file info: %v, skipping", err)
				continue
			}

			list = append(list, info)
			respCounter++
			if respCounter >= size {
				break
			}
		}
		counter++
	}

	if requestOwners {
		var participantIdSet = map[int64]bool{}
		for _, fileDto := range list {
			participantIdSet[fileDto.OwnerId] = true
		}
		var users = GetUsersRemotelyOrEmpty(participantIdSet, h.restClient, c)
		for _, fileDto := range list {
			user := users[fileDto.OwnerId]
			if user != nil {
				fileDto.Owner = user
			}
		}
	}

	return list, count, nil
}

func (h *FilesService) GetDownloadUrl(aKey string) (string, error) {
	ttl := viper.GetDuration("minio.publicDownloadTtl")

	u, err := h.minio.PresignedGetObject(context.Background(), h.minioConfig.Files, aKey, ttl, url.Values{})
	if err != nil {
		return "", err
	}

	err = ChangeUrl(u)
	if err != nil {
		return "", err
	}

	downloadUrl := fmt.Sprintf("%v", u)
	return downloadUrl, nil
}

func ChangeUrl(url *url.URL) error {
	publicUrlPrefix := viper.GetString("minio.publicUrl")
	parsed, err := url.Parse(publicUrlPrefix)
	if err != nil {
		return err
	}

	url.Path = parsed.Path + url.Path
	url.Host = parsed.Host
	url.Scheme = parsed.Scheme
	return nil
}

func (h *FilesService) GetFileInfo(behalfUserId int64, objInfo minio.ObjectInfo, chatId int64, tagging *tags.Tags, hasAmzPrefix bool, originalFilename bool) (*dto.FileInfoDto, error) {
	previewUrl := h.GetPreviewUrlSmart(objInfo.Key)

	metadata := objInfo.UserMetadata

	_, fileOwnerId, fileId, _, err := DeserializeMetadata(metadata, hasAmzPrefix)
	if err != nil {
		Logger.Errorf("Error get metadata: %v", err)
		return nil, err
	}

	filename := ReadFilename(objInfo.Key)

	downloadUrl, err := h.GetDownloadUrl(objInfo.Key)
	if err != nil {
		Logger.Errorf("Error during getting downlad url %v", err)
		return nil, err
	}

	info := &dto.FileInfoDto{
		Id:           fileId,
		Filename:     filename,
		Url:          downloadUrl,
		Size:         objInfo.Size,
		CanDelete:    fileOwnerId == behalfUserId,
		CanEdit:      fileOwnerId == behalfUserId && strings.HasSuffix(objInfo.Key, ".txt"),
		CanShare:     fileOwnerId == behalfUserId,
		LastModified: objInfo.LastModified,
		OwnerId:      fileOwnerId,
		PreviewUrl:   previewUrl,
	}
	return info, nil
}

func (h *FilesService) getBaseUrlForDownload() string {
	return viper.GetString("server.contextPath") + "/storage"
}

func (h *FilesService) GetChatPrivateUrl(minioKey string, chatId int64) (string, string, error) {
	downloadUrl, err := url.Parse(h.getBaseUrlForDownload() + "/download")
	if err != nil {
		return "", "", err
	}

	query := downloadUrl.Query()
	query.Add(utils.FileParam, minioKey)
	downloadUrl.RawQuery = query.Encode()
	str := downloadUrl.String()

	withoutQuery := str
	str += "&original=true"

	return str, withoutQuery, nil
}

const Media_image = "image"
const Media_video = "video"

func (h *FilesService) GetPreviewUrlSmart(aKey string) *string {
	recognizedType := ""
	if utils.IsVideo(aKey) {
		recognizedType = Media_video
		return h.getPreviewUrl(aKey, recognizedType)
	} else if utils.IsImage(aKey) {
		recognizedType = Media_image
		return h.getPreviewUrl(aKey, recognizedType)
	}
	return nil
}

func GetType(itemUrl string) *string {
	var recognizedType string = ""
	if utils.IsVideo(itemUrl) {
		recognizedType = Media_video
	} else if utils.IsImage(itemUrl) {
		recognizedType = Media_image
	}

	if recognizedType != "" {
		return &recognizedType
	} else {
		return nil
	}
}

func (h *FilesService) getPreviewUrl(aKey string, requestedMediaType string) *string {
	var previewUrl *string = nil

	respUrl := url.URL{}
	respUrl.Path = "/api/storage/embed/preview"
	previewMinioKey := ""
	if requestedMediaType == Media_video {
		previewMinioKey = utils.SetVideoPreviewExtension(aKey)
	} else if requestedMediaType == Media_image {
		previewMinioKey = utils.SetImagePreviewExtension(aKey)
	}
	if previewMinioKey != "" {
		query := respUrl.Query()
		query.Set(utils.FileParam, previewMinioKey)

		obj, err := h.minio.StatObject(context.Background(), h.minioConfig.FilesPreview, previewMinioKey, minio.StatObjectOptions{})
		if err == nil {
			// if preview file presents we do set time. it is need to distinguish on front. it's required to update early requested file item without preview
			query.Set(utils.TimeParam, utils.Int64ToString(obj.LastModified.Unix()))
		}

		respUrl.RawQuery = query.Encode()

		tmp := respUrl.String()
		previewUrl = &tmp
	} else {
		Logger.Errorf("Unknown requested type %v", requestedMediaType)
	}

	return previewUrl
}

const publicKey = "public"

const fileIdKey = "fileid"
const ownerIdKey = "ownerid"
const chatIdKey = "chatid"
const correlationIdKey = "correlationid"

const originalKey = "originalkey"

func SerializeMetadataSimple(fileId uuid.UUID, userId int64, chatId int64, correlationId *string) map[string]string {
	var userMetadata = map[string]string{}
	userMetadata[fileIdKey] = fileId.String()
	userMetadata[ownerIdKey] = utils.Int64ToString(userId)
	userMetadata[chatIdKey] = utils.Int64ToString(chatId)
	if correlationId != nil {
		userMetadata[correlationIdKey] = *correlationId
	}
	return userMetadata
}

const xAmzMetaPrefix = "X-Amz-Meta-"

func SerializeMetadataAndStore(urlValues *url.Values, fileId uuid.UUID, userId int64, chatId int64, correlationId *string) {
	urlValues.Set(xAmzMetaPrefix+strings.Title(fileIdKey), fileId.String())
	urlValues.Set(xAmzMetaPrefix+strings.Title(ownerIdKey), utils.Int64ToString(userId))
	urlValues.Set(xAmzMetaPrefix+strings.Title(chatIdKey), utils.Int64ToString(chatId))
	if correlationId != nil {
		urlValues.Set(xAmzMetaPrefix+strings.Title(correlationIdKey), *correlationId)
	}
}

func DeserializeMetadata(userMetadata minio.StringMap, hasAmzPrefix bool) (int64, int64, uuid.UUID, string, error) {
	var prefix = ""
	if hasAmzPrefix {
		prefix = xAmzMetaPrefix
	}
	fileIdStr, ok := userMetadata[prefix+strings.Title(fileIdKey)]
	if !ok {
		return 0, 0, uuid.New(), "", errors.New("Unable to get fileId")
	}
	fileId, err := uuid.Parse(fileIdStr)
	if err != nil {
		return 0, 0, uuid.New(), "", errors.New("Unable to parse fileId")
	}

	ownerIdString, ok := userMetadata[prefix+strings.Title(ownerIdKey)]
	if !ok {
		return 0, 0, uuid.New(), "", errors.New("Unable to get owner id")
	}
	ownerId, err := utils.ParseInt64(ownerIdString)
	if err != nil {
		return 0, 0, uuid.New(), "", err
	}

	chatIdString, ok := userMetadata[prefix+strings.Title(chatIdKey)]
	if !ok {
		return 0, 0, uuid.New(), "", errors.New("Unable to get chat id")
	}
	chatId, err := utils.ParseInt64(chatIdString)
	if err != nil {
		return 0, 0, uuid.New(), "", err
	}
	correlationId := userMetadata[prefix+strings.Title(correlationIdKey)]

	return chatId, ownerId, fileId, correlationId, nil
}

func GenerateFilename(filename string, chatFileItemUuid string, chatId int64) string {
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

func FileIdKey(hasAmzPrefix bool) string {
	var prefix = ""
	if hasAmzPrefix {
		prefix = xAmzMetaPrefix
	}
	return prefix + strings.Title(fileIdKey)
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

func GetUsersRemotelyOrEmpty(userIdSet map[int64]bool, restClient *client.RestClient, c context.Context) map[int64]*dto.User {
	if remoteUsers, err := getUsersRemotely(userIdSet, restClient, c); err != nil {
		GetLogEntry(c).Warn("Error during getting users from aaa")
		return map[int64]*dto.User{}
	} else {
		return remoteUsers
	}
}

func getUsersRemotely(userIdSet map[int64]bool, restClient *client.RestClient, c context.Context) (map[int64]*dto.User, error) {
	var userIds = utils.SetToArray(userIdSet)
	length := len(userIds)
	GetLogEntry(c).Infof("Requested user length is %v", length)
	if length == 0 {
		return map[int64]*dto.User{}, nil
	}
	users, err := restClient.GetUsers(userIds, c)
	if err != nil {
		return nil, err
	}
	var ownersObjects = map[int64]*dto.User{}
	for _, u := range users {
		ownersObjects[u.Id] = u
	}
	return ownersObjects, nil
}
