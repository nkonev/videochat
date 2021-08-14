package handlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/tags"
	"github.com/spf13/viper"
	"mime/multipart"
	"net/http"
	"net/url"
	"nkonev.name/storage/auth"
	"nkonev.name/storage/client"
	"nkonev.name/storage/dto"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/utils"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type FilesHandler struct {
	serverUrl          string
	minio              *minio.Client
	chatClient *client.RestClient
}

type RenameDto struct {
	Newname string `json:"newname"`
}

const filesMultipartKey = "files"
const UrlStorageGetFile = "/storage/public/download"


type FileInfoDto struct {
	Id           string    `json:"id"`
	Filename     string    `json:"filename"`
	Url          string    `json:"url"`
	PublicUrl    string    `json:"publicUrl"`
	Size         int64     `json:"size"`
	CanRemove    bool      `json:"canRemove"`
	LastModified time.Time `json:"lastModified"`
	OwnerId      int64     `json:"ownerId"`
	Owner        *dto.User `json:"owner"`
}

const filenameKey = "filename"
const ownerIdKey = "ownerid"
const chatIdKey = "chatid"

const publicKey = "public"

func NewFilesHandler(
	minio *minio.Client,
	chatClient *client.RestClient,
) *FilesHandler {
	return &FilesHandler{
		minio:              minio,
		serverUrl:          viper.GetString("server.url"),
		chatClient: chatClient,
	}
}

func serializeMetadata(file *multipart.FileHeader, userPrincipalDto *auth.AuthResult, chatId int64) map[string]string {
	var userMetadata = map[string]string{}
	userMetadata[filenameKey] = file.Filename
	userMetadata[ownerIdKey] = utils.Int64ToString(userPrincipalDto.UserId)
	userMetadata[chatIdKey] = utils.Int64ToString(chatId)
	return userMetadata
}

func deserializeMetadata(userMetadata minio.StringMap, hasAmzPrefix bool) (int64, int64, string, error) {
	const xAmzMetaPrefix = "X-Amz-Meta-"
	var prefix = ""
	if hasAmzPrefix {
		prefix = xAmzMetaPrefix
	}
	filename, ok := userMetadata[prefix+strings.Title(filenameKey)]
	if ! ok {
		return 0, 0, "", errors.New("Unable to get filename")
	}
	ownerIdString, ok := userMetadata[prefix+strings.Title(ownerIdKey)]
	if ! ok {
		return 0, 0, "", errors.New("Unable to get owner id")
	}
	ownerId, err := utils.ParseInt64(ownerIdString)
	if err != nil {
		return 0, 0, "", err
	}

	chatIdString, ok := userMetadata[prefix+strings.Title(chatIdKey)]
	if ! ok {
		return 0, 0, "", errors.New("Unable to get chat id")
	}
	chatId, err := utils.ParseInt64(chatIdString)
	if err != nil {
		return 0, 0, "", err
	}
	return chatId, ownerId, filename, nil
}

func serializeTags(public bool) map[string]string {
	var userTags = map[string]string{}
	userTags[publicKey] = fmt.Sprintf("%v", public)
	return userTags
}

func deserializeTags(tags map[string]string) (bool, error) {
	publicString, ok := tags[strings.Title(publicKey)]
	if !ok {
		return false, errors.New("Unable to get public")
	}
	return utils.ParseBoolean(publicString)
}

