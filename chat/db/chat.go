package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/rotisserie/eris"
	"github.com/spf13/viper"
	"nkonev.name/chat/utils"
	"time"
)

const ReservedPublicallyAvailableForSearchChats = "__AVAILABLE_FOR_SEARCH"

// expects $1 is userId
func selectChatClause(performPersonalization bool) string {
	bldr := ""

	var p string
	if performPersonalization {
		p = "cp.user_id IS NOT NULL"
	} else {
		p = "($1::bigint != $1::bigint)" // to consume the given user_id
	}
	bldr += fmt.Sprintf(`
		SELECT 
			ch.id, 
			ch.title, 
			ch.avatar, 
			ch.avatar_big,
			ch.last_update_date_time,
			ch.tet_a_tet,
			ch.can_resend,
			ch.available_to_search,
			%s as pinned,
			ch.blog,
			ch.regular_participant_can_publish_message,
			ch.regular_participant_can_pin_message,
			ch.blog_about,
			ch.regular_participant_can_write_message,
			ch.can_react
	`, p)

	var pp string
	if performPersonalization {
		pp = "LEFT JOIN chat_pinned cp on (ch.id = cp.chat_id and cp.user_id = $1)"
	}

	bldr += fmt.Sprintf(` FROM chat ch %s `, pp)

	return bldr
}

// to use only with wrapped selectChatClause(), e. g.
// select * from (%s) ch
const chat_order = " ORDER BY (ch.pinned, ch.last_update_date_time, ch.id) "

const chat_of_participant = "SELECT chat_id FROM chat_participant WHERE user_id = $1"
const chat_where = "ch.id IN ( " + chat_of_participant + " )"

// db model
type Chat struct {
	Id                                  int64
	Title                               string
	LastUpdateDateTime                  time.Time
	TetATet                             bool
	CanResend                           bool
	Avatar                              *string
	AvatarBig                           *string
	AvailableToSearch                   bool
	Pinned                              bool
	Blog                                bool
	RegularParticipantCanPublishMessage bool
	RegularParticipantCanPinMessage     bool
	BlogAbout                           bool
	RegularParticipantCanWriteMessage   bool
	CanReact                            bool
}

type Blog struct {
	Id             int64
	Title          string
	CreateDateTime time.Time
	Avatar         *string
}

type ChatWithParticipants struct {
	Chat
	ParticipantsIds    []int64
	ParticipantsCount  int
	IsAdmin            bool
	LastMessagePreview *string
	LastMessageOwnerId *int64
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

	res := tx.QueryRowContext(ctx, `
		INSERT INTO chat(title, tet_a_tet, can_resend, available_to_search, blog, regular_participant_can_publish_message, regular_participant_can_pin_message, blog_about, regular_participant_can_write_message, can_react)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, last_update_date_time;
	`, u.Title, u.TetATet, u.CanResend, u.AvailableToSearch, u.Blog, u.RegularParticipantCanPublishMessage, u.RegularParticipantCanPinMessage, u.BlogAbout, u.RegularParticipantCanWriteMessage, u.CanReact)
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
		&chat.BlogAbout,
		&chat.RegularParticipantCanWriteMessage,
		&chat.CanReact,
	}
}

