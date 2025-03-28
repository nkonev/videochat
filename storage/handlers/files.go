package handlers

import (
	"context"
	"errors"
	"fmt"
	awsS3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/labstack/echo/v4"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/tags"
	"github.com/spf13/viper"
	"net/http"
	"net/url"
	"nkonev.name/storage/auth"
	"nkonev.name/storage/client"
	"nkonev.name/storage/dto"
	"nkonev.name/storage/logger"
	"nkonev.name/storage/s3"
	"nkonev.name/storage/services"
	"nkonev.name/storage/utils"
	"strconv"
	"strings"
	"time"
)

type FilesHandler struct {
	minio            *s3.InternalMinioClient
	awsS3            *awsS3.S3
	restClient       *client.RestClient
	minioConfig      *utils.MinioConfig
	filesService     *services.FilesService
	redisInfoService *services.RedisInfoService
	lgr              *logger.Logger
}

type RenameDto struct {
	Newname string `json:"newname"`
}

const NotFoundImage = "/images/covers/not_found.png"
const ConvertingImage = "/images/covers/ffmpeg_converting.jpg"

func NewFilesHandler(
	lgr *logger.Logger,
	minio *s3.InternalMinioClient,
	awsS3 *awsS3.S3,
	restClient *client.RestClient,
	minioConfig *utils.MinioConfig,
	filesService *services.FilesService,
	redisInfoService *services.RedisInfoService,
) *FilesHandler {
	return &FilesHandler{
		lgr:              lgr,
		minio:            minio,
		awsS3:            awsS3,
		restClient:       restClient,
		minioConfig:      minioConfig,
		filesService:     filesService,
		redisInfoService: redisInfoService,
	}
}

type EmbedDto struct {
	Url  string  `json:"url"`
	Type *string `json:"type"`
}

type UploadRequest struct {
	CorrelationId              *string `json:"correlationId"`
	FileItemUuid               *string `json:"fileItemUuid"`
	FileSize                   int64   `json:"fileSize"`
	FileName                   string  `json:"fileName"`
	ShouldAddDateToTheFilename bool    `json:"shouldAddDateToTheFilename"`
	IsMessageRecording         *bool   `json:"isMessageRecording"`
}

type UploadResponse struct {
	Url string `json:"url"`
}

type PresignedUrl struct {
	Url        string `json:"url"`
	PartNumber int    `json:"partNumber"`
}

type FinishMultipartRequest struct {
	Key      string                   `json:"key"`
	Parts    []MultipartFinishingPart `json:"parts"`
	UploadId string                   `json:"uploadId"`
}

type MultipartFinishingPart struct {
	Etag       string `json:"etag"`
	PartNumber int64  `json:"partNumber"`
}

