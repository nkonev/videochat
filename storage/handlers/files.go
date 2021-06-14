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

func deserializeTags(userMetadata minio.StringMap) (int64, int64, string, error) {
	filename, ok := userMetadata["X-Amz-Meta-"+strings.Title(filenameKey)]
	if ! ok {
		return 0, 0, "", errors.New("Unable to get filename")
	}
	ownerIdString, ok := userMetadata["X-Amz-Meta-"+strings.Title(ownerIdKey)]
	if ! ok {
		return 0, 0, "", errors.New("Unable to get owner id")
	}
	ownerId, err := utils.ParseInt64(ownerIdString)
	if err != nil {
		return 0, 0, "", err
	}

	chatIdString, ok := userMetadata["X-Amz-Meta-"+strings.Title(chatIdKey)]
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

	form, err := c.MultipartForm()
	if err != nil {
		return err
	}
	files := form.File[filesMultipartKey]

	for _, file := range files {
		bucketName, err := EnsureAndGetFilesBucket(h.minio)
		if err != nil {
			return err
		}

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
		filename := fmt.Sprintf("chat/%v/%v%v", chatId, fileUuid, dotExt)

		var userMetadata = serializeTags(file, userPrincipalDto, chatId)

		if _, err := h.minio.PutObject(context.Background(), bucketName, filename, src, file.Size, minio.PutObjectOptions{ContentType: contentType, UserMetadata: userMetadata}); err != nil {
			Logger.Errorf("Error during upload object: %v", err)
			return err
		}
	}

	return c.JSON(http.StatusOK, &utils.H{"status": "ok"})
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

	bucket, err := EnsureAndGetFilesBucket(h.minio)
	if err != nil {
		return err
	}

	Logger.Debugf("Listing bucket '%v':", bucket)

	filenameChatPrefix := fmt.Sprintf("chat/%v/", chatId)

	var objects <-chan minio.ObjectInfo = h.minio.ListObjects(context.Background(), bucket, minio.ListObjectsOptions{
		WithMetadata: true,
		Prefix:       filenameChatPrefix,
	})

	var list []FileInfoDto = make([]FileInfoDto, 0)
	for objInfo := range objects {
		Logger.Debugf("Object '%v'", objInfo.Key)

		downloadUrl, err := h.getChatPrivateUrlFromObject(objInfo, chatId)
		if err != nil {
			Logger.Errorf("Error get private url: %v", err)
			return err
		}
		metadata := objInfo.UserMetadata

		_, _, fileName, err := deserializeTags(metadata)
		if err != nil {
			Logger.Errorf("Error get file name url: %v", err)
			fileName = objInfo.Key
		}

		info := FileInfoDto{Id: objInfo.Key, Filename: fileName, Url: *downloadUrl, Size: objInfo.Size}
		list = append(list, info)
	}

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "files": list})
}

func (h *FilesHandler) getChatPrivateUrlFromObject(objInfo minio.ObjectInfo, chatId int64) (*string, error) {
	downloadUrl, err := url.Parse(h.serverUrl)
	if err != nil {
		return nil, err
	}

	filenameChatPrefix := fmt.Sprintf("chat/%v/", chatId)

	downloadUrl.Path += filenameChatPrefix + objInfo.Key
	str := downloadUrl.String()
	return &str, nil
}
