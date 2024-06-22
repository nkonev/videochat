package db

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/guregu/null"
	"github.com/spf13/viper"
	"github.com/rotisserie/eris"
	"nkonev.name/chat/auth"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
	"time"
)

const ReservedPublicallyAvailableForSearchChats = "__AVAILABLE_FOR_SEARCH"

// db model
type Chat struct {
	Id                           int64
	Title                        string
	LastUpdateDateTime           time.Time
	TetATet                      bool
	CanResend                    bool
	Avatar                       null.String
	AvatarBig                    null.String
	AvailableToSearch            bool
	Pinned                       bool
	Blog                                bool
	RegularParticipantCanPublishMessage bool
}

type Blog struct {
	Id             int64
	Title          string
	CreateDateTime time.Time
	Avatar         null.String
}

type ChatWithParticipants struct {
	Chat
	ParticipantsIds   []int64
	ParticipantsCount int
	IsAdmin           bool
}

// CreateChat creates a new chat.
// Returns an error if user is invalid or the tx fails.
func (tx *Tx) CreateChat(u *Chat) (int64, *time.Time, error) {
	// Validate the input.
	if u == nil {
		return 0, nil, eris.New("chat required")
	} else if u.Title == "" {
		return 0, nil, eris.New("title required")
	}

	// https://stackoverflow.com/questions/4547672/return-multiple-fields-as-a-record-in-postgresql-with-pl-pgsql/6085167#6085167
	res := tx.QueryRow(`SELECT chat_id, last_update_date_time FROM CREATE_CHAT($1, $2, $3, $4, $5, $6) AS (chat_id BIGINT, last_update_date_time TIMESTAMP)`, u.Title, u.TetATet, u.CanResend, u.AvailableToSearch, u.Blog, u.RegularParticipantCanPublishMessage)
	var id int64
	var lastUpdateDateTime time.Time
	if err := res.Scan(&id, &lastUpdateDateTime); err != nil {
		return 0, nil, eris.Wrap(err, "error during interacting with db")
	}

	return id, &lastUpdateDateTime, nil
}

func (tx *Tx) CreateTetATetChat(behalfUserId int64, toParticipantId int64) (int64, error) {
	tetATetChatName := fmt.Sprintf("tet_a_tet_%v_%v", behalfUserId, toParticipantId)
	chatId, _, err := tx.CreateChat(&Chat{Title: tetATetChatName, TetATet: true, CanResend: viper.GetBool("canResendFromTetATet")})
	return chatId, err
}

func (tx *Tx) IsExistsTetATet(participant1 int64, participant2 int64) (bool, int64, error) {
	res := tx.QueryRow("select b.chat_id from (select a.count >= 2 as exists, a.chat_id from ( (select cp.chat_id, count(cp.user_id) from chat_participant cp join chat ch on ch.id = cp.chat_id where ch.tet_a_tet = true and (cp.user_id = $1 or cp.user_id = $2) group by cp.chat_id)) a) b where b.exists is true;", participant1, participant2)
	var chatId int64
	if err := res.Scan(&chatId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// there were no rows, but otherwise no error occurred
			return false, 0, nil
		}
		return false, 0, eris.Wrap(err, "error during interacting with db")
	}
	return true, chatId, nil
}

// expects $1 is userId
func selectChatClause() string {
	return `SELECT 
				ch.id, 
				ch.title, 
				ch.avatar, 
				ch.avatar_big,
				ch.last_update_date_time,
				ch.tet_a_tet,
				ch.can_resend,
				ch.available_to_search,
				cp.user_id IS NOT NULL as pinned,
				ch.blog,
				ch.regular_participant_can_publish_message
			FROM chat ch 
				LEFT JOIN chat_pinned cp on (ch.id = cp.chat_id and cp.user_id = $1)
`
}

func provideScanToChat(chat *Chat) []any {
	return []any{
		&chat.Id,
		&chat.Title,
		&chat.Avatar,
		&chat.AvatarBig,
		&chat.LastUpdateDateTime,
		&chat.TetATet,
		&chat.CanResend,
		&chat.AvailableToSearch,
		&chat.Pinned,
		&chat.Blog,
		&chat.RegularParticipantCanPublishMessage,
	}
}

