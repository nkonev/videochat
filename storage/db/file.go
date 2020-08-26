package db

import (
	. "nkonev.name/storage/logger"
)

// Go enum
type AvatarType string

const (
	AVATAR_200x200 AvatarType = "AVATAR_200x200"
)

func (tx *Tx) CreateAvatarMetadata(userId int64, avatarType AvatarType, filename string) error {
	_, err := tx.Exec(`INSERT INTO avatar_metadata(avatar_type, owner_id, file_name) VALUES ($1, $2, $3) ON CONFLICT(owner_id, avatar_type) DO NOTHING`, avatarType, userId, filename)
	if err != nil {
		Logger.Errorf("Error during creating avatar metadata %v", err)
		return err
	}
	return nil
}
