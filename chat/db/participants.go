package db

import (
	"database/sql"
	"fmt"
	"github.com/rotisserie/eris"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
)

// db model

type ChatParticipant struct {
	Id     int64
	UserId int64
}

func (tx *Tx) AddParticipant(userId int64, chatId int64, admin bool) error {
	_, err := tx.Exec(`INSERT INTO chat_participant (chat_id, user_id, admin) VALUES ($1, $2, $3)`, chatId, userId, admin)
	return eris.Wrap(err, "error during interacting with db")
}

func (tx *Tx) DeleteParticipant(userId int64, chatId int64) error {
	_, err := tx.Exec(`DELETE FROM chat_participant WHERE chat_id = $1 AND user_id = $2`, chatId, userId)
	return eris.Wrap(err, "error during interacting with db")
}

func (tx *Tx) DeleteUserAsAParticipantFromAllChats(userId int64) error {
	_, err := tx.Exec(`DELETE FROM chat_participant WHERE user_id = $1`, userId)
	return eris.Wrap(err, "error during interacting with db")
}

func getParticipantIdsCommon(qq CommonOperations, chatId int64, participantsSize, participantsOffset int) ([]int64, error) {
	if rows, err := qq.Query("SELECT user_id FROM chat_participant WHERE chat_id = $1 ORDER BY create_date_time DESC LIMIT $2 OFFSET $3", chatId, participantsSize, participantsOffset); err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		defer rows.Close()
		list := make([]int64, 0)
		for rows.Next() {
			var participantId int64
			if err := rows.Scan(&participantId); err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
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

	if rows, err := qq.Query(`
		select ch.chat_id, usr.user_id from 
		   ( select distinct(cp.chat_id) from chat_participant cp where cp.chat_id = any($2) ) ch
		   join lateral ( select cpi.user_id from chat_participant cpi where cpi.chat_id = ch.chat_id order by create_date_time desc limit $1 ) usr on true 
		   order by chat_id;
	`, participantsSize, chatIds); err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		defer rows.Close()
		prevChatId := int64(-1)
		var aParticipantIds *ParticipantIds
		for rows.Next() {
			var userId, chatId int64
			if err := rows.Scan(&chatId, &userId); err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			} else {
				if chatId != prevChatId {
					if aParticipantIds != nil {
						res = append(res, aParticipantIds)
					}
					prevChatId = chatId
					aParticipantIds = new(ParticipantIds)
					aParticipantIds.ChatId = chatId
				}
				aParticipantIds.ParticipantIds = append(aParticipantIds.ParticipantIds, userId)
			}
		}
		if aParticipantIds != nil {
			res = append(res, aParticipantIds)
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

func getChatParticipantIdsCommon(qq CommonOperations, chatId int64, consumer func(participantIds []int64) error) error {
	shouldContinue := true
	var lastError error
	for page := 0; shouldContinue; page++ {
		offset := utils.GetOffset(page, utils.DefaultSize)
		participantIds, err := getParticipantIdsCommon(qq, chatId, utils.DefaultSize, offset)
		if err != nil {
			logger.Logger.Errorf("Got error during getting portion %v", err)
			lastError = err
			break
		}
		if len(participantIds) == 0 {
			return nil
		}
		if len(participantIds) < utils.DefaultSize {
			shouldContinue = false
		}
		err = consumer(participantIds)
		if err != nil {
			logger.Logger.Errorf("Got error during invoking consumer portion %v", err)
			lastError = err
			break
		}
	}
	return lastError
}

func (tx *Tx) IterateOverChatParticipantIds(chatId int64, consumer func(participantIds []int64) error) error {
	return getChatParticipantIdsCommon(tx, chatId, consumer)
}

func (db *DB) IterateOverChatParticipantIds(chatId int64, consumer func(participantIds []int64) error) error {
	return getChatParticipantIdsCommon(db, chatId, consumer)
}

func getPortionOfAllParticipantIdsCommon(qq CommonOperations, participantsSize, participantsOffset int) ([]int64, error) {
	if rows, err := qq.Query("SELECT distinct (user_id) FROM chat_participant ORDER BY user_id LIMIT $1 OFFSET $2", participantsSize, participantsOffset); err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		defer rows.Close()
		list := make([]int64, 0)
		for rows.Next() {
			var participantId int64
			if err := rows.Scan(&participantId); err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			} else {
				list = append(list, participantId)
			}
		}
		return list, nil
	}
}