func (h *FilesHandler) InitMultipartUpload(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}
	if ok, err := h.restClient.CheckAccess(c.Request().Context(), &userPrincipalDto.UserId, chatId); err != nil {
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

	// generated for the first file
	chatFileItemUuid := utils.GetFileItemId()
	fileItemUuidString := reqDto.FileItemUuid
	if fileItemUuidString != nil && *fileItemUuidString != "" {
		// and reused for the subsequent
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

	filteredFilename := utils.CleanFilename(c.Request().Context(), h.lgr, reqDto.FileName, reqDto.ShouldAddDateToTheFilename)

	aKey := services.GetKey(filteredFilename, chatFileItemUuid, chatId)

	// check that this file does not exist
	exists, _, err := h.minio.FileExists(c.Request().Context(), bucketName, aKey)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf(err.Error())
		return c.JSON(http.StatusInternalServerError, &utils.H{"status": "error", "message": err.Error()})
	}
	if exists {
		h.lgr.WithTracing(c.Request().Context()).Infof("Conflict for: %v", aKey)
		return c.JSON(http.StatusConflict, &utils.H{"status": "error", "message": fmt.Sprintf("Already exists: %v", aKey)})
	}

	// check enough size taking on account free disk space probe (see LimitsHandler)
	ok, consumption, available, err := checkUserLimit(c.Request().Context(), h.lgr, h.minio, bucketName, userPrincipalDto, reqDto.FileSize, h.restClient)
	if err != nil {
		return err
	}
	if !ok {
		return c.JSON(http.StatusOK, &utils.H{"status": "oversized", "used": consumption, "available": available})
	}

	metadata := services.SerializeMetadataSimple(userPrincipalDto.UserId, chatId, reqDto.CorrelationId, nil, reqDto.IsMessageRecording)

	expire := viper.GetDuration("minio.multipart.expire")
	expTime := time.Now().UTC().Add(expire)
	converted := convertMetadata(&metadata)
	mpu := awsS3.CreateMultipartUploadInput{
		Expires:  &expTime,
		Bucket:   &bucketName,
		Key:      &aKey,
		Metadata: converted,
	}
	upload, err := h.awsS3.CreateMultipartUpload(&mpu)
	if err != nil {
		return err
	}

	uploadDuration := viper.GetDuration("minio.presignUploadTtl")

	chunkSize := viper.GetInt64("minio.multipart.chunkSize")
	chunksNum := int(reqDto.FileSize / chunkSize)
	if reqDto.FileSize%chunkSize != 0 {
		chunksNum++
	}

	presignedUrls := []PresignedUrl{}
	for i := 1; i <= chunksNum; i++ {
		var urlVals = url.Values{}
		urlVals.Set("partNumber", utils.IntToString(i))
		urlVals.Set("uploadId", *upload.UploadId)

		u, err := h.minio.Presign(c.Request().Context(), "PUT", bucketName, aKey, uploadDuration, urlVals)
		if err != nil {
			h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting downlad url %v", err)
			return err
		}

		stringUrl, err := services.ChangeMinioUrl(u)
		if err != nil {
			return err
		}

		presignedUrls = append(presignedUrls, PresignedUrl{stringUrl, i})
	}
	existingCount := h.getCountFilesInFileItem(c.Request().Context(), bucketName, filenameChatPrefix)

	previewable := utils.IsPreviewable(aKey)

	return c.JSON(http.StatusOK, &utils.H{
		"status":        "ready",
		"uploadId":      upload.UploadId,
		"presignedUrls": presignedUrls,
		"chunkSize":     chunkSize,
		"fileItemUuid":  chatFileItemUuid,
		"existingCount": existingCount,
		"newFileName":   filteredFilename,
		"key":           aKey,
		"chatId":        chatId,
		"previewable":   previewable,
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
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}
	if ok, err := h.restClient.CheckAccess(c.Request().Context(), &userPrincipalDto.UserId, chatId); err != nil {
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
			ETag:       &part.Etag,
			PartNumber: &part.PartNumber,
		})
	}

	input := awsS3.CompleteMultipartUploadInput{
		Key:      &reqDto.Key,
		Bucket:   &bucketName,
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
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}
	if ok, err := h.restClient.CheckAccess(c.Request().Context(), &userPrincipalDto.UserId, chatId); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	var bindTo = new(ReplaceTextFileDto)
	if err := c.Bind(bindTo); err != nil {
		h.lgr.WithTracing(c.Request().Context()).Warnf("Error during binding to dto %v", err)
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
	userLimitOk, _, _, err := checkUserLimit(c.Request().Context(), h.lgr, h.minio, bucketName, userPrincipalDto, fileSize, h.restClient)
	if err != nil {
		return err
	}
	if !userLimitOk {
		return c.JSON(http.StatusRequestEntityTooLarge, &utils.H{"status": "fail"})
	}

	contentType := bindTo.ContentType

	h.lgr.WithTracing(c.Request().Context()).Debugf("Determined content type: %v", contentType)

	src := strings.NewReader(bindTo.Text)

	aKey := services.GetKey(bindTo.Filename, fileItemUuid, chatId)

	var userMetadata = services.SerializeMetadataSimple(userPrincipalDto.UserId, chatId, nil, nil, nil)

	if _, err := h.minio.PutObject(c.Request().Context(), bucketName, aKey, src, fileSize, minio.PutObjectOptions{ContentType: contentType, UserMetadata: userMetadata}); err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during upload object: %v", err)
		return err
	}

	return c.NoContent(http.StatusOK)
}

func (h *FilesHandler) getCountFilesInFileItem(ctx context.Context, bucketName string, filenameChatPrefix string) int {
	var count = 0
	var objectsNew <-chan minio.ObjectInfo = h.minio.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Prefix:    filenameChatPrefix,
		Recursive: true,
	})
	count = len(objectsNew)
	for oi := range objectsNew {
		h.lgr.WithTracing(ctx).Debugf("Processing %v", oi.Key)
		count++
	}
	return count
}

func getFileItemUuid(fileId string) string {
	split := strings.Split(fileId, "/")
	return split[2]
}

func (h *FilesHandler) ListHandler(c echo.Context) error {
	return h.listHandler(c, false)
}

func (h *FilesHandler) ListHandlerPublic(c echo.Context) error {
	return h.listHandler(c, true)
}

func (h *FilesHandler) listHandler(c echo.Context, public bool) error {
	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}

	var userId *int64
	var overrideChatId, overrideMessageId int64
	if !public {
		var userPrincipalDto, _ = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
		if userPrincipalDto == nil {
			return c.NoContent(http.StatusUnauthorized)
		} else {
			userId = &userPrincipalDto.UserId
		}
		overrideChatId = utils.ChatIdNonExistent
		overrideMessageId = utils.MessageIdNonExistent
	} else { // userPrincipalDto == nil and userId == nil
		overrideChatId = getOverrideChatIdPublic(c)
		overrideMessageId = getOverrideMessageIdPublic(c)
	}

	fileItemUuid := c.QueryParam("fileItemUuid")

	if ok, err := h.restClient.CheckAccessExtended(c.Request().Context(), userId, chatId, overrideChatId, overrideMessageId, fileItemUuid); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	filesPage := utils.FixPageString(c.QueryParam("page"))
	filesSize := utils.FixSizeString(c.QueryParam("size"))
	filesOffset := utils.GetOffset(filesPage, filesSize)

	searchString := c.QueryParam("searchString")
	searchString = strings.TrimSpace(searchString)
	searchString = strings.ToLower(searchString)

	bucketName := h.minioConfig.Files

	h.lgr.WithTracing(c.Request().Context()).Debugf("Listing bucket '%v':", bucketName)

	filenameChatPrefix := h.getFilenameChatPrefix(chatId, fileItemUuid)

	filter := h.getFilterFunction(searchString)

	list, count, err := h.filesService.GetListFilesInFileItem(c.Request().Context(), public, overrideChatId, overrideMessageId, userId, bucketName, filenameChatPrefix, chatId, filter, true, filesSize, filesOffset)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "items": list, "count": count})
}

