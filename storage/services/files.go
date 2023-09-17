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
	"time"
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

			info, err := h.GetFileInfo(behalfUserId, objInfo, chatId, tagging, true)
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

type SimpleFileItem struct {
	Id             string    `json:"id"`
	Filename       string    `json:"filename"`
	LastModified   time.Time `json:"time"`
}

type GroupedByFileItemUuid struct {
	FileItemUuid uuid.UUID `json:"fileItemUuid"`
	Files []SimpleFileItem `json:"files"`
}

func (h *FilesService) GetListFilesItemUuids(
	bucket, filenameChatPrefix string,
	c context.Context,
	size, offset int,
) ([]*GroupedByFileItemUuid, int, error) {
	var objects <-chan minio.ObjectInfo = h.minio.ListObjects(context.Background(), bucket, minio.ListObjectsOptions{
		WithMetadata: true,
		Prefix:       filenameChatPrefix,
		Recursive:    true,
	})

	tmpMap := make(map[uuid.UUID][]SimpleFileItem)
	for m := range objects {
		itemUuid, err := utils.ParseFileItemUuid(m.Key)
		if err != nil {
			GetLogEntry(c).Errorf("Unable for %v to get fileItemUuid '%v'", m.Key, err)
		} else {
			if _, ok := tmpMap[itemUuid]; !ok {
				tmpMap[itemUuid] = []SimpleFileItem{}
			}

			tmpMap[itemUuid] = append(tmpMap[itemUuid], SimpleFileItem{
				m.Key,
				ReadFilename(m.Key),
				m.LastModified,
			})
		}
	}

	tmlList := make([]*GroupedByFileItemUuid, 0)
	for k, v := range tmpMap {
		tmlList = append(tmlList, &GroupedByFileItemUuid{k, v})
	}
	sort.SliceStable(tmlList, func(i, j int) bool {
		first := tmlList[i]
		second := tmlList[j]
		if len(first.Files) > 0 && len(second.Files) > 0 {
			return first.Files[0].LastModified.Unix() > second.Files[0].LastModified.Unix()
		} else {
			return false
		}
	})

	count := len(tmlList)

	var list []*GroupedByFileItemUuid = make([]*GroupedByFileItemUuid, 0)
	var counter = 0
	var respCounter = 0

	for _, item := range tmlList {

		if counter >= offset {
			list = append(list, item)
			respCounter++
			if respCounter >= size {
				break
			}
		}
		counter++
	}

	return list, count, nil
}

func (h *FilesService) GetCount(ctx context.Context, filenameChatPrefix string) (int, error) {
	var objects <-chan minio.ObjectInfo = h.minio.ListObjects(context.Background(), h.minioConfig.Files, minio.ListObjectsOptions{
		Prefix:    filenameChatPrefix,
		Recursive: true,
	})

	var count int = 0
	for objInfo := range objects {
		GetLogEntry(ctx).Debugf("Object '%v'", objInfo.Key)
		count++
	}
	return count, nil
}

func (h *FilesService) GetTemporaryDownloadUrl(aKey string) (string, error) {
	ttl := viper.GetDuration("minio.publicDownloadTtl")

	u, err := h.minio.PresignedGetObject(context.Background(), h.minioConfig.Files, aKey, ttl, url.Values{})
	if err != nil {
		return "", err
	}

	err = ChangeMinioUrl(u)
	if err != nil {
		return "", err
	}

	downloadUrl := fmt.Sprintf("%v", u)
	return downloadUrl, nil
}

func (h *FilesService) GetConstantDownloadUrl(aKey string) (string, error) {
	downloadUrl, err := url.Parse(viper.GetString("server.contextPath") + utils.UrlStorageGetFile)
	if err != nil {
		return "", err
	}

	query := downloadUrl.Query()
	query.Add(utils.FileParam, aKey)
	downloadUrl.RawQuery = query.Encode()

	downloadUrlStr := fmt.Sprintf("%v", downloadUrl)
	return downloadUrlStr, nil
}

func ChangeMinioUrl(url *url.URL) error {
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

func (h *FilesService) getPublicUrl(public bool, fileName string) (*string, error) {
	if !public {
		return nil, nil
	}

	downloadUrl, err := url.Parse(h.getBaseUrlForDownload() + utils.UrlStorageGetFilePublicExternal)
	if err != nil {
		return nil, err
	}

	query := downloadUrl.Query()
	query.Add(utils.FileParam, fileName)
	downloadUrl.RawQuery = query.Encode()
	str := downloadUrl.String()
	return &str, nil
}

func (h *FilesService) GetFileInfo(behalfUserId int64, objInfo minio.ObjectInfo, chatId int64, tagging *tags.Tags, hasAmzPrefix bool) (*dto.FileInfoDto, error) {
	previewUrl := h.GetPreviewUrlSmart(objInfo.Key)

	metadata := objInfo.UserMetadata

	_, fileOwnerId, _, err := DeserializeMetadata(metadata, hasAmzPrefix)
	if err != nil {
		Logger.Errorf("Error get metadata: %v", err)
		return nil, err
	}

	filename := ReadFilename(objInfo.Key)

	public, err := DeserializeTags(tagging)
	if err != nil {
		Logger.Errorf("Error get tags: %v", err)
		return nil, err
	}

	publicUrl, err := h.getPublicUrl(public, objInfo.Key)
	if err != nil {
		Logger.Errorf("Error get public url: %v", err)
		return nil, err
	}

	downloadUrl, err := h.GetConstantDownloadUrl(objInfo.Key)
	if err != nil {
		Logger.Errorf("Error during getting downlad url %v", err)
		return nil, err
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
	}
	return info, nil
}

func (h *FilesService) getBaseUrlForDownload() string {
	return viper.GetString("server.contextPath") + "/storage"
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

const ownerIdKey = "ownerid"
const chatIdKey = "chatid"
const correlationIdKey = "correlationid"

const originalKey = "originalkey"

func SerializeMetadataSimple(userId int64, chatId int64, correlationId *string) map[string]string {
	var userMetadata = map[string]string{}
	userMetadata[ownerIdKey] = utils.Int64ToString(userId)
	userMetadata[chatIdKey] = utils.Int64ToString(chatId)
	if correlationId != nil {
		userMetadata[correlationIdKey] = *correlationId
	}
	return userMetadata
}

const xAmzMetaPrefix = "X-Amz-Meta-"

func SerializeMetadataAndStore(urlValues *url.Values, userId int64, chatId int64, correlationId *string) {
	urlValues.Set(xAmzMetaPrefix+strings.Title(ownerIdKey), utils.Int64ToString(userId))
	urlValues.Set(xAmzMetaPrefix+strings.Title(chatIdKey), utils.Int64ToString(chatId))
	if correlationId != nil {
		urlValues.Set(xAmzMetaPrefix+strings.Title(correlationIdKey), *correlationId)
	}
}

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