func getAllParticipantIdsCommon(qq CommonOperations, consumer func(participantIds []int64) error) error {
	shouldContinue := true
	var lastError error
	for page := 0; shouldContinue; page++ {
		offset := utils.GetOffset(page, utils.DefaultSize)
		participantIds, err := getPortionOfAllParticipantIdsCommon(qq, utils.DefaultSize, offset)
		if err != nil {
			logger.Logger.Errorf("Got error during getting portion %v", err)
			lastError = err
			break
		}

		if len(participantIds) == 0 {
			return nil
		}
		if len(participantIds) < utils.DefaultSize {
			shouldContinue = false
		}
		err = consumer(participantIds)
		if err != nil {
			logger.Logger.Errorf("Got error during invoking consumer portion %v", err)
			lastError = err
			break
		}
	}
	return lastError
}

func (tx *Tx) IterateOverAllParticipantIds(consumer func(participantIds []int64) error) error {
	return getAllParticipantIdsCommon(tx, consumer)
}

func (db *DB) IterateOverAllParticipantIds(consumer func(participantIds []int64) error) error {
	return getAllParticipantIdsCommon(db, consumer)
}

func getParticipantsCountCommon(qq CommonOperations, chatId int64) (int, error) {
	var count int
	row := qq.QueryRow("SELECT count(*) FROM chat_participant WHERE chat_id = $1", chatId)

	if err := row.Scan(&count); err != nil {
		return 0, eris.Wrap(err, "error during interacting with db")
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
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		defer rows.Close()
		for _, cid := range chatIds {
			res[cid] = 0
		}
		for rows.Next() {
			var chatId int64
			var count int
			if err := rows.Scan(&chatId, &count); err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
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
		return false, eris.Wrap(err, "error during interacting with db")
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
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		defer rows.Close()

		for rows.Next() {
			var admin bool = false
			var chatId int64 = 0
			if err := rows.Scan(&chatId, &admin); err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
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


func getIsAdminBatchByParticipantsCommon(qq CommonOperations, userIds []int64, chatId int64) ([]UserAdminDbDTO, error) {
	var result = []UserAdminDbDTO{}

	if len(userIds) == 0 {
		return result, nil
	}

	if rows, err := qq.Query(fmt.Sprintf(`SELECT user_id, admin FROM chat_participant WHERE user_id = ANY($1) AND chat_id = $2 ORDER BY create_date_time DESC`), userIds, chatId); err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		defer rows.Close()

		for rows.Next() {
			var admin bool = false
			var userId int64 = 0
			if err := rows.Scan(&userId, &admin); err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			} else {
				result = append(result, UserAdminDbDTO{userId, admin})
			}
		}
		return result, nil
	}
}
func (db *DB) IsAdminBatchByParticipants(userIds []int64, chatId int64) ([]UserAdminDbDTO, error) {
	return getIsAdminBatchByParticipantsCommon(db, userIds, chatId)
}

func (tx *Tx) IsAdminBatchByParticipants(userIds []int64, chatId int64) ([]UserAdminDbDTO, error) {
	return getIsAdminBatchByParticipantsCommon(tx, userIds, chatId)
}

func isParticipantCommon(qq CommonOperations, userId int64, chatId int64) (bool, error) {
	var exists bool = false
	row := qq.QueryRow(`SELECT exists(SELECT * FROM chat_participant WHERE user_id = $1 AND chat_id = $2 LIMIT 1)`, userId, chatId)
	if err := row.Scan(&exists); err != nil {
		return false, eris.Wrap(err, "error during interacting with db")
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
		return 0, eris.Wrap(err, "error during interacting with db")
	} else {
		return pid, nil
	}
}

func (db *DB) GetCoChattedParticipantIdsCommon(participantId int64, limit, offset int) ([]int64, error) {
	if rows, err := db.Query("SELECT DISTINCT user_id FROM chat_participant WHERE chat_id IN (SELECT chat_id FROM chat_participant WHERE user_id = $1) ORDER BY user_id LIMIT $2 OFFSET $3", participantId, limit, offset); err != nil {
		return nil, err
	} else {
		defer rows.Close()
		list := make([]int64, 0)
		for rows.Next() {
			var coParticipantId int64
			if err := rows.Scan(&coParticipantId); err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			} else {
				list = append(list, coParticipantId)
			}
		}
		return list, nil
	}
}

// returns only found participant ids
func (db *DB) ParticipantsExistence(chatId int64, participantIds []int64) ([]int64, error) {
	if rows, err := db.Query("SELECT user_id FROM chat_participant WHERE chat_id = $1 AND user_id = ANY ($2)", chatId, participantIds); err != nil {
		return nil, err
	} else {
		defer rows.Close()
		list := make([]int64, 0)
		for rows.Next() {
			var coParticipantId int64
			if err := rows.Scan(&coParticipantId); err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			} else {
				list = append(list, coParticipantId)
			}
		}
		return list, nil
	}
}

