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
	"time"
)

type FilesHandler struct {
	minio        *minio.Client
	restClient   *client.RestClient
	minioConfig  *utils.MinioConfig
	filesService *FilesService
}

type RenameDto struct {
	Newname string `json:"newname"`
}

const filesMultipartKey = "files"
const UrlStorageGetFile = "/storage/public/download"
const UrlStorageGetFilePublicExternal = "/public/download"

type FileInfoDto struct {
	Id           string    `json:"id"`
	Filename     string    `json:"filename"`
	Url          string    `json:"url"`
	PublicUrl    *string   `json:"publicUrl"`
	Size         int64     `json:"size"`
	CanDelete    bool      `json:"canDelete"`
	CanEdit      bool      `json:"canEdit"`
	CanShare     bool      `json:"canShare"`
	LastModified time.Time `json:"lastModified"`
	OwnerId      int64     `json:"ownerId"`
	Owner        *dto.User `json:"owner"`
}

const publicKey = "public"

func NewFilesHandler(
	minio *minio.Client,
	restClient *client.RestClient,
	minioConfig *utils.MinioConfig,
	filesService *FilesService,
) *FilesHandler {
	return &FilesHandler{
		minio:        minio,
		restClient:   restClient,
		minioConfig:  minioConfig,
		filesService: filesService,
	}
}

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

func serializeTags(public bool) map[string]string {
	var userTags = map[string]string{}
	userTags[publicKey] = fmt.Sprintf("%v", public)
	return userTags
}

func deserializeTags(tagging *tags.Tags) (bool, error) {
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

func (h *FilesHandler) UploadHandler(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}
	if ok, err := h.restClient.CheckAccess(userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	bucketName := h.minioConfig.Files

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
		userLimitOk, _, _, err := checkUserLimit(h.minio, bucketName, userPrincipalDto, file.Size)
		if err != nil {
			return err
		}
		if !userLimitOk {
			return c.JSON(http.StatusRequestEntityTooLarge, &utils.H{"status": "fail"})
		}

		contentType := file.Header.Get("Content-Type")
		dotExt := getDotExtension(file)

		GetLogEntry(c.Request().Context()).Debugf("Determined content type: %v", contentType)

		src, err := file.Open()
		if err != nil {
			return err
		}
		defer src.Close()

		fileUuid := uuid.New().String()
		filename := fmt.Sprintf("chat/%v/%v/%v%v", chatId, fileItemUuid, fileUuid, dotExt)

		var userMetadata = serializeMetadata(file, userPrincipalDto, chatId)

		if _, err := h.minio.PutObject(context.Background(), bucketName, filename, src, file.Size, minio.PutObjectOptions{ContentType: contentType, UserMetadata: userMetadata}); err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during upload object: %v", err)
			return err
		}
	}

	// get count
	count := h.getCountFilesInFileItem(bucketName, filenameChatPrefix)

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "fileItemUuid": fileItemUuid, "count": count})
}

type ReplaceTextFileDto struct {
	Id          string `json:"id"` // file id
	Text        string `json:"text"`
	ContentType string `json:"contentType"`
	Filename    string `json:"filename"`
}

func (h *FilesHandler) ReplaceHandler(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}
	if ok, err := h.restClient.CheckAccess(userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	var bindTo = new(ReplaceTextFileDto)
	if err := c.Bind(bindTo); err != nil {
		GetLogEntry(c.Request().Context()).Warnf("Error during binding to dto %v", err)
		return err
	}

	bucketName := h.minioConfig.Files

	fileItemUuid := getFileItemUuid(bindTo.Id)

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

	fileSize := int64(len(bindTo.Text))
	userLimitOk, _, _, err := checkUserLimit(h.minio, bucketName, userPrincipalDto, fileSize)
	if err != nil {
		return err
	}
	if !userLimitOk {
		return c.JSON(http.StatusRequestEntityTooLarge, &utils.H{"status": "fail"})
	}

	contentType := bindTo.ContentType
	dotExt := GetDotExtensionStr(bindTo.Filename)

	GetLogEntry(c.Request().Context()).Debugf("Determined content type: %v", contentType)

	src := strings.NewReader(bindTo.Text)

	fileUuid := getFileId(bindTo.Id)
	filename := fmt.Sprintf("chat/%v/%v/%v%v", chatId, fileItemUuid, fileUuid, dotExt)

	var userMetadata = serializeMetadataByArgs(bindTo.Filename, userPrincipalDto, chatId)

	if _, err := h.minio.PutObject(context.Background(), bucketName, filename, src, fileSize, minio.PutObjectOptions{ContentType: contentType, UserMetadata: userMetadata}); err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during upload object: %v", err)
		return err
	}

	return c.NoContent(http.StatusOK)
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

