package handlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/minio/minio-go/v7"
	"net/http"
	"nkonev.name/storage/auth"
	"nkonev.name/storage/client"
	"nkonev.name/storage/dto"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/services"
	"nkonev.name/storage/utils"
	"strconv"
)

type EmbedHandler struct {
	minio        *minio.Client
	restClient   *client.RestClient
	minioConfig  *utils.MinioConfig
	filesService *services.FilesService
}

func NewEmbedHandler(
	minio *minio.Client,
	restClient *client.RestClient,
	minioConfig *utils.MinioConfig,
	filesService *services.FilesService,
) *EmbedHandler {
	return &EmbedHandler{
		minio:        minio,
		restClient:   restClient,
		minioConfig:  minioConfig,
		filesService: filesService,
	}
}

type MediaDto struct {
	Id         string  `json:"id"`
	Filename   string  `json:"filename"`
	Url        string  `json:"url"`
	PreviewUrl *string `json:"previewUrl"`
}

func (h *EmbedHandler) ListCandidatesForEmbed(c echo.Context) error {
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
	belongs, err := h.restClient.CheckAccess(userPrincipalDto.UserId, chatId, c.Request().Context())
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

	items, count, err := h.filesService.GetListFilesInFileItem(userPrincipalDto.UserId, bucketName, filenameChatPrefix, chatId, c.Request().Context(), filter, false, false, filesSize, filesOffset)
	if err != nil {
		return err
	}

	var list []*MediaDto = make([]*MediaDto, 0)

	for _, item := range items {
		list = append(list, convert(item, requestedMediaType))
	}

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "files": list, "count": count})
}

func convert(item *dto.FileInfoDto, requestedMediaType string) *MediaDto {
	if item == nil {
		return nil
	}
	var previewUrl *string = services.GetPreviewUrl(item.Url, requestedMediaType)

	return &MediaDto{
		Id:         item.Id,
		Filename:   item.Filename,
		Url:        item.Url,
		PreviewUrl: previewUrl,
	}
}

func (h *EmbedHandler) PreviewDownloadHandler(c echo.Context) error {
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
		GetLogEntry(c.Request().Context()).Errorf("Error during getting object %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	chatId, err := utils.ParseChatId(fileId)
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

	object, e := h.minio.GetObject(context.Background(), bucketName, fileId, minio.GetObjectOptions{})
	if e != nil {
		return c.JSON(http.StatusInternalServerError, &utils.H{"status": "fail"})
	}
	defer object.Close()

	c.Response().Header().Set(echo.HeaderContentLength, strconv.FormatInt(objectInfo.Size, 10))
	c.Response().Header().Set(echo.HeaderContentType, objectInfo.ContentType)

	return c.Stream(http.StatusOK, objectInfo.ContentType, object)
}
