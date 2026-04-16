package cqrs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/preview"
	"nkonev.name/chat/sanitizer"
	"nkonev.name/chat/utils"

	"github.com/qdm12/reprint"

	"github.com/georgysavva/scany/v2/sqlscan"
	"github.com/jackc/pgtype"
)

func (m *CommonProjection) GetChatIds(ctx context.Context, tx *db.Tx, size int32, offset int64) ([]int64, error) {
	ma := []int64{}

	err := sqlscan.Select(ctx, tx, &ma, `
		select c.id
		from chat_common c
		order by c.id asc 
		limit $1 offset $2
	`, size, offset)

	if err != nil {
		return ma, err
	}
	return ma, nil
}

func (m *CommonProjection) OnChatCreated(ctx context.Context, event *ChatCreated) error {
	// we don't check chat existence for the chat creation

	errOuter := db.Transact(ctx, m.db, func(tx *db.Tx) error {
		if event.TetATet {
			if event.TetATetOppositeUserId != nil {
				tetATetTwoExists, _, errInner := m.IsExistsTetATetTwo(ctx, tx, event.AdditionalData.BehalfUserId, *event.TetATetOppositeUserId)
				if errInner != nil {
					return errInner
				}

				if tetATetTwoExists {
					m.lgr.InfoContext(ctx,
						"Not created common chat because 2-participant tet-a-tet esists",
						logger.AttributeChatId, event.ChatId,
						"title", event.Title,
					)

					return nil
				}
			} else {
				tetATetOneExists, _, errInner := m.IsExistsTetATetOne(ctx, tx, event.AdditionalData.BehalfUserId)
				if errInner != nil {
					return errInner
				}

				if tetATetOneExists {
					m.lgr.InfoContext(ctx,
						"Not created common chat because 1-participant tet-a-tet esists",
						logger.AttributeChatId, event.ChatId,
						"title", event.Title,
					)

					return nil
				}
			}
		}

		_, errInner := tx.ExecContext(ctx, `
		insert into chat_common(
			 id
			,title
			,create_date_time
			,tet_a_tet
			,avatar
			,avatar_big
			,can_resend
			,can_react
			,available_to_search
			,regular_participant_can_publish_message
			,regular_participant_can_pin_message
			,regular_participant_can_write_message
			,regular_participant_can_add_participant
		) values (
			$1
			,$2
			,$3
			,$4
			,$5
		    ,$6
		    ,$7
		    ,$8
		    ,$9
		    ,$10
		    ,$11
		    ,$12
		    ,$13
		)
		on conflict(id) do update set 
		    title = excluded.title
		    ,tet_a_tet = excluded.tet_a_tet
		    ,avatar = excluded.avatar
		    ,avatar_big = excluded.avatar_big
			,can_resend = excluded.can_resend
			,can_react = excluded.can_react
			,available_to_search = excluded.available_to_search
			,regular_participant_can_publish_message = excluded.regular_participant_can_publish_message
			,regular_participant_can_pin_message = excluded.regular_participant_can_pin_message
			,regular_participant_can_write_message = excluded.regular_participant_can_write_message
			,regular_participant_can_add_participant = excluded.regular_participant_can_add_participant
	`, event.ChatId, event.Title, event.AdditionalData.CreatedAt, event.TetATet, event.Avatar, event.AvatarBig, event.CanResend, event.CanReact, event.AvailableToSearch, event.RegularParticipantCanPublishMessage, event.RegularParticipantCanPinMessage, event.RegularParticipantCanWriteMessage, event.RegularParticipantCanAddParticipant)
		if errInner != nil {
			return errInner
		}

		if event.Blog {
			// add blog
			_, errInner = m.refreshBlog(ctx, tx, event.ChatId, event.AdditionalData.CreatedAt, &event.BlogAbout)
			if errInner != nil {
				return errInner
			}
		}

		return nil
	})

	if errOuter != nil {
		return errOuter
	}

	m.lgr.InfoContext(ctx,
		"Common chat created",
		logger.AttributeChatId, event.ChatId,
		"title", event.Title,
	)

	return nil
}

func (m *CommonProjection) OnChatEdited(ctx context.Context, event *ChatEdited) (*int64, error) {
	var previousBlogAbout *int64
	errOuter := db.Transact(ctx, m.db, func(tx *db.Tx) error {
		chatExists, err := m.checkChatExists(ctx, tx, event.ChatId)
		if err != nil {
			return err
		}
		if !chatExists {
			m.lgr.InfoContext(ctx, "Skipping ChatEdited because there is no chat", logger.AttributeChatId, event.ChatId)
			return nil
		}

		blog, errInner := m.isChatBlog(ctx, tx, event.ChatId)
		if errInner != nil {
			return errInner
		}

		_, errInner = tx.ExecContext(ctx, `
			update chat_common
			set title = $2
			    ,avatar = $3
			    ,avatar_big = $4
				,can_resend = $5
				,can_react = $6
				,available_to_search = $7
				,regular_participant_can_publish_message = $8
				,regular_participant_can_pin_message = $9
				,regular_participant_can_write_message = $10
				,regular_participant_can_add_participant = $11
			where id = $1
		`, event.ChatId, event.Title, event.Avatar, event.AvatarBig, event.CanResend, event.CanReact, event.AvailableToSearch, event.RegularParticipantCanPublishMessage, event.RegularParticipantCanPinMessage, event.RegularParticipantCanWriteMessage, event.RegularParticipantCanAddParticipant)
		if errInner != nil {
			return errInner
		}
		m.lgr.InfoContext(ctx,
			"Common chat edited",
			logger.AttributeChatId, event.ChatId,
			"title", event.Title,
		)

		if blog && !event.Blog {
			// rm blog
			err = m.removeBlog(ctx, tx, event.ChatId)
			if errInner != nil {
				return errInner
			}
		} else if !blog && event.Blog {
			// add blog
			previousBlogAbout, errInner = m.refreshBlog(ctx, tx, event.ChatId, event.AdditionalData.CreatedAt, &event.BlogAbout)
			if errInner != nil {
				return errInner
			}
		} else if blog && event.Blog {
			// update blog
			previousBlogAbout, errInner = m.refreshBlog(ctx, tx, event.ChatId, event.AdditionalData.CreatedAt, &event.BlogAbout)
			if errInner != nil {
				return errInner
			}
		}

		return nil
	})

	if errOuter != nil {
		return nil, errOuter
	}

	return previousBlogAbout, nil
}

