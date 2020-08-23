package db

import (
	"errors"
	"github.com/google/uuid"
	. "nkonev.name/storage/logger"
	"time"
)

// Go enum
type AvatarType string

const (
	AVATAR_200x200 AvatarType = "AVATAR_200x200"
)

// db model
type FileMetadata struct {
	Id                 uuid.UUID
	FileName           string
	LastUpdateDateTime time.Time
}

func (tx *Tx) CreateFileMetadata(u *FileMetadata, userId int64) (uuid.UUID, error) {
	// Validate the input.
	if u == nil {
		return uuid.New(), errors.New("file_metadata required")
	} else if u.FileName == "" {
		return uuid.New(), errors.New("file name required")
	}

	res := tx.QueryRow(`INSERT INTO file_metadata(file_name, owner_id) VALUES ($1, $2) RETURNING id`, u.FileName, userId)
	var id uuid.UUID
	if err := res.Scan(&id); err != nil {
		Logger.Errorf("Error during creating file metadata %v", err)
		return uuid.New(), err
	}
	return id, nil
}

func (tx *Tx) CreateAvatarMetadata(userId int64, avatarType AvatarType) (error) {
	_, err := tx.Exec(`INSERT INTO avatar_metadata(avatar_type, owner_id) VALUES ($1, $2) ON CONFLICT(owner_id, avatar_type) DO NOTHING`, avatarType, userId)
	if err != nil {
		Logger.Errorf("Error during creating avatar metadata %v", err)
		return err
	}
	return nil
}