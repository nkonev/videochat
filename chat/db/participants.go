package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/ztrue/tracerr"
	"nkonev.name/chat/utils"
)

// db model

type ChatParticipant struct {
	Id     int64
	UserId int64
}

func (tx *Tx) AddParticipant(userId int64, chatId int64, admin bool) error {
	_, err := tx.Exec(`INSERT INTO chat_participant (chat_id, user_id, admin) VALUES ($1, $2, $3)`, chatId, userId, admin)
	return tracerr.Wrap(err)
}

func (tx *Tx) DeleteParticipant(userId int64, chatId int64) error {
	_, err := tx.Exec(`DELETE FROM chat_participant WHERE chat_id = $1 AND user_id = $2`, chatId, userId)
	return tracerr.Wrap(err)
}

func getParticipantIdsCommon(qq CommonOperations, chatId int64, participantsSize, participantsOffset int) ([]int64, error) {
	if rows, err := qq.Query("SELECT user_id FROM chat_participant WHERE chat_id = $1 ORDER BY user_id LIMIT $2 OFFSET $3", chatId, participantsSize, participantsOffset); err != nil {
		return nil, tracerr.Wrap(err)
	} else {
		defer rows.Close()
		list := make([]int64, 0)
		for rows.Next() {
			var participantId int64
			if err := rows.Scan(&participantId); err != nil {
				return nil, tracerr.Wrap(err)
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

func getParticipantIdsBatchCommon(qq CommonOperations, chatIds []int64, participantsSize int) ([]*ParticipantIds, error) {
	res := make([]*ParticipantIds, 0)
	if len(chatIds) == 0 {
		return res, nil
	}

	var builder = ""
	var first = true
	for _, chatId := range chatIds {
		if !first {
			builder += ", "
		}
		builder += utils.Int64ToString(chatId)
		first = false
	}
	if rows, err := qq.Query(fmt.Sprintf("SELECT chat_id, jsonb_agg(user_id) FROM chat_participant WHERE chat_id in (%v) group by chat_id;", builder)); err != nil {
		return nil, tracerr.Wrap(err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var pi = new(ParticipantIds)
			var arr string
			if err := rows.Scan(&pi.ChatId, &arr); err != nil {
				return nil, tracerr.Wrap(err)
			} else {
				err := json.Unmarshal([]byte(arr), &pi.ParticipantIds)
				if err != nil {
					return nil, tracerr.Wrap(err)
				}

				pi.ParticipantIds = pi.ParticipantIds[:utils.Min(len(pi.ParticipantIds), int(participantsSize))]
				res = append(res, pi)
			}
		}
		return res, nil
	}
}

func (tx *Tx) GetParticipantIdsBatch(chatIds []int64, participantsSize int) ([]*ParticipantIds, error) {
	return getParticipantIdsBatchCommon(tx, chatIds, participantsSize)
}

func (db *DB) GetParticipantIdsBatch(chatIds []int64, participantsSize int) ([]*ParticipantIds, error) {
	return getParticipantIdsBatchCommon(db, chatIds, participantsSize)
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
				return nil, tracerr.Wrap(err)
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
		return 0, tracerr.Wrap(err)
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

func getParticipantsCountBatchCommon(qq CommonOperations, chatIds []int64) (map[int64]int, error) {
	res := map[int64]int{}

	if len(chatIds) == 0 {
		return res, nil
	}

	var builder = ""
	var first = true
	for _, chatId := range chatIds {
		if !first {
			builder += " union "
		}
		builder += fmt.Sprintf("(SELECT %v, count(*) FROM chat_participant WHERE chat_id = %v)", chatId, chatId)

		first = false
	}

	var rows *sql.Rows
	var err error
	rows, err = qq.Query(builder)
	if err != nil {
		return nil, tracerr.Wrap(err)
	} else {
		defer rows.Close()
		for _, cid := range chatIds {
			res[cid] = 0
		}
		for rows.Next() {
			var chatId int64
			var count int
			if err := rows.Scan(&chatId, &count); err != nil {
				return nil, tracerr.Wrap(err)
			} else {
				res[chatId] = count
			}
		}
		return res, nil
	}
}

func (tx *Tx) GetParticipantsCountBatch(chatIds []int64) (map[int64]int, error) {
	return getParticipantsCountBatchCommon(tx, chatIds)
}

func (db *DB) GetParticipantsCountBatch(chatIds []int64) (map[int64]int, error) {
	return getParticipantsCountBatchCommon(db, chatIds)
}

func getIsAdminCommon(qq CommonOperations, userId int64, chatId int64) (bool, error) {
	var admin bool = false
	row := qq.QueryRow(`SELECT exists(SELECT * FROM chat_participant WHERE user_id = $1 AND chat_id = $2 AND admin = true LIMIT 1)`, userId, chatId)
	if err := row.Scan(&admin); err != nil {
		return false, tracerr.Wrap(err)
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

func getIsAdminBatchCommon(qq CommonOperations, userId int64, chatIds []int64) (map[int64]bool, error) {
	var result = map[int64]bool{}

	if len(chatIds) == 0 {
		return result, nil
	}

	for _, chatId := range chatIds {
		result[chatId] = false // prefill all with false
	}

	var builder = ""
	var first = true
	for _, chatId := range chatIds {
		if !first {
			builder += ", "
		}
		builder += utils.Int64ToString(chatId)
		first = false
	}

	if rows, err := qq.Query(fmt.Sprintf(`SELECT chat_id, admin FROM chat_participant WHERE user_id = $1 AND chat_id IN (%v) AND admin = true`, builder), userId); err != nil {
		return nil, tracerr.Wrap(err)
	} else {
		defer rows.Close()

		for rows.Next() {
			var admin bool = false
			var chatId int64 = 0
			if err := rows.Scan(&chatId, &admin); err != nil {
				return nil, tracerr.Wrap(err)
			} else {
				result[chatId] = admin
			}
		}
		return result, nil
	}
}

func (tx *Tx) IsAdminBatch(userId int64, chatIds []int64) (map[int64]bool, error) {
	return getIsAdminBatchCommon(tx, userId, chatIds)
}

func (db *DB) IsAdminBatch(userId int64, chatIds []int64) (map[int64]bool, error) {
	return getIsAdminBatchCommon(db, userId, chatIds)
}

func isParticipantCommon(qq CommonOperations, userId int64, chatId int64) (bool, error) {
	var exists bool = false
	row := qq.QueryRow(`SELECT exists(SELECT * FROM chat_participant WHERE user_id = $1 AND chat_id = $2 LIMIT 1)`, userId, chatId)
	if err := row.Scan(&exists); err != nil {
		return false, tracerr.Wrap(err)
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
		return 0, tracerr.Wrap(err)
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
				return nil, tracerr.Wrap(err)
			} else {
				list = append(list, participantId)
			}
		}
		return list, nil
	}
}

func setAdminCommon(qq CommonOperations, userId int64, chatId int64, newAdmin bool) error {
	if _, err := qq.Exec("UPDATE chat_participant SET admin = $3 WHERE user_id = $1 AND chat_id = $2", userId, chatId, newAdmin); err != nil {
		return tracerr.Wrap(err)
	}
	return nil
}

func (tx *Tx) SetAdmin(userId int64, chatId int64, newAdmin bool) error {
	return setAdminCommon(tx, userId, chatId, newAdmin)
}

func (db *DB) SetAdmin(userId int64, chatId int64, newAdmin bool) error {
	return setAdminCommon(db, userId, chatId, newAdmin)
}

func (db *DB) GetChatsWithMe(userId int64) ([]int64, error) {
	if rows, err := db.Query("SELECT DISTINCT chat_id FROM chat_participant WHERE user_id = $1", userId); err != nil {
		return nil, tracerr.Wrap(err)
	} else {
		defer rows.Close()
		list := make([]int64, 0)
		for rows.Next() {
			var chatId int64
			if err := rows.Scan(&chatId); err != nil {
				return nil, tracerr.Wrap(err)
			} else {
				list = append(list, chatId)
			}
		}
		return list, nil
	}

}
