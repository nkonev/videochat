package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/guregu/null"
	"github.com/rotisserie/eris"
	"github.com/spf13/viper"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
	"time"
)

const ReservedPublicallyAvailableForSearchChats = "__AVAILABLE_FOR_SEARCH"

const real_chat_columns = `
	id, 
	title, 
	avatar, 
	avatar_big,
	last_update_date_time,
	tet_a_tet,
	can_resend,
	available_to_search,
	pinned,
	blog,
	regular_participant_can_publish_message,
	regular_participant_can_pin_message
`

const select_chat = `
SELECT 
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
	ch.regular_participant_can_publish_message,
	ch.regular_participant_can_pin_message

`
const chat_order = " ORDER BY (cp.user_id is not null, ch.last_update_date_time, ch.id) "
const chat_from = `
FROM chat ch 
LEFT JOIN chat_pinned cp on (ch.id = cp.chat_id and cp.user_id = $1)
`
const chat_of_participant = "SELECT chat_id FROM chat_participant WHERE user_id = $1"
const chat_where = "ch.id IN ( " + chat_of_participant + " )"

// db model
type Chat struct {
	Id                                  int64
	Title                               string
	LastUpdateDateTime                  time.Time
	TetATet                             bool
	CanResend                           bool
	Avatar                              null.String
	AvatarBig                           null.String
	AvailableToSearch                   bool
	Pinned                              bool
	Blog                                bool
	RegularParticipantCanPublishMessage bool
	RegularParticipantCanPinMessage     bool
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
func (tx *Tx) CreateChat(ctx context.Context, u *Chat) (int64, *time.Time, error) {
	// Validate the input.
	if u == nil {
		return 0, nil, eris.New("chat required")
	} else if u.Title == "" {
		return 0, nil, eris.New("title required")
	}

	// https://stackoverflow.com/questions/4547672/return-multiple-fields-as-a-record-in-postgresql-with-pl-pgsql/6085167#6085167
	res := tx.QueryRowContext(ctx, `SELECT chat_id, last_update_date_time FROM CREATE_CHAT($1, $2, $3, $4, $5, $6, $7) AS (chat_id BIGINT, last_update_date_time TIMESTAMP)`, u.Title, u.TetATet, u.CanResend, u.AvailableToSearch, u.Blog, u.RegularParticipantCanPublishMessage, u.RegularParticipantCanPinMessage)
	var id int64
	var lastUpdateDateTime time.Time
	if err := res.Scan(&id, &lastUpdateDateTime); err != nil {
		return 0, nil, eris.Wrap(err, "error during interacting with db")
	}

	return id, &lastUpdateDateTime, nil
}

func (tx *Tx) CreateTetATetChat(ctx context.Context, behalfUserId int64, toParticipantId int64) (int64, error) {
	tetATetChatName := fmt.Sprintf("tet_a_tet_%v_%v", behalfUserId, toParticipantId)
	chatId, _, err := tx.CreateChat(ctx, &Chat{Title: tetATetChatName, TetATet: true, CanResend: viper.GetBool("canResendFromTetATet")})
	return chatId, err
}

func (tx *Tx) IsExistsTetATet(ctx context.Context, participant1 int64, participant2 int64) (bool, int64, error) {
	res := tx.QueryRowContext(ctx, "select b.chat_id from (select a.count >= 2 as exists, a.chat_id from ( (select cp.chat_id, count(cp.user_id) from chat_participant cp join chat ch on ch.id = cp.chat_id where ch.tet_a_tet = true and (cp.user_id = $1 or cp.user_id = $2) group by cp.chat_id)) a) b where b.exists is true;", participant1, participant2)
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
func selectChatWithRowNumbersClause(orderDirection string) string {
	return select_chat + `
			, row_number() over ( ` + chat_order + orderDirection + ` ) as rn		
` + chat_from
}

// expects $1 is userId
func selectChatClause() string {
	return select_chat + chat_from
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
		&chat.RegularParticipantCanPinMessage,
	}
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
	return fmt.Sprintf(" ( ( %s AND ( ch.title ILIKE $2 %s ) ) OR ( (ch.available_to_search = TRUE OR ch.blog = TRUE) AND $3 = '%s' ) )",
		chat_where, additionalUserIdsClause, ReservedPublicallyAvailableForSearchChats,
	)
}

func convertToWithParticipants(ctx context.Context, db CommonOperations, chat *Chat, behalfUserId int64, participantsSize, participantsOffset int) (*ChatWithParticipants, error) {
	if ids, err := db.GetParticipantIds(ctx, chat.Id, participantsSize, participantsOffset); err != nil {
		return nil, err
	} else {
		admin, err := db.IsAdmin(ctx, behalfUserId, chat.Id)
		if err != nil {
			return nil, err
		}
		participantsCount, err := db.GetParticipantsCount(ctx, chat.Id)
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

func convertToWithoutParticipants(ctx context.Context, db CommonOperations, chat *Chat, behalfUserId int64) (*ChatWithParticipants, error) {
	admin, err := db.IsAdmin(ctx, behalfUserId, chat.Id)
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

func getChats(ctx context.Context, co CommonOperations, participantId int64, limit int, leftRowNumber, rightRowNumber int64, orderDirection string, searchString, searchStringPercents string, additionalFoundUserIds []int64) ([]*Chat, error) {
	list := make([]*Chat, 0)

	var rows *sql.Rows
	var err error

	if searchString != "" {
		rows, err = co.QueryContext(ctx, fmt.Sprintf(`
					select %s from (
						%v
						WHERE 
								%s
					) inn
					WHERE	inn.rn >= $5
						AND inn.rn <= $6 
					LIMIT $4`, real_chat_columns, selectChatWithRowNumbersClause(orderDirection), getChatSearchClause(additionalFoundUserIds)),
			participantId, searchStringPercents, searchString,
			limit, leftRowNumber, rightRowNumber)
		if err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		}
		defer rows.Close()
	} else {
		rows, err = co.QueryContext(ctx, fmt.Sprintf(`
					select %s from (
						%v
						WHERE 
								%s
					) inn
					WHERE	inn.rn >= $3 
						AND inn.rn <= $4
					LIMIT $2`, real_chat_columns, selectChatWithRowNumbersClause(orderDirection), chat_where),
			participantId,
			limit, leftRowNumber, rightRowNumber)
		if err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		}
		defer rows.Close()
	}
	for rows.Next() {
		chat := Chat{}
		if err = rows.Scan(provideScanToChat(&chat)[:]...); err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		} else {
			list = append(list, &chat)
		}
	}
	return list, nil
}

func getChatsSimple(ctx context.Context, co CommonOperations, participantId int64, limit int, reverse bool, searchString, searchStringPercents string, additionalFoundUserIds []int64) ([]*Chat, error) {
	list := make([]*Chat, 0)

	order := "desc"
	if reverse {
		order = "asc"
	}
	var err error
	var rows *sql.Rows

	if searchString != "" {
		rows, err = co.QueryContext(ctx, fmt.Sprintf(`%v
			WHERE   
					%s
			%s %s 
			LIMIT $4`, selectChatClause(), getChatSearchClause(additionalFoundUserIds), chat_order, order),
			participantId, searchStringPercents, searchString,
			limit)
		if err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		}
		defer rows.Close()
	} else {
		rows, err = co.QueryContext(ctx, fmt.Sprintf(`%v
			WHERE 
			         %s
			%s %s 
			LIMIT $2`, selectChatClause(), chat_where, chat_order, order),
			participantId,
			limit)
		if err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		}
		defer rows.Close()
	}

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

