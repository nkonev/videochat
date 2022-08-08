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

func (tx *Tx) DeleteParticipant(userId int64, chatId int64) error {
	_, err := tx.Exec(`DELETE FROM chat_participant WHERE chat_id = $1 AND user_id = $2`, chatId, userId)
	return err
}

func getParticipantIdsCommon(qq CommonOperations, chatId int64, participantsSize, participantsOffset int) ([]int64, error) {
	if rows, err := qq.Query("SELECT user_id FROM chat_participant WHERE chat_id = $1 ORDER BY user_id LIMIT $2 OFFSET $3", chatId, participantsSize, participantsOffset); err != nil {
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

func (tx *Tx) GetParticipantIds(chatId int64, participantsSize, participantsOffset int) ([]int64, error) {
	return getParticipantIdsCommon(tx, chatId, participantsSize, participantsOffset)
}

func (db *DB) GetParticipantIds(chatId int64, participantsSize, participantsOffset int) ([]int64, error) {
	return getParticipantIdsCommon(db, chatId, participantsSize, participantsOffset)
}

func getAllParticipantIdsCommon(qq CommonOperations, chatId int64) ([]int64, error) {
	if rows, err := qq.Query("SELECT user_id FROM chat_participant WHERE chat_id = $1 ORDER BY user_id", chatId); err != nil {
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

func (tx *Tx) GetAllParticipantIds(chatId int64) ([]int64, error) {
	return getAllParticipantIdsCommon(tx, chatId)
}

func (db *DB) GetAllParticipantIds(chatId int64) ([]int64, error) {
	return getAllParticipantIdsCommon(db, chatId)
}

func getParticipantsCountCommon(qq CommonOperations, chatId int64) (int, error) {
	var count int
	row := qq.QueryRow("SELECT count(*) FROM chat_participant WHERE chat_id = $1", chatId)

	if err := row.Scan(&count); err != nil {
		return 0, err
	} else {
		return count, nil
	}
}

func (tx *Tx) GetParticipantsCount(chatId int64) (int, error) {
	return getParticipantsCountCommon(tx, chatId)
}

func (db *DB) GetParticipantsCount(chatId int64) (int, error) {
	return getParticipantsCountCommon(db, chatId)
}

func getIsAdminCommon(qq CommonOperations, userId int64, chatId int64) (bool, error) {
	var admin bool = false
	row := qq.QueryRow(`SELECT exists(SELECT * FROM chat_participant WHERE user_id = $1 AND chat_id = $2 AND admin = true LIMIT 1)`, userId, chatId)
	if err := row.Scan(&admin); err != nil {
		return false, err
	} else {
		return admin, nil
	}
}

func (tx *Tx) IsAdmin(userId int64, chatId int64) (bool, error) {
	return getIsAdminCommon(tx, userId, chatId)
}

func (db *DB) IsAdmin(userId int64, chatId int64) (bool, error) {
	return getIsAdminCommon(db, userId, chatId)
}

func isParticipantCommon(qq CommonOperations, userId int64, chatId int64) (bool, error) {
	var exists bool = false
	row := qq.QueryRow(`SELECT exists(SELECT * FROM chat_participant WHERE user_id = $1 AND chat_id = $2 LIMIT 1)`, userId, chatId)
	if err := row.Scan(&exists); err != nil {
		return false, err
	} else {
		return exists, nil
	}
}

func (tx *Tx) IsParticipant(userId int64, chatId int64) (bool, error) {
	return isParticipantCommon(tx, userId, chatId)
}

func (db *DB) IsParticipant(userId int64, chatId int64) (bool, error) {
	return isParticipantCommon(db, userId, chatId)
}

func (tx *Tx) GetFirstParticipant(chatId int64) (int64, error) {
	var pid int64
	row := tx.QueryRow(`SELECT user_id FROM chat_participant WHERE chat_id = $1 LIMIT 1`, chatId)
	if err := row.Scan(&pid); err != nil {
		return 0, err
	} else {
		return pid, nil
	}
}

func (db *DB) GetCoChattedParticipantIdsCommon(participantId int64) ([]int64, error) {
	if rows, err := db.Query("SELECT DISTINCT user_id FROM chat_participant WHERE chat_id IN (SELECT chat_id FROM chat_participant WHERE user_id = $1) ORDER BY user_id", participantId); err != nil {
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

func setAdminCommon(qq CommonOperations, userId int64, chatId int64, newAdmin bool) error {
	if _, err := qq.Exec("UPDATE chat_participant SET admin = $3 WHERE user_id = $1 AND chat_id = $2", userId, chatId, newAdmin); err != nil {
		Logger.Errorf("Error during editing participant admin id %v", err)
		return err
	}
	return nil
}

func (tx *Tx) SetAdmin(userId int64, chatId int64, newAdmin bool) error {
	return setAdminCommon(tx, userId, chatId, newAdmin)
}

func (db *DB) SetAdmin(userId int64, chatId int64, newAdmin bool) error {
	return setAdminCommon(db, userId, chatId, newAdmin)
}
