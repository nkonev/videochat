package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	awsS3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/tags"
	"github.com/spf13/viper"
	"nkonev.name/storage/auth"
	"nkonev.name/storage/client"
	"nkonev.name/storage/db"
	"nkonev.name/storage/dto"
	"nkonev.name/storage/logger"
	"nkonev.name/storage/producer"
	"nkonev.name/storage/s3"
	"nkonev.name/storage/services"
	"nkonev.name/storage/utils"
)

const headerCorrelationId = "X-CorrelationId"

type FilesHandler struct {
	minio            *s3.InternalMinioClient
	awsS3            *awsS3.S3
	restClient       *client.RestClient
	minioConfig      *utils.MinioConfig
	filesService     *services.FilesService
	redisInfoService *services.RedisInfoService
	dba              *db.DB
	lgr              *logger.Logger
	publisher        *producer.RabbitFileUploadedPublisher
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
	dba *db.DB,
	publisher *producer.RabbitFileUploadedPublisher,
) *FilesHandler {
	return &FilesHandler{
		lgr:              lgr,
		minio:            minio,
		awsS3:            awsS3,
		restClient:       restClient,
		minioConfig:      minioConfig,
		filesService:     filesService,
		redisInfoService: redisInfoService,
		dba:              dba,
		publisher:        publisher,
	}
}

type EmbedDto struct {
	Url  string  `json:"url"`
	Type *string `json:"type"`
}

type UploadRequest struct {
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
	belongs, err := h.checkFileItemBelongsToUser(chatFileItemUuid, c, chatId, bucketName, userPrincipalDto)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during checking belongs, userId = %v, chatId = %v: %v", userPrincipalDto.UserId, chatId, err)
		return c.NoContent(http.StatusInternalServerError)
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

	correlationId := c.Request().Header.Get(headerCorrelationId)
	var correlationIdP *string

	if correlationId != "" {
		_, err = uuid.Parse(correlationId)
		if err != nil {
			return c.NoContent(http.StatusBadRequest)
		}

		correlationIdP = &correlationId
	}

	metadata := services.SerializeMetadataSimple(userPrincipalDto.UserId, correlationIdP, nil, reqDto.IsMessageRecording, utils.GetUnixMilliUtc())

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

	existingCount, err := db.GetCount(c.Request().Context(), h.dba, chatId, chatFileItemUuid, nil)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting count %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

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
		kk := k
		vv := v
		res[kk] = &vv
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
	belongs, err := h.checkFileItemBelongsToUser(fileItemUuid, c, chatId, bucketName, userPrincipalDto)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during checking belongs, userId = %v, chatId = %v: %v", userPrincipalDto.UserId, chatId, err)
		return c.NoContent(http.StatusInternalServerError)
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

	var userMetadata = services.SerializeMetadataSimple(userPrincipalDto.UserId, nil, nil, nil, utils.GetUnixMilliUtc())

	if _, err := h.minio.PutObject(c.Request().Context(), bucketName, aKey, src, fileSize, minio.PutObjectOptions{ContentType: contentType, UserMetadata: userMetadata}); err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during upload object: %v", err)
		return err
	}

	return c.NoContent(http.StatusOK)
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

	filterObj := db.NewFilterBySearchString(searchString)

	list, count, err := h.filesService.GetListFilesInFileItem(c.Request().Context(), public, overrideChatId, overrideMessageId, userId, bucketName, chatId, fileItemUuid, filterObj, true, filesSize, filesOffset)
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

	filterObj := db.NewFilterByType(services.GetPreviewableExtensions())

	viewListLimit := viper.GetInt("viewList.maxSize")
	metadatas, err := db.GetList(c.Request().Context(), h.dba, chatId, fileItemUuid, filterObj, true, viewListLimit, 0)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting list, userId = %v, chatId = %v: %v", userId, chatId, err)
		return c.NoContent(http.StatusInternalServerError)
	}

	for _, metadata := range metadatas {
		aKey := utils.BuildNormalizedKey(&metadata)
		var downloadUrl string
		var previewUrl *string
		if !isAnonymous {
			downloadUrl, err = h.filesService.GetConstantDownloadUrl(aKey)
			if err != nil {
				h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting downlad url %v", err)
				continue
			}
			previewUrl = h.filesService.GetPreviewUrlSmart(c.Request().Context(), aKey)
		} else {
			downloadUrl, err = h.filesService.GetAnonymousUrl(aKey, overrideChatId, overrideMessageId)
			if err != nil {
				h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting public downlad url %v", err)
				continue
			}

			previewUrl, err = h.filesService.GetAnonymousPreviewUrl(c.Request().Context(), aKey, overrideChatId, overrideMessageId)
			if err != nil {
				h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting public downlad url %v", err)
				continue
			}
		}

		filename := services.ReadFilename(aKey)
		retList = append(retList, ViewItem{
			Url:            downloadUrl,
			Filename:       filename,
			PreviewUrl:     previewUrl,
			This:           aKey == fileId,
			CanPlayAsVideo: utils.IsVideo(aKey),
			CanShowAsImage: utils.IsImage(aKey),
		})

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

	list, count, err := h.filesService.GetListFilesItemUuids(c.Request().Context(), chatId, filesSize, filesOffset)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "files": list, "count": count})
}