func (db *DB) IterateOverCoChattedParticipantIds(participantId int64, consumer func(participantIds []int64) error) error {
	shouldContinue := true
	var lastError error
	for page := 0; shouldContinue; page++ {
		offset := utils.GetOffset(page, utils.DefaultSize)
		participantIds, err := db.GetCoChattedParticipantIdsCommon(participantId, utils.DefaultSize, offset)
		if err != nil {
			logger.Logger.Errorf("Got error during getting portion %v", err)
			lastError = err
			break
		}
		if len(participantIds) == 0 {
			return nil
		}
		if len(participantIds) < utils.DefaultSize {
			shouldContinue = false
		}
		err = consumer(participantIds)
		if err != nil {
			logger.Logger.Errorf("Got error during invoking consumer portion %v", err)
			lastError = err
			break
		}
	}
	return lastError
}

func setAdminCommon(qq CommonOperations, userId int64, chatId int64, newAdmin bool) error {
	if _, err := qq.Exec("UPDATE chat_participant SET admin = $3 WHERE user_id = $1 AND chat_id = $2", userId, chatId, newAdmin); err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func (tx *Tx) SetAdmin(userId int64, chatId int64, newAdmin bool) error {
	return setAdminCommon(tx, userId, chatId, newAdmin)
}

func (db *DB) SetAdmin(userId int64, chatId int64, newAdmin bool) error {
	return setAdminCommon(db, userId, chatId, newAdmin)
}

func (tx *Tx) HasParticipants(chatIds []int64) (map[int64]bool, error) {
	response := map[int64]bool{}
	for _, chatId := range chatIds {
		response[chatId] = false
	}
	if rows, err := tx.Query("SELECT DISTINCT(chat_id) FROM chat_participant WHERE chat_id = ANY ($1)", chatIds); err != nil {
		return nil, err
	} else {
		defer rows.Close()

		for rows.Next() {
			var chatId int64
			if err := rows.Scan(&chatId); err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			} else {
				response[chatId] = true
			}
		}
		return response, nil
	}
}

func (tx *Tx) GetAmIParticipantBatch(chatIds []int64, userId int64) (map[int64]bool, error) {
	if rows, err := tx.Query("SELECT chat_id FROM chat_participant WHERE user_id = $1 AND chat_id = ANY ($2)", userId, chatIds); err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		defer rows.Close()
		result := map[int64]bool{}
		for _, chatId := range chatIds {
			result[chatId] = false
		}
		for rows.Next() {
			var chatId int64
			if err := rows.Scan(&chatId); err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			} else {
				result[chatId] = true
			}
		}
		return result, nil
	}
}