// rowNumber is in [1, count]
func (tx *Tx) GetChatRowNumber(itemId, userId int64, orderDirection, searchString string) (int, error) {

	var searchStringWithPercents = "%" + searchString + "%"
	var theQuery = `
		SELECT al.nrow FROM (
			SELECT 
				ch.id as cid,
				ROW_NUMBER () OVER (ORDER BY (cp.user_id is not null, ch.last_update_date_time, ch.id) ` + orderDirection + `) as nrow
			FROM 
				chat ch 
				LEFT JOIN chat_pinned cp ON (ch.id = cp.chat_id AND cp.user_id = $1)
			WHERE `+ getChatSearchClause([]int64{}) +`
		) al WHERE al.cid = $4
	`
	var position int
	row := tx.QueryRow(theQuery, userId, searchStringWithPercents, searchString, itemId)
	err := row.Scan(&position)
	if err != nil {
		return 0, eris.Wrap(err, "error during interacting with db")
	} else {
		return position, nil
	}
}

func getChatIdsByLimitOffsetCommon(co CommonOperations, participantId int64, limit int, offset int) ([]int64, error) {
	var rows *sql.Rows
	var err error
	rows, err = co.Query(`SELECT ch.id from chat ch
		WHERE ch.id IN ( SELECT chat_id FROM chat_participant WHERE user_id = $1 ) 
		ORDER BY ch.id
		LIMIT $2 OFFSET $3
	`, participantId, limit, offset)
	if err != nil {
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

func getChatsByLimitOffsetCommon(co CommonOperations, participantId int64, limit int, offset int, orderDirection string) ([]*Chat, error) {
	var rows *sql.Rows
	var err error
	rows, err = co.Query(fmt.Sprintf(selectChatClause()+`
		WHERE ch.id IN ( SELECT chat_id FROM chat_participant WHERE user_id = $1 ) 
		ORDER BY (cp.user_id is not null, ch.last_update_date_time, ch.id) %s 
		LIMIT $2 OFFSET $3
	`, orderDirection), participantId, limit, offset)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		defer rows.Close()
		list := make([]*Chat, 0)
		for rows.Next() {
			chat := Chat{}
			if err := rows.Scan(provideScanToChat(&chat)[:]...); err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			} else {
				list = append(list, &chat)
			}
		}
		return list, nil
	}
}

func (db *DB) GetChatsByLimitOffset(participantId int64, limit int, offset int, orderDirection string) ([]*Chat, error) {
	return getChatsByLimitOffsetCommon(db, participantId, limit, offset, orderDirection)
}

func (tx *Tx) GetChatsByLimitOffset(participantId int64, limit int, offset int, orderDirection string) ([]*Chat, error) {
	return getChatsByLimitOffsetCommon(tx, participantId, limit, offset, orderDirection)
}

// requires
// $1 - owner_id
// $2 - searchStringWithPercents
// $3 - searchString
func getChatSearchClause(additionalFoundUserIds []int64) string {
	var additionalUserIds = ""
	first := true
	for _, userId := range additionalFoundUserIds {
		if !first {
			additionalUserIds = additionalUserIds + ","
		}
		additionalUserIds = additionalUserIds + utils.Int64ToString(userId)
		first = false
	}

	var additionalUserIdsClause = ""
	if len(additionalFoundUserIds) > 0 {
		additionalUserIdsClause = fmt.Sprintf(" OR ( ch.tet_a_tet IS true AND ch.id IN ( SELECT chat_id FROM chat_participant WHERE user_id IN (%s) ) ) ", additionalUserIds)
	}
	return fmt.Sprintf(" ( ( ch.id IN ( SELECT chat_id FROM chat_participant WHERE user_id = $1 ) AND ( ch.title ILIKE $2 %s ) ) OR ( ch.available_to_search IS TRUE AND $3 = '%s' ) )",
		additionalUserIdsClause, ReservedPublicallyAvailableForSearchChats,
	)
}