type ViewItem struct {
	Url            string  `json:"url"`
	Filename       string  `json:"filename"`
	PreviewUrl     *string `json:"previewUrl"`
	This           bool    `json:"this"`
	CanPlayAsVideo bool    `json:"canPlayAsVideo"`
	CanShowAsImage bool    `json:"canShowAsImage"`
}

type ListViewRequest struct {
	Url string `json:"url"`
}

func (h *FilesHandler) ViewListHandler(c echo.Context) error {
	var userPrincipalDto, _ = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)

	reqDto := new(ListViewRequest)
	err := c.Bind(reqDto)
	if err != nil {
		return err
	}

	anUrl, err := url.Parse(reqDto.Url)
	if err != nil {
		return err
	}

	selfUrls := strings.Split(viper.GetString("selfUrls"), ",")

	var retList = []ViewItem{}

	if !utils.ContainsUrl(h.lgr, selfUrls, reqDto.Url) {
		return c.JSON(http.StatusOK, &utils.H{"status": "ok", "items": retList})
	}

	fileId := anUrl.Query().Get(utils.FileParam)
	if fileId == "" {
		return c.JSON(http.StatusOK, &utils.H{"status": "ok", "items": retList})
	}

	fileItemUuid, err := utils.ParseFileItemUuid(fileId)
	if err != nil {
		return err
	}

	overrideChatId, err := getOverrideChatIdPublicFromUrl(anUrl)
	if err != nil {
		return err
	}

	overrideMessageId, err := getOverrideMessageIdPublicFromUrl(anUrl)
	if err != nil {
		return err
	}

	chatId, err := utils.ParseChatId(fileId)
	if err != nil {
		return err
	}

	var userId *int64 = nil
	var isAnonymous = false // public message or blog
	if userPrincipalDto != nil {
		userId = &userPrincipalDto.UserId
	} else {
		isAnonymous = true
	}
	if ok, err := h.restClient.CheckAccessExtended(c.Request().Context(), userId, chatId, overrideChatId, overrideMessageId, fileItemUuid); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	bucketName := h.minioConfig.Files

	filenameChatPrefix := h.getFilenameChatPrefix(chatId, fileItemUuid)

	var filter = func(objInfo *minio.ObjectInfo) bool {
		return utils.IsPreviewable(objInfo.Key)
	}

	var objects <-chan minio.ObjectInfo = h.minio.ListObjects(c.Request().Context(), bucketName, minio.ListObjectsOptions{
		WithMetadata: true,
		Prefix:       filenameChatPrefix,
		Recursive:    true,
	})

	for objInfo := range objects {
		if filter(&objInfo) {

			var downloadUrl string
			var previewUrl *string
			if !isAnonymous {
				downloadUrl, err = h.filesService.GetConstantDownloadUrl(objInfo.Key)
				if err != nil {
					h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting downlad url %v", err)
					continue
				}
				previewUrl = h.filesService.GetPreviewUrlSmart(c.Request().Context(), objInfo.Key)
			} else {
				downloadUrl, err = h.filesService.GetAnonymousUrl(objInfo.Key, overrideChatId, overrideMessageId)
				if err != nil {
					h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting public downlad url %v", err)
					continue
				}

				previewUrl, err = h.filesService.GetAnonymousPreviewUrl(c.Request().Context(), objInfo.Key, overrideChatId, overrideMessageId)
				if err != nil {
					h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting public downlad url %v", err)
					continue
				}
			}

			filename := services.ReadFilename(objInfo.Key)
			retList = append(retList, ViewItem{
				Url:            downloadUrl,
				Filename:       filename,
				PreviewUrl:     previewUrl,
				This:           objInfo.Key == fileId,
				CanPlayAsVideo: utils.IsVideo(objInfo.Key),
				CanShowAsImage: utils.IsImage(objInfo.Key),
			})
		}
	}

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "items": retList})
}

type StatusViewRequest struct {
	Url string `json:"url"`
}