func (m *CommonProjection) OnChatRemoved(ctx context.Context, event *ChatDeleted) error {
	errOuter := db.Transact(ctx, m.db, func(tx *db.Tx) error {
		// we don't check IsChatAdmin because a participant was already removed

		blog, errInner := m.isChatBlog(ctx, tx, event.ChatId)
		if errInner != nil {
			return errInner
		}

		_, errInner = m.db.ExecContext(ctx, `
			delete from chat_common
			where id = $1
		`, event.ChatId)
		if errInner != nil {
			return errInner
		}

		_, errInner = m.db.ExecContext(ctx, `
			delete from message
			where chat_id = $1
		`, event.ChatId)
		if errInner != nil {
			return errInner
		}

		if blog {
			err := m.removeBlog(ctx, tx, event.ChatId)
			if err != nil {
				return err
			}
		}

		m.lgr.InfoContext(ctx,
			"Common chat removed",
			logger.AttributeChatId, event.ChatId,
		)
		return nil
	})

	if errOuter != nil {
		return errOuter
	}
	return nil
}

func (m *CommonProjection) OnChatPinned(ctx context.Context, event *UserChatPinned) error {
	errOuter := db.Transact(ctx, m.db, func(tx *db.Tx) error {
		participant, err := m.IsParticipant(ctx, tx, event.AdditionalData.BehalfUserId, event.ChatId)
		if err != nil {
			return err
		}
		if !participant {
			m.lgr.InfoContext(ctx, "Skipping ChatPinned because participant isn't participant", logger.AttributeUserId, event.AdditionalData.BehalfUserId, logger.AttributeChatId, event.ChatId)
			return nil
		}

		_, err = tx.ExecContext(ctx, `
		update chat_user_view
		set pinned = $3
		where (id, user_id) = ($1, $2)
	`, event.ChatId, event.AdditionalData.BehalfUserId, event.Pinned)
		if err != nil {
			return err
		}
		return nil
	})
	if errOuter != nil {
		return errOuter
	}

	m.lgr.InfoContext(ctx,
		"Chat pinned",
		logger.AttributeUserId, event.AdditionalData.BehalfUserId,
		logger.AttributeChatId, event.ChatId,
		"pinned", event.Pinned,
	)

	return nil
}

func (m *CommonProjection) OnChatNotificationSettingsSetted(ctx context.Context, event *UserChatNotificationSettingsSetted) error {

	errOuter := db.Transact(ctx, m.db, func(tx *db.Tx) error {
		participant, err := m.IsParticipant(ctx, tx, event.AdditionalData.BehalfUserId, event.ChatId)
		if err != nil {
			return err
		}
		if !participant {
			m.lgr.InfoContext(ctx, "Skipping ChatNotificationSettingsSetted because participant isn't participant", logger.AttributeUserId, event.AdditionalData.BehalfUserId, logger.AttributeChatId, event.ChatId)
			return nil
		}

		_, err = tx.ExecContext(ctx, `
		update chat_user_view 
		set consider_messages_as_unread = $3
		where id = $1 and user_id = $2 
	`, event.ChatId, event.AdditionalData.BehalfUserId, event.Setted)
		if err != nil {
			return err
		}

		m.lgr.InfoContext(ctx,
			"Chat notification settings setted",
			logger.AttributeUserId, event.AdditionalData.BehalfUserId,
			logger.AttributeChatId, event.ChatId,
			"setted", event.Setted,
		)

		err = m.updateHasUnreads(ctx, tx, event.AdditionalData.BehalfUserId)
		if err != nil {
			return err
		}

		return nil
	})

	return errOuter
}

// called in cases when chat should lift because of changing update_date_time
// in other cases (for example, read all the messages in the chat), when no need to update th timestamp - we should use another method
func (m *CommonProjection) OnChatViewRefreshedForPartitionUser(
	ctx context.Context,
	updatedAt time.Time,
	participantId int64, // current participant
	chatId int64,
) error {
	errOuter := db.Transact(ctx, m.db, func(tx *db.Tx) error {
		// in order not to have a potential race condition
		// for example "by upserting refresh view we can resurrect view of the newly removed participant in case message add"
		// we shouldn't upsert into chat_user_view
		// we can only update it here

		// for the cases like renaming chat, ...
		// the db was updated earlier, here we need to update chat_user_view.update_date_time

		// to eliminate unnecessary chat_user_view writes in participant changed
		_, err := tx.ExecContext(ctx, `
				update chat_user_view set update_date_time = $3 where user_id = $1 and id = $2
			`, participantId, chatId, updatedAt)
		if err != nil {
			return err
		}

		return nil
	})

	if errOuter != nil {
		return errOuter
	}
	return nil
}

func (m *CommonProjection) checkChatExists(ctx context.Context, co db.CommonOperations, chatId int64) (bool, error) {
	res, err := m.checkAreChatsExist(ctx, co, []int64{chatId})
	if err != nil {
		return false, err
	}

	return res[chatId], nil
}

func (m *CommonProjection) checkAreChatsExist(ctx context.Context, co db.CommonOperations, chatIds []int64) (map[int64]bool, error) {
	var existedChatIds []int64

	err := sqlscan.Select(ctx, co, &existedChatIds, "select id from chat_common where id = any($1)", chatIds)

	if err != nil {
		return nil, err
	}

	res := map[int64]bool{}
	for _, chatId := range chatIds {
		res[chatId] = false
	}

	for _, chatId := range existedChatIds {
		res[chatId] = true
	}

	return res, nil
}