func (h *FilesHandler) UploadHandler(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}
	if ok, err := h.chatClient.CheckAccess(userPrincipalDto.UserId, chatId); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	bucketName, err := EnsureAndGetFilesBucket(h.minio)
	if err != nil {
		return err
	}

	fileItemUuid := uuid.New().String()

	fileItemUuidString := c.Param("fileItemUuid")
	if fileItemUuidString != "" {
		fileItemUuid = fileItemUuidString
	}

	// check this fileItem belongs to user
	filenameChatPrefix := fmt.Sprintf("chat/%v/%v/", chatId, fileItemUuid)
	belongs, err := h.checkFileItemBelongsToUser(filenameChatPrefix, c, chatId, bucketName, userPrincipalDto)
	if err != nil {
		return err
	}
	if !belongs {
		return c.NoContent(http.StatusUnauthorized)
	}
	// end check

	form, err := c.MultipartForm()
	if err != nil {
		return err
	}
	files := form.File[filesMultipartKey]

	for _, file := range files {
		userLimitOk, err := h.checkUserLimit(bucketName, userPrincipalDto, file)
		if err != nil {
			return err
		}
		if !userLimitOk {
			return c.JSON(http.StatusRequestEntityTooLarge, &utils.H{"status": "fail"})
		}

		contentType := file.Header.Get("Content-Type")
		dotExt := getDotExtension(file)

		Logger.Debugf("Determined content type: %v", contentType)

		src, err := file.Open()
		if err != nil {
			return err
		}
		defer src.Close()

		fileUuid := uuid.New().String()
		filename := fmt.Sprintf("chat/%v/%v/%v%v", chatId, fileItemUuid, fileUuid, dotExt)

		var userMetadata = serializeMetadata(file, userPrincipalDto, chatId)

		if _, err := h.minio.PutObject(context.Background(), bucketName, filename, src, file.Size, minio.PutObjectOptions{ContentType: contentType, UserMetadata: userMetadata}); err != nil {
			Logger.Errorf("Error during upload object: %v", err)
			return err
		}
	}

	// get count
	count := h.getCountFilesInFileItem(bucketName, filenameChatPrefix)

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "fileItemUuid": fileItemUuid, "count": count})
}

func (h *FilesHandler) getCountFilesInFileItem(bucketName string, filenameChatPrefix string) int {
	var count = 0
	var objectsNew <-chan minio.ObjectInfo = h.minio.ListObjects(context.Background(), bucketName, minio.ListObjectsOptions{
		Prefix:    filenameChatPrefix,
		Recursive: true,
	})
	count = len(objectsNew)
	for oi := range objectsNew {
		Logger.Debugf("Processing %v", oi.Key)
		count++
	}
	return count
}

func getDotExtension(file *multipart.FileHeader) string {
	split := strings.Split(file.Filename, ".")
	if len(split) > 1 {
		return "."+split[len(split)-1]
	} else {
		return ""
	}
}

func (h *FilesHandler) checkUserLimit(bucketName string, userPrincipalDto *auth.AuthResult, file *multipart.FileHeader) (bool, error) {
	consumption := h.calcUserFilesConsumption(bucketName)
	maxAllowed, err := h.getMaxAllowedConsumption(userPrincipalDto)
	if err != nil {
		Logger.Errorf("Error during calculating max allowed %v", err)
		return false, err
	}
	if consumption+file.Size > maxAllowed {
		Logger.Infof("Upload too large %v+%v>%v bytes", consumption, file.Size, maxAllowed)
		return false, nil
	}
	return true, nil
}

func (h *FilesHandler) calcUserFilesConsumption(bucketName string) int64 {
	var totalBucketConsumption int64

	doneCh := make(chan struct{})
	defer close(doneCh)

	Logger.Debugf("Listing bucket '%v':", bucketName)
	for objInfo := range h.minio.ListObjects(context.Background(), bucketName, minio.ListObjectsOptions{Recursive: true}) {
		totalBucketConsumption += objInfo.Size
	}
	return totalBucketConsumption
}

func (h *FilesHandler) getMaxAllowedConsumption(userPrincipalDto *auth.AuthResult) (int64, error) {
	isUnlimited := userPrincipalDto != nil && userPrincipalDto.HasRole("ROLE_ADMIN")
	if isUnlimited {
		var stat syscall.Statfs_t
		wd := viper.GetString("limits.stat.dir")
		err := syscall.Statfs(wd, &stat)
		if err != nil {
			return 0, err
		}
		// Available blocks * size per block = available space in bytes
		u := int64(stat.Bavail * uint64(stat.Bsize))
		return u, nil
	} else {
		return viper.GetInt64("limits.default.per.user.max"), nil
	}
}

