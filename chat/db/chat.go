package db

import (
	"errors"
	"fmt"
	. "nkonev.name/chat/logger"
	"strings"
)

// db model
type Chat struct {
	Id    int64
	Title string
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

	res := tx.QueryRow(`INSERT INTO chat (title) VALUES ($1) RETURNING id`, u.Title)
	var id int64
	if err := res.Scan(&id); err != nil {
		Logger.Errorf("Error during getting chat id %v", err)
		return 0, err
	}
	return id, nil
}

func (db *DB) GetChats(chatIds []int64) ([]*Chat, error) {
	chatIdsStrings := make([]string, len(chatIds))
	for i, id := range chatIds {
		chatIdsStrings[i] = fmt.Sprintf("%v", id)
	}
	ids := strings.Join(chatIdsStrings, ", ")
	stmt := fmt.Sprintf(`SELECT * FROM chat WHERE id IN ( %s ) ORDER BY id`, ids)

	if rows, err := db.Query(stmt); err != nil {
		Logger.Errorf("Error during get chat rows %v", err)
		return nil, err
	} else {
		defer rows.Close()
		list := make([]*Chat, 0)
		for rows.Next() {
			chat := Chat{}
			if err := rows.Scan(&chat.Id, &chat.Title); err != nil {
				Logger.Errorf("Error during scan chat rows %v", err)
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

func (tx *Tx) DeleteChat(id int64) error {
	if _, err := tx.Exec("DELETE FROM chat WHERE id = $1", id); err != nil {
		Logger.Errorf("Error during delete chat %v %v", id, err)
		return err
	} else {
		return nil
	}
}

func (tx *Tx) EditChat(id int64, newTitle string) error {
	if _, err := tx.Exec("UPDATE chat SET title = $2 WHERE id = $1", id, newTitle); err != nil {
		Logger.Errorf("Error during update chat %v %v", id, err)
		return err
	} else {
		return nil
	}
}

func (db *DB) GetChat(chatId int64) (*Chat, error) {
	row := db.QueryRow(`SELECT * FROM chat WHERE id = $1`, chatId)
	chat := Chat{}
	err := row.Scan(&chat.Id, &chat.Title)
	if err != nil {
		Logger.Errorf("Error during get chat row %v", err)
		return nil, err
	} else {
		return &chat, nil
	}
}
