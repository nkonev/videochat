package db

import (
	"errors"
	. "github.com/nkonev/videochat/logger"
)

// db model
type Chat struct {
	Id      int64  `json:"id"`
	Title   string `json:"title"`
	OwnerId int64  `json:"ownerId"`
}

// CreateUser creates a new user.
// Returns an error if user is invalid or the tx fails.
func (tx *Tx) CreateChat(u *Chat) error {
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

func (tx *Tx) GetChats(owner int64, limit int, offset int) ([]Chat, error) {
	if rows, err := tx.Query(`SELECT * FROM chat WHERE owner_id = $1 ORDER BY id LIMIT $2 OFFSET $3`, owner, limit, offset); err != nil {
		return nil, err
	} else {
		defer rows.Close()
		list := make([]Chat, 0)
		for rows.Next() {
			chat := Chat{}
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