func (h *FilesHandler) ListHandler(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}
	if ok, err := h.chatClient.CheckAccess(userPrincipalDto.UserId, chatId); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	fileItemUuid := c.QueryParam("fileItemUuid")

	bucket, err := EnsureAndGetFilesBucket(h.minio)
	if err != nil {
		return err
	}

	Logger.Debugf("Listing bucket '%v':", bucket)

	var filenameChatPrefix string
	if fileItemUuid == "" {
		filenameChatPrefix = fmt.Sprintf("chat/%v/", chatId)
	} else {
		filenameChatPrefix = fmt.Sprintf("chat/%v/%v/", chatId, fileItemUuid)
	}

	list, err := h.getListFilesInFileItem(userPrincipalDto.UserId, bucket, filenameChatPrefix, chatId)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "files": list})
}

func getUsersRemotelyOrEmpty(userIdSet map[int64]bool, restClient *client.RestClient) map[int64]*dto.User {
	if remoteUsers, err := getUsersRemotely(userIdSet, restClient); err != nil {
		Logger.Warn("Error during getting users from aaa")
		return map[int64]*dto.User{}
	} else {
		return remoteUsers
	}
}

func getUsersRemotely(userIdSet map[int64]bool, restClient *client.RestClient) (map[int64]*dto.User, error) {
	var userIds = utils.SetToArray(userIdSet)
	length := len(userIds)
	Logger.Infof("Requested user length is %v", length)
	if length == 0 {
		return map[int64]*dto.User{}, nil
	}
	users, err := restClient.GetUsers(userIds)
	if err != nil {
		return nil, err
	}
	var ownersObjects = map[int64]*dto.User{}
	for _, u := range users {
		ownersObjects[u.Id] = u
	}
	return ownersObjects, nil
}

func (h *FilesHandler) getListFilesInFileItem(behalfUserId int64, bucket, filenameChatPrefix string, chatId int64) ([]*FileInfoDto, error) {
	var objects <-chan minio.ObjectInfo = h.minio.ListObjects(context.Background(), bucket, minio.ListObjectsOptions{
		WithMetadata: true,
		Prefix:       filenameChatPrefix,
		Recursive: true,
	})

	var list []*FileInfoDto = make([]*FileInfoDto, 0)
	for objInfo := range objects {
		Logger.Debugf("Object '%v'", objInfo.Key)

		downloadUrl, err := h.getChatPrivateUrlFromObject(objInfo, chatId)
		if err != nil {
			Logger.Errorf("Error get private url: %v", err)
			return nil, err
		}
		metadata := objInfo.UserMetadata

		_, fileOwnerId, fileName, err := deserializeMetadata(metadata, true)
		if err != nil {
			Logger.Errorf("Error get file name url: %v, skipping", err)
			continue
		}

		info := &FileInfoDto{
			Id: objInfo.Key,
			Filename: fileName,
			Url: *downloadUrl,
			Size: objInfo.Size,
			CanRemove: fileOwnerId == behalfUserId,
			LastModified: objInfo.LastModified,
			OwnerId: fileOwnerId,
		}
		list = append(list, info)
	}
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].LastModified.Unix() < list[j].LastModified.Unix()
	})

	var participantIdSet = map[int64]bool{}
	for _, fileDto := range list {
		participantIdSet[fileDto.OwnerId] = true
	}
	var users = getUsersRemotelyOrEmpty(participantIdSet, h.chatClient)
	for _, fileDto := range list {
		user := users[fileDto.OwnerId]
		if user != nil {
			fileDto.Owner = user
		}
	}

	return list, nil
}

func (h *FilesHandler) getChatPrivateUrlFromObject(objInfo minio.ObjectInfo, chatId int64) (*string, error) {
	downloadUrl, err := url.Parse(h.serverUrl + viper.GetString("server.contextPath") + "/storage/download")
	if err != nil {
		return nil, err
	}

	query := downloadUrl.Query()
	query.Add("file", objInfo.Key)
	downloadUrl.RawQuery = query.Encode()
	str := downloadUrl.String()
	return &str, nil
}

type DeleteObjectDto struct {
	Id     string     `json:"id"` // file id
}