func (m *EnrichingProjection) ChatFilter(ctx context.Context, co db.CommonOperations, behalfUserId, chatId int64, searchString string) (bool, error) {
	participant, err := m.cp.IsParticipant(ctx, co, behalfUserId, chatId)
	if err != nil {
		return false, err
	}
	if !participant {
		return false, NewUnauthorizedError(fmt.Sprintf("user %v is not a participant of chat %v", behalfUserId, chatId))
	}

	searchString = sanitizer.TrimAmdSanitize(m.policy, searchString)

	additionalFoundUserIds := m.searchForUsers(ctx, searchString)

	queryArgs := []any{chatId, behalfUserId}

	var searchClause = ""
	var searchCte = ""
	if len(searchString) > 0 {
		searchClause += " and ("

		searchClauseT, searchCteT, queryArgsT := processAdditionalUserIds(queryArgs, additionalFoundUserIds, searchString)
		searchClause += searchClauseT
		searchCte = searchCteT
		queryArgs = queryArgsT

		searchClause += " ) "
	}

	var found bool
	err = sqlscan.Get(ctx, co, &found, fmt.Sprintf(`
		%s
		SELECT EXISTS (
			select 1
			from chat_common cc
			join chat_user_view ch on (cc.id = ch.id and ch.user_id = $2)
			left join blog b on ch.id = b.id
			where ch.id = $1
			%s
		)
	`, searchCte, searchClause), queryArgs...)
	if err != nil {
		return false, err
	}

	return found, nil
}

func isSearchForPublic(searchString string) bool {
	return searchString == dto.ReservedPublicallyAvailableForSearchChats
}

func processAdditionalUserIds(queryArgsInput []any, additionalFoundUserIds []int64, searchString string) (searchClause string, searchCte string, queryArgs []any) {
	queryArgs = queryArgsInput
	var additionalUserIdsClause = ""
	searchForPublic := isSearchForPublic(searchString)
	if len(additionalFoundUserIds) > 0 {
		queryArgs = append(queryArgs, additionalFoundUserIds)
		searchCte = fmt.Sprintf(`
			with tet_a_tet_chats_ids as materialized (
				SELECT distinct (cp.chat_id) as chat_id
				FROM chat_common cc 
				join chat_participant cp
				on cc.id = cp.chat_id
				WHERE cc.tet_a_tet IS true AND cp.user_id = any($%d)
			)
			`, len(queryArgs))
		additionalUserIdsClause = fmt.Sprintf(" ( cc.id = any(array(SELECT chat_id FROM tet_a_tet_chats_ids)) ) or ")
	}
	searchClause = fmt.Sprintf(" ( ( %s cc.title ILIKE $%d ) OR ( (cc.available_to_search = TRUE OR b.id is not null) AND $%d = true ) )", additionalUserIdsClause, len(queryArgs)+1, len(queryArgs)+2)
	searchStringPercents := "%" + searchString + "%"
	queryArgs = append(queryArgs, searchStringPercents)
	queryArgs = append(queryArgs, searchForPublic)

	return
}

// contract: either multiple chats
// or one chatId != nil
func (m *EnrichingProjection) GetChatsEnriched(ctx context.Context, behalfParticipantIds []int64, size int32, startingFromItemId *dto.ChatId, includeStartingFrom, reverse bool, searchString string, chatId *int64, forceNonParticipant bool) ([]dto.ChatViewEnrichedDto, map[int64]*dto.User, error) {
	if len(behalfParticipantIds) == 0 {
		return nil, nil, errors.New("Wrong invariant: len(behalfParticipantIds) == 0")
	}
	multipleBehalfUserId := len(behalfParticipantIds) > 1
	if multipleBehalfUserId && chatId == nil {
		return nil, nil, errors.New("Wrong invariant: multipleBehalfUserId is true and null chatId")
	}

	searchString = sanitizer.TrimAmdSanitize(m.policy, searchString)

	additionalFoundUserIds := m.searchForUsers(ctx, searchString)

	type tupleDto struct {
		resultChats       []dto.ChatViewEnrichedDto
		intermediateUsers map[int64]*dto.User
	}

	d, errOuter := db.TransactWithResult(ctx, m.cp.db, func(tx *db.Tx) (*tupleDto, error) {
		chats, err := m.cp.GetChats(ctx, tx, behalfParticipantIds, size, startingFromItemId, includeStartingFrom, reverse, searchString, additionalFoundUserIds, chatId)
		if err != nil {
			m.lgr.ErrorContext(ctx, "Error getting chats", logger.AttributeError, err)
			return nil, err
		}

		participantIds, participantOfTetAtetId := getUserIdsFromChats(chats) // max num of users should fit aaa's limitation
		users, err := m.aaaRestClient.GetUsers(ctx, participantIds)
		if err != nil {
			m.lgr.WarnContext(ctx, "unable to get users")
		}

		usersMap := utils.ToMap(users)

		var areAdminsOfUserIds = map[int64]bool{}
		var areAdminsOfChatIds = map[int64]bool{}
		if multipleBehalfUserId {
			areAdminsOfUserIds, err = m.cp.getAreAdminsOfUserIds(ctx, tx, behalfParticipantIds, *chatId)
			if err != nil {
				return nil, err
			}
		} else {
			chatIds := getChatIdsFromChats(chats)

			areAdminsOfChatIds, err = m.cp.getAreAdminsOfChatIds(ctx, tx, behalfParticipantIds[0], chatIds)
			if err != nil {
				return nil, err
			}
		}

		tetATetOnlines, err := m.getParticipantsOnlineForTetATetMap(ctx, participantOfTetAtetId)
		if err != nil {
			m.lgr.WarnContext(ctx, "Something bad during getting tetATetOnlines", logger.AttributeError, err)
		}

		chatsEnriched := make([]dto.ChatViewEnrichedDto, 0, len(chats))
		for _, ch := range chats {
			var admin bool
			if multipleBehalfUserId {
				admin = areAdminsOfUserIds[ch.BehalfUserId]
			} else {
				admin = areAdminsOfChatIds[ch.Id]
			}

			che := m.enrichChat(ch.BehalfUserId, ch, usersMap, admin, tetATetOnlines, forceNonParticipant)
			chatsEnriched = append(chatsEnriched, che)
		}

		return &tupleDto{
			resultChats:       chatsEnriched,
			intermediateUsers: usersMap,
		}, nil
	})
	if errOuter != nil {
		return nil, nil, errOuter
	}
	return d.resultChats, d.intermediateUsers, nil
}

func (m *EnrichingProjection) getChatInfoForMessageNotification(ctx context.Context, co db.CommonOperations, chatId int64) (*dto.ChatInfoForNotification, error) {
	var chatBasic dto.ChatInfoForNotification

	err := sqlscan.Get(ctx, co, &chatBasic, `
		select 
		    c.title,
		    c.avatar
		from chat_common c
		where c.id = $1
	`, chatId)

	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &chatBasic, nil
}