func getChatsByLimitOffsetSearchCommon(commonOps CommonOperations, participantId int64, limit int, offset int, orderDirection, searchString string, additionalFoundUserIds []int64) ([]*Chat, error) {
	var rows *sql.Rows
	var err error
	searchStringWithPercents := "%" + searchString + "%"

	rows, err = commonOps.Query(selectChatClause() + " WHERE " + getChatSearchClause(additionalFoundUserIds) + `
			ORDER BY (cp.user_id is not null, ch.last_update_date_time, ch.id) ` + orderDirection + ` 
			LIMIT $4 OFFSET $5
	`, participantId, searchStringWithPercents, searchString, limit, offset)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		defer rows.Close()
		list := make([]*Chat, 0)
		for rows.Next() {
			chat := Chat{}
			if err := rows.Scan(provideScanToChat(&chat)[:]...); err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			} else {
				list = append(list, &chat)
			}
		}
		return list, nil
	}
}

func (db *DB) GetChatsByLimitOffsetSearch(participantId int64, limit int, offset int, orderDirection, searchString string, additionalFoundUserIds []int64) ([]*Chat, error) {
	return getChatsByLimitOffsetSearchCommon(db, participantId, limit, offset, orderDirection, searchString, additionalFoundUserIds)
}

func (tx *Tx) GetChatsByLimitOffsetSearch(participantId int64, limit int, offset int, orderDirection, searchString string, additionalFoundUserIds []int64) ([]*Chat, error) {
	return getChatsByLimitOffsetSearchCommon(tx, participantId, limit, offset, orderDirection, searchString, additionalFoundUserIds)
}

func convertToWithParticipants(db CommonOperations, chat *Chat, behalfUserId int64, participantsSize, participantsOffset int) (*ChatWithParticipants, error) {
	if ids, err := db.GetParticipantIds(chat.Id, participantsSize, participantsOffset); err != nil {
		return nil, err
	} else {
		admin, err := db.IsAdmin(behalfUserId, chat.Id)
		if err != nil {
			return nil, err
		}
		participantsCount, err := db.GetParticipantsCount(chat.Id)
		if err != nil {
			return nil, err
		}
		ccc := &ChatWithParticipants{
			Chat:              *chat,
			ParticipantsIds:   ids,
			IsAdmin:           admin,
			ParticipantsCount: participantsCount,
		}
		return ccc, nil
	}
}

func convertToWithoutParticipants(db CommonOperations, chat *Chat, behalfUserId int64) (*ChatWithParticipants, error) {
	admin, err := db.IsAdmin(behalfUserId, chat.Id)
	if err != nil {
		return nil, err
	}
	ccc := &ChatWithParticipants{
		Chat:              *chat,
		ParticipantsIds:   []int64{}, // to be set in callee
		IsAdmin:           admin,
		ParticipantsCount: 0, // to be set in callee
	}
	return ccc, nil
}

type ChatQueryByLimitOffset struct {
	Limit  int
	Offset int
}

type ChatQueryByIds struct {
	Ids []int64
}

type ParticipantIds struct {
	ChatId         int64
	ParticipantIds []int64
}

func convertToWithParticipantsBatch(chat *Chat, participantIdsBatch []*ParticipantIds, isAdminBatch map[int64]bool, participantsCountBatch map[int64]int) (*ChatWithParticipants, error) {
	participantsCount := participantsCountBatch[chat.Id]

	var participantsIds []int64 = make([]int64, 0)
	for _, pidsb := range participantIdsBatch {
		if pidsb.ChatId == chat.Id {
			participantsIds = pidsb.ParticipantIds
			break
		}
	}

	admin := isAdminBatch[chat.Id]

	ccc := &ChatWithParticipants{
		Chat:              *chat,
		ParticipantsIds:   participantsIds,
		IsAdmin:           admin,
		ParticipantsCount: participantsCount,
	}
	return ccc, nil
}

