package handlers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/labstack/echo/v4"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
	"image"
	"image/jpeg"
	"net/http"
	"nkonev.name/storage/auth"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/utils"
	"strconv"
	"time"
)

const FormFile = "data"

// Go enum
type AvatarType string

const (
	AVATAR_200x200 AvatarType = "AVATAR_200x200"
	AVATAR_640x640 AvatarType = "AVATAR_640x640"
)

type abstractMethods interface {
	getAvatarFileName(c echo.Context, avatarType AvatarType) (string, error)
	ensureAndGetAvatarBucket() (string, error)
	GetUrlPath() string
}

type abstractAvatarHandler struct {
	minio       *minio.Client
	minioConfig *utils.MinioConfig
	delegate    abstractMethods
}

func (h *abstractAvatarHandler) PutAvatar(c echo.Context) error {
	filePart, err := c.FormFile(FormFile)
	if err != nil {
		GetLogEntry(c.Request()).Errorf("Error during extracting form %v parameter: %v", FormFile, err)
		return err
	}

	bucketName, err := h.delegate.ensureAndGetAvatarBucket()
	if err != nil {
		GetLogEntry(c.Request()).Errorf("Error during get bucket: %v", err)
		return err
	}

	contentType := filePart.Header.Get("Content-Type")

	GetLogEntry(c.Request()).Debugf("Determined content type: %v", contentType)

	src, err := filePart.Open()
	if err != nil {
		GetLogEntry(c.Request()).Errorf("Error during opening multipart file: %v", err)
		return err
	}
	defer src.Close()

	srcImage, _, err := image.Decode(src)
	if err != nil {
		GetLogEntry(c.Request()).Errorf("Error during decoding image: %v", err)
		return err
	}

	currTime := time.Now().Unix()
	filename200, relativeUrl, err := h.putSizedFile(c, srcImage, err, bucketName, contentType, 200, 200, AVATAR_200x200, currTime)
	if err != nil {
		return err
	}
	filename640, relativeBigUrl, err := h.putSizedFile(c, srcImage, err, bucketName, contentType, 640, 640, AVATAR_640x640, currTime)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "filename": filename200, "filenameBig": filename640, "relativeUrl": relativeUrl, "relativeBigUrl": relativeBigUrl})
}

func (h *abstractAvatarHandler) putSizedFile(c echo.Context, srcImage image.Image, err error, bucketName string, contentType string, width, height int, avatarType AvatarType, currTime int64) (string, string, error) {
	dstImage := imaging.Resize(srcImage, width, height, imaging.Lanczos)
	byteBuffer := new(bytes.Buffer)
	err = jpeg.Encode(byteBuffer, dstImage, nil)
	if err != nil {
		GetLogEntry(c.Request()).Errorf("Error during encoding image: %v", err)
		return "", "", err
	}
	filename, err := h.getAvatarFileName(c, avatarType)
	if err != nil {
		GetLogEntry(c.Request()).Errorf("Error during get avatar filename: %v", err)
		return "", "", err
	}
	if _, err := h.minio.PutObject(context.Background(), bucketName, filename, byteBuffer, int64(byteBuffer.Len()), minio.PutObjectOptions{ContentType: contentType}); err != nil {
		GetLogEntry(c.Request()).Errorf("Error during upload object: %v", err)
		return "", "", err
	}
	relativeUrl := fmt.Sprintf("%v%v/%v?time=%v", viper.GetString("server.contextPath"), h.delegate.GetUrlPath(), filename, currTime)

	return filename, relativeUrl, nil
}

func (r *abstractAvatarHandler) getAvatarFileName(c echo.Context, avatarType AvatarType) (string, error) {
	return r.delegate.getAvatarFileName(c, avatarType)
}

func (h *abstractAvatarHandler) Download(c echo.Context) error {
	bucketName, err := h.delegate.ensureAndGetAvatarBucket()
	if err != nil {
		GetLogEntry(c.Request()).Errorf("Error during get bucket: %v", err)
		return err
	}

	objId := c.Param("filename")

	info, e := h.minio.StatObject(context.Background(), bucketName, objId, minio.StatObjectOptions{})
	if e != nil {
		return c.JSON(http.StatusNotFound, &utils.H{"status": "stat fail"})
	}

	c.Response().Header().Set(echo.HeaderContentLength, strconv.FormatInt(info.Size, 10))
	c.Response().Header().Set(echo.HeaderContentType, info.ContentType)
	//c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; Filename=\""+mongoDto.Filename+"\"")

	object, e := h.minio.GetObject(context.Background(), bucketName, objId, minio.GetObjectOptions{})
	defer object.Close()
	if e != nil {
		return c.JSON(http.StatusInternalServerError, &utils.H{"status": "fail"})
	}

	return c.Stream(http.StatusOK, info.ContentType, object)
}

type UserAvatarHandler struct {
	abstractAvatarHandler
}

func NewUserAvatarHandler(minio *minio.Client, minioConfig *utils.MinioConfig) *UserAvatarHandler {
	uah := UserAvatarHandler{}
	uah.minio = minio
	uah.delegate = &uah
	uah.minioConfig = minioConfig
	return &uah
}

const urlStorageGetUserAvatar = "/storage/public/user/avatar"

func (h *UserAvatarHandler) ensureAndGetAvatarBucket() (string, error) {
	return h.minioConfig.UserAvatar, nil
}

func (r *UserAvatarHandler) getAvatarFileName(c echo.Context, avatarType AvatarType) (string, error) {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request()).Errorf("Error during getting auth context")
		return "", errors.New("Error during getting auth context")
	}
	return fmt.Sprintf("%v_%v.jpg", userPrincipalDto.UserId, avatarType), nil
}

func (r *UserAvatarHandler) GetUrlPath() string {
	return urlStorageGetUserAvatar
}

type ChatAvatarHandler struct {
	abstractAvatarHandler
}

func NewChatAvatarHandler(minio *minio.Client, minioConfig *utils.MinioConfig) *ChatAvatarHandler {
	uah := ChatAvatarHandler{}
	uah.minio = minio
	uah.delegate = &uah
	uah.minioConfig = minioConfig
	return &uah
}

const urlStorageGetChatAvatar = "/storage/public/chat/avatar"

func (h *ChatAvatarHandler) ensureAndGetAvatarBucket() (string, error) {
	return h.minioConfig.ChatAvatar, nil
}

func (r *ChatAvatarHandler) getAvatarFileName(c echo.Context, avatarType AvatarType) (string, error) {
	return fmt.Sprintf("%v_%v.jpg", c.Param("chatId"), avatarType), nil
}

func (r *ChatAvatarHandler) GetUrlPath() string {
	return urlStorageGetChatAvatar
}
