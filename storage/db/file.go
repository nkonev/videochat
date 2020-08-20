package db

import (
	"errors"
	. "nkonev.name/storage/logger"
	"time"
)

// Go enum
type StorageType int

const (
	AVATAR StorageType = iota
	FILE
)

// db model
type FileMetadata struct {
	Id                 int64
	FileName           string
	StorageType		   StorageType
	LastUpdateDateTime time.Time
}

func (tx *Tx) CreateFileMetadata(u *FileMetadata, userId int64) (int64, error) {
	// Validate the input.
	if u == nil {
		return 0, errors.New("file_metadata required")
	} else if u.FileName == "" {
		return 0, errors.New("file name required")
	}

	res := tx.QueryRow(`INSERT INTO file_metadata(storage_type, file_name, owner_id) VALUES ($1, $2) RETURNING id`, u.StorageType, u.FileName, userId)
	var id int64
	if err := res.Scan(&id); err != nil {
		Logger.Errorf("Error during getting chat id %v", err)
		return 0, err
	}
	return id, nil
}