func getChatsWithParticipantsCommon(commonOps CommonOperations, participantId int64, limit, offset int, orderDirection, searchString string, additionalFoundUserIds []int64, userPrincipalDto *auth.AuthResult, participantsSize, participantsOffset int) ([]*ChatWithParticipants, error) {
	var err error
	var chats []*Chat

	if searchString == "" {
		chats, err = commonOps.GetChatsByLimitOffset(participantId, limit, offset, orderDirection)
	} else {
		chats, err = commonOps.GetChatsByLimitOffsetSearch(participantId, limit, offset, orderDirection, searchString, additionalFoundUserIds)
	}

	if err != nil {
		return nil, err
	} else {
		var chatIds []int64 = make([]int64, 0)
		for _, cc := range chats {
			chatIds = append(chatIds, cc.Id)
		}

		fixedParticipantsSize := utils.FixSize(participantsSize)
		participantIdsBatch, err := commonOps.GetParticipantIdsBatch(chatIds, fixedParticipantsSize)
		if err != nil {
			return nil, err
		}

		isAdminBatch, err := commonOps.IsAdminBatch(userPrincipalDto.UserId, chatIds)
		if err != nil {
			return nil, err
		}

		participantsCountBatch, err := commonOps.GetParticipantsCountBatch(chatIds)
		if err != nil {
			return nil, err
		}

		list := make([]*ChatWithParticipants, 0)

		for _, cc := range chats {
			if ccc, err := convertToWithParticipantsBatch(cc, participantIdsBatch, isAdminBatch, participantsCountBatch); err != nil {
				return nil, err
			} else {
				list = append(list, ccc)
			}
		}
		return list, nil
	}
}
func (db *DB) GetChatsWithParticipants(participantId int64, limit, offset int, orderDirection, searchString string, additionalFoundUserIds []int64, userPrincipalDto *auth.AuthResult, participantsSize, participantsOffset int) ([]*ChatWithParticipants, error) {
	return getChatsWithParticipantsCommon(db, participantId, limit, offset, orderDirection, searchString, additionalFoundUserIds, userPrincipalDto, participantsSize, participantsOffset)
}

func (tx *Tx) GetChatsWithParticipants(participantId int64, limit, offset int, orderDirection, searchString string, additionalFoundUserIds []int64, userPrincipalDto *auth.AuthResult, participantsSize, participantsOffset int) ([]*ChatWithParticipants, error) {
	return getChatsWithParticipantsCommon(tx, participantId, limit, offset, orderDirection, searchString, additionalFoundUserIds, userPrincipalDto, participantsSize, participantsOffset)
}

func (tx *Tx) GetChatWithParticipants(behalfParticipantId, chatId int64, participantsSize, participantsOffset int) (*ChatWithParticipants, error) {
	return getChatWithParticipantsCommon(tx, behalfParticipantId, chatId, participantsSize, participantsOffset)
}

func (db *DB) GetChatWithParticipants(behalfParticipantId, chatId int64, participantsSize, participantsOffset int) (*ChatWithParticipants, error) {
	return getChatWithParticipantsCommon(db, behalfParticipantId, chatId, participantsSize, participantsOffset)
}

func getChatWithParticipantsCommon(commonOps CommonOperations, behalfParticipantId, chatId int64, participantsSize, participantsOffset int) (*ChatWithParticipants, error) {
	if chat, err := commonOps.GetChat(behalfParticipantId, chatId); err != nil {
		return nil, err
	} else if chat == nil {
		return nil, nil
	} else {
		return convertToWithParticipants(commonOps, chat, behalfParticipantId, participantsSize, participantsOffset)
	}
}

func (tx *Tx) GetChatWithoutParticipants(behalfParticipantId, chatId int64) (*ChatWithParticipants, error) {
	return getChatWithoutParticipantsCommon(tx, behalfParticipantId, chatId)
}

func (db *DB) GetChatWithoutParticipants(behalfParticipantId, chatId int64) (*ChatWithParticipants, error) {
	return getChatWithoutParticipantsCommon(db, behalfParticipantId, chatId)
}

func getChatWithoutParticipantsCommon(commonOps CommonOperations, behalfParticipantId, chatId int64) (*ChatWithParticipants, error) {
	if chat, err := commonOps.GetChat(behalfParticipantId, chatId); err != nil {
		return nil, err
	} else if chat == nil {
		return nil, nil
	} else {
		return convertToWithoutParticipants(commonOps, chat, behalfParticipantId)
	}
}

