package db

import (
	"errors"
	. "github.com/nkonev/videochat/logger"
	"github.com/nkonev/videochat/models"
)

// CreateUser creates a new user.
// Returns an error if user is invalid or the tx fails.
func (tx *Tx) CreateChat(u *models.Chat) error {
	// Validate the input.
	if u == nil {
		return errors.New("chat required")
	} else if u.Title == "" {
		return errors.New("title required")
	}

	// Perform the actual insert and return any errors.
	_, e := tx.Exec(`INSERT INTO chat (title, owner_id) VALUES`, u.Title, u.OwnerId)
	return e
}

func (tx *Tx) GetChats(limit int, offset int) ([]models.Chat, error) {
	if rows, err := tx.Query(`SELECT * FROM chat ORDER BY id LIMIT $1 OFFSET $2`, limit, offset); err != nil {
		return nil, err
	} else {
		defer rows.Close()
		list := make([]models.Chat, 0)
		for rows.Next() {
			chat := models.Chat{}
			if err := rows.Scan(&chat.Id, &chat.Title, &chat.OwnerId); err != nil {
				Logger.Errorf("Error during scan chat rows", err)
				return nil, err
			} else {
				list = append(list, chat)
			}
		}
		return list, nil
	}

}
