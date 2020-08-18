package db

import (
	"errors"
	. "nkonev.name/storage/logger"
	"time"
)

// db model
type FileMetadata struct {
	Id                 int64
	Name               string
	LastUpdateDateTime time.Time
}

func (tx *Tx) CreateFileMetadata(u *FileMetadata, userId int64) (int64, error) {
	// Validate the input.
	if u == nil {
		return 0, errors.New("file_metadata required")
	} else if u.Name == "" {
		return 0, errors.New("file name required")
	}

	res := tx.QueryRow(`INSERT INTO file_metadata(name, owner_id) VALUES ($1, $2) RETURNING id`, u.Name, userId)
	var id int64
	if err := res.Scan(&id); err != nil {
		Logger.Errorf("Error during getting chat id %v", err)
		return 0, err
	}
	return id, nil
}