// returns status: ok, error, not_found, converting
func (h *FilesHandler) ViewStatusHandler(c echo.Context) error {
	var userPrincipalDto, _ = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)

	reqDto := new(StatusViewRequest)
	err := c.Bind(reqDto)
	if err != nil {
		return err
	}

	anUrl, err := url.Parse(reqDto.Url)
	if err != nil {
		return err
	}

	selfUrls := strings.Split(viper.GetString("selfUrls"), ",")

	if !utils.ContainsUrl(h.lgr, selfUrls, reqDto.Url) {
		return c.JSON(http.StatusOK, &utils.H{"status": "ok"})
	}

	fileId := anUrl.Query().Get(utils.FileParam)
	if fileId == "" {
		return c.JSON(http.StatusOK, &utils.H{"status": "ok"})
	}

	fileItemUuid, err := utils.ParseFileItemUuid(fileId)
	if err != nil {
		return err
	}

	overrideChatId, err := getOverrideChatIdPublicFromUrl(anUrl)
	if err != nil {
		return err
	}

	overrideMessageId, err := getOverrideMessageIdPublicFromUrl(anUrl)
	if err != nil {
		return err
	}

	chatId, err := utils.ParseChatId(fileId)
	if err != nil {
		return err
	}

	var userId *int64 = nil
	if userPrincipalDto != nil {
		userId = &userPrincipalDto.UserId
	}
	if ok, err := h.restClient.CheckAccessExtended(c.Request().Context(), userId, chatId, overrideChatId, overrideMessageId, fileItemUuid); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	bucketName := h.minioConfig.Files

	exists, obj, err := h.minio.FileExists(c.Request().Context(), bucketName, fileId)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Unable to check existence of %v: %v", fileId, err)
		return c.JSON(http.StatusOK, StatusItem{
			Status:       "error",
			FileItemUuid: &fileItemUuid,
		})
	}
	// obj is nil in case when video is still converting

	if exists {
		filename := services.ReadFilename(obj.Key)
		return c.JSON(http.StatusOK, StatusItem{
			Status:       "ok",
			Filename:     filename,
			FileItemUuid: &fileItemUuid,
		})
	} else {
		converting, err := h.redisInfoService.GetConvertedConverting(c.Request().Context(), fileId)
		if err != nil {
			h.lgr.WithTracing(c.Request().Context()).Errorf("Unable to check converting of %v: %v", fileId, err)
			return c.JSON(http.StatusOK, StatusItem{
				Status:       "error",
				FileItemUuid: &fileItemUuid,
			})
		}
		if converting {
			i := ConvertingImage
			return c.JSON(http.StatusOK, StatusItem{
				Status:       "converting",
				StatusImage:  &i,
				FileItemUuid: &fileItemUuid,
			})
		} else {
			i := NotFoundImage
			return c.JSON(http.StatusOK, StatusItem{
				Status:       "not_found",
				StatusImage:  &i,
				FileItemUuid: &fileItemUuid,
			})
		}
	}
}

type StatusItem struct {
	Status       string  `json:"status"`
	FileItemUuid *string `json:"fileItemUuid"`
	Filename     string  `json:"filename"`
	StatusImage  *string `json:"statusImage"`
}

func (h *FilesHandler) getFilenameChatPrefix(chatId int64, fileItemUuid string) string {
	var filenameChatPrefix string
	if fileItemUuid == "" {
		filenameChatPrefix = fmt.Sprintf("chat/%v/", chatId)
	} else {
		filenameChatPrefix = fmt.Sprintf("chat/%v/%v/", chatId, fileItemUuid)
	}
	return filenameChatPrefix
}

func (h *FilesHandler) getFilterFunction(searchString string) func(info *minio.ObjectInfo) bool {
	var filter func(info *minio.ObjectInfo) bool = nil
	if searchString != "" {
		filter = func(info *minio.ObjectInfo) bool {
			normalizedFileName := strings.ToLower(services.ReadFilename(info.Key))
			return strings.Contains(normalizedFileName, searchString)
		}
	}
	return filter
}