func (h *FilesHandler) DeleteHandler(c echo.Context) error {
	var bindTo = new(DeleteObjectDto)
	if err := c.Bind(bindTo); err != nil {
		GetLogEntry(c.Request()).Warnf("Error during binding to dto %v", err)
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}
	if ok, err := h.chatClient.CheckAccess(userPrincipalDto.UserId, chatId); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	bucketName, err := EnsureAndGetFilesBucket(h.minio)
	if err != nil {
		return err
	}

	// check this fileItem belongs to user
	objectInfo, err := h.minio.StatObject(context.Background(), bucketName, bindTo.Id, minio.StatObjectOptions{})
	if err != nil {
		Logger.Errorf("Error during getting object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	belongs, err := h.checkFileBelongsToUser(objectInfo, chatId, userPrincipalDto, false)
	if err != nil {
		Logger.Errorf("Error during checking belong object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !belongs {
		Logger.Errorf("Object '%v' is not belongs to user %v", objectInfo.Key, userPrincipalDto.UserId)
		return c.NoContent(http.StatusUnauthorized)
	}
	// end check

	err = h.minio.RemoveObject(context.Background(), bucketName, bindTo.Id, minio.RemoveObjectOptions{})
	if err != nil {
		Logger.Errorf("Error during removing object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	fileItemUuid := c.QueryParam("fileItemUuid")
	var filenameChatPrefix string
	if fileItemUuid == "" {
		filenameChatPrefix = fmt.Sprintf("chat/%v/", chatId)
	} else {
		filenameChatPrefix = fmt.Sprintf("chat/%v/%v/", chatId, fileItemUuid)
	}

	list, err := h.getListFilesInFileItem(userPrincipalDto.UserId, bucketName, filenameChatPrefix, chatId)
	if err != nil {
		return err
	}

	if len(list) == 0 {
		h.chatClient.RemoveFileItem(chatId, fileItemUuid, userPrincipalDto.UserId)
	}

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "files": list})
}

func (h *FilesHandler) checkFileItemBelongsToUser(filenameChatPrefix string, c echo.Context, chatId int64, bucketName string, userPrincipalDto *auth.AuthResult) (bool, error) {
	var objects <-chan minio.ObjectInfo = h.minio.ListObjects(context.Background(), bucketName, minio.ListObjectsOptions{
		WithMetadata: true,
		Prefix:       filenameChatPrefix,
		Recursive:    true,
	})
	for objInfo := range objects {
		b, err := h.checkFileBelongsToUser(objInfo, chatId, userPrincipalDto, true)
		if err != nil {
			return false, err
		}
		if !b {
			return false, nil
		}
	}
	return true, nil
}

func (h *FilesHandler) checkFileBelongsToUser(objInfo minio.ObjectInfo, chatId int64, userPrincipalDto *auth.AuthResult, hasAmzPrefix bool) (bool, error) {
	gotChatId, gotOwnerId, _, err := deserializeMetadata(objInfo.UserMetadata, hasAmzPrefix)
	if err != nil {
		Logger.Errorf("Error deserializeMetadata: %v", err)
		return false, err
	}

	if gotChatId != chatId {
		Logger.Infof("Wrong chatId: expected %v but got %v", chatId, gotChatId)
		return false, nil
	}

	if gotOwnerId != userPrincipalDto.UserId {
		Logger.Infof("Wrong ownerId: expected %v but got %v", userPrincipalDto.UserId, gotOwnerId)
		return false, nil
	}
	return true, nil
}


func (h *FilesHandler) DownloadHandler(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	bucketName, err := EnsureAndGetFilesBucket(h.minio)
	if err != nil {
		return err
	}

	// check user belongs to chat
	fileId := c.QueryParam("file")
	objectInfo, err := h.minio.StatObject(context.Background(), bucketName, fileId, minio.StatObjectOptions{})
	if err != nil {
		Logger.Errorf("Error during getting object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	chatId, _, fileName, err := deserializeMetadata(objectInfo.UserMetadata, false)
	if err != nil {
		Logger.Errorf("Error during deserializing object metadata %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	belongs, err := h.chatClient.CheckAccess(userPrincipalDto.UserId, chatId)
	if err != nil {
		Logger.Errorf("Error during checking user auth to chat %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !belongs {
		Logger.Errorf("User %v is not belongs to chat %v", userPrincipalDto.UserId, chatId)
		return c.NoContent(http.StatusUnauthorized)
	}
	// end check


	c.Response().Header().Set(echo.HeaderContentLength, strconv.FormatInt(objectInfo.Size, 10))
	c.Response().Header().Set(echo.HeaderContentType, objectInfo.ContentType)
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; Filename=\""+fileName+"\"")

	object, e := h.minio.GetObject(context.Background(), bucketName, fileId, minio.GetObjectOptions{})
	if e != nil {
		return c.JSON(http.StatusInternalServerError, &utils.H{"status": "fail"})
	}
	defer object.Close()

	return c.Stream(http.StatusOK, objectInfo.ContentType, object)
}

func (h *FilesHandler) SetPublic(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	bucketName, err := EnsureAndGetFilesBucket(h.minio)
	if err != nil {
		return err
	}

	// check user is owner
	fileId := c.QueryParam("file")
	objectInfo, err := h.minio.StatObject(context.Background(), bucketName, fileId, minio.StatObjectOptions{})
	if err != nil {
		Logger.Errorf("Error during getting object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	_, ownerId, _, err := deserializeMetadata(objectInfo.UserMetadata, false)
	if err != nil {
		Logger.Errorf("Error during deserializing object metadata %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if ownerId != userPrincipalDto.UserId {
		Logger.Errorf("User %v is not owner of file %v", userPrincipalDto.UserId, fileId)
		return c.NoContent(http.StatusUnauthorized)
	}
	// end check

	publicStr := c.QueryParam("public")
	public, err := utils.ParseBoolean(publicStr)
	if err != nil {
		Logger.Errorf("Error during deserializing request param %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	tagsMap := serializeTags(public)
	objectTags, err := tags.MapToObjectTags(tagsMap)
	if err != nil {
		Logger.Errorf("Error during mapping tags %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	err = h.minio.PutObjectTagging(context.Background(), bucketName, fileId, objectTags, minio.PutObjectTaggingOptions{})
	if err != nil {
		Logger.Errorf("Error during saving tags %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}

func (h *FilesHandler) PublicDownloadHandler(c echo.Context) error {
	bucketName, err := EnsureAndGetFilesBucket(h.minio)
	if err != nil {
		return err
	}

	// check file is public
	fileId := c.QueryParam("file")
	objectInfo, err := h.minio.StatObject(context.Background(), bucketName, fileId, minio.StatObjectOptions{})
	if err != nil {
		Logger.Errorf("Error during getting object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	_, _, fileName, err := deserializeMetadata(objectInfo.UserMetadata, false)
	if err != nil {
		Logger.Errorf("Error during deserializing object metadata %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	isPublic, err := deserializeTags(objectInfo.UserTags)
	if err != nil {
		Logger.Errorf("Error during deserializing object tags %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if !isPublic {
		Logger.Errorf("File %v is not public", fileId)
		return c.NoContent(http.StatusUnauthorized)
	}
	// end check


	c.Response().Header().Set(echo.HeaderContentLength, strconv.FormatInt(objectInfo.Size, 10))
	c.Response().Header().Set(echo.HeaderContentType, objectInfo.ContentType)
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; Filename=\""+fileName+"\"")

	object, e := h.minio.GetObject(context.Background(), bucketName, fileId, minio.GetObjectOptions{})
	if e != nil {
		return c.JSON(http.StatusInternalServerError, &utils.H{"status": "fail"})
	}
	defer object.Close()

	return c.Stream(http.StatusOK, objectInfo.ContentType, object)
}

func (h *FilesHandler) LimitsHandler(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}
	if ok, err := h.chatClient.CheckAccess(userPrincipalDto.UserId, chatId); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	bucketName, err := EnsureAndGetFilesBucket(h.minio)
	if err != nil {
		return err
	}

	max, e := h.getMaxAllowedConsumption(userPrincipalDto)
	if e != nil {
		return e
	}
	consumption := h.calcUserFilesConsumption(bucketName)

	desiredSize, err := utils.ParseInt64(c.QueryParam("desiredSize"))
	if err != nil {
		return err
	}
	available := max - consumption
	if desiredSize > available {
		return c.JSON(http.StatusOK, &utils.H{"status": "oversized", "used": h.calcUserFilesConsumption(bucketName), "available": available})
	} else {
		return c.JSON(http.StatusOK, &utils.H{"status": "ok", "used": h.calcUserFilesConsumption(bucketName), "available": available})
	}
}
