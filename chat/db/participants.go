package db

import (
	. "nkonev.name/chat/logger"
)

// db model

type ChatParticipant struct {
	Id     int64
	UserId int64
}

func (tx *Tx) AddParticipant(userId int64, chatId int64, admin bool) error {
	_, err := tx.Exec(`INSERT INTO chat_participant (chat_id, user_id, admin) VALUES ($1, $2, $3)`, chatId, userId, admin)
	return err
}

func (tx *Tx) DeleteParticipantsExcept(userId int64, chatId int64) error {
	_, err := tx.Exec(`DELETE FROM chat_participant WHERE chat_id = $1 AND user_id != $2`, chatId, userId)
	return err
}

func (db *DB) GetParticipantIds(chatId int64) ([]int64, error) {
	if rows, err := db.Query("SELECT user_id FROM chat_participant WHERE chat_id = $1", chatId); err != nil {
		return nil, err
	} else {
		defer rows.Close()
		list := make([]int64, 0)
		for rows.Next() {
			var participantId int64
			if err := rows.Scan(&participantId); err != nil {
				Logger.Errorf("Error during scan chat rows %v", err)
				return nil, err
			} else {
				list = append(list, participantId)
			}
		}
		return list, nil
	}
}

func (tx *Tx) IsAdmin(userId int64, chatId int64) (bool, error) {
	var admin bool = false
	row := tx.QueryRow(`SELECT admin FROM chat_participant WHERE user_id = $1 AND chat_id = $2 LIMIT 1`, userId, chatId)
	if err := row.Scan(&admin); err != nil {
		return false, err
	} else {
		return admin, nil
	}
}

func (db *DB) IsParticipant(userId int64, chatId int64) (bool, error) {
	var exists bool = false
	row := db.QueryRow(`SELECT exists(SELECT 1 FROM chat_participant WHERE user_id = $1 AND chat_id = $2 LIMIT 1)`, userId, chatId)
	if err := row.Scan(&exists); err != nil {
		return false, err
	} else {
		return exists, nil
	}
}