func (h *FilesHandler) ListFileItemUuids(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}
	if ok, err := h.restClient.CheckAccess(c.Request().Context(), &userPrincipalDto.UserId, chatId); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	filesPage := utils.FixPageString(c.QueryParam("page"))
	filesSize := utils.FixSizeString(c.QueryParam("size"))
	filesOffset := utils.GetOffset(filesPage, filesSize)

	bucketName := h.minioConfig.Files

	filenameChatPrefix := fmt.Sprintf("chat/%v/", chatId)

	list, count, err := h.filesService.GetListFilesItemUuids(c.Request().Context(), bucketName, filenameChatPrefix, filesSize, filesOffset)
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
		h.lgr.WithTracing(c.Request().Context()).Warnf("Error during binding to dto %v", err)
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}
	if ok, err := h.restClient.CheckAccess(c.Request().Context(), &userPrincipalDto.UserId, chatId); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	bucketName := h.minioConfig.Files

	// check this fileItem belongs to user
	objectInfo, err := h.minio.StatObject(c.Request().Context(), bucketName, bindTo.Id, minio.StatObjectOptions{})
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	belongs, err := h.checkFileBelongsToUser(c.Request().Context(), objectInfo, chatId, userPrincipalDto, false)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during checking belong object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !belongs {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Object '%v' is not belongs to user %v", objectInfo.Key, userPrincipalDto.UserId)
		return c.NoContent(http.StatusUnauthorized)
	}
	// end check

	err = h.minio.RemoveObject(c.Request().Context(), bucketName, objectInfo.Key, minio.RemoveObjectOptions{})
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during removing object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if utils.IsVideo(objectInfo.Key) {
		previewToCheck := utils.SetVideoPreviewExtension(objectInfo.Key)
		err = h.minio.RemoveObject(c.Request().Context(), h.minioConfig.FilesPreview, previewToCheck, minio.RemoveObjectOptions{})
		if err != nil {
			h.lgr.WithTracing(c.Request().Context()).Errorf("Error during removing object %v", err)
		}
	} else if utils.IsImage(objectInfo.Key) {
		previewToCheck := utils.SetImagePreviewExtension(objectInfo.Key)
		err = h.minio.RemoveObject(c.Request().Context(), h.minioConfig.FilesPreview, previewToCheck, minio.RemoveObjectOptions{})
		if err != nil {
			h.lgr.WithTracing(c.Request().Context()).Errorf("Error during removing object %v", err)
		}
	}

	return c.NoContent(http.StatusOK)
}

func (h *FilesHandler) checkFileItemBelongsToUser(filenameChatPrefix string, c echo.Context, chatId int64, bucketName string, userPrincipalDto *auth.AuthResult) (bool, error) {
	var objects <-chan minio.ObjectInfo = h.minio.ListObjects(c.Request().Context(), bucketName, minio.ListObjectsOptions{
		WithMetadata: true,
		Prefix:       filenameChatPrefix,
		Recursive:    true,
	})
	for objInfo := range objects {
		b, err := h.checkFileBelongsToUser(c.Request().Context(), objInfo, chatId, userPrincipalDto, true)
		if err != nil {
			return false, err
		}
		if !b {
			return false, nil
		}
	}
	return true, nil
}

func (h *FilesHandler) checkFileBelongsToUser(ctx context.Context, objInfo minio.ObjectInfo, chatId int64, userPrincipalDto *auth.AuthResult, hasAmzPrefix bool) (bool, error) {
	gotChatId, gotOwnerId, _, err := services.DeserializeMetadata(objInfo.UserMetadata, hasAmzPrefix)
	if err != nil {
		h.lgr.WithTracing(ctx).Errorf("Error deserializeMetadata: %v", err)
		return false, err
	}

	if gotChatId != chatId {
		h.lgr.WithTracing(ctx).Infof("Wrong chatId: expected %v but got %v", chatId, gotChatId)
		return false, nil
	}

	if gotOwnerId != userPrincipalDto.UserId {
		h.lgr.WithTracing(ctx).Infof("Wrong ownerId: expected %v but got %v", userPrincipalDto.UserId, gotOwnerId)
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
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	bucketName := h.minioConfig.Files

	var bindTo = new(PublishRequest)
	if err := c.Bind(bindTo); err != nil {
		h.lgr.WithTracing(c.Request().Context()).Warnf("Error during binding to dto %v", err)
		return err
	}

	// check user is owner
	fileId := bindTo.Id
	objectInfo, err := h.minio.StatObject(c.Request().Context(), bucketName, fileId, minio.StatObjectOptions{})
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	_, ownerId, _, err := services.DeserializeMetadata(objectInfo.UserMetadata, false)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during deserializing object metadata %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if ownerId != userPrincipalDto.UserId {
		h.lgr.WithTracing(c.Request().Context()).Errorf("User %v is not owner of file %v", userPrincipalDto.UserId, fileId)
		return c.NoContent(http.StatusUnauthorized)
	}
	// end check

	tagsMap := services.SerializeTags(bindTo.Public)
	objectTags, err := tags.MapToObjectTags(tagsMap)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during mapping tags %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	err = h.minio.PutObjectTagging(c.Request().Context(), bucketName, fileId, objectTags, minio.PutObjectTaggingOptions{})
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during saving tags %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	publishedUrl, err := h.filesService.GetPublishedUrl(bindTo.Public, objectInfo.Key)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error get public url: %v", err)
		return err
	}

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "publishedUrl": publishedUrl})
}

type CountResponse struct {
	Count int  `json:"count"`
	Found bool `json:"found"`
}

type CountRequest struct {
	SearchString string `json:"searchString"`
	FileItemUuid string `json:"fileItemUuid"`
}