func (m *EnrichingProjection) patchChatInfoForMessageNotification(ctx context.Context, inp *dto.ChatInfoForNotification, allPortionUsersMap map[int64]*dto.User, oppositeTetATetUserId *int64) *dto.ChatInfoForNotification {
	if oppositeTetATetUserId != nil {
		var copyInp *dto.ChatInfoForNotification
		err := reprint.FromTo(&inp, &copyInp)
		if err != nil {
			m.lgr.WarnContext(ctx, "Unable to copy", logger.AttributeError, err)
			return inp
		} else {
			us, ok := allPortionUsersMap[*oppositeTetATetUserId]
			if !ok {
				m.lgr.InfoContext(ctx, "Opposite user isn't found in the map", logger.AttributeUserId, *oppositeTetATetUserId)
			} else {
				copyInp.ChatName = us.Login
				copyInp.ChatAvatar = us.Avatar
			}

			return copyInp
		}
	} else {
		return inp
	}
}

func (m *EnrichingProjection) getTetATetOpposites(ctx context.Context, co db.CommonOperations, chatId int64, behalfUserIds []int64) (map[int64]*int64, error) {
	var res = []struct {
		RequestedParticipantId int64  `db:"requested_participant_id"`
		OppositeParticipantId  *int64 `db:"opposite_participant_id"`
	}{}

	err := sqlscan.Select(ctx, co, &res, `
		with requested_participants as (
			select * from unnest(cast ($1 as bigint[])) as t(user_id)
		)
		select 
			rp.user_id as requested_participant_id,
		    cp.user_id as opposite_participant_id
		from requested_participants rp 
		left join chat_participant cp on (cp.user_id != rp.user_id and cp.chat_id = $2)
		join chat_common cc on (cc.id = cp.chat_id and cc.tet_a_tet = true and cc.id = $2)
	`, behalfUserIds, chatId)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	ret := map[int64]*int64{}
	for _, r := range res {
		ret[r.RequestedParticipantId] = r.OppositeParticipantId
	}

	return ret, nil
}

func (m *EnrichingProjection) getParticipantsOnlineForTetATetMap(ctx context.Context, userIds []int64) (map[int64]bool, error) {
	ret := map[int64]bool{}

	if len(userIds) == 0 {
		return ret, nil
	}

	onlines, err := m.aaaRestClient.GetOnlines(ctx, userIds) // get online for opposite user
	if err != nil {
		m.lgr.WarnContext(ctx, "Unable to get online for", "user_ids", userIds, logger.AttributeError, err)
		// nothing
		return ret, nil
	}

	for _, onl := range onlines {
		ret[onl.Id] = onl.Online
	}
	return ret, err
}

func (m *EnrichingProjection) GetChat(ctx context.Context, userId, chatId int64) (res *dto.ChatViewEnrichedDto, shouldJoin bool, err error) {
	size := int32(1)
	reverse := false

	var startingFromItemId *dto.ChatId = nil
	includeStartingFrom := true
	searchString := ""

	chats, _, errG := m.GetChatsEnriched(ctx, []int64{userId}, size, startingFromItemId, includeStartingFrom, reverse, searchString, &chatId, false)
	if errG != nil {
		m.lgr.ErrorContext(ctx, "Error getting chats", logger.AttributeError, errG)
		err = errG
		return
	}

	if len(chats) == 0 {
		basic, errB := m.cp.GetChatBasic(ctx, m.cp.db, chatId)
		if errB != nil {
			m.lgr.ErrorContext(ctx, "Error getting basic chat", logger.AttributeError, errB)
			err = errB
			return
		}
		if basic != nil && (basic.AvailableToSearch || basic.IsBlog) {
			shouldJoin = true
			return
		} else {
			res = nil
			return
		}
	} else if len(chats) > 1 {
		err = errors.New("Wrong invariant: More than 1 chats got")
		return
	}

	chat := chats[0]
	res = &chat
	return
}

func (m *CommonProjection) GetBasicInfo(ctx context.Context, chatId int64) (*dto.BasicChatDto, error) {
	ret, errOuter := db.TransactWithResult(ctx, m.db, func(tx *db.Tx) (*dto.BasicChatDto, error) {
		chatBasic, err := m.GetChatBasic(ctx, tx, chatId)
		if err != nil {
			return &dto.BasicChatDto{}, err
		}

		participantIds, err := m.GetParticipantIds(ctx, tx, chatId, utils.FixSize(0), utils.FixPage(0))
		if err != nil {
			return &dto.BasicChatDto{}, err
		}

		ret := dto.BasicChatDto{
			TetATet:        chatBasic.TetATet,
			ParticipantIds: participantIds,
		}
		return &ret, nil
	})
	if errOuter != nil {
		return nil, errOuter
	}
	return ret, nil
}

func (m *EnrichingProjection) GetNameForInvite(ctx context.Context, chatId, behalfUserId int64, participantIds []int64) ([]dto.ChatName, error) {
	ret := []dto.ChatName{}

	type txDto struct {
		chatBasic           *dto.ChatBasic
		tetATetOppositeUser *int64
	}

	tr, errOuter := db.TransactWithResult(ctx, m.cp.db, func(tx *db.Tx) (*txDto, error) {
		cb, err := m.cp.GetChatBasic(ctx, tx, chatId)
		if err != nil {
			return nil, err
		}
		if cb == nil {
			return nil, nil
		}

		var tetATetOppositeUser *int64
		if cb.TetATet {
			participantIds, err := m.cp.GetParticipantIds(ctx, tx, chatId, utils.FixSize(0), utils.FixPage(0))
			if err != nil {
				return nil, err
			}
			tetATetOppositeUser = tetATetOpposite(participantIds, behalfUserId)
		}

		return &txDto{
			chatBasic:           cb,
			tetATetOppositeUser: tetATetOppositeUser,
		}, nil
	})
	if errOuter != nil {
		return nil, errOuter
	}

	if tr == nil {
		return ret, nil
	}

	var userMap = map[int64]*dto.User{}
	var participantIdsToQuery = []int64{behalfUserId}
	if tr.tetATetOppositeUser != nil {
		participantIdsToQuery = append(participantIdsToQuery, *tr.tetATetOppositeUser)
	}

	users, err := m.aaaRestClient.GetUsers(ctx, participantIdsToQuery)
	if err != nil {
		return nil, err
	}
	userMap = utils.ToMap(users)

	for _, userId := range participantIds {
		cn := dto.ChatName{
			Name:   tr.chatBasic.Title,
			Avatar: tr.chatBasic.Avatar,
			UserId: userId,
		}

		if tr.chatBasic.TetATet {
			behalfOppUser := userMap[behalfUserId]
			if behalfOppUser == nil {
				m.lgr.WarnContext(ctx, "Skipping an behalfOppUser because it doesn't present in aaa response", logger.AttributeChatId, chatId, logger.AttributeUserId, behalfUserId)
				continue
			}

			if userId != behalfUserId {
				cn.Name = behalfOppUser.Login
				cn.Avatar = behalfOppUser.Avatar
			} else {
				itselfUser := userMap[userId]
				if itselfUser == nil {
					m.lgr.WarnContext(ctx, "Skipping an itselfUser because it doesn't present in aaa response", logger.AttributeChatId, chatId, logger.AttributeUserId, userId)
					continue
				}
				cn.Name = itselfUser.Login
				cn.Avatar = itselfUser.Avatar
			}
		}

		ret = append(ret, cn)
	}

	return ret, nil
}

