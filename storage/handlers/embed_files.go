package handlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
	"net/http"
	"nkonev.name/storage/auth"
	"nkonev.name/storage/client"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/utils"
	"strconv"
)

type EmbedHandler struct {
	minio        *minio.Client
	restClient   *client.RestClient
	minioConfig  *utils.MinioConfig
	filesService *FilesService
}

const embedMultipartKey = "embed_file_header"
const RelativeEmbeddedUrl = "/api/storage/%v/embed/%v%v"

func NewEmbedHandler(
	minio *minio.Client,
	restClient *client.RestClient,
	minioConfig *utils.MinioConfig,
	filesService *FilesService,
) *EmbedHandler {
	return &EmbedHandler{
		minio:        minio,
		restClient:   restClient,
		minioConfig:  minioConfig,
		filesService: filesService,
	}
}

func (h *EmbedHandler) UploadHandler(c echo.Context) error {
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

	bucketName := h.minioConfig.Embedded

	formFile, err := c.FormFile(embedMultipartKey)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting multipart part: %v", err)
		return err
	}

	userLimitOk, _, _, err := checkUserLimit(h.minio, bucketName, userPrincipalDto, formFile.Size)
	if err != nil {
		return err
	}
	if !userLimitOk {
		return c.JSON(http.StatusRequestEntityTooLarge, &utils.H{"status": "fail"})
	}

	contentType := formFile.Header.Get("Content-Type")
	dotExt := getDotExtension(formFile)

	GetLogEntry(c.Request().Context()).Debugf("Determined content type: %v", contentType)

	src, err := formFile.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	fileUuid := uuid.New().String()
	filename := fmt.Sprintf("chat/%v/%v%v", chatId, fileUuid, dotExt)

	var userMetadata = serializeMetadata(formFile, userPrincipalDto, chatId)

	if _, err := h.minio.PutObject(context.Background(), bucketName, filename, src, formFile.Size, minio.PutObjectOptions{ContentType: contentType, UserMetadata: userMetadata}); err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during upload object: %v", err)
		return err
	}

	relUrl := fmt.Sprintf(RelativeEmbeddedUrl, chatId, fileUuid, dotExt)

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "relativeUrl": relUrl})
}

func (h *EmbedHandler) DownloadHandler(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	bucketName := h.minioConfig.Embedded

	// check user belongs to chat
	fileWithExt := c.Param("file")
	chatIdString := c.Param("chatId")
	chatId, err := utils.ParseInt64(chatIdString)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during parsing chatId")
		return err
	}

	originalString := c.QueryParam("original")
	original, _ := utils.ParseBoolean(originalString)

	fileId := fmt.Sprintf("chat/%v/%v", chatId, fileWithExt)

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

	c.Response().Header().Set(echo.HeaderContentLength, strconv.FormatInt(objectInfo.Size, 10))
	c.Response().Header().Set(echo.HeaderContentType, objectInfo.ContentType)

	if original {
		c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; Filename=\""+fileName+"\"")
	}

	object, e := h.minio.GetObject(context.Background(), bucketName, fileId, minio.GetObjectOptions{})
	if e != nil {
		return c.JSON(http.StatusInternalServerError, &utils.H{"status": "fail"})
	}
	defer object.Close()

	return c.Stream(http.StatusOK, objectInfo.ContentType, object)
}

type MediaDto struct {
	Id         string  `json:"id"`
	Filename   string  `json:"filename"`
	Url        string  `json:"url"`
	PreviewUrl *string `json:"previewUrl"`
}

func (h *EmbedHandler) DownloadHandlerList(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	filesPage := utils.FixPageString(c.QueryParam("page"))
	filesSize := utils.FixSizeString(c.QueryParam("size"))
	filesOffset := utils.GetOffset(filesPage, filesSize)

	mediaType := c.QueryParam("type")

	var filenameChatPrefix string = ""

	bucketName := h.minioConfig.Files

	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}

	imageTypes := viper.GetStringSlice("types.image")
	videoTypes := viper.GetStringSlice("types.video")

	filter := func(info *minio.ObjectInfo) bool {
		switch mediaType {
		case "image":
			return utils.StringContains(imageTypes, GetDotExtensionStr(info.Key))
		case "video":
			return utils.StringContains(videoTypes, GetDotExtensionStr(info.Key))
		default:
			return false
		}
	}

	items, err := h.filesService.getListFilesInFileItem(userPrincipalDto.UserId, bucketName, filenameChatPrefix, chatId, c.Request().Context(), filter, filesSize, filesOffset)
	if err != nil {
		return err
	}

	var list []*MediaDto = make([]*MediaDto, 0)

	for _, item := range items {
		list = append(list, convert(item))
	}

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "files": list, "count": 4})
}

func convert(item *FileInfoDto) *MediaDto {
	if item == nil {
		return nil
	}
	return &MediaDto{
		Id:         item.Id,
		Filename:   item.Filename,
		Url:        item.Url,
		PreviewUrl: nil, // TODO
	}
}