func (h *FilesHandler) CountHandler(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	var bindTo = new(CountRequest)
	if err := c.Bind(bindTo); err != nil {
		h.lgr.WithTracing(c.Request().Context()).Warnf("Error during binding to dto %v", err)
		return err
	}

	bucketName := h.minioConfig.Files

	searchString := bindTo.SearchString
	searchString = strings.TrimSpace(searchString)
	searchString = strings.ToLower(searchString)

	// check user belongs to chat
	fileItemUuid := bindTo.FileItemUuid
	chatIdString := c.Param("chatId")
	chatId, err := utils.ParseInt64(chatIdString)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during parsing chatId %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	belongs, err := h.restClient.CheckAccess(c.Request().Context(), &userPrincipalDto.UserId, chatId)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during checking user auth to chat %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !belongs {
		h.lgr.WithTracing(c.Request().Context()).Errorf("User %v is not belongs to chat %v", userPrincipalDto.UserId, chatId)
		return c.NoContent(http.StatusUnauthorized)
	}
	// end check

	filenameChatPrefix := h.getFilenameChatPrefix(chatId, fileItemUuid)

	filter := h.getFilterFunction(searchString)

	var objects <-chan minio.ObjectInfo = h.minio.ListObjects(c.Request().Context(), bucketName, minio.ListObjectsOptions{
		WithMetadata: false,
		Prefix:       filenameChatPrefix,
		Recursive:    true,
	})

	var counter = 0
	var exists bool
	for objInfo := range objects {
		if (filter != nil && filter(&objInfo)) || filter == nil {
			counter++
		}
	}

	var countDto = CountResponse{
		Count: counter,
		Found: exists,
	}

	return c.JSON(http.StatusOK, countDto)
}

type FilterRequest struct {
	SearchString string `json:"searchString"`
	FileItemUuid string `json:"fileItemUuid"`
	FileId       string `json:"fileId"`
}

type FilterResponseItem struct {
	Id string `json:"id"`
}

func (h *FilesHandler) FilterHandler(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	var bindTo = new(FilterRequest)
	if err := c.Bind(bindTo); err != nil {
		h.lgr.WithTracing(c.Request().Context()).Warnf("Error during binding to dto %v", err)
		return err
	}

	bucketName := h.minioConfig.Files

	searchString := bindTo.SearchString
	searchString = strings.TrimSpace(searchString)
	searchString = strings.ToLower(searchString)

	// check user belongs to chat
	fileItemUuid := bindTo.FileItemUuid
	chatIdString := c.Param("chatId")
	chatId, err := utils.ParseInt64(chatIdString)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during parsing chatId %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	fileId := bindTo.FileId

	belongs, err := h.restClient.CheckAccess(c.Request().Context(), &userPrincipalDto.UserId, chatId)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during checking user auth to chat %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !belongs {
		h.lgr.WithTracing(c.Request().Context()).Errorf("User %v is not belongs to chat %v", userPrincipalDto.UserId, chatId)
		return c.NoContent(http.StatusUnauthorized)
	}
	// end check

	filenameChatPrefix := h.getFilenameChatPrefix(chatId, fileItemUuid)

	filter := h.getFilterFunction(searchString)

	var objects <-chan minio.ObjectInfo = h.minio.ListObjects(c.Request().Context(), bucketName, minio.ListObjectsOptions{
		WithMetadata: false,
		Prefix:       filenameChatPrefix,
		Recursive:    true,
	})

	var filterResponseItemArray = make([]FilterResponseItem, 0)
	for objInfo := range objects {
		if (filter != nil && filter(&objInfo)) || filter == nil {
			if objInfo.Key == fileId {
				filterResponseItemArray = append(filterResponseItemArray, FilterResponseItem{
					Id: objInfo.Key,
				})
			}
		}
	}

	return c.JSON(http.StatusOK, filterResponseItemArray)
}

func (h *FilesHandler) LimitsHandler(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}
	if ok, err := h.restClient.CheckAccess(c.Request().Context(), &userPrincipalDto.UserId, chatId); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	bucketName := h.minioConfig.Files

	desiredSize, err := utils.ParseInt64(c.QueryParam("desiredSize"))
	if err != nil {
		return err
	}
	ok, consumption, available, err := checkUserLimit(c.Request().Context(), h.lgr, h.minio, bucketName, userPrincipalDto, desiredSize, h.restClient)
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
		h.lgr.WithTracing(c.Request().Context()).Warnf("Error during binding to dto %v", err)
		return err
	}

	endpoint := viper.GetString("minio.interContainerUrl")
	accessKeyID := viper.GetString("minio.accessKeyId")
	secretAccessKey := viper.GetString("minio.secretAccessKey")

	isConferenceRecording := true
	metadata := services.SerializeMetadataSimple(bindTo.OwnerId, bindTo.ChatId, nil, &isConferenceRecording, nil)

	chatFileItemUuid := utils.GetFileItemId()

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
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting auth context")
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
	belongs, err := h.restClient.CheckAccess(c.Request().Context(), &userPrincipalDto.UserId, chatId)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during checking user auth to chat %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !belongs {
		h.lgr.WithTracing(c.Request().Context()).Errorf("User %v is not belongs to chat %v", userPrincipalDto.UserId, chatId)
		return c.NoContent(http.StatusUnauthorized)
	}

	var filenameChatPrefix string = fmt.Sprintf("chat/%v/", chatId)

	filter := h.getFilterByType(requestedMediaType)

	items, count, err := h.filesService.GetListFilesInFileItem(c.Request().Context(), false, utils.ChatIdNonExistent, utils.MessageIdNonExistent, &userPrincipalDto.UserId, bucketName, filenameChatPrefix, chatId, filter, false, filesSize, filesOffset)
	if err != nil {
		return err
	}

	var list []*MediaDto = make([]*MediaDto, 0)

	for _, item := range items {
		list = append(list, convert(item))
	}

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "items": list, "count": count})
}