func (m *EnrichingProjection) searchForUsers(ctx context.Context, searchString string) []int64 {
	var additionalFoundUserIds = []int64{}

	if searchString != "" && searchString != dto.ReservedPublicallyAvailableForSearchChats {
		users, _, err := m.aaaRestClient.SearchGetUsers(ctx, searchString, true, []int64{}, 0, 0)
		if err != nil {
			m.lgr.ErrorContext(ctx, "Error get users from aaa", logger.AttributeError, err)
		}
		for _, u := range users {
			additionalFoundUserIds = append(additionalFoundUserIds, u.Id)
		}
	}
	return additionalFoundUserIds
}

func getUserIdsFromChats(chats []dto.ChatViewDto) ([]int64, []int64) {
	m := map[int64]struct{}{}
	mt := map[int64]struct{}{}

	for _, ch := range chats {
		for _, p := range ch.ParticipantIds {
			m[p] = struct{}{}

			if ch.TetATet {
				mt[p] = struct{}{}
			}
		}

		if ch.LastMessageOwnerId != nil {
			m[*ch.LastMessageOwnerId] = struct{}{}
		}
	}

	r := []int64{}
	rt := []int64{}

	for k, _ := range m {
		r = append(r, k)
	}

	for k, _ := range mt {
		rt = append(rt, k)
	}

	return r, rt
}

func getChatIdsFromChats(chats []dto.ChatViewDto) []int64 {
	m := map[int64]struct{}{}

	for _, ch := range chats {
		m[ch.Id] = struct{}{}
	}

	r := []int64{}

	for k, _ := range m {
		r = append(r, k)
	}
	return r
}

func tetATetOpposite(participantIds []int64, behalfUserId int64) *int64 {
	oppa := utils.GetSliceWithout(behalfUserId, participantIds)
	if len(oppa) == 1 {
		oppositeUserId := oppa[0]
		return &oppositeUserId
	}
	return nil
}

func (m *EnrichingProjection) enrichChat(behalfUserId int64, ch dto.ChatViewDto, users map[int64]*dto.User, admin bool, tetATetOnlines map[int64]bool, forceNonParticipant bool) dto.ChatViewEnrichedDto {
	che := dto.ChatViewEnrichedDto{
		ChatViewDto:  ch,
		Participants: makeParticipants(ch.ParticipantIds, users),
	}
	if che.ChatViewDto.TetATet {
		var displayableUser *dto.User
		if che.ChatViewDto.ParticipantsCount == 1 {
			oppositeUserId := che.ChatViewDto.ParticipantIds[0]
			displayableUser = users[oppositeUserId]
		} else {
			tetATetOpposite := tetATetOpposite(che.ParticipantIds, behalfUserId)
			if tetATetOpposite != nil {
				oppositeUserId := *tetATetOpposite
				displayableUser = users[oppositeUserId]
			}
		}

		if displayableUser != nil {
			che.Title = displayableUser.Login
			che.Avatar = displayableUser.Avatar

			che.ShortInfo = displayableUser.ShortInfo
			che.LoginColor = displayableUser.LoginColor
			che.AdditionalData = displayableUser.AdditionalData

			if displayableUser.Id != behalfUserId {
				che.LastSeenDateTime = displayableUser.LastSeenDateTime

				onl, ok := tetATetOnlines[displayableUser.Id]
				if ok {
					if onl { // if the opposite user is online we don't need to show last login
						che.LastSeenDateTime = nil
					}
				}
			}
		}
	}

	var isParticipant = ch.IsParticipant
	if forceNonParticipant {
		isParticipant = false

		che.UnreadMessages = 0
	}

	SetChatPersonalizedFields(&che, behalfUserId, admin, isParticipant)

	if ch.LastMessageOwnerId != nil && ch.LastMessageContent != nil {
		u := users[*ch.LastMessageOwnerId]
		if u != nil {
			previewStr := preview.CreateMessagePreview(m.stripAllTags, m.cfg.Message.PreviewMaxTextSize, *ch.LastMessageContent, u.Login)
			che.LastMessagePreview = &previewStr
		}
	}

	return che
}

func SetChatPersonalizedFields(copied *dto.ChatViewEnrichedDto, behalfUserId int64, admin bool, participant bool) {
	canEdit := CanEditChat(admin, copied.TetATet)
	copied.CanEdit = &canEdit
	canDelete := CanDeleteChat(admin)
	copied.CanDelete = &canDelete
	canLeave := CanLeaveChat(admin, copied.TetATet, participant)
	copied.CanLeave = &canLeave
	copied.CanVideoKick = admin
	copied.CanAudioMute = admin
	copied.CanChangeChatAdmins = CanChangeParticipant(behalfUserId, admin, copied.TetATet, dto.NonExistentUser)
	copied.CanBroadcast = CanBroadcast(admin)

	// yes, mutate the fields
	copied.CanReact = CanReactOnMessage(copied.CanReact, participant)
	copied.CanResend = CanResendMessage(copied.CanResend, participant)

	// participant can be false in case result from search for publicly available chats
	copied.IsResultFromSearch = !participant

	copied.CanWriteMessage = CanWriteMessage(participant, admin, copied.RegularParticipantCanWriteMessage)
	copied.CanAddParticipant = CanAddParticipant(admin, copied.TetATet, false, copied.AvailableToSearch, copied.Blog, false, participant, copied.RegularParticipantCanAddParticipant)
}