func (db *DB) CountChats() (int64, error) {
	var count int64
	row := db.QueryRow("SELECT count(*) FROM chat")
	err := row.Scan(&count)
	if err != nil {
		return 0, eris.Wrap(err, "error during interacting with db")
	} else {
		return count, nil
	}
}

func countChatsPerUser(commonOps CommonOperations, userId int64) (int64, error) {
	var count int64
	row := commonOps.QueryRow("SELECT count(*) FROM chat_participant WHERE user_id = $1", userId)
	err := row.Scan(&count)
	if err != nil {
		return 0, eris.Wrap(err, "error during interacting with db")
	} else {
		return count, nil
	}
}

func (db *DB) CountChatsPerUser(userId int64) (int64, error) {
	return countChatsPerUser(db, userId)
}

func (tx *Tx) CountChatsPerUser(userId int64) (int64, error) {
	return countChatsPerUser(tx, userId)
}

func (tx *Tx) DeleteChat(id int64) error {
	if _, err := tx.Exec(`CALL DELETE_CHAT($1)`, id); err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func (tx *Tx) EditChat(id int64, newTitle string, avatar, avatarBig null.String, canResend bool, availableToSearch bool, blog* bool, regularParticipantCanPublish bool) (*time.Time, error) {
	var res sql.Result
	var err error
	if blog != nil {
		isBlog := utils.NullableToBoolean(blog)
		res, err = tx.Exec(`UPDATE chat SET title = $2, avatar = $3, avatar_big = $4, last_update_date_time = utc_now(), can_resend = $5, available_to_search = $6, blog = $7, regular_participant_can_publish_message = $8 WHERE id = $1`, id, newTitle, avatar, avatarBig, canResend, availableToSearch, isBlog, regularParticipantCanPublish)
	} else {
		res, err = tx.Exec(`UPDATE chat SET title = $2, avatar = $3, avatar_big = $4, last_update_date_time = utc_now(), can_resend = $5, available_to_search = $6, regular_participant_can_publish_message = $7 WHERE id = $1`, id, newTitle, avatar, avatarBig, canResend, availableToSearch, regularParticipantCanPublish)
	}
	if err != nil {
		Logger.Errorf("Error during editing chat id %v", err)
		return nil, err
	} else {
		affected, err := res.RowsAffected()
		if err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		}
		if affected == 0 {
			return nil, eris.New("No rows affected")
		}
	}

	var lastUpdateDateTime time.Time
	res2 := tx.QueryRow(`SELECT last_update_date_time FROM chat WHERE id = $1`, id)
	if err := res2.Scan(&lastUpdateDateTime); err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}

	return &lastUpdateDateTime, nil
}

func getChatCommon(co CommonOperations, participantId, chatId int64) (*Chat, error) {
	row := co.QueryRow(selectChatClause()+` WHERE ch.id in (SELECT chat_id FROM chat_participant WHERE user_id = $1 AND chat_id = $2)`, participantId, chatId)
	chat := Chat{}
	err := row.Scan(provideScanToChat(&chat)[:]...)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil
	}
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		return &chat, nil
	}
}

func (db *DB) GetChat(participantId, chatId int64) (*Chat, error) {
	return getChatCommon(db, participantId, chatId)
}

func (tx *Tx) GetChat(participantId, chatId int64) (*Chat, error) {
	return getChatCommon(tx, participantId, chatId)
}

func getChatBasicCommon(co CommonOperations, chatId int64) (*BasicChatDto, error) {
	row := co.QueryRow(`SELECT 
				ch.id, 
				ch.title, 
				ch.tet_a_tet,
				ch.can_resend,
				ch.available_to_search,
				ch.blog,
				ch.create_date_time,
				ch.regular_participant_can_publish_message
			FROM chat ch 
			WHERE ch.id = $1
`, chatId)
	chat := BasicChatDto{}
	err := row.Scan(&chat.Id, &chat.Title, &chat.IsTetATet, &chat.CanResend, &chat.AvailableToSearch, &chat.IsBlog, &chat.CreateDateTime, &chat.RegularParticipantCanPublishMessage)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil
	}
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		return &chat, nil
	}
}

