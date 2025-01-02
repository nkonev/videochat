package db

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/rotisserie/eris"
	"nkonev.name/chat/utils"
)

// db model

type ChatParticipant struct {
	Id     int64
	UserId int64
}

func (tx *Tx) AddParticipant(ctx context.Context, userId int64, chatId int64, admin bool) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO chat_participant (chat_id, user_id, admin) VALUES ($1, $2, $3)`, chatId, userId, admin)
	return eris.Wrap(err, "error during interacting with db")
}

func (tx *Tx) DeleteParticipant(ctx context.Context, userId int64, chatId int64) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM chat_participant WHERE chat_id = $1 AND user_id = $2`, chatId, userId)
	return eris.Wrap(err, "error during interacting with db")
}

func (tx *Tx) DeleteUserAsAParticipantFromAllChats(ctx context.Context, userId int64) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM chat_participant WHERE user_id = $1`, userId)
	return eris.Wrap(err, "error during interacting with db")
}

func getParticipantIdsCommon(ctx context.Context, qq CommonOperations, chatId int64, participantsSize, participantsOffset int) ([]int64, error) {
	if rows, err := qq.QueryContext(ctx, "SELECT user_id FROM chat_participant WHERE chat_id = $1 ORDER BY create_date_time DESC LIMIT $2 OFFSET $3", chatId, participantsSize, participantsOffset); err != nil {
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

func (tx *Tx) GetParticipantIds(ctx context.Context, chatId int64, participantsSize, participantsOffset int) ([]int64, error) {
	return getParticipantIdsCommon(ctx, tx, chatId, participantsSize, participantsOffset)
}

func (db *DB) GetParticipantIds(ctx context.Context, chatId int64, participantsSize, participantsOffset int) ([]int64, error) {
	return getParticipantIdsCommon(ctx, db, chatId, participantsSize, participantsOffset)
}

func getParticipantIdsBatchCommon(ctx context.Context, qq CommonOperations, chatIds []int64, participantsSize int) ([]*ParticipantIds, error) {
	res := make([]*ParticipantIds, 0)
	if len(chatIds) == 0 {
		return res, nil
	}

	if rows, err := qq.QueryContext(ctx, `
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

func (tx *Tx) GetParticipantIdsBatch(ctx context.Context, chatIds []int64, participantsSize int) ([]*ParticipantIds, error) {
	return getParticipantIdsBatchCommon(ctx, tx, chatIds, participantsSize)
}

func (db *DB) GetParticipantIdsBatch(ctx context.Context, chatIds []int64, participantsSize int) ([]*ParticipantIds, error) {
	return getParticipantIdsBatchCommon(ctx, db, chatIds, participantsSize)
}

func getChatParticipantIdsCommon(ctx context.Context, qq CommonOperations, chatId int64, consumer func(participantIds []int64) error) error {
	shouldContinue := true
	var lastError error
	for page := 0; shouldContinue; page++ {
		offset := utils.GetOffset(page, utils.DefaultSize)
		participantIds, err := getParticipantIdsCommon(ctx, qq, chatId, utils.DefaultSize, offset)
		if err != nil {
			qq.logger().WithTracing(ctx).Errorf("Got error during getting portion %v", err)
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
			qq.logger().WithTracing(ctx).Errorf("Got error during invoking consumer portion %v", err)
			lastError = err
			break
		}
	}
	return lastError
}

func (tx *Tx) IterateOverChatParticipantIds(ctx context.Context, chatId int64, consumer func(participantIds []int64) error) error {
	return getChatParticipantIdsCommon(ctx, tx, chatId, consumer)
}

func (db *DB) IterateOverChatParticipantIds(ctx context.Context, chatId int64, consumer func(participantIds []int64) error) error {
	return getChatParticipantIdsCommon(ctx, db, chatId, consumer)
}

