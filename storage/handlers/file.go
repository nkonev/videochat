package handlers

import (
	"context"
	"errors"
	"github.com/labstack/echo/v4"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
	"net/http"
	"nkonev.name/storage/auth"
	"nkonev.name/storage/db"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/utils"
	"time"
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

const FormFile = "file"

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

func (h *FileHandler) ensureAndGetBucket() (string, error) {
	bucketName := viper.GetString("minio.avatars.bucket")
	bucketLocation := viper.GetString("minio.location")
	err := h.ensureBucket(bucketName, bucketLocation)
	return bucketName, err
}


func (fh *FileHandler) PutFile(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}


	file, err := c.FormFile(FormFile)
	if err != nil {
		Logger.Errorf("Error during extracting form %v parameter: %v", FormFile, err)
		return err
	}

	bucketName, err := fh.ensureAndGetBucket()
	if err != nil {
		return err
	}

	contentType := file.Header.Get("Content-Type")

	Logger.Debugf("Determined content type: %v", contentType)

	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	result, err := db.TransactWithResult(fh.db, func(tx *db.Tx) (interface{}, error) {
		metadata := db.FileMetadata{Name: file.Filename}
		id, err := tx.CreateFileMetadata(&metadata, userPrincipalDto.UserId)
		if err != nil {
			return 0, err
		} else {
			return id, nil
		}
	})
	id := result.(int64)

	if _, err := fh.minio.PutObject(context.Background(), bucketName, id, src, file.Size, minio.PutObjectOptions{ContentType: contentType}); err != nil {
		Logger.Errorf("Error during upload object: %v", err)
		return err
	}

	return c.JSON(http.StatusOK, &utils.H{"status": "ok", "id": id})
}
