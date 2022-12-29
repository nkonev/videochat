package handlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/minio/minio-go/v7"
	"net/http"
	"net/url"
	"nkonev.name/storage/auth"
	"nkonev.name/storage/client"
	"nkonev.name/storage/dto"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/services"
	"nkonev.name/storage/utils"
	"os/exec"
	"strconv"
	"time"
)

type EmbedHandler struct {
	minio        *minio.Client
	restClient   *client.RestClient
	minioConfig  *utils.MinioConfig
	filesService *services.FilesService
}

const embedMultipartKey = "embed_file_header"
const RelativeEmbeddedUrl = "/api/storage/%v/embed/%v%v"

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
	dotExt := utils.GetDotExtension(formFile)

	GetLogEntry(c.Request().Context()).Debugf("Determined content type: %v", contentType)

	src, err := formFile.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	fileUuid := uuid.New().String()
	filename := fmt.Sprintf("chat/%v/%v%v", chatId, fileUuid, dotExt)

	var userMetadata = services.SerializeMetadata(formFile, userPrincipalDto, chatId)

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
	_, _, fileName, err := services.DeserializeMetadata(objectInfo.UserMetadata, false)
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

const media_image = "image"
const media_video = "video"

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
		case media_image:
			return utils.IsImage(info.Key)
		case media_video:
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
		list = append(list, convert(item, requestedMediaType))
	}

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "files": list, "count": count})
}

func convert(item *dto.FileInfoDto, requestedMediaType string) *MediaDto {
	if item == nil {
		return nil
	}
	var previewUrl *string = nil
	if requestedMediaType == media_image {
		previewUrl = &item.Url
	} else if requestedMediaType == media_video {
		parsedUrl, err := url.Parse(item.Url)
		if err == nil {
			parsedUrl.Path = "/api/storage/preview"
			tmp := parsedUrl.String()
			previewUrl = &tmp
		} else {
			Logger.Errorf("Error during parse url %v", err)
		}
	}
	// TODO video
	//  use 	h.minio.PresignedGetObject() to get an url, then pass it to the ffmpeg ang get an thumbnail
	// TODO research
	//  https://medium.com/@tiwari_nitish/lambda-computing-with-minio-and-kafka-de928897ccdf
	//  https://min.io/docs/minio/linux/administration/monitoring/publish-events-to-amqp.html#minio-bucket-notifications-publish-amqp
	return &MediaDto{
		Id:         item.Id,
		Filename:   item.Filename,
		Url:        item.Url,
		PreviewUrl: previewUrl,
	}
}

func (h *EmbedHandler) PreviewDownloadHandler(c echo.Context) error {

	bucketName := h.minioConfig.Files

	fileId := c.QueryParam("file")

	d, _ := time.ParseDuration("10m")
	presignedUrl, err := h.minio.PresignedGetObject(c.Request().Context(), bucketName, fileId, d, url.Values{})
	if err != nil {
		return err
	}
	stringPresingedUrl := presignedUrl.String()

	ffCmd := exec.Command("ffmpeg",
		"-i", stringPresingedUrl, "-vf", "thumbnail", "-frames:v", "1",
		"-c:v", "png", "-f", "rawvideo", "-an", "-")

	// getting real error msg : https://stackoverflow.com/questions/18159704/how-to-debug-exit-status-1-error-when-running-exec-command-in-golang
	output, err := ffCmd.Output()
	if err != nil {
		Logger.Errorf("Error during getting thumbnail %v", err)
		return err
	}
	return c.Blob(http.StatusOK, "image/png", output)
}
