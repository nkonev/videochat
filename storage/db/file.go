package db

import (
	"errors"
	"github.com/google/uuid"
	. "nkonev.name/storage/logger"
	"time"
)

// Go enum
type StorageType string

const (
	AVATAR StorageType = "AVATAR"
	FILE StorageType = "FILE"
)

// db model
type FileMetadata struct {
	Id                 uuid.UUID
	FileName           string
	StorageType		   StorageType
	LastUpdateDateTime time.Time
}

func (tx *Tx) CreateFileMetadata(u *FileMetadata, userId int64) (uuid.UUID, error) {
	// Validate the input.
	if u == nil {
		return uuid.New(), errors.New("file_metadata required")
	} else if u.FileName == "" {
		return uuid.New(), errors.New("file name required")
	}

	res := tx.QueryRow(`INSERT INTO file_metadata(storage_type, file_name, owner_id) VALUES ($1, $2, $3) RETURNING id`, u.StorageType, u.FileName, userId)
	var id uuid.UUID
	if err := res.Scan(&id); err != nil {
		Logger.Errorf("Error during getting chat id %v", err)
		return uuid.New(), err
	}
	return id, nil
}