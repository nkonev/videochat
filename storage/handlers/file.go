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
)

type FileHandler struct {
	minio		*minio.Client
}

func NewFileHandler (minio *minio.Client) FileHandler {
	return FileHandler{
		minio: minio,
	}
}

const FormFile = "data"
const UrlStorageGetAvatar = "/storage/public/avatar"

func (h *FileHandler) ensureBucket(bucketName, location string) error {
	// Check to see if we already own this bucket (which happens if you run this twice)
	exists, err := h.minio.BucketExists(context.Background(), bucketName)
	if err == nil && exists {
		Logger.Debugf("Bucket '%s' already present", bucketName)
		return nil
	} else if err != nil {
		return err
	} else {
		if err := h.minio.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{
			Region:        location,
			ObjectLocking: false,
		}); err != nil {
			return err
		} else {
			Logger.Infof("Successfully created bucket '%s'", bucketName)
			return nil
		}
	}
}

func (h *FileHandler) ensureAndGetAvatarBucket() (string, error) {
	bucketName := viper.GetString("minio.bucket.avatar")
	bucketLocation := viper.GetString("minio.location")
	err := h.ensureBucket(bucketName, bucketLocation)
	return bucketName, err
}


// Go enum
type AvatarType string

const (
	AVATAR_200x200 AvatarType = "AVATAR_200x200"
	AVATAR_640x640 AvatarType = "AVATAR_640x640"
)

func (fh *FileHandler) PutAvatar(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	filePart, err := c.FormFile(FormFile)
	if err != nil {
		GetLogEntry(c.Request()).Errorf("Error during extracting form %v parameter: %v", FormFile, err)
		return err
	}

	bucketName, err := fh.ensureAndGetAvatarBucket()
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

	filename200, err := fh.putSizedFile(c, srcImage, err, userPrincipalDto, bucketName, contentType, 200, 200, AVATAR_200x200)
	if err != nil {
		return err
	}
	filename640, err := fh.putSizedFile(c, srcImage, err, userPrincipalDto, bucketName, contentType, 640, 640, AVATAR_640x640)
	if err != nil {
		return err
	}

	relativeUrl := fmt.Sprintf("%v%v/%v", viper.GetString("server.contextPath"), UrlStorageGetAvatar, filename200)
	relativeBigUrl := fmt.Sprintf("%v%v/%v", viper.GetString("server.contextPath"), UrlStorageGetAvatar, filename640)

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "filename": filename200, "filenameBig": filename640, "relativeUrl" : relativeUrl, "relativeBigUrl": relativeBigUrl})
}

func (fh *FileHandler) putSizedFile(c echo.Context, srcImage image.Image, err error, userPrincipalDto *auth.AuthResult, bucketName string, contentType string, width, height int, avatarType AvatarType) (string, error) {
	dstImage := imaging.Resize(srcImage, width, height, imaging.Lanczos)
	byteBuffer := new(bytes.Buffer)
	err = jpeg.Encode(byteBuffer, dstImage, nil)
	if err != nil {
		GetLogEntry(c.Request()).Errorf("Error during encoding image: %v", err)
		return "", err
	}
	filename := fmt.Sprintf("%v_%v.jpg", userPrincipalDto.UserId, avatarType)
	if _, err := fh.minio.PutObject(context.Background(), bucketName, filename, byteBuffer, int64(byteBuffer.Len()), minio.PutObjectOptions{ContentType: contentType}); err != nil {
		GetLogEntry(c.Request()).Errorf("Error during upload object: %v", err)
		return "", err
	}
	return filename, nil
}

func (h *FileHandler) Download(c echo.Context) error {
	bucketName, err := h.ensureAndGetAvatarBucket()
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