func (h *FilesHandler) CountEmbed(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	requestedMediaType := c.QueryParam("type")

	bucketName := h.minioConfig.Files

	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}
	belongs, err := h.restClient.CheckAccess(c.Request().Context(), &userPrincipalDto.UserId, chatId)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during checking user auth to chat %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !belongs {
		h.lgr.WithTracing(c.Request().Context()).Errorf("User %v is not belongs to chat %v", userPrincipalDto.UserId, chatId)
		return c.NoContent(http.StatusUnauthorized)
	}

	var filenameChatPrefix string = fmt.Sprintf("chat/%v/", chatId)

	filter := h.getFilterByType(requestedMediaType)

	_, count, err := h.filesService.GetListFilesInFileItem(c.Request().Context(), false, utils.ChatIdNonExistent, utils.MessageIdNonExistent, &userPrincipalDto.UserId, bucketName, filenameChatPrefix, chatId, filter, false, 10, 0)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "count": count})
}

func (h *FilesHandler) getFilterByType(requestedMediaType string) func(info *minio.ObjectInfo) bool {
	return func(info *minio.ObjectInfo) bool {
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
}

type CandidatesFilterRequest struct {
	Type   string `json:"type"`
	FileId string `json:"fileId"`
}

func (h *FilesHandler) FilterEmbed(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	chatIdString := c.Param("chatId")
	chatId, err := utils.ParseInt64(chatIdString)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during parsing chatId %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	belongs, err := h.restClient.CheckAccess(c.Request().Context(), &userPrincipalDto.UserId, chatId)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during checking user auth to chat %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !belongs {
		h.lgr.WithTracing(c.Request().Context()).Errorf("User %v is not belongs to chat %v", userPrincipalDto.UserId, chatId)
		return c.NoContent(http.StatusUnauthorized)
	}
	// end check

	bucketName := h.minioConfig.Files

	var bindTo = new(CandidatesFilterRequest)
	if err := c.Bind(bindTo); err != nil {
		h.lgr.WithTracing(c.Request().Context()).Warnf("Error during binding to dto %v", err)
		return err
	}

	var filenameChatPrefix string = fmt.Sprintf("chat/%v/", chatId)

	filter := h.getFilterByType(bindTo.Type)

	var objects <-chan minio.ObjectInfo = h.minio.ListObjects(c.Request().Context(), bucketName, minio.ListObjectsOptions{
		WithMetadata: false,
		Prefix:       filenameChatPrefix,
		Recursive:    true,
	})

	var filterResponseItemArray = make([]FilterResponseItem, 0)
	for objInfo := range objects {
		if (filter != nil && filter(&objInfo)) || filter == nil {
			if objInfo.Key == bindTo.FileId {
				filterResponseItemArray = append(filterResponseItemArray, FilterResponseItem{
					Id: objInfo.Key,
				})
			}
		}
	}

	return c.JSON(http.StatusOK, filterResponseItemArray)
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
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	bucketName := h.minioConfig.FilesPreview

	// check user belongs to chat
	fileId := c.QueryParam(utils.FileParam)

	exists, objectInfo, err := h.minio.FileExists(c.Request().Context(), bucketName, fileId)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !exists {
		return c.Redirect(http.StatusTemporaryRedirect, NotFoundImage)
	}

	chatId, err := utils.ParseChatId(fileId)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during deserializing object metadata %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	belongs, err := h.restClient.CheckAccess(c.Request().Context(), &userPrincipalDto.UserId, chatId)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during checking user auth to chat %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !belongs {
		h.lgr.WithTracing(c.Request().Context()).Errorf("User %v is not belongs to chat %v", userPrincipalDto.UserId, chatId)
		return c.NoContent(http.StatusUnauthorized)
	}
	// end check

	object, e := h.minio.GetObject(c.Request().Context(), bucketName, fileId, minio.GetObjectOptions{})
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
	exists, objectInfo, err := h.minio.FileExists(c.Request().Context(), bucketName, fileId)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !exists {
		return c.Redirect(http.StatusTemporaryRedirect, NotFoundImage)
	}

	chatId, err := utils.ParseChatId(fileId)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during deserializing object metadata %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	overrideChatId := getOverrideChatIdPublic(c)
	overrideMessageId := getOverrideMessageIdPublic(c)

	fileItemUuid, err := utils.ParseFileItemUuid(fileId)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting file item uuid %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	belongs, err := h.restClient.CheckAccessExtended(c.Request().Context(), nil, chatId, overrideChatId, overrideMessageId, fileItemUuid)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during checking user auth to chat %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !belongs {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Chat %v is not public", chatId)
		return c.NoContent(http.StatusUnauthorized)
	}
	// end check

	object, e := h.minio.GetObject(c.Request().Context(), bucketName, fileId, minio.GetObjectOptions{})
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
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	bucketName := h.minioConfig.Files

	// check user belongs to chat
	fileId := c.QueryParam(utils.FileParam)

	exists, objectInfo, err := h.minio.FileExists(c.Request().Context(), bucketName, fileId)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !exists {
		return c.Redirect(http.StatusTemporaryRedirect, NotFoundImage)
	}
	chatId, _, _, err := services.DeserializeMetadata(objectInfo.UserMetadata, false)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during deserializing object metadata %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	belongs, err := h.restClient.CheckAccess(c.Request().Context(), &userPrincipalDto.UserId, chatId)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during checking user auth to chat %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !belongs {
		h.lgr.WithTracing(c.Request().Context()).Errorf("User %v is not belongs to chat %v", userPrincipalDto.UserId, chatId)
		return c.NoContent(http.StatusUnauthorized)
	}
	// end check

	// send redirect to presigned
	downloadUrl, ttl, err := h.filesService.GetTemporaryDownloadUrl(c.Request().Context(), objectInfo.Key)
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

func getOverrideChatIdPublic(c echo.Context) int64 {
	parseInt64, err := utils.ParseInt64(c.QueryParam(utils.OverrideChatId))
	if err != nil {
		return utils.ChatIdNonExistent
	}
	return parseInt64
}

func getOverrideMessageIdPublic(c echo.Context) int64 {
	parseInt64, err := utils.ParseInt64(c.QueryParam(utils.OverrideMessageId))
	if err != nil {
		return utils.MessageIdNonExistent
	}
	return parseInt64
}

func getOverrideChatIdPublicFromUrl(anUrl *url.URL) (int64, error) {
	chatId := int64(utils.ChatIdNonExistent)
	if anUrl == nil {
		return chatId, errors.New("no url")
	}
	chatIdRaw := anUrl.Query().Get(utils.OverrideChatId)
	var err error
	if len(chatIdRaw) > 0 {
		chatId, err = utils.ParseInt64(chatIdRaw)
		if err != nil {
			return chatId, err
		}
	}
	return chatId, err
}

func getOverrideMessageIdPublicFromUrl(anUrl *url.URL) (int64, error) {
	messageId := int64(utils.MessageIdNonExistent)
	if anUrl == nil {
		return messageId, errors.New("no url")
	}
	messageIdRaw := anUrl.Query().Get(utils.OverrideMessageId)
	var err error
	if len(messageIdRaw) > 0 {
		messageId, err = utils.ParseInt64(messageIdRaw)
		if err != nil {
			return messageId, err
		}
	}
	return messageId, err
}

func (h *FilesHandler) PublicDownloadHandler(c echo.Context) error {
	bucketName := h.minioConfig.Files

	// check file is public
	fileId := c.QueryParam(utils.FileParam)
	exists, objectInfo, err := h.minio.FileExists(c.Request().Context(), bucketName, fileId)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !exists {
		return c.Redirect(http.StatusTemporaryRedirect, NotFoundImage)
	}

	tagging, err := h.minio.GetObjectTagging(c.Request().Context(), bucketName, fileId, minio.GetObjectTaggingOptions{})
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during deserializing object tags %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	isPublic, err := services.DeserializeTags(tagging)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during deserializing object tags %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if !isPublic {
		h.lgr.WithTracing(c.Request().Context()).Infof("File %v is not public, checking is chat blog", fileId)

		chatId, err := utils.ParseChatId(objectInfo.Key)
		if err != nil {
			h.lgr.WithTracing(c.Request().Context()).Errorf("Error during parsing chatId: %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}

		overrideChatId := getOverrideChatIdPublic(c)
		overrideMessageId := getOverrideMessageIdPublic(c)

		fileItemUuid, err := utils.ParseFileItemUuid(fileId)
		if err != nil {
			h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting file item uuid %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}

		belongs, err := h.restClient.CheckAccessExtended(c.Request().Context(), nil, chatId, overrideChatId, overrideMessageId, fileItemUuid)
		if err != nil {
			h.lgr.WithTracing(c.Request().Context()).Errorf("Error during checking user auth to chat %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
		if !belongs {
			h.lgr.WithTracing(c.Request().Context()).Errorf("File %v is not public", fileId)
			return c.NoContent(http.StatusUnauthorized)
		}
	}
	// end check

	// send redirect to presigned
	downloadUrl, ttl, err := h.filesService.GetTemporaryDownloadUrl(c.Request().Context(), objectInfo.Key)
	if err != nil {
		return err
	}

	cacheableResponse(c, ttl)
	c.Response().Header().Set("Location", downloadUrl)
	c.Response().WriteHeader(http.StatusTemporaryRedirect)
	return nil
}