func (db *DB) GetChatBasic(chatId int64) (*BasicChatDto, error) {
	return getChatBasicCommon(db, chatId)
}

func (tx *Tx) GetChatBasic(chatId int64) (*BasicChatDto, error) {
	return getChatBasicCommon(tx, chatId)
}

func getChatsBasicCommon(co CommonOperations, chatIds map[int64]bool, behalfParticipantId int64) (map[int64]*BasicChatDtoExtended, error) {
	result := map[int64]*BasicChatDtoExtended{}
	if len(chatIds) == 0 {
		return result, nil
	}

	inClause := ""
	first := true
	for chatId, val := range chatIds {
		if val {
			dtl := ""
			if !first {
				dtl = ","
			}
			dtl += utils.Int64ToString(chatId)
			inClause = inClause + dtl
		}

		first = false
	}
	rows, err := co.Query(fmt.Sprintf(`
		SELECT 
			c.id, 
			c.title,
			(cp.user_id is not null),
			c.tet_a_tet,
			c.can_resend,
			c.available_to_search,
			c.blog,
			c.create_date_time,
			c.regular_participant_can_publish_message
		FROM chat c 
		    LEFT JOIN chat_participant cp 
		        ON (c.id = cp.chat_id AND cp.user_id = $1) 
		WHERE c.id IN (%s)`, inClause),
		behalfParticipantId)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		defer rows.Close()
		list := make([]*BasicChatDtoExtended, 0)
		for rows.Next() {
			dto := new(BasicChatDtoExtended)
			if err := rows.Scan(&dto.Id, &dto.Title, &dto.BehalfUserIsParticipant, &dto.IsTetATet, &dto.CanResend, &dto.AvailableToSearch, &dto.IsBlog, &dto.CreateDateTime,  &dto.RegularParticipantCanPublishMessage); err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			} else {
				list = append(list, dto)
			}
		}
		for _, bc := range list {
			result[bc.Id] = bc
		}
		return result, nil
	}
}

func (db *DB) GetChatsBasic(chatIds map[int64]bool, behalfParticipantId int64) (map[int64]*BasicChatDtoExtended, error) {
	return getChatsBasicCommon(db, chatIds, behalfParticipantId)
}

func (tx *Tx) GetChatsBasic(chatIds map[int64]bool, behalfParticipantId int64) (map[int64]*BasicChatDtoExtended, error) {
	return getChatsBasicCommon(tx, chatIds, behalfParticipantId)
}

type BasicChatDto struct {
	Id                int64
	Title             string
	IsTetATet         bool
	CanResend         bool
	AvailableToSearch bool
	IsBlog            bool
	CreateDateTime    time.Time
	RegularParticipantCanPublishMessage bool
}

type BasicChatDtoExtended struct {
	BasicChatDto
	BehalfUserIsParticipant bool
}

func (tx *Tx) UpdateChatLastDatetimeChat(id int64) error {
	if _, err := tx.Exec("UPDATE chat SET last_update_date_time = utc_now() WHERE id = $1", id); err != nil {
		return eris.Wrap(err, "error during interacting with db")
	} else {
		return nil
	}
}

func (tx *Tx) GetChatLastDatetimeChat(chatId int64) (time.Time, error) {
	row := tx.QueryRow(`SELECT last_update_date_time FROM chat WHERE id = $1`, chatId)
	var lastUpdateDateTime time.Time
	err := row.Scan(&lastUpdateDateTime)
	if err != nil {
		return lastUpdateDateTime, eris.Wrap(err, "error during interacting with db")
	} else {
		return lastUpdateDateTime, nil
	}
}


func (db *DB) GetExistingChatIds(chatIds []int64) (*[]int64, error) {
	list := make([]int64, 0)

	if len(chatIds) == 0 {
		return &list, nil
	}

	var additionalChatIds = ""
	first := true
	for _, chatId := range chatIds {
		if !first {
			additionalChatIds = additionalChatIds + ","
		}
		additionalChatIds = additionalChatIds + utils.Int64ToString(chatId)
		first = false
	}

	rows, err := db.Query(fmt.Sprintf(`SELECT id FROM chat WHERE id IN (%s)`, additionalChatIds))
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	defer rows.Close()

	for rows.Next() {
		var chatId int64
		if err := rows.Scan(&chatId); err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		} else {
			list = append(list, chatId)
		}
	}

	return &list, nil
}