func getFileItemUuid(fileId string) string {
	split := strings.Split(fileId, "/")
	return split[2]
}

func getFileId(fileId string) string {
	split := strings.Split(fileId, "/")
	filenameWithExt := split[3]
	splitFn := strings.Split(filenameWithExt, ".")
	return splitFn[0]
}

func (h *FilesHandler) ListHandler(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}
	if ok, err := h.restClient.CheckAccess(userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	filesPage := utils.FixPageString(c.QueryParam("page"))
	filesSize := utils.FixSizeString(c.QueryParam("size"))
	filesOffset := utils.GetOffset(filesPage, filesSize)

	fileItemUuid := c.QueryParam("fileItemUuid")

	bucketName := h.minioConfig.Files

	GetLogEntry(c.Request().Context()).Debugf("Listing bucket '%v':", bucketName)

	var filenameChatPrefix string
	if fileItemUuid == "" {
		filenameChatPrefix = fmt.Sprintf("chat/%v/", chatId)
	} else {
		filenameChatPrefix = fmt.Sprintf("chat/%v/%v/", chatId, fileItemUuid)
	}

	list, count, err := h.filesService.getListFilesInFileItem(userPrincipalDto.UserId, bucketName, filenameChatPrefix, chatId, c.Request().Context(), nil, true, filesSize, filesOffset)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "files": list, "count": count})
}

