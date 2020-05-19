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
	if rows, err := tx.Query(`SELECT * FROM chat LIMIT $1 OFFSET $2`, limit, offset); err != nil {
		return nil, err
	} else {
		defer rows.Close()
		list := make([]models.Chat, 0)
		for rows.Next() {
			ch := models.Chat{}
			err := rows.Scan(&ch.Id, &ch.Title, &ch.OwnerId)
			if err != nil {
				Logger.Error(err)
			} else {
				list = append(list, ch)
			}
		}
		return list, nil
	}

}