// see also ChatFilter
func getRowNumbers(ctx context.Context, co CommonOperations, participantId int64, orderDirection string, startingFromItemId int64, limit, leftLimit, rightLimit int, searchString, searchStringPercents string, additionalFoundUserIds []int64) (*int64, *int64, bool, error) {
	var leftRowNumber, rightRowNumber *int64
	var noData bool

	var limitRes *sql.Row
	if searchString != "" {
		limitRes = co.QueryRowContext(ctx, fmt.Sprintf(`
				select inn4.minrn, inn4.maxrn from (
					select inn3.*, lag(rn, $5, inn3.mmin) over() as minrn, lead(rn, $6, inn3.mmax) over() as maxrn from (
						select inn2.*, id = $4 as central_element from (
							select id, rn, (min(inn.rn) over ()) as mmin, (max(inn.rn) over ()) as mmax FROM (
								select id, row_number() over (%s %s) as rn 
								%s
								where %s
							) inn
						) inn2
					) inn3
				) inn4 where central_element = true
			`, chat_order, orderDirection, chat_from, getChatSearchClause(additionalFoundUserIds)),
			participantId, searchStringPercents, searchString, startingFromItemId, leftLimit, rightLimit)
	} else {
		limitRes = co.QueryRowContext(ctx, fmt.Sprintf(`
				select inn4.minrn, inn4.maxrn from (
					select inn3.*, lag(rn, $3, inn3.mmin) over() as minrn, lead(rn, $4, inn3.mmax) over() as maxrn from (
						select inn2.*, id = $2 as central_element from (
							select id, rn, (min(inn.rn) over ()) as mmin, (max(inn.rn) over ()) as mmax FROM (
								select id, row_number() over (%s %s) as rn 
								%s
								where %s
							) inn
						) inn2
					) inn3
				) inn4 where central_element = true
			`, chat_order, orderDirection, chat_from, chat_where),
			participantId, startingFromItemId, leftLimit, rightLimit)
	}
	err := limitRes.Scan(&leftRowNumber, &rightRowNumber)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// there were no rows, but otherwise no error occurred
			return nil, nil, false, nil
		}
		return nil, nil, false, eris.Wrap(err, "error during interacting with db")
	}

	// 1001 962 (39) = ok
	// 1001 1   (1000) = not ok
	// 414 454  (40)   = ok
	if (rightRowNumber != nil && leftRowNumber != nil) && ((*rightRowNumber)-(*leftRowNumber) > int64(limit)) {
		noData = true
	}

	return leftRowNumber, rightRowNumber, noData, nil
}

