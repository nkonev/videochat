package handlers

import (
	"context"
	"errors"
	"fmt"
	awsS3 "github.com/aws/aws-sdk-go/service/s3"
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
	"time"
	"unicode"
)

type FilesHandler struct {
	minio        *s3.InternalMinioClient
	awsS3        *awsS3.S3
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
	awsS3 *awsS3.S3,
	restClient *client.RestClient,
	minioConfig *utils.MinioConfig,
	filesService *services.FilesService,
) *FilesHandler {
	return &FilesHandler{
		minio:        minio,
		awsS3: 		  awsS3,
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
	ShouldAddDateToTheFilename bool `json:"shouldAddDateToTheFilename"`
}

type UploadResponse struct {
	Url string `json:"url"`
}

func nonLetterSplit(c rune) bool {
	return !unicode.IsLetter(c) && !unicode.IsNumber(c) && c != '.' && c != '-' && c != '+' && c != '_' && c != ' '
}

// output of this fun eventually goes to sanitizer in chat
func cleanFilename(input string, shouldAddDateToTheFilename bool) string {
	words := strings.FieldsFunc(input, nonLetterSplit)
	tmp := strings.Join(words, "")
	trimmedFilename := strings.TrimSpace(tmp)

	filenameParts := strings.Split(trimmedFilename, ".")

	newFileName := ""
	if len(filenameParts) == 2 && shouldAddDateToTheFilename {
		newFileName = filenameParts[0] + "_" + time.Now().Format("20060102150405") + "." + filenameParts[1]
	} else {
		newFileName = trimmedFilename
	}

	return newFileName
}

type PresignedUrl struct {
	Url        string `json:"url"`
	PartNumber int    `json:"partNumber"`
}

type FinishMultipartRequest struct {
	Key string `json:"key"`
	Parts []MultipartFinishingPart `json:"parts"`
	UploadId string `json:"uploadId"`
}

type MultipartFinishingPart struct {
	Etag        string `json:"etag"`
	PartNumber int64 `json:"partNumber"`
}

func (h *FilesHandler) checkFileDoesNorExists(ctx context.Context, bucketName, aKey string) error {
	_, err := h.minio.StatObject(ctx, bucketName, aKey, minio.StatObjectOptions{})
	if err == nil {
		return errors.New(fmt.Sprintf("Already exists: %v", aKey))
	}
	return nil
}

func (h *FilesHandler) InitMultipartUpload(c echo.Context) error {
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

	chatFileItemUuid := uuid.New().String() // it's kinda last resort, actually it should be set on frontend in TipTapEditor.vue preallocatedCandidateFileItemId

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

	filteredFilename := cleanFilename(reqDto.FileName, reqDto.ShouldAddDateToTheFilename)

	aKey := services.GetKey(filteredFilename, chatFileItemUuid, chatId)

	// check that this file does not exist
	err = h.checkFileDoesNorExists(c.Request().Context(), bucketName, aKey)
	if err != nil {
		GetLogEntry(c.Request().Context()).Infof(err.Error())
		return c.JSON(http.StatusConflict, &utils.H{"status": "error", "message": err.Error()})
	}

	// check enough size taking on account free disk space probe (see LimitsHandler)
	ok, consumption, available, err := checkUserLimit(c.Request().Context(), h.minio, bucketName, userPrincipalDto, reqDto.FileSize)
	if err != nil {
		return err
	}
	if !ok {
		return c.JSON(http.StatusOK, &utils.H{"status": "oversized", "used": consumption, "available": available})
	}

	metadata := services.SerializeMetadataSimple(userPrincipalDto.UserId, chatId, reqDto.CorrelationId, nil)

	expire := viper.GetDuration("minio.multipart.expire")
	expTime := time.Now().Add(expire)
	converted := convertMetadata(&metadata)
	mpu := awsS3.CreateMultipartUploadInput{
		Expires: &expTime,
		Bucket: &bucketName,
		Key: &aKey,
		Metadata: converted,
	}
	upload, err := h.awsS3.CreateMultipartUpload(&mpu)
	if err != nil {
		return err
	}

	uploadDuration := viper.GetDuration("minio.publicUploadTtl")

	chunkSize := viper.GetInt64("minio.multipart.chunkSize")
	chunksNum := int(reqDto.FileSize / chunkSize)
	if reqDto.FileSize % chunkSize != 0 {
		chunksNum++
	}

	presignedUrls := []PresignedUrl{}
	for i := 1; i <= chunksNum; i++ {
		var urlVals = url.Values{}
		urlVals.Set("partNumber", utils.IntToString(i))
		urlVals.Set("uploadId", *upload.UploadId)

		u, err := h.minio.Presign(c.Request().Context(), "PUT", bucketName, aKey, uploadDuration, urlVals)
		if err != nil {
			Logger.Errorf("Error during getting downlad url %v", err)
			return err
		}

		err = services.ChangeMinioUrl(u)
		if err != nil {
			return err
		}

		presignedUrls = append(presignedUrls, PresignedUrl{u.String(), i})
	}
	existingCount := h.getCountFilesInFileItem(bucketName, filenameChatPrefix)

	return c.JSON(http.StatusOK, &utils.H{
		"status": "ready",
		"uploadId": upload.UploadId,
		"presignedUrls": presignedUrls,
		"chunkSize": chunkSize,
		"fileItemUuid": chatFileItemUuid,
		"existingCount": existingCount,
		"newFileName": filteredFilename,
		"key": aKey,
		"chatId": chatId,
	})
}

func convertMetadata(urlValues *map[string]string) map[string]*string {
	res := map[string]*string{}
	for k, v := range *urlValues {
		k := k
		v := v
		res[k] = &v
	}
	return res
}

func (h *FilesHandler) FinishMultipartUpload(c echo.Context) error {
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

	reqDto := new(FinishMultipartRequest)
	err = c.Bind(reqDto)
	if err != nil {
		return err
	}

	arr := []*awsS3.CompletedPart{}

	for _, part := range reqDto.Parts {
		part := part
		arr = append(arr, &awsS3.CompletedPart{
			ETag: &part.Etag,
			PartNumber: &part.PartNumber,
		})
	}

	input := awsS3.CompleteMultipartUploadInput{
		Key: &reqDto.Key,
		Bucket: &bucketName,
		UploadId: &reqDto.UploadId,
		MultipartUpload: &awsS3.CompletedMultipartUpload{
			Parts: arr,
		},
	}
	_, err = h.awsS3.CompleteMultipartUpload(&input)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
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
	userLimitOk, _, _, err := checkUserLimit(c.Request().Context(), h.minio, bucketName, userPrincipalDto, fileSize)
	if err != nil {
		return err
	}
	if !userLimitOk {
		return c.JSON(http.StatusRequestEntityTooLarge, &utils.H{"status": "fail"})
	}

	contentType := bindTo.ContentType

	GetLogEntry(c.Request().Context()).Debugf("Determined content type: %v", contentType)

	src := strings.NewReader(bindTo.Text)

	aKey := services.GetKey(bindTo.Filename, fileItemUuid, chatId)

	var userMetadata = services.SerializeMetadataSimple(userPrincipalDto.UserId, chatId, nil, nil)

	if _, err := h.minio.PutObject(context.Background(), bucketName, aKey, src, fileSize, minio.PutObjectOptions{ContentType: contentType, UserMetadata: userMetadata}); err != nil {
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

func (h *FilesHandler) ListFileItemUuids(c echo.Context) error {
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

	bucketName := h.minioConfig.Files

	filenameChatPrefix := fmt.Sprintf("chat/%v/", chatId)

	list, count, err := h.filesService.GetListFilesItemUuids( bucketName, filenameChatPrefix, c.Request().Context(), filesSize, filesOffset)
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

	showResponse := utils.ParseBooleanOr(c.QueryParam("response"), false)

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

	err = h.minio.RemoveObject(context.Background(), bucketName, objectInfo.Key, minio.RemoveObjectOptions{})
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during removing object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if !showResponse {
		return c.NoContent(http.StatusOK)
	}

	return c.NoContent(http.StatusOK)
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

	showResponse := utils.ParseBooleanOr(c.QueryParam("response"), false)

	// check user is owner
	fileId := bindTo.Id
	objectInfo, err := h.minio.StatObject(context.Background(), bucketName, fileId, minio.StatObjectOptions{})
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	_, ownerId, _, err := services.DeserializeMetadata(objectInfo.UserMetadata, false)
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

	if !showResponse {
		return c.NoContent(http.StatusOK)
	}

	return c.NoContent(http.StatusOK)
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
	ok, consumption, available, err := checkUserLimit(c.Request().Context(), h.minio, bucketName, userPrincipalDto, desiredSize)
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

	isRecording := true
	metadata := services.SerializeMetadataSimple(bindTo.OwnerId, bindTo.ChatId, nil, &isRecording)

	chatFileItemUuid := uuid.New().String()

	aKey := services.GetKey(bindTo.FileName, chatFileItemUuid, bindTo.ChatId)

	response := S3Response{
		AccessKey: accessKeyID,
		Secret:    secretAccessKey,
		Region:    viper.GetString("minio.location"),
		Endpoint:  endpoint,
		Bucket:    h.minioConfig.Files,
		Metadata:  metadata,
		Filepath:  aKey,
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
		case services.Media_audio:
			return utils.IsAudio(info.Key)
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
	h.previewCacheableResponse(c)
	return c.Stream(http.StatusOK, objectInfo.ContentType, object)
}

func (h *FilesHandler) PublicPreviewDownloadHandler(c echo.Context) error {
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

	belongs, err := h.restClient.CheckAccess(nil, chatId, c.Request().Context())
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during checking user auth to chat %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !belongs {
		GetLogEntry(c.Request().Context()).Errorf("Chat %v is not public", chatId)
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
	h.previewCacheableResponse(c)

	return c.Stream(http.StatusOK, objectInfo.ContentType, object)
}

func (h *FilesHandler) DownloadHandler(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
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

	// send redirect to presigned
	downloadUrl, ttl, err := h.filesService.GetTemporaryDownloadUrl(objectInfo.Key)
	if err != nil {
		return err
	}

	cacheableResponse(c, ttl)
	c.Response().Header().Set("Location", downloadUrl)
	c.Response().WriteHeader(http.StatusTemporaryRedirect)
	return nil
}

func (h *FilesHandler) previewCacheableResponse(c echo.Context) {
	cacheableResponse(c, viper.GetDuration("response.cache.preview"))
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
		GetLogEntry(c.Request().Context()).Infof("File %v is not public, checking is chat blog", fileId)

		chatId, err := utils.ParseChatId(objectInfo.Key)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during parsing chatId: %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}

		belongs, err := h.restClient.CheckAccess(nil, chatId, c.Request().Context())
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during checking user auth to chat %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
		if !belongs {
			GetLogEntry(c.Request().Context()).Errorf("File %v is not public", fileId)
			return c.NoContent(http.StatusUnauthorized)
		}
	}
	// end check

	// send redirect to presigned
	downloadUrl, ttl, err := h.filesService.GetTemporaryDownloadUrl(objectInfo.Key)
	if err != nil {
		return err
	}

	cacheableResponse(c, ttl)
	c.Response().Header().Set("Location", downloadUrl)
	c.Response().WriteHeader(http.StatusTemporaryRedirect)
	return nil
}
