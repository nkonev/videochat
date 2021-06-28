package handlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
	"mime/multipart"
	"net/http"
	"net/url"
	"nkonev.name/storage/auth"
	"nkonev.name/storage/client"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/utils"
	"strconv"
	"strings"
	"syscall"
)

type FilesHandler struct {
	serverUrl          string
	minio              *minio.Client
	chatClient *client.ChatAccessClient
}

type RenameDto struct {
	Newname string `json:"newname"`
}

const filesMultipartKey = "files";

type FileInfoDto struct {
	Id        string `json:"id"`
	Filename  string `json:"filename"`
	Url       string `json:"url"`
	PublicUrl string `json:"publicUrl"`
	Size      int64  `json:"size"`
	CanRemove bool   `json:"canRemove"`
}

const filenameKey = "filename"
const ownerIdKey = "ownerid"
const chatIdKey = "chatid"

func NewFilesHandler(
	minio *minio.Client,
	chatClient *client.ChatAccessClient,
) *FilesHandler {
	return &FilesHandler{
		minio:              minio,
		serverUrl:          viper.GetString("server.url"),
		chatClient: chatClient,
	}
}

func serializeTags(file *multipart.FileHeader, userPrincipalDto *auth.AuthResult, chatId int64) map[string]string {
	var userMetadata = map[string]string{}
	userMetadata[filenameKey] = file.Filename
	userMetadata[ownerIdKey] = utils.Int64ToString(userPrincipalDto.UserId)
	userMetadata[chatIdKey] = utils.Int64ToString(chatId)
	return userMetadata
}

func deserializeTags(userMetadata minio.StringMap, hasAmzPrefix bool) (int64, int64, string, error) {
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

		var userMetadata = serializeTags(file, userPrincipalDto, chatId)

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

func (h *FilesHandler) ListChatFilesHandler(c echo.Context) error {
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

	list, err2 := h.getListFilesInFileItem(userPrincipalDto.UserId, bucket, filenameChatPrefix, chatId)
	if err2 != nil {
		return err2
	}

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "files": list})
}

func (h *FilesHandler) getListFilesInFileItem(behalfUserId int64, bucket, filenameChatPrefix string, chatId int64) ([]FileInfoDto, error) {
	var objects <-chan minio.ObjectInfo = h.minio.ListObjects(context.Background(), bucket, minio.ListObjectsOptions{
		WithMetadata: true,
		Prefix:       filenameChatPrefix,
		Recursive: true,
	})

	var list []FileInfoDto = make([]FileInfoDto, 0)
	for objInfo := range objects {
		Logger.Debugf("Object '%v'", objInfo.Key)

		downloadUrl, err := h.getChatPrivateUrlFromObject(objInfo, chatId)
		if err != nil {
			Logger.Errorf("Error get private url: %v", err)
			return nil, err
		}
		metadata := objInfo.UserMetadata

		_, fileOwnerId, fileName, err := deserializeTags(metadata, true)
		if err != nil {
			Logger.Errorf("Error get file name url: %v, skipping", err)
			continue
		}

		info := FileInfoDto{Id: objInfo.Key, Filename: fileName, Url: *downloadUrl, Size: objInfo.Size, CanRemove: fileOwnerId == behalfUserId}
		list = append(list, info)
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
	fileItemUuid := c.Param("fileItemUuid")
	if fileItemUuid == "" {
		Logger.Errorf("fileItemUuid is required")
		return c.NoContent(http.StatusBadRequest)
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

	var filenameChatPrefix string = fmt.Sprintf("chat/%v/%v/", chatId, fileItemUuid)

	err = h.minio.RemoveObject(context.Background(), bucketName, bindTo.Id, minio.RemoveObjectOptions{})
	if err != nil {
		Logger.Errorf("Error during removing object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	list, err2 := h.getListFilesInFileItem(userPrincipalDto.UserId, bucketName, filenameChatPrefix, chatId)
	if err2 != nil {
		return err2
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
		b, err2 := h.checkFileBelongsToUser(objInfo, chatId, userPrincipalDto, true)
		if err2 != nil {
			return false, err2
		}
		if !b {
			return false, nil
		}
	}
	return true, nil
}

func (h *FilesHandler) checkFileBelongsToUser(objInfo minio.ObjectInfo, chatId int64, userPrincipalDto *auth.AuthResult, hasAmzPrefix bool) (bool, error) {
	gotChatId, gotOwnerId, _, err := deserializeTags(objInfo.UserMetadata, hasAmzPrefix)
	if err != nil {
		Logger.Errorf("Error deserializeTags: %v", err)
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
	chatId, _, fileName, err := deserializeTags(objectInfo.UserMetadata, false)
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
