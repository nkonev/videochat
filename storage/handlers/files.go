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
	"nkonev.name/storage/s3"
	"nkonev.name/storage/services"
	"nkonev.name/storage/utils"
	"strconv"
	"strings"
)

type FilesHandler struct {
	minio        *s3.InternalMinioClient
	restClient   *client.RestClient
	minioConfig  *utils.MinioConfig
	filesService *services.FilesService
}

type RenameDto struct {
	Newname string `json:"newname"`
}

const NotFoundImage = "/api/storage/assets/not_found.png"

func NewFilesHandler(
	minio *s3.InternalMinioClient,
	restClient *client.RestClient,
	minioConfig *utils.MinioConfig,
	filesService *services.FilesService,
) *FilesHandler {
	return &FilesHandler{
		minio:        minio,
		restClient:   restClient,
		minioConfig:  minioConfig,
		filesService: filesService,
	}
}

type EmbedDto struct {
	Url  string  `json:"url"`
	Type *string `json:"type"`
}

type UploadRequest struct {
	CorrelationId *string `json:"correlationId"`
	FileItemUuid  *string `json:"fileItemUuid"`
	FileSize      int64   `json:"fileSize"`
	FileName      string  `json:"fileName"`
}

type UploadResponse struct {
	Url string `json:"url"`
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
	if ok, err := h.restClient.CheckAccess(&userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	reqDto := new(UploadRequest)
	err = c.Bind(reqDto)
	if err != nil {
		return err
	}

	bucketName := h.minioConfig.Files

	chatFileItemUuid := uuid.New().String()

	fileItemUuidString := reqDto.FileItemUuid
	if fileItemUuidString != nil && *fileItemUuidString != "" {
		chatFileItemUuid = *fileItemUuidString
	}

	// check this fileItem belongs to user
	filenameChatPrefix := fmt.Sprintf("chat/%v/%v/", chatId, chatFileItemUuid)
	belongs, err := h.checkFileItemBelongsToUser(filenameChatPrefix, c, chatId, bucketName, userPrincipalDto)
	if err != nil {
		return err
	}
	if !belongs {
		return c.NoContent(http.StatusUnauthorized)
	}
	// end check

	filename := services.GenerateFilename(reqDto.FileName, chatFileItemUuid, chatId)

	// check that this file does not exist
	_, err = h.minio.StatObject(context.Background(), bucketName, filename, minio.StatObjectOptions{})
	if err == nil {
		GetLogEntry(c.Request().Context()).Errorf("Already exists: %v", filename)
		return c.NoContent(http.StatusConflict)
	}

	// check enough size taking on account free disk space probe (see LimitsHandler)
	ok, consumption, available, err := checkUserLimit(h.minio, bucketName, userPrincipalDto, reqDto.FileSize)
	if err != nil {
		return err
	}
	if !ok {
		return c.JSON(http.StatusOK, &utils.H{"status": "oversized", "used": consumption, "available": available})
	}

	uploadDuration := viper.GetDuration("minio.publicUploadTtl")
	var urlVals = url.Values{}

	services.SerializeMetadataAndStore(&urlVals, userPrincipalDto.UserId, chatId, reqDto.CorrelationId)

	u, err := h.minio.Presign(c.Request().Context(), "PUT", bucketName, filename, uploadDuration, urlVals)
	if err != nil {
		Logger.Errorf("Error during getting downlad url %v", err)
		return err
	}

	err = services.ChangeMinioUrl(u)
	if err != nil {
		return err
	}

	existingCount := h.getCountFilesInFileItem(bucketName, filenameChatPrefix)

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "url": fmt.Sprintf("%v", u), "fileItemUuid": chatFileItemUuid, "existingCount": existingCount})
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
	if ok, err := h.restClient.CheckAccess(&userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
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

	GetLogEntry(c.Request().Context()).Debugf("Determined content type: %v", contentType)

	src := strings.NewReader(bindTo.Text)

	chatFileItemUuid := getFileId(bindTo.Id)
	filename := services.GenerateFilename(bindTo.Filename, chatFileItemUuid, chatId)

	var userMetadata = services.SerializeMetadataSimple(userPrincipalDto.UserId, chatId, nil)

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
	if ok, err := h.restClient.CheckAccess(&userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	filesPage := utils.FixPageString(c.QueryParam("page"))
	filesSize := utils.FixSizeString(c.QueryParam("size"))
	filesOffset := utils.GetOffset(filesPage, filesSize)

	fileItemUuid := c.QueryParam("fileItemUuid")

	searchString := c.QueryParam("searchString")
	searchString = strings.TrimSpace(searchString)
	searchString = strings.ToLower(searchString)

	bucketName := h.minioConfig.Files

	GetLogEntry(c.Request().Context()).Debugf("Listing bucket '%v':", bucketName)

	var filenameChatPrefix string
	if fileItemUuid == "" {
		filenameChatPrefix = fmt.Sprintf("chat/%v/", chatId)
	} else {
		filenameChatPrefix = fmt.Sprintf("chat/%v/%v/", chatId, fileItemUuid)
	}

	var filter func(info *minio.ObjectInfo) bool = nil
	if searchString != "" {
		filter = func(info *minio.ObjectInfo) bool {
			normalizedFileName := strings.ToLower(services.ReadFilename(info.Key))
			return strings.Contains(normalizedFileName, searchString)
		}
	}

	list, count, err := h.filesService.GetListFilesInFileItem(userPrincipalDto.UserId, bucketName, filenameChatPrefix, chatId, c.Request().Context(), filter, true, filesSize, filesOffset)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "files": list, "count": count})
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
	if ok, err := h.restClient.CheckAccess(&userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
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

	list, count, err := h.filesService.GetListFilesInFileItem(userPrincipalDto.UserId, bucketName, filenameChatPrefix, chatId, c.Request().Context(), nil, true, filesSize, filesOffset)
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
	gotChatId, gotOwnerId, _, err := services.DeserializeMetadata(objInfo.UserMetadata, hasAmzPrefix)
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
	chatId, ownerId, _, err := services.DeserializeMetadata(objectInfo.UserMetadata, false)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during deserializing object metadata %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if ownerId != userPrincipalDto.UserId {
		GetLogEntry(c.Request().Context()).Errorf("User %v is not owner of file %v", userPrincipalDto.UserId, fileId)
		return c.NoContent(http.StatusUnauthorized)
	}
	// end check

	tagsMap := services.SerializeTags(bindTo.Public)
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

	info, err := h.filesService.GetFileInfo(userPrincipalDto.UserId, objectInfo, chatId, tagging, false)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getFileInfo %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	var participantIdSet = map[int64]bool{}
	participantIdSet[userPrincipalDto.UserId] = true
	var users = services.GetUsersRemotelyOrEmpty(participantIdSet, h.restClient, c.Request().Context())
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

	belongs, err := h.restClient.CheckAccess(&userPrincipalDto.UserId, chatId, c.Request().Context())
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
	if ok, err := h.restClient.CheckAccess(&userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
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

	endpoint := viper.GetString("minio.interContainerUrl")
	accessKeyID := viper.GetString("minio.accessKeyId")
	secretAccessKey := viper.GetString("minio.secretAccessKey")

	metadata := services.SerializeMetadataSimple(bindTo.OwnerId, bindTo.ChatId, nil)

	chatFileItemUuid := uuid.New().String()

	filename := services.GenerateFilename(bindTo.FileName, chatFileItemUuid, bindTo.ChatId)

	response := S3Response{
		AccessKey: accessKeyID,
		Secret:    secretAccessKey,
		Region:    viper.GetString("minio.location"),
		Endpoint:  endpoint,
		Bucket:    h.minioConfig.Files,
		Metadata:  metadata,
		Filepath:  filename,
	}

	return c.JSON(http.StatusOK, response)
}

type MediaDto struct {
	Id         string  `json:"id"`
	Filename   string  `json:"filename"`
	Url        string  `json:"url"`
	PreviewUrl *string `json:"previewUrl"`
}

func (h *FilesHandler) ListCandidatesForEmbed(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	filesPage := utils.FixPageString(c.QueryParam("page"))
	filesSize := utils.FixSizeString(c.QueryParam("size"))
	filesOffset := utils.GetOffset(filesPage, filesSize)

	requestedMediaType := c.QueryParam("type")

	bucketName := h.minioConfig.Files

	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}
	belongs, err := h.restClient.CheckAccess(&userPrincipalDto.UserId, chatId, c.Request().Context())
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during checking user auth to chat %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !belongs {
		GetLogEntry(c.Request().Context()).Errorf("User %v is not belongs to chat %v", userPrincipalDto.UserId, chatId)
		return c.NoContent(http.StatusUnauthorized)
	}

	var filenameChatPrefix string = fmt.Sprintf("chat/%v/", chatId)

	filter := func(info *minio.ObjectInfo) bool {
		switch requestedMediaType {
		case services.Media_image:
			return utils.IsImage(info.Key)
		case services.Media_video:
			return utils.IsVideo(info.Key)
		default:
			return false
		}
	}

	items, count, err := h.filesService.GetListFilesInFileItem(userPrincipalDto.UserId, bucketName, filenameChatPrefix, chatId, c.Request().Context(), filter, false, filesSize, filesOffset)
	if err != nil {
		return err
	}

	var list []*MediaDto = make([]*MediaDto, 0)

	for _, item := range items {
		list = append(list, convert(item))
	}

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "files": list, "count": count})
}