func getPortionOfAllParticipantIdsCommon(ctx context.Context, qq CommonOperations, participantsSize, participantsOffset int) ([]int64, error) {
	if rows, err := qq.QueryContext(ctx, "SELECT distinct (user_id) FROM chat_participant ORDER BY user_id LIMIT $1 OFFSET $2", participantsSize, participantsOffset); err != nil {
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

func getAllParticipantIdsCommon(ctx context.Context, qq CommonOperations, consumer func(participantIds []int64) error) error {
	shouldContinue := true
	var lastError error
	for page := 0; shouldContinue; page++ {
		offset := utils.GetOffset(page, utils.DefaultSize)
		participantIds, err := getPortionOfAllParticipantIdsCommon(ctx, qq, utils.DefaultSize, offset)
		if err != nil {
			qq.logger().WithTracing(ctx).Errorf("Got error during getting portion %v", err)
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
			qq.logger().WithTracing(ctx).Errorf("Got error during invoking consumer portion %v", err)
			lastError = err
			break
		}
	}
	return lastError
}

func (tx *Tx) IterateOverAllParticipantIds(ctx context.Context, consumer func(participantIds []int64) error) error {
	return getAllParticipantIdsCommon(ctx, tx, consumer)
}

func (db *DB) IterateOverAllParticipantIds(ctx context.Context, consumer func(participantIds []int64) error) error {
	return getAllParticipantIdsCommon(ctx, db, consumer)
}

func getPortionOfAllChatIdsCommon(ctx context.Context, qq CommonOperations, participantId int64, chatsSize, chatsOffset int) ([]int64, error) {
	if rows, err := qq.QueryContext(ctx, "SELECT chat_id FROM chat_participant WHERE user_id = $1 ORDER BY chat_id LIMIT $2 OFFSET $3", participantId, chatsSize, chatsOffset); err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		defer rows.Close()
		list := make([]int64, 0)
		for rows.Next() {
			var chatId int64
			if err := rows.Scan(&chatId); err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			} else {
				list = append(list, chatId)
			}
		}
		return list, nil
	}
}

func getAllMyChatIdsCommon(ctx context.Context, qq CommonOperations, participantId int64, consumer func(chatIds []int64) error) error {
	shouldContinue := true
	var lastError error
	for page := 0; shouldContinue; page++ {
		offset := utils.GetOffset(page, utils.DefaultSize)
		chatIds, err := getPortionOfAllChatIdsCommon(ctx, qq, participantId, utils.DefaultSize, offset)
		if err != nil {
			qq.logger().WithTracing(ctx).Errorf("Got error during getting portion %v", err)
			lastError = err
			break
		}

		if len(chatIds) == 0 {
			return nil
		}
		if len(chatIds) < utils.DefaultSize {
			shouldContinue = false
		}
		err = consumer(chatIds)
		if err != nil {
			qq.logger().WithTracing(ctx).Errorf("Got error during invoking consumer portion %v", err)
			lastError = err
			break
		}
	}
	return lastError
}

func (tx *Tx) IterateOverAllMyChatIds(ctx context.Context, participantId int64, consumer func(chatIds []int64) error) error {
	return getAllMyChatIdsCommon(ctx, tx, participantId, consumer)
}

func (db *DB) IterateOverAllMyChatIds(ctx context.Context, participantId int64, consumer func(chatIds []int64) error) error {
	return getAllMyChatIdsCommon(ctx, db, participantId, consumer)
}

func getParticipantsCountCommon(ctx context.Context, qq CommonOperations, chatId int64) (int, error) {
	var count int
	row := qq.QueryRowContext(ctx, "SELECT count(*) FROM chat_participant WHERE chat_id = $1", chatId)

	if err := row.Scan(&count); err != nil {
		return 0, eris.Wrap(err, "error during interacting with db")
	} else {
		return count, nil
	}
}

func (tx *Tx) GetParticipantsCount(ctx context.Context, chatId int64) (int, error) {
	return getParticipantsCountCommon(ctx, tx, chatId)
}

func (db *DB) GetParticipantsCount(ctx context.Context, chatId int64) (int, error) {
	return getParticipantsCountCommon(ctx, db, chatId)
}

