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
	"nkonev.name/storage/db"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/utils"
	"strconv"
)

type FileHandler struct {
	db          db.DB
	minio		*minio.Client
}

func NewFileHandler (db db.DB, minio *minio.Client) FileHandler {
	return FileHandler{
		db: db,
		minio: minio,
	}
}

const FormFile = "data"

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

	dstImage := imaging.Resize(srcImage, 200, 200, imaging.Lanczos)

	byteBuffer := new(bytes.Buffer)
	err = jpeg.Encode(byteBuffer, dstImage, nil)
	if err != nil {
		GetLogEntry(c.Request()).Errorf("Error during encoding image: %v", err)
		return err
	}

	avatarType := db.AVATAR_200x200
	filename := fmt.Sprintf("%v_%v.jpg", userPrincipalDto.UserId, avatarType)
	err = db.Transact(fh.db, func(tx *db.Tx) (error) {
		return tx.CreateAvatarMetadata(userPrincipalDto.UserId, avatarType, filename)
	})
	if err != nil {
		GetLogEntry(c.Request()).Errorf("Error during inserting into database: %v", err)
		return err
	}

	if _, err := fh.minio.PutObject(context.Background(), bucketName, filename, byteBuffer, int64(byteBuffer.Len()), minio.PutObjectOptions{ContentType: contentType}); err != nil {
		GetLogEntry(c.Request()).Errorf("Error during upload object: %v", err)
		return err
	}

	relativeUrl := fmt.Sprintf("%v/storage/public/avatar/%v", viper.GetString("server.contextPath"), filename)

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "filename": filename, "relativeUrl": relativeUrl})
}

func (h *FileHandler) Download(c echo.Context) error {
	bucketName, err := h.ensureAndGetAvatarBucket()
	if err != nil {
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