func getChatsCommon(ctx context.Context, co CommonOperations, participantId int64, limit int, startingFromItemId *int64, reverse, hasHash bool, searchString string, additionalFoundUserIds []int64) ([]*Chat, error) {
	list := make([]*Chat, 0)
	var err error
	orderDirection := "desc"
	if reverse {
		orderDirection = "asc"
	}
	var searchStringPercents = ""
	if searchString != "" {
		searchStringPercents = "%" + searchString + "%"
	}

	if hasHash {
		// has hash means that frontend's page has chat hash
		// it means we need to calculate page/2 to the top and to the bottom
		// to respond page containing from two halves
		leftLimit := limit / 2
		rightLimit := limit / 2

		if leftLimit == 0 {
			leftLimit = 1
		}
		if rightLimit == 0 {
			rightLimit = 1
		}

		var leftRowNumber, rightRowNumber *int64
		var noData bool
		if startingFromItemId != nil {
			leftRowNumber, rightRowNumber, noData, err = getRowNumbers(ctx, co, participantId, orderDirection, *startingFromItemId, limit, leftLimit, rightLimit, searchString, searchStringPercents, additionalFoundUserIds)
			if err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			}
		}

		if noData {
			// leave empty list
		} else if startingFromItemId == nil || (leftRowNumber == nil || rightRowNumber == nil) {
			co.logger().Infof("Got leftItemId=%v, rightItemId=%v startingFromItemId=%v, reverse=%v, searchString=%v, fallback to simple", leftRowNumber, rightRowNumber, startingFromItemId, reverse, searchString)
			list, err = getChatsSimple(ctx, co, participantId, limit, reverse, searchString, searchStringPercents, additionalFoundUserIds)
			if err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			}
		} else {
			list, err = getChats(ctx, co, participantId, limit, *leftRowNumber, *rightRowNumber, orderDirection, searchString, searchStringPercents, additionalFoundUserIds)
			if err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			}
		}
	} else {
		// otherwise, startingFromItemId is used as the top or the bottom limit of the portion
		leftLimit := -1 // not to send the element with startingFromItemId to response
		rightLimit := limit

		var leftRowNumber, rightRowNumber *int64
		var noData bool

		if startingFromItemId != nil {
			leftRowNumber, rightRowNumber, noData, err = getRowNumbers(ctx, co, participantId, orderDirection, *startingFromItemId, limit, leftLimit, rightLimit, searchString, searchStringPercents, additionalFoundUserIds)
			if err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			}
		}

		if noData {
			// leave empty list
		} else if startingFromItemId == nil || (leftRowNumber == nil || rightRowNumber == nil) {
			co.logger().Infof("Got leftItemId=%v, rightItemId=%v startingFromItemId=%v, reverse=%v, searchString=%v, fallback to simple", leftRowNumber, rightRowNumber, startingFromItemId, reverse, searchString)
			list, err = getChatsSimple(ctx, co, participantId, limit, reverse, searchString, searchStringPercents, additionalFoundUserIds)
			if err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			}
		} else {
			list, err = getChats(ctx, co, participantId, limit, *leftRowNumber, *rightRowNumber, orderDirection, searchString, searchStringPercents, additionalFoundUserIds)
			if err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			}
		}
	}

	return list, nil
}