func convert(item *dto.FileInfoDto) *MediaDto {
	if item == nil {
		return nil
	}

	return &MediaDto{
		Id:         item.Id,
		Filename:   item.Filename,
		Url:        item.Url,
		PreviewUrl: item.PreviewUrl,
	}
}

func (h *FilesHandler) PreviewDownloadHandler(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	bucketName := h.minioConfig.FilesPreview

	// check user belongs to chat
	fileId := c.QueryParam(utils.FileParam)
	objectInfo, err := h.minio.StatObject(context.Background(), bucketName, fileId, minio.StatObjectOptions{})
	if err != nil {
		if errTyped, ok := err.(minio.ErrorResponse); ok {
			if errTyped.Code == "NoSuchKey" {
				return c.Redirect(http.StatusTemporaryRedirect, NotFoundImage)
			}
		}
		GetLogEntry(c.Request().Context()).Errorf("Error during getting object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	chatId, err := utils.ParseChatId(fileId)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during deserializing object metadata %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	belongs, err := h.restClient.CheckAccess(&userPrincipalDto.UserId, chatId, c.Request().Context())
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during checking user auth to chat %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !belongs {
		GetLogEntry(c.Request().Context()).Errorf("User %v is not belongs to chat %v", userPrincipalDto.UserId, chatId)
		return c.NoContent(http.StatusUnauthorized)
	}
	// end check

	object, e := h.minio.GetObject(context.Background(), bucketName, fileId, minio.GetObjectOptions{})
	if e != nil {
		return c.JSON(http.StatusInternalServerError, &utils.H{"status": "fail"})
	}
	defer object.Close()

	c.Response().Header().Set(echo.HeaderContentLength, strconv.FormatInt(objectInfo.Size, 10))
	c.Response().Header().Set(echo.HeaderContentType, objectInfo.ContentType)

	return c.Stream(http.StatusOK, objectInfo.ContentType, object)
}

func (h *FilesHandler) DownloadHandler(c echo.Context) error {
	var userId *int64
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if ok {
		userId = &userPrincipalDto.UserId
	}

	bucketName := h.minioConfig.Files

	// check user belongs to chat
	fileId := c.QueryParam(utils.FileParam)
	objectInfo, err := h.minio.StatObject(context.Background(), bucketName, fileId, minio.StatObjectOptions{})
	if err != nil {
		if errTyped, ok := err.(minio.ErrorResponse); ok {
			if errTyped.Code == "NoSuchKey" {
				return c.Redirect(http.StatusTemporaryRedirect, NotFoundImage)
			}
		}
		GetLogEntry(c.Request().Context()).Errorf("Error during getting object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	chatId, _, _, err := services.DeserializeMetadata(objectInfo.UserMetadata, false)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during deserializing object metadata %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	belongs, err := h.restClient.CheckAccess(userId, chatId, c.Request().Context())
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during checking user auth to chat %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !belongs {
		GetLogEntry(c.Request().Context()).Errorf("User %v is not belongs to chat %v", userId, chatId)
		return c.NoContent(http.StatusUnauthorized)
	}
	// end check

	// send redirect to presigned
	downloadUrl, err := h.filesService.GetTemporaryDownloadUrl(objectInfo.Key)
	if err != nil {
		return err
	}

	return c.Redirect(http.StatusTemporaryRedirect, downloadUrl)
}

func (h *FilesHandler) PublicDownloadHandler(c echo.Context) error {
	bucketName := h.minioConfig.Files

	// check file is public
	fileId := c.QueryParam(utils.FileParam)
	objectInfo, err := h.minio.StatObject(context.Background(), bucketName, fileId, minio.StatObjectOptions{})
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	tagging, err := h.minio.GetObjectTagging(context.Background(), bucketName, fileId, minio.GetObjectTaggingOptions{})
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during deserializing object tags %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	isPublic, err := services.DeserializeTags(tagging)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during deserializing object tags %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if !isPublic {
		GetLogEntry(c.Request().Context()).Errorf("File %v is not public", fileId)
		return c.NoContent(http.StatusUnauthorized)
	}
	// end check

	// send redirect to presigned
	downloadUrl, err := h.filesService.GetTemporaryDownloadUrl(objectInfo.Key)
	if err != nil {
		return err
	}

	return c.Redirect(http.StatusTemporaryRedirect, downloadUrl)
}
