package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/tags"
	"github.com/spf13/viper"
	"mime/multipart"
	"net/url"
	"nkonev.name/storage/auth"
	"nkonev.name/storage/client"
	"nkonev.name/storage/dto"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/utils"
	"sort"
	"strings"
)

const UrlStorageGetFilePublicExternal = "/public/download"

type FilesService struct {
	minio      *minio.Client
	restClient *client.RestClient
}

func NewFilesService(
	minio *minio.Client,
	chatClient *client.RestClient,
) *FilesService {
	return &FilesService{
		minio:      minio,
		restClient: chatClient,
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

func (h *FilesService) GetFileInfo(behalfUserId int64, objInfo minio.ObjectInfo, chatId int64, tagging *tags.Tags, hasAmzPrefix bool, originalFilename bool) (*dto.FileInfoDto, error) {
	downloadUrl, err := h.GetChatPrivateUrlFromObject(objInfo.Key, chatId, originalFilename)
	if err != nil {
		Logger.Errorf("Error get chat private url: %v", err)
		return nil, err
	}
	metadata := objInfo.UserMetadata

	_, fileOwnerId, fileName, err := DeserializeMetadata(metadata, hasAmzPrefix)
	if err != nil {
		Logger.Errorf("Error get metadata: %v", err)
		return nil, err
	}

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

	info := &dto.FileInfoDto{
		Id:           objInfo.Key,
		Filename:     fileName,
		Url:          *downloadUrl,
		Size:         objInfo.Size,
		CanDelete:    fileOwnerId == behalfUserId,
		CanEdit:      fileOwnerId == behalfUserId && strings.HasSuffix(fileName, ".txt"),
		CanShare:     fileOwnerId == behalfUserId,
		LastModified: objInfo.LastModified,
		OwnerId:      fileOwnerId,
		PublicUrl:    publicUrl,
	}
	return info, nil
}

func (h *FilesService) getPublicUrl(public bool, fileName string) (*string, error) {
	if !public {
		return nil, nil
	}

	downloadUrl, err := url.Parse(h.getBaseUrlForDownload() + UrlStorageGetFilePublicExternal)
	if err != nil {
		return nil, err
	}

	query := downloadUrl.Query()
	query.Add(utils.FileParam, fileName)
	downloadUrl.RawQuery = query.Encode()
	str := downloadUrl.String()
	return &str, nil
}

func (h *FilesService) getBaseUrlForDownload() string {
	return viper.GetString("server.contextPath") + "/storage"
}

func (h *FilesService) GetChatPrivateUrlFromObject(minioKey string, chatId int64, originalFilename bool) (*string, error) {
	downloadUrl, err := url.Parse(h.getBaseUrlForDownload() + "/download")
	if err != nil {
		return nil, err
	}

	query := downloadUrl.Query()
	query.Add(utils.FileParam, minioKey)
	downloadUrl.RawQuery = query.Encode()
	str := downloadUrl.String()

	if originalFilename {
		str += "&original=true"
	}

	return &str, nil
}

const publicKey = "public"

const filenameKey = "filename"
const ownerIdKey = "ownerid"
const chatIdKey = "chatid"

const originalKey = "originalkey"

func SerializeMetadata(file *multipart.FileHeader, userPrincipalDto *auth.AuthResult, chatId int64) map[string]string {
	return SerializeMetadataByArgs(file.Filename, userPrincipalDto, chatId)
}

func SerializeMetadataByArgs(filename string, userPrincipalDto *auth.AuthResult, chatId int64) map[string]string {
	return SerializeMetadataSimple(filename, userPrincipalDto.UserId, chatId)
}

func SerializeMetadataSimple(filename string, userId int64, chatId int64) map[string]string {
	var userMetadata = map[string]string{}
	userMetadata[filenameKey] = filename
	userMetadata[ownerIdKey] = utils.Int64ToString(userId)
	userMetadata[chatIdKey] = utils.Int64ToString(chatId)
	return userMetadata
}

func SerializeOriginalKeyToMetadata(originalKeyParam string) map[string]string {
	var userMetadata = map[string]string{}
	userMetadata[originalKey] = originalKeyParam
	return userMetadata
}

const xAmzMetaPrefix = "X-Amz-Meta-"

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

func DeserializeMetadata(userMetadata minio.StringMap, hasAmzPrefix bool) (int64, int64, string, error) {
	var prefix = ""
	if hasAmzPrefix {
		prefix = xAmzMetaPrefix
	}
	filename, ok := userMetadata[prefix+strings.Title(filenameKey)]
	if !ok {
		return 0, 0, "", errors.New("Unable to get filename")
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
	return chatId, ownerId, filename, nil
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