type DeleteObjectDto struct {
	FileId       *string `json:"id"`           // file id
	FileItemUuid *string `json:"fileItemUuid"` // file item uuid
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

	if bindTo.FileId != nil {
		fileId := *bindTo.FileId
		// check this fileItem belongs to user
		fileItemUuid, err := utils.ParseFileItemUuid(fileId)
		if err != nil {
			return err
		}
		belongs, err := db.CheckFileItemBelongsToUser(c.Request().Context(), h.dba, chatId, fileItemUuid, userPrincipalDto.UserId)
		if err != nil {
			h.lgr.WithTracing(c.Request().Context()).Errorf("Error during checking belong object %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
		if !belongs {
			return c.NoContent(http.StatusUnauthorized)
		}
		// end check

		err = h.minio.RemoveObject(c.Request().Context(), bucketName, fileId, minio.RemoveObjectOptions{})
		if err != nil {
			h.lgr.WithTracing(c.Request().Context()).Errorf("Error during removing object %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}

		if utils.IsVideo(fileId) {
			previewToCheck := utils.SetVideoPreviewExtension(fileId)
			err = h.minio.RemoveObject(c.Request().Context(), h.minioConfig.FilesPreview, previewToCheck, minio.RemoveObjectOptions{})
			if err != nil {
				h.lgr.WithTracing(c.Request().Context()).Errorf("Error during removing object %v", err)
			}
		} else if utils.IsImage(fileId) {
			previewToCheck := utils.SetImagePreviewExtension(fileId)
			err = h.minio.RemoveObject(c.Request().Context(), h.minioConfig.FilesPreview, previewToCheck, minio.RemoveObjectOptions{})
			if err != nil {
				h.lgr.WithTracing(c.Request().Context()).Errorf("Error during removing object %v", err)
			}
		}
	} else if bindTo.FileItemUuid != nil {
		fileItemUuid := *bindTo.FileItemUuid
		// check this fileItem belongs to user
		belongs, err := db.CheckFileItemBelongsToUser(c.Request().Context(), h.dba, chatId, fileItemUuid, userPrincipalDto.UserId)
		if err != nil {
			h.lgr.WithTracing(c.Request().Context()).Errorf("Error during checking belong object %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
		if !belongs {
			return c.NoContent(http.StatusUnauthorized)
		}
		// end check

		prefix := fmt.Sprintf("chat/%v/%v/", chatId, fileItemUuid)

		// send events (they won't be sent via minio subscription, so we send them manually)
		var fileObjects <-chan minio.ObjectInfo = h.minio.ListObjects(c.Request().Context(), bucketName, minio.ListObjectsOptions{
			Prefix:       prefix,
			Recursive:    true,
			WithMetadata: true,
		})

		batch := []minio.ObjectInfo{}
		for fileOjInfo := range fileObjects {
			batch = append(batch, fileOjInfo)
			if len(batch) == utils.DefaultSize {
				h.sendFileDeletedToUsers(c.Request().Context(), chatId, fileItemUuid, batch)
				batch = []minio.ObjectInfo{}
			}
		}
		if len(batch) > 0 {
			h.sendFileDeletedToUsers(c.Request().Context(), chatId, fileItemUuid, batch)
		}

		err = db.RemoveFileItem(c.Request().Context(), h.dba, chatId, fileItemUuid)
		if err != nil {
			h.lgr.WithTracing(c.Request().Context()).Errorf("Error during removing object metadata %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}

		err = h.minio.RemoveObject(c.Request().Context(), bucketName, prefix, minio.RemoveObjectOptions{ForceDelete: true})
		if err != nil {
			h.lgr.WithTracing(c.Request().Context()).Errorf("Error during removing object %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
	} else {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Unknown invariant")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}

func (h *FilesHandler) sendFileDeletedToUsers(c context.Context, chatId int64, fileItemUuid string, batch []minio.ObjectInfo) {
	bucketName := h.minioConfig.Files

	eventType := utils.FILE_DELETED

	err := h.restClient.GetChatParticipantIds(c, chatId, func(participantIds []int64) error {
		for _, fileOjInfo := range batch {
			normalizedKey := utils.StripBucketName(fileOjInfo.Key, bucketName)

			fileInfo := &dto.FileInfoDto{
				Id:           normalizedKey,
				FileItemUuid: fileItemUuid,
				LastModified: time.Now().UTC(),
			}

			for _, participantId := range participantIds {
				err := h.publisher.PublishFileEvent(c, participantId, chatId, &dto.WrappedFileInfoDto{
					FileInfoDto: fileInfo,
				}, eventType, nil)
				if err != nil {
					h.lgr.WithTracing(c).Errorf("Error during sending object %v", err)
				}
			}
		}
		return nil
	})
	if err != nil {
		h.lgr.WithTracing(c).Errorf("Error during GetChatParticipantIds %v", err)
	}
}

func (h *FilesHandler) checkFileItemBelongsToUser(fileItemUuid string, c echo.Context, chatId int64, bucketName string, userPrincipalDto *auth.AuthResult) (bool, error) {
	return db.CheckFileItemBelongsToUser(c.Request().Context(), h.dba, chatId, fileItemUuid, userPrincipalDto.UserId)
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

	mcid, err := utils.BuildMetadataCacheId(fileId)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting metadata cache id %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	mce, err := db.Get(c.Request().Context(), h.dba, *mcid, nil)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting metadata cache %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if mce == nil {
		h.lgr.WithTracing(c.Request().Context()).Info("not found metadata cache")
		return c.NoContent(http.StatusNotFound)
	}

	ownerId := mce.OwnerId

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

	publishedUrl, err := h.filesService.GetPublishedUrl(bindTo.Public, fileId)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error get public url: %v", err)
		return err
	}

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "publishedUrl": publishedUrl})
}

type CountResponse struct {
	Count int64 `json:"count"`
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

	filterObj := db.NewFilterBySearchString(searchString)

	count, err := db.GetCount(c.Request().Context(), h.dba, chatId, fileItemUuid, filterObj)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting count %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var countDto = CountResponse{
		Count: count,
	}

	return c.JSON(http.StatusOK, countDto)
}

type FilterRequest struct {
	SearchString string `json:"searchString"`
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

	searchString := bindTo.SearchString
	searchString = strings.TrimSpace(searchString)
	searchString = strings.ToLower(searchString)

	// check user belongs to chat
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

	filename, err := utils.ParseFileName(fileId)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during parsing filename %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	fileItemUuid, err := utils.ParseFileItemUuid(bindTo.FileId)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during parsing fileItemUuid %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	filterObj := db.NewFilterBySearchString(searchString)

	metadataCache, err := db.Get(c.Request().Context(), h.dba, dto.MetadataCacheId{
		ChatId:       chatId,
		FileItemUuid: fileItemUuid,
		Filename:     filename,
	}, filterObj)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting metadataCache getting from db: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var filterResponseItemArray = make([]FilterResponseItem, 0)
	if metadataCache != nil {
		filterResponseItemArray = append(filterResponseItemArray, FilterResponseItem{
			Id: utils.BuildNormalizedKey(metadataCache),
		})
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
	metadata := services.SerializeMetadataSimple(bindTo.OwnerId, nil, &isConferenceRecording, nil, utils.GetUnixMilliUtc())

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

	filterObj := db.NewFilterByType(services.GetTypeExtensions(requestedMediaType))

	items, count, err := h.filesService.GetListFilesInFileItem(c.Request().Context(), false, utils.ChatIdNonExistent, utils.MessageIdNonExistent, &userPrincipalDto.UserId, bucketName, chatId, dto.NoFileItemUuid, filterObj, false, filesSize, filesOffset)
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

	filterObj := db.NewFilterByType(services.GetTypeExtensions(requestedMediaType))

	count, err := db.GetCount(c.Request().Context(), h.dba, chatId, "", filterObj)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting count %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "count": count})
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

	var bindTo = new(CandidatesFilterRequest)
	if err := c.Bind(bindTo); err != nil {
		h.lgr.WithTracing(c.Request().Context()).Warnf("Error during binding to dto %v", err)
		return err
	}

	fileItemUuid, err := utils.ParseFileItemUuid(bindTo.FileId)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during parsing fileItemUuid %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	filename, err := utils.ParseFileName(bindTo.FileId)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during parsing filename %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	filterObj := db.NewFilterByType(services.GetTypeExtensions(bindTo.Type))

	metadataCache, err := db.Get(c.Request().Context(), h.dba, dto.MetadataCacheId{
		ChatId:       chatId,
		FileItemUuid: fileItemUuid,
		Filename:     filename,
	}, filterObj)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting metadataCache getting from db: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var filterResponseItemArray = make([]FilterResponseItem, 0)
	if metadataCache != nil {
		filterResponseItemArray = append(filterResponseItemArray, FilterResponseItem{
			Id: utils.BuildNormalizedKey(metadataCache),
		})
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

	exists, _, err := h.minio.FileExists(c.Request().Context(), bucketName, fileId)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !exists {
		return c.Redirect(http.StatusTemporaryRedirect, NotFoundImage)
	}

	mcid, err := utils.BuildMetadataCacheId(fileId)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting metadata cache id %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	mce, err := db.Get(c.Request().Context(), h.dba, *mcid, nil)
	if err != nil {
		h.lgr.WithTracing(c.Request().Context()).Errorf("Error during getting metadata cache %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if mce == nil {
		h.lgr.WithTracing(c.Request().Context()).Info("not found metadata cache")
		return c.NoContent(http.StatusNotFound)
	}

	chatId := mce.ChatId

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
	downloadUrl, ttl, err := h.filesService.GetTemporaryDownloadUrl(c.Request().Context(), fileId)
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