func getChatsWithParticipantsCommon(ctx context.Context, commonOps CommonOperations, participantId int64, limit int, startingFromItemId *int64, reverse, hasHash bool, searchString string, additionalFoundUserIds []int64, participantsSize, participantsOffset int) ([]*ChatWithParticipants, error) {
	var err error
	var chats []*Chat

	chats, err = getChatsCommon(ctx, commonOps, participantId, limit, startingFromItemId, reverse, hasHash, searchString, additionalFoundUserIds)

	if err != nil {
		return nil, err
	} else {
		var chatIds []int64 = make([]int64, 0)
		for _, cc := range chats {
			chatIds = append(chatIds, cc.Id)
		}

		fixedParticipantsSize := utils.FixSize(participantsSize)
		participantIdsBatch, err := commonOps.GetParticipantIdsBatch(ctx, chatIds, fixedParticipantsSize)
		if err != nil {
			return nil, err
		}

		isAdminBatch, err := commonOps.IsAdminBatch(ctx, participantId, chatIds)
		if err != nil {
			return nil, err
		}

		participantsCountBatch, err := commonOps.GetParticipantsCountBatch(ctx, chatIds)
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
func (db *DB) GetChatsWithParticipants(ctx context.Context, participantId int64, limit int, startingFromItemId *int64, reverse, hasHash bool, searchString string, additionalFoundUserIds []int64, participantsSize, participantsOffset int) ([]*ChatWithParticipants, error) {
	return getChatsWithParticipantsCommon(ctx, db, participantId, limit, startingFromItemId, reverse, hasHash, searchString, additionalFoundUserIds, participantsSize, participantsOffset)
}

func (tx *Tx) GetChatsWithParticipants(ctx context.Context, participantId int64, limit int, startingFromItemId *int64, reverse, hasHash bool, searchString string, additionalFoundUserIds []int64, participantsSize, participantsOffset int) ([]*ChatWithParticipants, error) {
	return getChatsWithParticipantsCommon(ctx, tx, participantId, limit, startingFromItemId, reverse, hasHash, searchString, additionalFoundUserIds, participantsSize, participantsOffset)
}

func (tx *Tx) GetChatWithParticipants(ctx context.Context, behalfParticipantId, chatId int64, participantsSize, participantsOffset int) (*ChatWithParticipants, error) {
	return getChatWithParticipantsCommon(ctx, tx, behalfParticipantId, chatId, participantsSize, participantsOffset)
}

func (db *DB) GetChatWithParticipants(ctx context.Context, behalfParticipantId, chatId int64, participantsSize, participantsOffset int) (*ChatWithParticipants, error) {
	return getChatWithParticipantsCommon(ctx, db, behalfParticipantId, chatId, participantsSize, participantsOffset)
}

func getChatWithParticipantsCommon(ctx context.Context, commonOps CommonOperations, behalfParticipantId, chatId int64, participantsSize, participantsOffset int) (*ChatWithParticipants, error) {
	if chat, err := commonOps.GetChat(ctx, behalfParticipantId, chatId); err != nil {
		return nil, err
	} else if chat == nil {
		return nil, nil
	} else {
		return convertToWithParticipants(ctx, commonOps, chat, behalfParticipantId, participantsSize, participantsOffset)
	}
}

func (tx *Tx) GetChatWithoutParticipants(ctx context.Context, behalfParticipantId, chatId int64) (*ChatWithParticipants, error) {
	return getChatWithoutParticipantsCommon(ctx, tx, behalfParticipantId, chatId)
}

func (db *DB) GetChatWithoutParticipants(ctx context.Context, behalfParticipantId, chatId int64) (*ChatWithParticipants, error) {
	return getChatWithoutParticipantsCommon(ctx, db, behalfParticipantId, chatId)
}

func getChatWithoutParticipantsCommon(ctx context.Context, commonOps CommonOperations, behalfParticipantId, chatId int64) (*ChatWithParticipants, error) {
	if chat, err := commonOps.GetChat(ctx, behalfParticipantId, chatId); err != nil {
		return nil, err
	} else if chat == nil {
		return nil, nil
	} else {
		return convertToWithoutParticipants(ctx, commonOps, chat, behalfParticipantId)
	}
}

func (db *DB) CountChats(ctx context.Context) (int64, error) {
	var count int64
	row := db.QueryRowContext(ctx, "SELECT count(*) FROM chat")
	err := row.Scan(&count)
	if err != nil {
		return 0, eris.Wrap(err, "error during interacting with db")
	} else {
		return count, nil
	}
}

func countChatsPerUser(ctx context.Context, commonOps CommonOperations, userId int64) (int64, error) {
	var count int64
	row := commonOps.QueryRowContext(ctx, "SELECT count(*) FROM chat_participant WHERE user_id = $1", userId)
	err := row.Scan(&count)
	if err != nil {
		return 0, eris.Wrap(err, "error during interacting with db")
	} else {
		return count, nil
	}
}

func (db *DB) CountChatsPerUser(ctx context.Context, userId int64) (int64, error) {
	return countChatsPerUser(ctx, db, userId)
}

func (tx *Tx) CountChatsPerUser(ctx context.Context, userId int64) (int64, error) {
	return countChatsPerUser(ctx, tx, userId)
}

func (tx *Tx) DeleteChat(ctx context.Context, id int64) error {
	if _, err := tx.ExecContext(ctx, `CALL DELETE_CHAT($1)`, id); err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func (tx *Tx) EditChat(ctx context.Context, id int64, newTitle string, avatar, avatarBig null.String, canResend bool, availableToSearch bool, blog *bool, regularParticipantCanPublishMessage bool, regularParticipantCanPinMessage bool) (*time.Time, error) {
	var res sql.Result
	var err error
	if blog != nil {
		isBlog := utils.NullableToBoolean(blog)
		res, err = tx.ExecContext(ctx, `UPDATE chat SET title = $2, avatar = $3, avatar_big = $4, last_update_date_time = utc_now(), can_resend = $5, available_to_search = $6, blog = $7, regular_participant_can_publish_message = $8, regular_participant_can_pin_message = $9 WHERE id = $1`, id, newTitle, avatar, avatarBig, canResend, availableToSearch, isBlog, regularParticipantCanPublishMessage, regularParticipantCanPinMessage)
	} else {
		res, err = tx.ExecContext(ctx, `UPDATE chat SET title = $2, avatar = $3, avatar_big = $4, last_update_date_time = utc_now(), can_resend = $5, available_to_search = $6, regular_participant_can_publish_message = $7, regular_participant_can_pin_message = $8 WHERE id = $1`, id, newTitle, avatar, avatarBig, canResend, availableToSearch, regularParticipantCanPublishMessage, regularParticipantCanPinMessage)
	}
	if err != nil {
		GetLogEntry(ctx, tx.lgr).Errorf("Error during editing chat id %v", err)
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
	res2 := tx.QueryRowContext(ctx, `SELECT last_update_date_time FROM chat WHERE id = $1`, id)
	if err := res2.Scan(&lastUpdateDateTime); err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}

	return &lastUpdateDateTime, nil
}

func getChatCommon(ctx context.Context, co CommonOperations, participantId, chatId int64) (*Chat, error) {
	row := co.QueryRowContext(ctx, selectChatClause()+` WHERE ch.id in (`+chat_of_participant+` AND chat_id = $2)`, participantId, chatId)
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

func (db *DB) GetChat(ctx context.Context, participantId, chatId int64) (*Chat, error) {
	return getChatCommon(ctx, db, participantId, chatId)
}

func (tx *Tx) GetChat(ctx context.Context, participantId, chatId int64) (*Chat, error) {
	return getChatCommon(ctx, tx, participantId, chatId)
}

func getChatBasicCommon(ctx context.Context, co CommonOperations, chatId int64) (*BasicChatDto, error) {
	row := co.QueryRowContext(ctx, `SELECT 
				ch.id, 
				ch.title, 
				ch.tet_a_tet,
				ch.can_resend,
				ch.available_to_search,
				ch.blog,
				ch.regular_participant_can_publish_message,
				ch.regular_participant_can_pin_message
			FROM chat ch 
			WHERE ch.id = $1
`, chatId)
	chat := BasicChatDto{}
	err := row.Scan(&chat.Id, &chat.Title, &chat.IsTetATet, &chat.CanResend, &chat.AvailableToSearch, &chat.IsBlog, &chat.RegularParticipantCanPublishMessage, &chat.RegularParticipantCanPinMessage)
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

func (db *DB) GetChatBasic(ctx context.Context, chatId int64) (*BasicChatDto, error) {
	return getChatBasicCommon(ctx, db, chatId)
}

func (tx *Tx) GetChatBasic(ctx context.Context, chatId int64) (*BasicChatDto, error) {
	return getChatBasicCommon(ctx, tx, chatId)
}

func getBlogChatBasicCommon(ctx context.Context, co CommonOperations, chatId int64) (*BasicBlogDto, error) {
	row := co.QueryRowContext(ctx, `SELECT 
				ch.id, 
				ch.title, 
				ch.blog,
				ch.create_date_time
			FROM chat ch 
			WHERE ch.id = $1
`, chatId)
	chat := BasicBlogDto{}
	err := row.Scan(&chat.Id, &chat.Title, &chat.IsBlog, &chat.CreateDateTime)
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

func (db *DB) GetBlogBasic(ctx context.Context, chatId int64) (*BasicBlogDto, error) {
	return getBlogChatBasicCommon(ctx, db, chatId)
}

func (tx *Tx) GetBlogBasic(ctx context.Context, chatId int64) (*BasicBlogDto, error) {
	return getBlogChatBasicCommon(ctx, tx, chatId)
}

func getChatsBasicCommon(ctx context.Context, co CommonOperations, chatIds map[int64]bool, behalfParticipantId int64) (map[int64]*BasicChatDtoExtended, error) {
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
	rows, err := co.QueryContext(ctx, fmt.Sprintf(`
		SELECT 
			c.id, 
			c.title,
			(cp.user_id is not null),
			c.tet_a_tet,
			c.can_resend,
			c.available_to_search,
			c.blog,
			c.regular_participant_can_publish_message,
			c.regular_participant_can_pin_message
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
			if err := rows.Scan(&dto.Id, &dto.Title, &dto.BehalfUserIsParticipant, &dto.IsTetATet, &dto.CanResend, &dto.AvailableToSearch, &dto.IsBlog, &dto.RegularParticipantCanPublishMessage, &dto.RegularParticipantCanPinMessage); err != nil {
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

func (db *DB) GetChatsBasic(ctx context.Context, chatIds map[int64]bool, behalfParticipantId int64) (map[int64]*BasicChatDtoExtended, error) {
	return getChatsBasicCommon(ctx, db, chatIds, behalfParticipantId)
}

func (tx *Tx) GetChatsBasic(ctx context.Context, chatIds map[int64]bool, behalfParticipantId int64) (map[int64]*BasicChatDtoExtended, error) {
	return getChatsBasicCommon(ctx, tx, chatIds, behalfParticipantId)
}

type BasicChatDto struct {
	Id                                  int64
	Title                               string
	IsTetATet                           bool
	CanResend                           bool
	AvailableToSearch                   bool
	IsBlog                              bool
	CreateDateTime                      time.Time
	RegularParticipantCanPublishMessage bool
	RegularParticipantCanPinMessage     bool
}

type BasicBlogDto struct {
	Id             int64
	Title          string
	IsBlog         bool
	CreateDateTime time.Time
}

type BasicChatDtoExtended struct {
	BasicChatDto
	BehalfUserIsParticipant bool
}

func (tx *Tx) UpdateChatLastDatetimeChat(ctx context.Context, id int64) error {
	if _, err := tx.ExecContext(ctx, "UPDATE chat SET last_update_date_time = utc_now() WHERE id = $1", id); err != nil {
		return eris.Wrap(err, "error during interacting with db")
	} else {
		return nil
	}
}

func (tx *Tx) GetChatLastDatetimeChat(ctx context.Context, chatId int64) (time.Time, error) {
	row := tx.QueryRowContext(ctx, `SELECT last_update_date_time FROM chat WHERE id = $1`, chatId)
	var lastUpdateDateTime time.Time
	err := row.Scan(&lastUpdateDateTime)
	if err != nil {
		return lastUpdateDateTime, eris.Wrap(err, "error during interacting with db")
	} else {
		return lastUpdateDateTime, nil
	}
}

func (db *DB) GetExistingChatIds(ctx context.Context, chatIds []int64) (*[]int64, error) {
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

	rows, err := db.QueryContext(ctx, fmt.Sprintf(`SELECT id FROM chat WHERE id IN (%s)`, additionalChatIds))
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

func pinChatCommon(ctx context.Context, co CommonOperations, chatId int64, userId int64, pin bool) error {
	if pin {
		_, err := co.ExecContext(ctx, "insert into chat_pinned(user_id, chat_id) values ($1, $2) on conflict do nothing", userId, chatId)
		if err != nil {
			return eris.Wrap(err, "error during interacting with db")
		}
	} else {
		_, err := co.ExecContext(ctx, "delete from chat_pinned where user_id = $1 and chat_id = $2", userId, chatId)
		if err != nil {
			return eris.Wrap(err, "error during interacting with db")
		}
	}
	return nil
}

func (db *DB) PinChat(ctx context.Context, chatId int64, userId int64, pin bool) error {
	return pinChatCommon(ctx, db, chatId, userId, pin)
}

func (tx *Tx) PinChat(ctx context.Context, chatId int64, userId int64, pin bool) error {
	return pinChatCommon(ctx, tx, chatId, userId, pin)
}

func (tx *Tx) IsChatPinnedBatch(ctx context.Context, userIds []int64, chatId int64) (map[int64]bool, error) {
	res := map[int64]bool{}

	var rows *sql.Rows
	var err error
	rows, err = tx.QueryContext(ctx, `
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

func (tx *Tx) DeleteChatsPinned(ctx context.Context, userId int64) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM chat_pinned WHERE user_id = $1", userId); err != nil {
		return eris.Wrap(err, "error during interacting with db")
	} else {
		return nil
	}
}

func (tx *Tx) RenameChat(ctx context.Context, chatId int64, title string) error {
	_, err := tx.ExecContext(ctx, "update chat set title = $1 where id = $2", title, chatId)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func getChatIdsByParticipantIdCommon(ctx context.Context, co CommonOperations, participantId int64, limit int, offset int) ([]int64, error) {
	var rows *sql.Rows
	var err error
	rows, err = co.QueryContext(ctx, fmt.Sprintf(`SELECT ch.id from chat ch
		WHERE %s
		ORDER BY ch.id
		LIMIT $2 OFFSET $3
	`, chat_where), participantId, limit, offset)
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

func getChatIdsCommon(ctx context.Context, qq CommonOperations, chatsSize, chatsOffset int) ([]int64, error) {
	if rows, err := qq.QueryContext(ctx, "SELECT id FROM chat ORDER BY id LIMIT $1 OFFSET $2", chatsSize, chatsOffset); err != nil {
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

func (tx *Tx) GetChatIds(ctx context.Context, chatsSize, chatsOffset int) ([]int64, error) {
	return getChatIdsCommon(ctx, tx, chatsSize, chatsOffset)
}

func (db *DB) GetChatIds(ctx context.Context, chatsSize, chatsOffset int) ([]int64, error) {
	return getChatIdsCommon(ctx, db, chatsSize, chatsOffset)
}

func getBlogPostsByLimitOffsetCommon(ctx context.Context, co CommonOperations, reverse bool, limit int, offset int) ([]*Blog, error) {
	var rows *sql.Rows
	var err error
	var sort string
	if reverse {
		sort = "asc"
	} else {
		sort = "desc"
	}
	rows, err = co.QueryContext(ctx, fmt.Sprintf(`SELECT 
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

func (tx *Tx) GetBlogPostsByLimitOffset(ctx context.Context, reverse bool, limit int, offset int) ([]*Blog, error) {
	return getBlogPostsByLimitOffsetCommon(ctx, tx, reverse, limit, offset)
}

func (db *DB) GetBlogPostsByLimitOffset(ctx context.Context, reverse bool, limit int, offset int) ([]*Blog, error) {
	return getBlogPostsByLimitOffsetCommon(ctx, db, reverse, limit, offset)
}

func (db *DB) CountBlogs(ctx context.Context) (int64, error) {
	res := db.QueryRowContext(ctx, "SELECT count(*) FROM chat ch WHERE ch.blog IS TRUE")
	var count int64
	if err := res.Scan(&count); err != nil {
		return 0, eris.Wrap(err, "error during interacting with db")
	}
	return count, nil
}

type BlogPost struct {
	ChatId    int64
	MessageId int64
	OwnerId   int64
	Text      string
}

func getBlogPostsByChatIdsCommon(ctx context.Context, co CommonOperations, ids []int64) ([]*BlogPost, error) {
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
	rows, err = co.QueryContext(ctx, builder)
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

func (tx *Tx) GetBlogPostsByChatIds(ctx context.Context, ids []int64) ([]*BlogPost, error) {
	return getBlogPostsByChatIdsCommon(ctx, tx, ids)
}

func (db *DB) GetBlogPostsByChatIds(ctx context.Context, ids []int64) ([]*BlogPost, error) {
	return getBlogPostsByChatIdsCommon(ctx, db, ids)
}

func (db *DB) GetBlogPostMessageId(ctx context.Context, chatId int64) (int64, error) {
	res := db.QueryRowContext(ctx, fmt.Sprintf("(select id from message_chat_%v where blog_post is true order by id limit 1)", chatId))
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

func (db *DB) GetBlobPostModifiedDates(ctx context.Context, chatIds []int64) (map[int64]time.Time, error) {
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
	rows, err = db.QueryContext(ctx, builder)
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

func (db *DB) InitUserChatNotificationSettings(ctx context.Context, userId, chatId int64) error {
	if _, err := db.ExecContext(ctx, `insert into chat_participant_notification(user_id, chat_id) values($1, $2) on conflict(user_id, chat_id) do nothing`, userId, chatId); err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func (db *DB) PutUserChatNotificationSettings(ctx context.Context, considerMessagesOfThisChatAsUnread *bool, userId, chatId int64) error {
	_, err := db.ExecContext(ctx, "update chat_participant_notification set consider_messages_as_unread = $1 where user_id = $2 and chat_id = $3", considerMessagesOfThisChatAsUnread, userId, chatId)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func (db *DB) GetUserChatNotificationSettings(ctx context.Context, userId, chatId int64) (*bool, error) {
	res := db.QueryRowContext(ctx, `SELECT consider_messages_as_unread FROM chat_participant_notification where user_id = $1 and chat_id = $2`, userId, chatId)
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

// see also getRowNumbers
func (tx *Tx) ChatFilter(ctx context.Context, participantId int64, chatId, edgeChatId int64, pageSize int, reverse bool, searchString string, additionalFoundUserIds []int64) (bool, error) {

	orderDirection := "desc"
	if reverse {
		orderDirection = "asc"
	}

	var searchStringWithPercents = ""
	if searchString != "" {
		searchStringWithPercents = "%" + searchString + "%"
	}

	var row *sql.Row
	if searchString != "" {
		row = tx.QueryRowContext(ctx, fmt.Sprintf(`
			with first_page as (
				select inn3.* from (
					select inn2.* from (
						select id, rn FROM (
							select id, row_number() over (%s %s) as rn 
							%s
							where %s
						) inn
					) inn2 limit $4
				) inn3
			)
			select exists (select * from first_page where id = $5) -- chat id to probe
				and exists (select * from first_page where id = $6 and rn in (1, 2)) -- edge on the screen - here we ensure that this is the first page, in (1, 2) means the first place for the toppest element or the second place after sorting
		`, chat_order, orderDirection, chat_from, getChatSearchClause(additionalFoundUserIds)),
			participantId, searchStringWithPercents, searchString, pageSize, chatId, edgeChatId)
	} else {
		row = tx.QueryRowContext(ctx, fmt.Sprintf(`
			with first_page as (
				select inn3.* from (
					select inn2.* from (
						select id, rn FROM (
							select id, row_number() over (%s %s) as rn 
							%s
							where %s
						) inn
					) inn2 limit $2
			  	) inn3
			)
			select exists (select * from first_page where id = $3) -- chat id to probe
				and exists (select * from first_page where id = $4 and rn in (1, 2)) -- edge on the screen - here we ensure that this is the first page, in (1, 2) means the first place for the toppest element or the second place after sorting
		`, chat_order, orderDirection, chat_from, chat_where),
			participantId, pageSize, chatId, edgeChatId)
	}
	if row.Err() != nil {
		GetLogEntry(ctx, tx.lgr).Errorf("Error during get Search %v", row.Err())
		return false, eris.Wrap(row.Err(), "error during interacting with db")
	}

	var found bool
	err := row.Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, eris.Wrap(err, "error during interacting with db")
	}
	return found, nil
}

func (db *DB) DeleteAllParticipants(ctx context.Context) error {
	// see aaa/src/main/resources/db/demo/V32000__demo.sql
	// 1 admin
	// 2 nikita
	// 3 alice
	// 4 bob
	// 5 John Smith
	_, err := db.ExecContext(ctx, "DELETE FROM chat_participant WHERE user_id > 5")
	return eris.Wrap(err, "error during interacting with db")
}