func getParticipantsCountBatchCommon(ctx context.Context, qq CommonOperations, chatIds []int64) (map[int64]int, error) {
	res := map[int64]int{}

	if len(chatIds) == 0 {
		return res, nil
	}

	var builder = ""
	var first = true
	for _, chatId := range chatIds {
		if !first {
			builder += " UNION ALL "
		}
		builder += fmt.Sprintf("(SELECT %v, count(*) FROM chat_participant WHERE chat_id = %v)", chatId, chatId)

		first = false
	}

	var rows *sql.Rows
	var err error
	rows, err = qq.QueryContext(ctx, builder)
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

func (tx *Tx) GetParticipantsCountBatch(ctx context.Context, chatIds []int64) (map[int64]int, error) {
	return getParticipantsCountBatchCommon(ctx, tx, chatIds)
}

func (db *DB) GetParticipantsCountBatch(ctx context.Context, chatIds []int64) (map[int64]int, error) {
	return getParticipantsCountBatchCommon(ctx, db, chatIds)
}

func getIsAdminCommon(ctx context.Context, qq CommonOperations, userId int64, chatId int64) (bool, error) {
	var admin bool = false
	row := qq.QueryRowContext(ctx, `SELECT exists(SELECT * FROM chat_participant WHERE user_id = $1 AND chat_id = $2 AND admin = true LIMIT 1)`, userId, chatId)
	if err := row.Scan(&admin); err != nil {
		return false, eris.Wrap(err, "error during interacting with db")
	} else {
		return admin, nil
	}
}

func (tx *Tx) IsAdmin(ctx context.Context, userId int64, chatId int64) (bool, error) {
	return getIsAdminCommon(ctx, tx, userId, chatId)
}

func (db *DB) IsAdmin(ctx context.Context, userId int64, chatId int64) (bool, error) {
	return getIsAdminCommon(ctx, db, userId, chatId)
}

func getIsAdminBatchCommon(ctx context.Context, qq CommonOperations, userId int64, chatIds []int64) (map[int64]bool, error) {
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

	if rows, err := qq.QueryContext(ctx, fmt.Sprintf(`SELECT chat_id, admin FROM chat_participant WHERE user_id = $1 AND chat_id IN (%v) AND admin = true`, builder), userId); err != nil {
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

func (tx *Tx) IsAdminBatch(ctx context.Context, userId int64, chatIds []int64) (map[int64]bool, error) {
	return getIsAdminBatchCommon(ctx, tx, userId, chatIds)
}

func (db *DB) IsAdminBatch(ctx context.Context, userId int64, chatIds []int64) (map[int64]bool, error) {
	return getIsAdminBatchCommon(ctx, db, userId, chatIds)
}

func getIsAdminBatchByParticipantsCommon(ctx context.Context, qq CommonOperations, userIds []int64, chatId int64) ([]UserAdminDbDTO, error) {
	var result = []UserAdminDbDTO{}

	if len(userIds) == 0 {
		return result, nil
	}

	if rows, err := qq.QueryContext(ctx, fmt.Sprintf(`SELECT user_id, admin FROM chat_participant WHERE user_id = ANY($1) AND chat_id = $2 ORDER BY create_date_time DESC`), userIds, chatId); err != nil {
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
func (db *DB) IsAdminBatchByParticipants(ctx context.Context, userIds []int64, chatId int64) ([]UserAdminDbDTO, error) {
	return getIsAdminBatchByParticipantsCommon(ctx, db, userIds, chatId)
}

func (tx *Tx) IsAdminBatchByParticipants(ctx context.Context, userIds []int64, chatId int64) ([]UserAdminDbDTO, error) {
	return getIsAdminBatchByParticipantsCommon(ctx, tx, userIds, chatId)
}

func isParticipantCommon(ctx context.Context, qq CommonOperations, userId int64, chatId int64) (bool, error) {
	var exists bool = false
	row := qq.QueryRowContext(ctx, `SELECT exists(SELECT * FROM chat_participant WHERE user_id = $1 AND chat_id = $2 LIMIT 1)`, userId, chatId)
	if err := row.Scan(&exists); err != nil {
		return false, eris.Wrap(err, "error during interacting with db")
	} else {
		return exists, nil
	}
}

func (tx *Tx) IsParticipant(ctx context.Context, userId int64, chatId int64) (bool, error) {
	return isParticipantCommon(ctx, tx, userId, chatId)
}

func (db *DB) IsParticipant(ctx context.Context, userId int64, chatId int64) (bool, error) {
	return isParticipantCommon(ctx, db, userId, chatId)
}

func (tx *Tx) GetFirstParticipant(ctx context.Context, chatId int64) (int64, error) {
	var pid int64
	row := tx.QueryRowContext(ctx, `SELECT user_id FROM chat_participant WHERE chat_id = $1 LIMIT 1`, chatId)
	if err := row.Scan(&pid); err != nil {
		return 0, eris.Wrap(err, "error during interacting with db")
	} else {
		return pid, nil
	}
}

func getCoChattedParticipantIdsCommon(ctx context.Context, co CommonOperations, participantId int64, limit, offset int) ([]int64, error) {
	if rows, err := co.QueryContext(ctx, "SELECT DISTINCT user_id FROM chat_participant WHERE chat_id IN (SELECT chat_id FROM chat_participant WHERE user_id = $1) ORDER BY user_id LIMIT $2 OFFSET $3", participantId, limit, offset); err != nil {
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
func (db *DB) ParticipantsExistence(ctx context.Context, chatId int64, participantIds []int64) ([]int64, error) {
	if rows, err := db.QueryContext(ctx, "SELECT user_id FROM chat_participant WHERE chat_id = $1 AND user_id = ANY ($2)", chatId, participantIds); err != nil {
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

func iterateOverCoChattedParticipantIdsCommon(ctx context.Context, co CommonOperations, participantId int64, consumer func(participantIds []int64) error) error {
	shouldContinue := true
	var lastError error
	for page := 0; shouldContinue; page++ {
		offset := utils.GetOffset(page, utils.DefaultSize)
		participantIds, err := getCoChattedParticipantIdsCommon(ctx, co, participantId, utils.DefaultSize, offset)
		if err != nil {
			co.logger().WithTracing(ctx).Errorf("Got error during getting portion %v", err)
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
			co.logger().WithTracing(ctx).Errorf("Got error during invoking consumer portion %v", err)
			lastError = err
			break
		}
	}
	return lastError
}

func (db *DB) IterateOverCoChattedParticipantIds(ctx context.Context, participantId int64, consumer func(participantIds []int64) error) error {
	return iterateOverCoChattedParticipantIdsCommon(ctx, db, participantId, consumer)
}

func (tx *Tx) IterateOverCoChattedParticipantIds(ctx context.Context, participantId int64, consumer func(participantIds []int64) error) error {
	return iterateOverCoChattedParticipantIdsCommon(ctx, tx, participantId, consumer)
}

func setAdminCommon(ctx context.Context, qq CommonOperations, userId int64, chatId int64, newAdmin bool) error {
	if _, err := qq.ExecContext(ctx, "UPDATE chat_participant SET admin = $3 WHERE user_id = $1 AND chat_id = $2", userId, chatId, newAdmin); err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func (tx *Tx) SetAdmin(ctx context.Context, userId int64, chatId int64, newAdmin bool) error {
	return setAdminCommon(ctx, tx, userId, chatId, newAdmin)
}

func (db *DB) SetAdmin(ctx context.Context, userId int64, chatId int64, newAdmin bool) error {
	return setAdminCommon(ctx, db, userId, chatId, newAdmin)
}

func (tx *Tx) HasParticipants(ctx context.Context, chatIds []int64) (map[int64]bool, error) {
	response := map[int64]bool{}
	for _, chatId := range chatIds {
		response[chatId] = false
	}
	if rows, err := tx.QueryContext(ctx, "SELECT DISTINCT(chat_id) FROM chat_participant WHERE chat_id = ANY ($1)", chatIds); err != nil {
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

func (tx *Tx) GetAmIParticipantBatch(ctx context.Context, chatIds []int64, userId int64) (map[int64]bool, error) {
	if rows, err := tx.QueryContext(ctx, "SELECT chat_id FROM chat_participant WHERE user_id = $1 AND chat_id = ANY ($2)", userId, chatIds); err != nil {
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