func pinChatCommon(co CommonOperations, chatId int64, userId int64, pin bool) error {
	if pin {
		_, err := co.Exec("insert into chat_pinned(user_id, chat_id) values ($1, $2) on conflict do nothing", userId, chatId)
		if err != nil {
			return eris.Wrap(err, "error during interacting with db")
		}
	} else {
		_, err := co.Exec("delete from chat_pinned where user_id = $1 and chat_id = $2", userId, chatId)
		if err != nil {
			return eris.Wrap(err, "error during interacting with db")
		}
	}
	return nil
}

func (db *DB) PinChat(chatId int64, userId int64, pin bool) error {
	return pinChatCommon(db, chatId, userId, pin)
}

func (tx *Tx) PinChat(chatId int64, userId int64, pin bool) error {
	return pinChatCommon(tx, chatId, userId, pin)
}

func (tx *Tx) IsChatPinnedBatch(userIds []int64, chatId int64) (map[int64]bool, error) {
	res := map[int64]bool{}

	var rows *sql.Rows
	var err error
	rows, err = tx.Query(`
		SELECT 
			cp.user_id
			FROM chat_pinned cp WHERE cp.user_id = ANY($1) AND cp.chat_id = $2
	`, userIds, chatId)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		defer rows.Close()
		for _, uid := range userIds {
			res[uid] = false // init map
		}
		for rows.Next() {
			var userId int64
			if err := rows.Scan(&userId); err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			} else {
				res[userId] = true
			}
		}
		return res, nil
	}
}

func (tx *Tx) DeleteChatsPinned(userId int64) error {
	if _, err := tx.Exec("DELETE FROM chat_pinned WHERE user_id = $1", userId); err != nil {
		return eris.Wrap(err, "error during interacting with db")
	} else {
		return nil
	}
}

func (tx *Tx) RenameChat(chatId int64, title string) error {
	_, err := tx.Exec("update chat set title = $1 where id = $2", title, chatId)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func getChatIdsCommon(qq CommonOperations, chatsSize, chatsOffset int) ([]int64, error) {
	if rows, err := qq.Query("SELECT id FROM chat ORDER BY id LIMIT $1 OFFSET $2", chatsSize, chatsOffset); err != nil {
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

func (tx *Tx) GetChatIds(chatsSize, chatsOffset int) ([]int64, error) {
	return getChatIdsCommon(tx, chatsSize, chatsOffset)
}

func (db *DB) GetChatIds(chatsSize, chatsOffset int) ([]int64, error) {
	return getChatIdsCommon(db, chatsSize, chatsOffset)
}



func getBlogPostsByLimitOffsetCommon(co CommonOperations, reverse bool, limit int, offset int) ([]*Blog, error) {
	var rows *sql.Rows
	var err error
	var sort string
	if reverse {
		sort = "asc"
	} else {
		sort = "desc"
	}
	rows, err = co.Query(fmt.Sprintf(`SELECT 
			ch.id, 
			ch.title,
			ch.create_date_time,
			ch.avatar
		FROM chat ch 
		WHERE ch.blog is TRUE 
		ORDER BY ch.id %s 
		LIMIT $1 OFFSET $2`, sort),
		limit, offset)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		defer rows.Close()
		list := make([]*Blog, 0)
		for rows.Next() {
			chat := Blog{}
			if err := rows.Scan(&chat.Id, &chat.Title, &chat.CreateDateTime, &chat.Avatar); err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			} else {
				list = append(list, &chat)
			}
		}
		return list, nil
	}
}

func (tx *Tx) GetBlogPostsByLimitOffset(reverse bool, limit int, offset int) ([]*Blog, error) {
	return getBlogPostsByLimitOffsetCommon(tx, reverse, limit, offset)
}

func (db *DB) GetBlogPostsByLimitOffset(reverse bool, limit int, offset int) ([]*Blog, error) {
	return getBlogPostsByLimitOffsetCommon(db, reverse, limit, offset)
}

type BlogPost struct {
	ChatId    int64
	MessageId int64
	OwnerId   int64
	Text      string
}

func getBlogPostsByChatIdsCommon(co CommonOperations, ids []int64) ([]*BlogPost, error) {
	var builder = ""
	var first = true
	for _, chatId := range ids {
		if !first {
			builder += " UNION ALL "
		}
		builder += fmt.Sprintf("(select %v, id, owner_id, text from message_chat_%v where blog_post is true order by id limit 1)", chatId, chatId)

		first = false
	}

	var rows *sql.Rows
	var err error
	rows, err = co.Query(builder)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		defer rows.Close()
		list := make([]*BlogPost, 0)
		for rows.Next() {
			chat := BlogPost{}
			if err := rows.Scan(&chat.ChatId, &chat.MessageId, &chat.OwnerId, &chat.Text); err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			} else {
				list = append(list, &chat)
			}
		}
		return list, nil
	}
}

