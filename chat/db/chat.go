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
func (tx *Tx) CreateChat(u *Chat) (int64, error) {
	// Validate the input.
	if u == nil {
		return 0, errors.New("chat required")
	} else if u.Title == "" {
		return 0, errors.New("title required")
	}

	// Perform the actual insert and return any errors.
	res := tx.QueryRow(`INSERT INTO chat (title, owner_id) VALUES ($1, $2) RETURNING id`, u.Title, u.OwnerId)
	var id int64
	if err := res.Scan(&id); err != nil {
		Logger.Errorf("Error during getting chat id")
		return 0, err
	}
	return id, nil
}

func (db *DB) GetChats(owner int64, limit int, offset int) ([]*Chat, error) {
	if rows, err := db.Query(`SELECT * FROM chat WHERE owner_id = $1 ORDER BY id LIMIT $2 OFFSET $3`, owner, limit, offset); err != nil {
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

func (db *DB) DeleteChat(id int64) error {
	if _, err := db.Exec("DELETE FROM chat WHERE id = $1", id); err != nil {
		Logger.Errorf("Error during delete chat %v", id, err)
		return err
	} else {
		return nil
	}
}

func (tx *Tx) EditChat(id int64, ownerId int64, newTitle string) error {
	if _, err := tx.Exec("UPDATE chat SET title = $3 WHERE id = $1 AND owner_id = $2", id, ownerId, newTitle); err != nil {
		Logger.Errorf("Error during update chat %v", id, err)
		return err
	} else {
		return nil
	}
}

func (db *DB) GetChat(userId int64, chatId int64) (*Chat, error) {
	row := db.QueryRow(`SELECT * FROM chat WHERE owner_id = $1 AND id = $2`, userId, chatId)
	chat := Chat{}
	err := row.Scan(&chat.Id, &chat.Title, &chat.OwnerId)
	if err != nil {
		Logger.Errorf("Error during get chat row", err)
		return nil, err
	} else {
		return &chat, nil
	}
}