// We use pure functions for authorization, for sake simplicity and composability
func CanEditChat(isAdmin, tetATet bool) bool {
	return isChatAdminInternal(isAdmin) && !tetATet
}

func CanDeleteChat(isAdmin bool) bool {
	return isAdmin
}

func CanLeaveChat(isAdmin, tetATet, isParticipant bool) bool {
	return !isAdmin && !tetATet && isParticipant
}

func CanBroadcast(isAdmin bool) bool {
	return isAdmin
}

func CanReactOnMessage(chatCanReact bool, isParticipant bool) bool {
	return chatCanReact && isParticipant
}

func CanResendMessage(chatCanResend bool, isParticipant bool) bool {
	return chatCanResend && isParticipant
}

func (m *CommonProjection) IterateOverAllChats(ctx context.Context, co db.CommonOperations, consumer func(chatIdsPortion []int64) error) error {
	shouldContinue := true
	var lastError error
	for page := int64(0); shouldContinue; page++ {
		offset := utils.GetOffset(page, utils.DefaultSize)

		list := []int64{}

		sqlArgs := []any{utils.DefaultSize, offset}
		sqlQuery := `
			SELECT id FROM chat_common ORDER BY id LIMIT $1 OFFSET $2
		`
		err := sqlscan.Select(ctx, co, &list, sqlQuery, sqlArgs...)
		if err != nil {
			m.lgr.ErrorContext(ctx, "Got error during getting portion", logger.AttributeError, err)
			lastError = err
			break
		}
		if len(list) == 0 {
			return nil
		}
		if len(list) < utils.DefaultSize {
			shouldContinue = false
		}

		err = consumer(list)
		if err != nil {
			m.lgr.ErrorContext(ctx, "Got error during invoking consumer portion", logger.AttributeError, err)
			lastError = err
			break
		}
	}
	return lastError
}

func (m *CommonProjection) GetChatDataForAuthorization(ctx context.Context, co db.CommonOperations, userId, chatId int64) (dto.ChatAuthorizationData, error) {
	d := dto.ChatAuthorizationData{}
	err := sqlscan.Get(ctx, co, &d, `
		with
		provided as (
			select 
				 cast($2 as bigint) as chat_id
		),
		chat_participant_row as (
			SELECT user_id, chat_admin FROM chat_participant WHERE user_id = $1 AND chat_id = $2 LIMIT 1
		),
		chat_info as (
			select * from chat_common where id = $2
		)
		SELECT 
			cc.id is not null as is_chat_found
			,(SELECT exists(SELECT * FROM chat_participant_row) as is_chat_participant)
			,(SELECT exists(SELECT * FROM chat_participant_row WHERE chat_admin) as is_chat_admin)
			,coalesce(cc.regular_participant_can_write_message, false) as chat_can_write_message
			,coalesce(cc.tet_a_tet, false) as chat_is_tet_a_tet
			,coalesce(cc.can_resend, false) as chat_can_resend_message
			,coalesce(cc.can_react, false) as chat_can_react_on_message
			,coalesce(cc.available_to_search, false) as chat_is_available_to_search
			,coalesce(cc.regular_participant_can_add_participant, false) as regular_participant_can_add_participant
			,b.id is not null as chat_is_blog
		FROM provided pr
		LEFT JOIN chat_info cc on pr.chat_id = cc.id
		left join blog b on cc.id = b.id
	`, userId, chatId)
	if err != nil {
		return d, err
	}
	return d, nil
}