func getUsersRemotelyOrEmpty(userIdSet map[int64]bool, restClient *client.RestClient, c context.Context) map[int64]*dto.User {
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

func (h *FilesService) getListFilesInFileItem(
	behalfUserId int64,
	bucket, filenameChatPrefix string,
	chatId int64,
	c context.Context,
	filter func(*minio.ObjectInfo) bool,
	requestOwners bool,
	size, offset int,
) ([]*FileInfoDto, int, error) {
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

	var list []*FileInfoDto = make([]*FileInfoDto, 0)
	var counter = 0
	var respCounter = 0

	for _, objInfo := range intermediateList {

		if counter >= offset {
			tagging, err := h.minio.GetObjectTagging(c, bucket, objInfo.Key, minio.GetObjectTaggingOptions{})
			if err != nil {
				GetLogEntry(c).Errorf("Error during getting tags %v", err)
				return nil, 0, err
			}

			info, err := h.getFileInfo(behalfUserId, objInfo, chatId, tagging, true)
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
		var users = getUsersRemotelyOrEmpty(participantIdSet, h.restClient, c)
		for _, fileDto := range list {
			user := users[fileDto.OwnerId]
			if user != nil {
				fileDto.Owner = user
			}
		}
	}

	return list, count, nil
}

func (h *FilesService) getFileInfo(behalfUserId int64, objInfo minio.ObjectInfo, chatId int64, tagging *tags.Tags, hasAmzPrefix bool) (*FileInfoDto, error) {
	downloadUrl, err := h.getChatPrivateUrlFromObject(objInfo, chatId)
	if err != nil {
		Logger.Errorf("Error get private url: %v", err)
		return nil, err
	}
	metadata := objInfo.UserMetadata

	_, fileOwnerId, fileName, err := deserializeMetadata(metadata, hasAmzPrefix)
	if err != nil {
		Logger.Errorf("Error get metadata: %v", err)
		return nil, err
	}

	public, err := deserializeTags(tagging)
	if err != nil {
		Logger.Errorf("Error get tags: %v", err)
		return nil, err
	}

	publicUrl, err := h.getPublicUrl(public, objInfo.Key)
	if err != nil {
		Logger.Errorf("Error get public url: %v", err)
		return nil, err
	}

	info := &FileInfoDto{
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
	query.Add("file", fileName)
	downloadUrl.RawQuery = query.Encode()
	str := downloadUrl.String()
	return &str, nil
}

func (h *FilesService) getBaseUrlForDownload() string {
	return viper.GetString("server.contextPath") + "/storage"
}

func (h *FilesService) getChatPrivateUrlFromObject(objInfo minio.ObjectInfo, chatId int64) (*string, error) {
	downloadUrl, err := url.Parse(h.getBaseUrlForDownload() + "/download")
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
	Id string `json:"id"` // file id
}

func (h *FilesHandler) DeleteHandler(c echo.Context) error {
	var bindTo = new(DeleteObjectDto)
	if err := c.Bind(bindTo); err != nil {
		GetLogEntry(c.Request().Context()).Warnf("Error during binding to dto %v", err)
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}
	if ok, err := h.restClient.CheckAccess(userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	bucketName := h.minioConfig.Files

	// check this fileItem belongs to user
	objectInfo, err := h.minio.StatObject(context.Background(), bucketName, bindTo.Id, minio.StatObjectOptions{})
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	belongs, err := h.checkFileBelongsToUser(objectInfo, chatId, userPrincipalDto, false)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during checking belong object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !belongs {
		GetLogEntry(c.Request().Context()).Errorf("Object '%v' is not belongs to user %v", objectInfo.Key, userPrincipalDto.UserId)
		return c.NoContent(http.StatusUnauthorized)
	}
	// end check

	formerFileItemUuid := getFileItemUuid(objectInfo.Key)

	err = h.minio.RemoveObject(context.Background(), bucketName, objectInfo.Key, minio.RemoveObjectOptions{})
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during removing object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	filesPage := utils.FixPageString(c.QueryParam("page"))
	filesSize := utils.FixSizeString(c.QueryParam("size"))
	filesOffset := utils.GetOffset(filesPage, filesSize)

	// this fileItemUuid used for display list in response
	fileItemUuid := c.QueryParam("fileItemUuid")
	var filenameChatPrefix string
	if fileItemUuid == "" {
		filenameChatPrefix = fmt.Sprintf("chat/%v/", chatId)
	} else {
		filenameChatPrefix = fmt.Sprintf("chat/%v/%v/", chatId, fileItemUuid)
	}

	list, count, err := h.filesService.getListFilesInFileItem(userPrincipalDto.UserId, bucketName, filenameChatPrefix, chatId, c.Request().Context(), nil, true, filesSize, filesOffset)
	if err != nil {
		return err
	}

	// this fileItemUuid used for remove orphans
	if h.countFilesUnderFileUuid(chatId, formerFileItemUuid, bucketName) == 0 {
		h.restClient.RemoveFileItem(chatId, formerFileItemUuid, userPrincipalDto.UserId, c.Request().Context())
	}

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "files": list, "count": count})
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
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	bucketName := h.minioConfig.Files

	// check user belongs to chat
	fileId := c.QueryParam("file")
	objectInfo, err := h.minio.StatObject(context.Background(), bucketName, fileId, minio.StatObjectOptions{})
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	chatId, _, fileName, err := deserializeMetadata(objectInfo.UserMetadata, false)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during deserializing object metadata %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	belongs, err := h.restClient.CheckAccess(userPrincipalDto.UserId, chatId, c.Request().Context())
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during checking user auth to chat %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !belongs {
		GetLogEntry(c.Request().Context()).Errorf("User %v is not belongs to chat %v", userPrincipalDto.UserId, chatId)
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

type PublishRequest struct {
	Public bool   `json:"public"`
	Id     string `json:"id"`
}

func (h *FilesHandler) SetPublic(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	bucketName := h.minioConfig.Files

	var bindTo = new(PublishRequest)
	if err := c.Bind(bindTo); err != nil {
		GetLogEntry(c.Request().Context()).Warnf("Error during binding to dto %v", err)
		return err
	}

	// check user is owner
	fileId := bindTo.Id
	objectInfo, err := h.minio.StatObject(context.Background(), bucketName, fileId, minio.StatObjectOptions{})
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	chatId, ownerId, _, err := deserializeMetadata(objectInfo.UserMetadata, false)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during deserializing object metadata %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if ownerId != userPrincipalDto.UserId {
		GetLogEntry(c.Request().Context()).Errorf("User %v is not owner of file %v", userPrincipalDto.UserId, fileId)
		return c.NoContent(http.StatusUnauthorized)
	}
	// end check

	tagsMap := serializeTags(bindTo.Public)
	objectTags, err := tags.MapToObjectTags(tagsMap)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during mapping tags %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	err = h.minio.PutObjectTagging(context.Background(), bucketName, fileId, objectTags, minio.PutObjectTaggingOptions{})
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during saving tags %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	objectInfo, err = h.minio.StatObject(context.Background(), bucketName, fileId, minio.StatObjectOptions{})
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during stat %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	tagging, err := h.minio.GetObjectTagging(context.Background(), bucketName, fileId, minio.GetObjectTaggingOptions{})
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting tags %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	info, err := h.filesService.getFileInfo(userPrincipalDto.UserId, objectInfo, chatId, tagging, false)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getFileInfo %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	var participantIdSet = map[int64]bool{}
	participantIdSet[userPrincipalDto.UserId] = true
	var users = getUsersRemotelyOrEmpty(participantIdSet, h.restClient, c.Request().Context())
	user, ok := users[userPrincipalDto.UserId]
	if ok {
		info.Owner = user
	}

	return c.JSON(http.StatusOK, info)
}

type CountResponse struct {
	Count int `json:"count"`
}

func (h *FilesHandler) CountHandler(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	bucketName := h.minioConfig.Files

	// check user belongs to chat
	fileItemUuid := c.Param("fileItemUuid")
	chatIdString := c.Param("chatId")
	chatId, err := utils.ParseInt64(chatIdString)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during parsing chatId %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	belongs, err := h.restClient.CheckAccess(userPrincipalDto.UserId, chatId, c.Request().Context())
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during checking user auth to chat %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !belongs {
		GetLogEntry(c.Request().Context()).Errorf("User %v is not belongs to chat %v", userPrincipalDto.UserId, chatId)
		return c.NoContent(http.StatusUnauthorized)
	}
	// end check

	counter := h.countFilesUnderFileUuid(chatId, fileItemUuid, bucketName)

	var countDto = CountResponse{
		Count: counter,
	}

	return c.JSON(http.StatusOK, countDto)
}

func (h *FilesHandler) countFilesUnderFileUuid(chatId int64, fileItemUuid string, bucketName string) int {
	var filenameChatPrefix = fmt.Sprintf("chat/%v/%v/", chatId, fileItemUuid)
	var objects <-chan minio.ObjectInfo = h.minio.ListObjects(context.Background(), bucketName, minio.ListObjectsOptions{
		WithMetadata: false,
		Prefix:       filenameChatPrefix,
		Recursive:    true,
	})

	var counter = 0
	for _ = range objects {
		counter++
	}
	return counter
}

func (h *FilesHandler) PublicDownloadHandler(c echo.Context) error {
	bucketName := h.minioConfig.Files

	// check file is public
	fileId := c.QueryParam("file")
	objectInfo, err := h.minio.StatObject(context.Background(), bucketName, fileId, minio.StatObjectOptions{})
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	_, _, fileName, err := deserializeMetadata(objectInfo.UserMetadata, false)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during deserializing object metadata %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	tagging, err := h.minio.GetObjectTagging(context.Background(), bucketName, fileId, minio.GetObjectTaggingOptions{})
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during deserializing object tags %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	isPublic, err := deserializeTags(tagging)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during deserializing object tags %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if !isPublic {
		GetLogEntry(c.Request().Context()).Errorf("File %v is not public", fileId)
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
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}
	if ok, err := h.restClient.CheckAccess(userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	bucketName := h.minioConfig.Files

	desiredSize, err := utils.ParseInt64(c.QueryParam("desiredSize"))
	if err != nil {
		return err
	}
	ok, consumption, available, err := checkUserLimit(h.minio, bucketName, userPrincipalDto, desiredSize)
	if err != nil {
		return err
	}

	if !ok {
		return c.JSON(http.StatusOK, &utils.H{"status": "oversized", "used": consumption, "available": available})
	} else {
		return c.JSON(http.StatusOK, &utils.H{"status": "ok", "used": consumption, "available": available})
	}
}

type S3Response struct {
	AccessKey string            `json:"accessKey"`
	Secret    string            `json:"secret"`
	Region    string            `json:"region"`
	Endpoint  string            `json:"endpoint"`
	Bucket    string            `json:"bucket"`
	Metadata  map[string]string `json:"metadata"`
	Filepath  string            `json:"filepath"`
}

type S3Request struct {
	FileName string `json:"fileName"`
	ChatId   int64  `json:"chatId"`
	OwnerId  int64  `json:"ownerId"`
}

func (h *FilesHandler) S3Handler(c echo.Context) error {
	bindTo := new(S3Request)
	if err := c.Bind(bindTo); err != nil {
		GetLogEntry(c.Request().Context()).Warnf("Error during binding to dto %v", err)
		return err
	}

	endpoint := viper.GetString("minio.containerEndpoint")
	accessKeyID := viper.GetString("minio.accessKeyId")
	secretAccessKey := viper.GetString("minio.secretAccessKey")

	metadata := serializeMetadataSimple(bindTo.FileName, bindTo.OwnerId, bindTo.ChatId)

	fileItemUuid := uuid.New().String()
	fileUuid := uuid.New().String()
	dotExt := GetDotExtensionStr(bindTo.FileName)

	minioFilename := fmt.Sprintf("/chat/%v/%v/%v%v", bindTo.ChatId, fileItemUuid, fileUuid, dotExt)

	response := S3Response{
		AccessKey: accessKeyID,
		Secret:    secretAccessKey,
		Region:    viper.GetString("minio.location"),
		Endpoint:  endpoint,
		Bucket:    h.minioConfig.Files,
		Metadata:  metadata,
		Filepath:  minioFilename,
	}

	return c.JSON(http.StatusOK, response)
}
