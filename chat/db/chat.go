package db

import (
	"errors"
	. "nkonev.name/chat/logger"
)

// db model
type Chat struct {
	Id      int64
	Title   string
	OwnerId int64
}

// CreateChat creates a new chat.
// Returns an error if user is invalid or the tx fails.
func (tx *Tx) CreateChat(u *Chat) error {
	// Validate the input.
	if u == nil {
		return errors.New("chat required")
	} else if u.Title == "" {
		return errors.New("title required")
	}

	// Perform the actual insert and return any errors.
	_, e := tx.Exec(`INSERT INTO chat (title, owner_id) VALUES ($1, $2)`, u.Title, u.OwnerId)
	return e
}

func (tx *Tx) GetChats(owner int64, limit int, offset int) ([]*Chat, error) {
	if rows, err := tx.Query(`SELECT * FROM chat WHERE owner_id = $1 ORDER BY id LIMIT $2 OFFSET $3`, owner, limit, offset); err != nil {
		Logger.Errorf("Error during get chat rows", err)
		return nil, err
	} else {
		defer rows.Close()
		list := make([]*Chat, 0)
		for rows.Next() {
			chat := Chat{}
			if err := rows.Scan(&chat.Id, &chat.Title, &chat.OwnerId); err != nil {
				Logger.Errorf("Error during scan chat rows", err)
				return nil, err
			} else {
				list = append(list, &chat)
			}
		}
		return list, nil
	}
}

func (db *DB) CountChats() (int64, error) {
	var count int64
	row := db.QueryRow("SELECT count(*) FROM chat")
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	} else {
		return count, nil
	}
}