func (m *CommonProjection) GetChats(ctx context.Context, co db.CommonOperations, participantIds []int64, size int32, startingFromItemId *dto.ChatId, includeStartingFrom, reverse bool, searchString string, additionalFoundUserIds []int64, chatId *int64) ([]dto.ChatViewDto, error) {
	type chatDto struct {
		Id                                  int64            `db:"id"`
		UserId                              int64            `db:"user_id"`
		Title                               string           `db:"title"`
		Pinned                              bool             `db:"pinned"`
		UnreadMessages                      int64            `db:"unread_messages"`
		LastMessageId                       *int64           `db:"last_message_id"`
		LastMessageOwnerId                  *int64           `db:"last_message_owner_id"`
		LastMessageContent                  *string          `db:"last_message_content"`
		ParticipantsCount                   int64            `db:"participants_count"`
		ParticipantIds                      pgtype.Int8Array `db:"last_n_participant_ids"` // ids of last N participants
		Blog                                bool             `db:"blog"`
		BlogAbout                           bool             `db:"blog_about"`
		UpdateDateTime                      *time.Time       `db:"update_date_time"`
		TetATet                             bool             `db:"tet_a_tet"`
		Avatar                              *string          `db:"avatar"`
		AvatarBig                           *string          `db:"avatar_big"`
		ConsiderMessagesAsUnread            bool             `db:"consider_messages_as_unread"`
		CanResend                           bool             `db:"can_resend"`
		CanReact                            bool             `db:"can_react"`
		RegularParticipantCanPublishMessage bool             `db:"regular_participant_can_publish_message"`
		RegularParticipantCanPinMessage     bool             `db:"regular_participant_can_pin_message"`
		RegularParticipantCanWriteMessage   bool             `db:"regular_participant_can_write_message"`
		AvailableToSearch                   bool             `db:"available_to_search"`
		IsParticipant                       bool             `db:"is_participant"`
		RegularParticipantCanAddParticipant bool             `db:"regular_participant_can_add_participant"`
	}

	if size == dto.NoSize {
		return nil, fmt.Errorf("wrong invariant: NoSize is not implemented")
	}

	list := []chatDto{}
	res := []dto.ChatViewDto{}

	var searchForPublic bool = isSearchForPublic(searchString)

	queryArgs := []any{size, participantIds, dto.NonExistentUser}

	order := "desc"
	offset := " offset 1" // to make behaviour the same as in users, messages (there is > or <)
	if reverse {
		order = "asc"
	}

	const personalOrder = "ch.pinned, ch.update_date_time, ch.id"
	const publicOrder = "cc.create_date_time, cc.id"

	var orderClause string
	if !searchForPublic {
		orderClause = fmt.Sprintf("order by (%s) %s", personalOrder, order)
	} else {
		orderClause = fmt.Sprintf("order by (%s) %s", publicOrder, order)
	}
	// see also getSafeDefaultUserId() in aaa
	if includeStartingFrom || startingFromItemId == nil {
		offset = ""
	}

	nonEquality := "<="
	if reverse {
		nonEquality = ">="
	}

	conditionClause := " true "

	var joinClause string

	if startingFromItemId != nil && chatId != nil {
		return nil, fmt.Errorf("wrong invariant: both startingFromItemId and chatId provided")
	}

	if len(searchString) > 0 && chatId != nil {
		return nil, fmt.Errorf("wrong invariant: both searchString and chatId provided")
	}

	if startingFromItemId != nil {
		var paginationKeyset string
		if !searchForPublic {
			paginationKeyset = fmt.Sprintf(` and (%s) %s ($%d, $%d, $%d)`, personalOrder, nonEquality, len(queryArgs)+1, len(queryArgs)+2, len(queryArgs)+3)
			queryArgs = append(queryArgs, startingFromItemId.Pinned, startingFromItemId.LastUpdateDateTime, startingFromItemId.Id)
		} else {
			paginationKeyset = fmt.Sprintf(` and (%s) %s ($%d, $%d)`, publicOrder, nonEquality, len(queryArgs)+1, len(queryArgs)+2)
			queryArgs = append(queryArgs, startingFromItemId.LastUpdateDateTime, startingFromItemId.Id)
		}

		conditionClause += paginationKeyset
	}

	var searchClause = ""
	var searchCte = ""
	if len(searchString) > 0 {
		searchClause = " and ("

		searchClauseT, searchCteT, queryArgsT := processAdditionalUserIds(queryArgs, additionalFoundUserIds, searchString)
		searchClause += searchClauseT
		searchCte = searchCteT
		queryArgs = queryArgsT
		searchClause += " or "

		queryArgs = append(queryArgs, searchString)
		searchClause += fmt.Sprintf(`
		exists( 
			select 1 from (select * from (select unnest(tsvector_to_array(cc.fts_title))) t(av)) inq 
			where
				   ( inq.av %% plainto_tsquery('russian', $%d)::text )
			    or ( cyrillic_transliterate(inq.av) %% cyrillic_transliterate(plainto_tsquery('russian', $%d)::text) ) 
		) `, len(queryArgs), len(queryArgs))

		searchClause += " ) "
	}

	if chatId != nil {
		chatIdV := *chatId
		queryArgs = append(queryArgs, chatIdV)
		chatIdClause := fmt.Sprintf(" and ch.id = $%d", len(queryArgs))

		conditionClause += chatIdClause
		orderClause = "order by ch.update_date_time desc, ch.user_id" // to prevent flaky tests. the same as in projection_participantv :: getParticipantsCommonExcepting()
	}

	if !searchForPublic {
		conditionClause += " and ch.user_id = any($2) "
		joinClause = " join "
	} else {
		joinClause = " left join "
	}

	var dateTimeClause string
	if !searchForPublic {
		dateTimeClause = "ch.update_date_time"
	} else {
		dateTimeClause = "cc.create_date_time"
	}

	// it is optimized (all order by in the same table)
	// so querying a page (using keyset) from a large amount of chats is fast
	// it's the root cause why we use cqrs
	q := fmt.Sprintf(`
		%s
		select 
		    cc.id,
			coalesce(ch.user_id, $3) as user_id,
		    cc.title,
		    coalesce(ch.pinned, false) as pinned,
		    coalesce(ch.unread_messages, 0) as unread_messages,
		    cc.last_message_id,
		    cc.last_message_owner_id,
		    cc.last_message_content,
		    cc.participants_count,
		    cc.last_n_participant_ids,
		    b.id is not null as blog,
		    coalesce(b.blog_about, false) as blog_about,
		    %s as update_date_time,
		    cc.tet_a_tet,
			cc.avatar,
			cc.avatar_big,
			coalesce(ch.consider_messages_as_unread, true) as consider_messages_as_unread,
			cc.can_resend,
			cc.can_react,
			cc.regular_participant_can_publish_message,
			cc.regular_participant_can_pin_message,
			cc.regular_participant_can_write_message,
			cc.available_to_search,
			ch.id is not null as is_participant,
			cc.regular_participant_can_add_participant
		from chat_common cc
		%s chat_user_view ch on (cc.id = ch.id and ch.user_id = any($2))
		left join blog b on cc.id = b.id
		where %s
		%s
		%s
		limit $1
		%s
		`, searchCte, dateTimeClause, joinClause, conditionClause, searchClause, orderClause, offset)
	err := sqlscan.Select(ctx, co, &list, q, queryArgs...)
	if err != nil {
		return res, err
	}

	for i, de := range list {
		mapped := dto.ChatViewDto{
			Id:                                  de.Id,
			BehalfUserId:                        de.UserId,
			Title:                               de.Title,
			Pinned:                              de.Pinned,
			UnreadMessages:                      de.UnreadMessages,
			LastMessageId:                       de.LastMessageId,
			LastMessageOwnerId:                  de.LastMessageOwnerId,
			LastMessageContent:                  de.LastMessageContent,
			ParticipantsCount:                   de.ParticipantsCount,
			Blog:                                de.Blog,
			BlogAbout:                           de.BlogAbout,
			UpdateDateTime:                      de.UpdateDateTime,
			TetATet:                             de.TetATet,
			Avatar:                              de.Avatar,
			AvatarBig:                           de.AvatarBig,
			ConsiderMessagesAsUnread:            de.ConsiderMessagesAsUnread,
			CanResend:                           de.CanResend,
			CanReact:                            de.CanReact,
			RegularParticipantCanPublishMessage: de.RegularParticipantCanPublishMessage,
			RegularParticipantCanPinMessage:     de.RegularParticipantCanPinMessage,
			RegularParticipantCanWriteMessage:   de.RegularParticipantCanWriteMessage,
			AvailableToSearch:                   de.AvailableToSearch,
			IsParticipant:                       de.IsParticipant,
			CanPin:                              de.IsParticipant,
			RegularParticipantCanAddParticipant: de.RegularParticipantCanAddParticipant,
		}
		err = de.ParticipantIds.AssignTo(&mapped.ParticipantIds)
		if err != nil {
			return res, fmt.Errorf("error during mapping on index %d: %w", i, err)
		}

		res = append(res, mapped)
	}

	return res, nil
}