func (tx *Tx) GetBlogPostsByChatIds(ids []int64) ([]*BlogPost, error) {
	return getBlogPostsByChatIdsCommon(tx, ids)
}

func (db *DB) GetBlogPostsByChatIds(ids []int64) ([]*BlogPost, error) {
	return getBlogPostsByChatIdsCommon(db, ids)
}

func (db *DB) GetBlogPostMessageId(chatId int64) (int64, error) {
	res := db.QueryRow(fmt.Sprintf("(select id from message_chat_%v where blog_post is true order by id limit 1)", chatId))
	var messageId int64
	if err := res.Scan(&messageId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// there were no rows, but otherwise no error occurred
			return 0, nil
		}
		return 0, eris.Wrap(err, "error during interacting with db")
	}
	return messageId, nil
}

func (db *DB) GetBlobPostModifiedDates(chatIds []int64) (map[int64]time.Time, error) {
	res := map[int64]time.Time{}

	if len(chatIds) == 0 {
		return res, nil
	}

	var builder = ""
	var first = true
	for _, chatId := range chatIds {
		if !first {
			builder += " UNION ALL "
		}
		builder += fmt.Sprintf("(select %v, coalesce(edit_date_time, create_date_time) from message_chat_%v where blog_post is true order by id limit 1)", chatId, chatId)

		first = false
	}

	var rows *sql.Rows
	var err error
	rows, err = db.Query(builder)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	defer rows.Close()
	for rows.Next() {
		var chatId int64
		var modifiedDateTime time.Time
		if err := rows.Scan(&chatId, &modifiedDateTime); err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		} else {
			res[chatId] = modifiedDateTime
		}
	}
	return res, nil
}

func (db *DB) InitUserChatNotificationSettings(userId, chatId int64) error {
	if _, err := db.Exec(`insert into chat_participant_notification(user_id, chat_id) values($1, $2) on conflict(user_id, chat_id) do nothing`, userId, chatId); err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func (db *DB) PutUserChatNotificationSettings(considerMessagesOfThisChatAsUnread *bool, userId, chatId int64) error {
	_, err := db.Exec("update chat_participant_notification set consider_messages_as_unread = $1 where user_id = $2 and chat_id = $3", considerMessagesOfThisChatAsUnread, userId, chatId)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func (db *DB) GetUserChatNotificationSettings(userId, chatId int64) (*bool, error) {
	res := db.QueryRow(`SELECT consider_messages_as_unread FROM chat_participant_notification where user_id = $1 and chat_id = $2`, userId, chatId)
	var consider *bool
	if err := res.Scan(&consider); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// if there is no rows then return default
			return nil, nil
		}
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	return consider, nil
}

func (db *DB) DeleteAllParticipants() error {
	// see aaa/src/main/resources/db/demo/V32000__demo.sql
	// 1 admin
	// 2 nikita
	// 3 alice
	// 4 bob
	// 5 John Smith
	_, err := db.Exec("DELETE FROM chat_participant WHERE user_id > 5")
	return eris.Wrap(err, "error during interacting with db")
}