// requires
// $1 - owner_id
// $2 - searchStringWithPercents
// $3 - searchString
func getChatSearchWhereClause(additionalFoundUserIds []int64) string {
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

func convertToWithParticipants(ctx context.Context, co CommonOperations, chat *Chat, behalfUserId int64, participantsSize, participantsOffset int) (*ChatWithParticipants, error) {
	if ids, err := co.GetParticipantIds(ctx, chat.Id, participantsSize, participantsOffset); err != nil {
		return nil, err
	} else {
		admin, err := co.IsAdmin(ctx, behalfUserId, chat.Id)
		if err != nil {
			return nil, err
		}
		participantsCount, err := co.GetParticipantsCount(ctx, chat.Id)
		if err != nil {
			return nil, err
		}

		messagePreviews, err := getLastMessagePreview(ctx, co, []int64{chat.Id})
		if err != nil {
			return nil, err
		}

		ccc := &ChatWithParticipants{
			Chat:              *chat,
			ParticipantsIds:   ids,
			IsAdmin:           admin,
			ParticipantsCount: participantsCount,
		}

		messagePreview := messagePreviews[chat.Id]

		if messagePreview != nil {
			ccc.LastMessagePreview = &messagePreview.LastMessagePreview
			ccc.LastMessageOwnerId = &messagePreview.LastMessageOwnerId
		}

		return ccc, nil
	}
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

func convertToWithParticipantsBatch(chat *Chat, participantIdsBatch []*ParticipantIds, isAdminBatch map[int64]bool, participantsCountBatch map[int64]int, messagePreviewsBatch map[int64]*LastMessagePreview) (*ChatWithParticipants, error) {
	participantsCount := participantsCountBatch[chat.Id]

	var participantsIds []int64 = make([]int64, 0)
	for _, pidsb := range participantIdsBatch {
		if pidsb.ChatId == chat.Id {
			participantsIds = pidsb.ParticipantIds
			break
		}
	}

	admin := isAdminBatch[chat.Id]

	messagePreview := messagePreviewsBatch[chat.Id]

	ccc := &ChatWithParticipants{
		Chat:              *chat,
		ParticipantsIds:   participantsIds,
		IsAdmin:           admin,
		ParticipantsCount: participantsCount,
	}

	if messagePreview != nil {
		ccc.LastMessagePreview = &messagePreview.LastMessagePreview
		ccc.LastMessageOwnerId = &messagePreview.LastMessageOwnerId
	}

	return ccc, nil
}

type LastMessagePreview struct {
	LastMessagePreview string
	LastMessageOwnerId int64
}

func getLastMessagePreview(ctx context.Context, co CommonOperations, chatIds []int64) (map[int64]*LastMessagePreview, error) {
	ret := map[int64]*LastMessagePreview{}
	if len(chatIds) == 0 {
		return ret, nil
	}

	maxPrevSizeDb := viper.GetInt("previewMaxTextSizeDb")

	rows, err := co.QueryContext(ctx, `
		select m.chat_id, substring(strip_tags(m.text), 0, $2), m.owner_id 
		from message m
		join (
			select chat_id, max(id) as message_id from message where chat_id = any($1) group by chat_id
		) inn on m.id = inn.message_id and m.chat_id = inn.chat_id
	`, chatIds, maxPrevSizeDb)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	defer rows.Close()

	for rows.Next() {
		item := &LastMessagePreview{}
		var chatId int64
		if err = rows.Scan(&chatId, &item.LastMessagePreview, &item.LastMessageOwnerId); err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		} else if chatId != 0 {
			ret[chatId] = item
		}
	}
	return ret, nil
}

func getPaginationWhereAndClause(startingFromItemId *ChatId, reverse bool) string {
	if startingFromItemId == nil {
		return ""
	}
	bldr := ""

	nonEquality := "<="
	if reverse {
		nonEquality = ">="
	}

	bldr += fmt.Sprintf("(pinned, last_update_date_time, id) %s (%v, '%s', %v)", nonEquality,
		startingFromItemId.Pinned, startingFromItemId.LastUpdateDateTime.Format("2006-01-02T15:04:05.999999Z"), startingFromItemId.Id)
	bldr += " AND"
	return bldr
}

// implements keyset pagination
func getChatsSimple(ctx context.Context, co CommonOperations, participantId int64, limit int, startingFromItemId *ChatId, includeStartingFrom, reverse bool, searchString, searchStringPercents string, additionalFoundUserIds []int64) ([]*Chat, error) {
	list := make([]*Chat, 0)

	order := "desc"
	offset := " OFFSET 1" // to make behaviour the same as in users, messages (there is > or <)
	if reverse {
		order = "asc"
	}
	// see also getSafeDefaultUserId() in aaa
	if includeStartingFrom || startingFromItemId == nil {
		offset = ""
	}

	var err error
	var rows *sql.Rows

	// we wrap an existing ch and cp into select * from (%s) ch in order to search by pinned (there is no such column)
	if searchString != "" {
		q := fmt.Sprintf(`select * from (%s) ch
			WHERE   %s
					%s
			%s %s 
			LIMIT $4 %s`, selectChatClause(true), getPaginationWhereAndClause(startingFromItemId, reverse), getChatSearchWhereClause(additionalFoundUserIds), chat_order, order, offset)
		rows, err = co.QueryContext(ctx, q,
			participantId, searchStringPercents, searchString,
			limit)
		if err != nil {
			return nil, eris.Wrap(err, "error during interacting with db")
		}
		defer rows.Close()
	} else {
		q := fmt.Sprintf(`select * from (%s) ch
			WHERE 	 %s
			         %s
			%s %s 
			LIMIT $2 %s`, selectChatClause(true), getPaginationWhereAndClause(startingFromItemId, reverse), chat_where, chat_order, order, offset)
		rows, err = co.QueryContext(ctx, q,
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

type ChatId struct {
	Pinned             bool
	LastUpdateDateTime time.Time
	Id                 int64
}

func getChatsCommon(ctx context.Context, co CommonOperations, participantId int64, limit int, startingFromItemId *ChatId, includeStartingFrom, reverse bool, searchString string, additionalFoundUserIds []int64) ([]*Chat, error) {
	list := make([]*Chat, 0)
	var err error
	var searchStringPercents = ""
	if searchString != "" {
		searchStringPercents = "%" + searchString + "%"
	}

	list, err = getChatsSimple(ctx, co, participantId, limit, startingFromItemId, includeStartingFrom, reverse, searchString, searchStringPercents, additionalFoundUserIds)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}

	return list, nil
}

func getChatsWithParticipantsCommon(ctx context.Context, commonOps CommonOperations, participantId int64, limit int, startingFromItemId *ChatId, includeStartingFrom, reverse bool, searchString string, additionalFoundUserIds []int64, participantsSize, participantsOffset int) ([]*ChatWithParticipants, error) {
	var err error
	var chats []*Chat

	chats, err = getChatsCommon(ctx, commonOps, participantId, limit, startingFromItemId, includeStartingFrom, reverse, searchString, additionalFoundUserIds)

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

		messagePreviewsBatch, err := getLastMessagePreview(ctx, commonOps, chatIds)
		if err != nil {
			return nil, err
		}

		list := make([]*ChatWithParticipants, 0)

		for _, cc := range chats {
			if ccc, err := convertToWithParticipantsBatch(cc, participantIdsBatch, isAdminBatch, participantsCountBatch, messagePreviewsBatch); err != nil {
				return nil, err
			} else {
				list = append(list, ccc)
			}
		}
		return list, nil
	}
}
func (db *DB) GetChatsWithParticipants(ctx context.Context, participantId int64, limit int, startingFromItemId *ChatId, includeStartingFrom, reverse bool, searchString string, additionalFoundUserIds []int64, participantsSize, participantsOffset int) ([]*ChatWithParticipants, error) {
	return getChatsWithParticipantsCommon(ctx, db, participantId, limit, startingFromItemId, includeStartingFrom, reverse, searchString, additionalFoundUserIds, participantsSize, participantsOffset)
}

func (tx *Tx) GetChatsWithParticipants(ctx context.Context, participantId int64, limit int, startingFromItemId *ChatId, includeStartingFrom, reverse bool, searchString string, additionalFoundUserIds []int64, participantsSize, participantsOffset int) ([]*ChatWithParticipants, error) {
	return getChatsWithParticipantsCommon(ctx, tx, participantId, limit, startingFromItemId, includeStartingFrom, reverse, searchString, additionalFoundUserIds, participantsSize, participantsOffset)
}

func (tx *Tx) GetChatWithParticipants(ctx context.Context, performPersonalization bool, behalfParticipantId, chatId int64, participantsSize, participantsOffset int) (*ChatWithParticipants, error) {
	return getChatWithParticipantsCommon(ctx, tx, performPersonalization, behalfParticipantId, chatId, participantsSize, participantsOffset)
}

func (db *DB) GetChatWithParticipants(ctx context.Context, performPersonalization bool, behalfParticipantId, chatId int64, participantsSize, participantsOffset int) (*ChatWithParticipants, error) {
	return getChatWithParticipantsCommon(ctx, db, performPersonalization, behalfParticipantId, chatId, participantsSize, participantsOffset)
}

func getChatWithParticipantsCommon(ctx context.Context, commonOps CommonOperations, performPersonalization bool, behalfParticipantId, chatId int64, participantsSize, participantsOffset int) (*ChatWithParticipants, error) {
	if chat, err := commonOps.GetChat(ctx, performPersonalization, behalfParticipantId, chatId); err != nil {
		return nil, err
	} else if chat == nil {
		return nil, nil
	} else {
		return convertToWithParticipants(ctx, commonOps, chat, behalfParticipantId, participantsSize, participantsOffset)
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

func (tx *Tx) EditChat(
	ctx context.Context,
	id int64,
	newTitle string,
	avatar, avatarBig *string,
	canResend bool,
	availableToSearch bool,
	blog *bool, // null is whether to change blog or not
	regularParticipantCanPublishMessage bool,
	regularParticipantCanPinMessage bool,
	blogAbout bool,
	regularParticipantCanWriteMessage bool,
	canReact bool,
) (*time.Time, error) {
	var res sql.Result
	var err error
	if blog != nil {
		isBlog := utils.NullableToBoolean(blog)
		res, err = tx.ExecContext(ctx, `UPDATE chat SET title = $2, avatar = $3, avatar_big = $4, last_update_date_time = utc_now(), can_resend = $5, available_to_search = $6, blog = $7, regular_participant_can_publish_message = $8, regular_participant_can_pin_message = $9, blog_about = $10, regular_participant_can_write_message = $11, can_react = $12 WHERE id = $1`, id, newTitle, avatar, avatarBig, canResend, availableToSearch, isBlog, regularParticipantCanPublishMessage, regularParticipantCanPinMessage, blogAbout, regularParticipantCanWriteMessage, canReact)
	} else {
		res, err = tx.ExecContext(ctx, `UPDATE chat SET title = $2, avatar = $3, avatar_big = $4, last_update_date_time = utc_now(), can_resend = $5, available_to_search = $6, regular_participant_can_publish_message = $7, regular_participant_can_pin_message = $8, regular_participant_can_write_message = $9, can_react = $10 WHERE id = $1`, id, newTitle, avatar, avatarBig, canResend, availableToSearch, regularParticipantCanPublishMessage, regularParticipantCanPinMessage, regularParticipantCanWriteMessage, canReact)
	}
	if err != nil {
		tx.lgr.WithTracing(ctx).Errorf("Error during editing chat id %v", err)
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

func getChatCommon(ctx context.Context, co CommonOperations, performPersonalization bool, participantId, chatId int64) (*Chat, error) {
	s := selectChatClause(performPersonalization) + ` WHERE ch.id = $2 `
	if performPersonalization {
		s += " AND ch.id in (" + chat_of_participant + ")"
	}
	row := co.QueryRowContext(ctx, s, participantId, chatId)
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

func (db *DB) GetChat(ctx context.Context, performPersonalization bool, participantId, chatId int64) (*Chat, error) {
	return getChatCommon(ctx, db, performPersonalization, participantId, chatId)
}

func (tx *Tx) GetChat(ctx context.Context, performPersonalization bool, participantId, chatId int64) (*Chat, error) {
	return getChatCommon(ctx, tx, performPersonalization, participantId, chatId)
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
				ch.regular_participant_can_pin_message,
				ch.regular_participant_can_write_message,
				ch.can_react
			FROM chat ch 
			WHERE ch.id = $1
`, chatId)
	chat := BasicChatDto{}
	err := row.Scan(&chat.Id, &chat.Title, &chat.IsTetATet, &chat.CanResend, &chat.AvailableToSearch, &chat.IsBlog, &chat.RegularParticipantCanPublishMessage, &chat.RegularParticipantCanPinMessage, &chat.RegularParticipantCanWriteMessage, &chat.CanReact)
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
				ch.create_date_time,
				ch.regular_participant_can_write_message
			FROM chat ch 
			WHERE ch.id = $1
`, chatId)
	chat := BasicBlogDto{}
	err := row.Scan(&chat.Id, &chat.Title, &chat.IsBlog, &chat.CreateDateTime, &chat.CanWriteMessage)
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
			c.regular_participant_can_pin_message,
			c.regular_participant_can_write_message,
			c.can_react
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
			if err := rows.Scan(&dto.Id, &dto.Title, &dto.BehalfUserIsParticipant, &dto.IsTetATet, &dto.CanResend, &dto.AvailableToSearch, &dto.IsBlog, &dto.RegularParticipantCanPublishMessage, &dto.RegularParticipantCanPinMessage, &dto.RegularParticipantCanWriteMessage, &dto.CanReact); err != nil {
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
	RegularParticipantCanWriteMessage   bool
	CanReact                            bool
}

type BasicBlogDto struct {
	Id              int64
	Title           string
	IsBlog          bool
	CreateDateTime  time.Time
	CanWriteMessage bool
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
	rows, err = co.QueryContext(ctx, fmt.Sprintf(`SELECT cp.chat_id from chat_participant cp
	 	WHERE cp.user_id = $1
		ORDER BY cp.chat_id
		LIMIT $2 OFFSET $3
	`), participantId, limit, offset)
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
	ChatId       int64
	MessageId    int64
	OwnerId      int64
	Text         string
	FileItemUuid *string
}

func getBlogPostsByChatIdsCommon(ctx context.Context, co CommonOperations, chatIds []int64) ([]*BlogPost, error) {
	var rows *sql.Rows
	var err error
	rows, err = co.QueryContext(ctx, `
		select m.chat_id, m.id, m.owner_id, m.text, m.file_item_uuid from message m where chat_id = any($1) and blog_post = true
	`, chatIds)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		defer rows.Close()
		list := make([]*BlogPost, 0)
		for rows.Next() {
			post := BlogPost{}
			if err := rows.Scan(&post.ChatId, &post.MessageId, &post.OwnerId, &post.Text, &post.FileItemUuid); err != nil {
				return nil, eris.Wrap(err, "error during interacting with db")
			} else {
				list = append(list, &post)
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
	res := db.QueryRowContext(ctx, fmt.Sprintf("(select id from message where chat_id = %v and blog_post is true order by id limit 1)", chatId))
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

func (db *DB) GetBlogPostModifiedDates(ctx context.Context, chatIds []int64) (map[int64]time.Time, error) {
	res := map[int64]time.Time{}

	if len(chatIds) == 0 {
		return res, nil
	}

	var rows *sql.Rows
	var err error
	rows, err = db.QueryContext(ctx, `
		select m.chat_id, coalesce(m.edit_date_time, m.create_date_time) from message m where chat_id = any($1) and blog_post = true
	`, chatIds)
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
func (tx *Tx) ChatFilter(ctx context.Context, participantId int64, chatId int64, reverse bool, searchString string, additionalFoundUserIds []int64) (bool, error) {

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
			with a_page as (
							select * from (%s) ch
							where %s
							%s %s
			)
			select exists (select * from a_page where id = $4)
		`, selectChatClause(true), getChatSearchWhereClause(additionalFoundUserIds), chat_order, orderDirection),
			participantId, searchStringWithPercents, searchString, chatId)
		// last line:
		// edge on the screen - here we ensure that this is the first page, in (1, 2) means the first place for the toppest element or the second place after sorting
		// checking ($6::bigint is null) is needed for the case no items on the screen so frontend has edgeChatId == null
		// casing to bigint needed because of https://github.com/jackc/pgx/issues/281
	} else {
		row = tx.QueryRowContext(ctx, fmt.Sprintf(`
			with a_page as (
							select * from (%s) ch
							where %s
							%s %s
			)
			select exists (select * from a_page where id = $2)
		`, selectChatClause(true), chat_where, chat_order, orderDirection),
			participantId, chatId)
	}
	if row.Err() != nil {
		tx.lgr.WithTracing(ctx).Errorf("Error during get Search %v", row.Err())
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

func getBlogAboutChatIdCommon(ctx context.Context, co CommonOperations) (*int64, *string, error) {
	row := co.QueryRowContext(ctx, `
							SELECT 
								ch.id,
								ch.title
							FROM chat ch 
							WHERE 
							    ch.blog IS TRUE AND
								ch.blog_about IS TRUE
							ORDER BY id LIMIT 1
						`,
	)
	var id *int64
	var title *string
	err := row.Scan(&id, &title)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, eris.Wrap(err, "error during interacting with db")
	} else {
		return id, title, nil
	}
}

func (db *DB) GetBlogAboutChatId(ctx context.Context) (*int64, *string, error) {
	return getBlogAboutChatIdCommon(ctx, db)
}

func (tx *Tx) GetBlogAboutChatId(ctx context.Context) (*int64, *string, error) {
	return getBlogAboutChatIdCommon(ctx, tx)
}

func (tx *Tx) SetBlogAbout(ctx context.Context, chatId int64, desiredValue bool) error {
	_, err := tx.ExecContext(ctx, "UPDATE chat SET blog_about = $2 WHERE id = $1", chatId, desiredValue)
	if err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
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