func (m *CommonProjection) GetHasUnreadMessages(ctx context.Context, userIds []int64) (map[int64]bool, error) {
	var has = map[int64]bool{}

	type hasDto struct {
		UserId int64 `db:"user_id"`
		Has    bool  `db:"has"`
	}
	list := []hasDto{}
	err := sqlscan.Select(ctx, m.db, &list, `
	with
	normalized_user as (
		select unnest(cast ($1 as bigint[])) as user_id
	)
	select 
		nu.user_id,
		coalesce(h.has, false) as has
	from has_unread_messages h
	right join normalized_user nu on h.user_id = nu.user_id
	where h.user_id = any($1)
	`, userIds)
	if err != nil {
		return nil, err
	}
	for _, hd := range list {
		has[hd.UserId] = hd.Has
	}
	return has, nil
}

func (m *CommonProjection) IsChatUserViewExists(ctx context.Context, co db.CommonOperations, chatId, userId int64) (bool, error) {
	var t bool
	err := sqlscan.Get(ctx, co, &t, "select exists(select c.* from chat_user_view ch join chat_common c on ch.id = c.id where ch.user_id = $1 and ch.id = $2)", userId, chatId)
	if err != nil {
		return false, err
	}
	return t, nil
}

func (m *CommonProjection) IsChatExists(ctx context.Context, co db.CommonOperations, chatId int64) (bool, error) {
	var t bool
	err := sqlscan.Get(ctx, co, &t, "select exists(select c.* from chat_common c where c.id = $1)", chatId)
	if err != nil {
		return false, err
	}
	return t, nil
}

func (m *CommonProjection) GetChatUserViewBasic(ctx context.Context, co db.CommonOperations, chatId, participantId int64) (dto.ChatUserViewBasic, error) {
	var t dto.ChatUserViewBasic
	err := sqlscan.Get(ctx, co, &t, "select ch.id, ch.update_date_time, ch.unread_messages from chat_user_view ch where ch.user_id = $1 and ch.id = $2", participantId, chatId)
	if err != nil {
		return t, err
	}
	return t, nil

}

func (m *CommonProjection) GetChatBasic(ctx context.Context, co db.CommonOperations, chatId int64) (*dto.ChatBasic, error) {
	var cht dto.ChatBasic

	err := sqlscan.Get(ctx, co, &cht, `
		select 
		    c.id,
		    c.title,
			coalesce(c.avatar_big, c.avatar) as avatar,
		    c.can_resend,
		    c.tet_a_tet,
			b.id is not null as blog,
			c.available_to_search,
			c.regular_participant_can_publish_message,
			c.regular_participant_can_pin_message,
			c.regular_participant_can_write_message
		from chat_common c
		left join blog b on c.id = b.id
		where c.id = $1
	`, chatId)

	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &cht, nil
}

func getDeletedChatName(chatId int64) string {
	return fmt.Sprintf("deleted_chat_%d", chatId)
}

// result: map[userId][chatId]*dto.BasicChatDtoExtended
func (m *CommonProjection) GetChatsBasicExtended(ctx context.Context, co db.CommonOperations, chatIds []int64, behalfParticipantIds []int64) (map[int64]map[int64]*dto.BasicChatDtoExtended, error) {
	result := map[int64]map[int64]*dto.BasicChatDtoExtended{}
	if len(chatIds) == 0 {
		return result, nil
	}

	list := []dto.BasicChatDtoExtended{}
	err := sqlscan.Select(ctx, co, &list, `
		with requested_participants as (
			select * from unnest(cast ($1 as bigint[])) as t(user_id)
		),
		chats_participants as (
			select 
				user_id,
				chat_id 
			from chat_participant cp 
			where cp.chat_id = any($2) AND cp.user_id = any($1)
		)
		SELECT 
			re.user_id,
			c.id,
			c.title,
			coalesce(c.avatar_big, c.avatar) as avatar,
			(cp.user_id is not null) as behalf_user_is_participant,
			c.tet_a_tet,
			c.can_resend,
			b.id is not null as blog,
			c.available_to_search,
			c.regular_participant_can_publish_message,
			c.regular_participant_can_pin_message,
			c.regular_participant_can_write_message
		FROM chat_common c
		CROSS JOIN requested_participants re
		LEFT JOIN chats_participants cp ON (c.id = cp.chat_id and re.user_id = cp.user_id)
		left join blog b on c.id = b.id
		WHERE c.id = any($2)
	`,
		behalfParticipantIds, chatIds)
	if err != nil {
		return nil, err
	}
	for _, bc := range list {
		innerMap, ok := result[bc.BehalfUserId]
		if !ok {
			innerMap = map[int64]*dto.BasicChatDtoExtended{}
			result[bc.BehalfUserId] = innerMap
		}
		innerMap[bc.Id] = &bc
	}
	return result, nil
}

func (m *CommonProjection) GetChatNotificationSettings(ctx context.Context, behalfParticipantId int64, chatId int64) (*dto.UserChatNotificationSettings, error) {
	value := dto.UserChatNotificationSettings{}
	err := sqlscan.Get(ctx, m.db, &value, "select ch.consider_messages_as_unread from chat_user_view ch where ch.user_id = $1 and ch.id = $2", behalfParticipantId, chatId)

	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &value, err
}

func (m *CommonProjection) GetExistingChatIds(ctx context.Context, co db.CommonOperations, chatIds []int64) ([]int64, error) {
	list := []int64{}
	err := sqlscan.Select(ctx, co, &list, `
	select id from chat_common
	where id = any($1)
	`, chatIds)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (m *CommonProjection) getChatNameForNotification(ctx context.Context, co db.CommonOperations, chatId int64) (string, error) {
	chatBasic, err := m.GetChatBasic(ctx, co, chatId)
	if err != nil {
		return "", err
	}
	var chatName string
	if chatBasic != nil {
		chatName = chatBasic.Title
		if chatBasic.TetATet {
			chatName = ""
		}
	} else {
		chatName = getDeletedChatName(chatId)
	}
	return chatName, nil
}
