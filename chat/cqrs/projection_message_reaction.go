package cqrs

import (
	"context"
	"fmt"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"

	"github.com/georgysavva/scany/v2/sqlscan"
	"github.com/jackc/pgtype"
)

func (m *CommonProjection) OnMessageReactionCreated(ctx context.Context, additionalData *AdditionalData, metadata *Metadata, chatId int64, messageId int64, reactionStr string) (bool, error) {
	wasAdded, errOuter := db.TransactWithResult(ctx, m.db, func(tx *db.Tx) (bool, error) {
		var wasAddedInner bool

		messageExists, errInner := m.checkMessageExists(ctx, tx, chatId, messageId)
		if errInner != nil {
			return false, errInner
		}

		if !messageExists {
			m.lgr.InfoContext(ctx, "Skipping MessageReactionCreated because there is no message", logger.AttributeChatId, chatId, logger.AttributeMessageId, messageId)
			return false, nil
		}

		_, errInner = tx.ExecContext(ctx, `
				insert into message_reaction(chat_id, message_id, user_id, reaction, create_date_time)
				values ($1, $2, $3, $4, $5)
				on conflict (chat_id, message_id, user_id, reaction) do nothing
			`, chatId, messageId, additionalData.BehalfUserId, reactionStr, additionalData.CreatedAt)
		if errInner != nil {
			return false, errInner
		}
		wasAddedInner = true

		return wasAddedInner, nil
	})
	return wasAdded, errOuter
}

func (m *CommonProjection) OnMessageReactionDeleted(ctx context.Context, additionalData *AdditionalData, metadata *Metadata, chatId int64, messageId int64, reactionStr string) (bool, error) {
	wasAdded, errOuter := db.TransactWithResult(ctx, m.db, func(tx *db.Tx) (bool, error) {
		var wasAddedInner bool

		_, errInner := tx.ExecContext(ctx, "DELETE FROM message_reaction WHERE chat_id = $1 AND message_id = $2 AND user_id = $3 AND reaction = $4", chatId, messageId, additionalData.BehalfUserId, reactionStr)
		if errInner != nil {
			return false, errInner
		}

		return wasAddedInner, nil
	})
	return wasAdded, errOuter
}

func getReactionsCommon(ctx context.Context, co db.CommonOperations, chatId int64, messageIds []int64, reaction *string, maxDisplayableUsers int) ([]dto.ReactionDto, error) {
	type reactionDto struct {
		MessageId int64            `db:"message_id"`
		UserIds   pgtype.Int8Array `db:"user_ids"`
		Reaction  string           `db:"reaction"`
		Count     int64            `db:"count"`
	}

	reactions := []reactionDto{}
	res := []dto.ReactionDto{}

	sqlArgs := []any{chatId, messageIds, maxDisplayableUsers}

	var additionalCondition string
	if reaction != nil {
		sqlArgs = append(sqlArgs, *reaction)
		additionalCondition = fmt.Sprintf("and reaction = $%d", len(sqlArgs))
	}

	q := fmt.Sprintf(`
		with
		requested_message_reactions as (
			select * from message_reaction where chat_id = $1 and message_id = any($2) %s
		),
		reaction_counts as (
			select message_id, reaction, count(user_id) as count
			from requested_message_reactions group by message_id, reaction
		),
		reaction_users_last_n as (
			select
				message_id,
				reaction,
				(array_agg(user_id order by create_date_time))[:$3] as user_ids,
				min(create_date_time) as create_date_time
			from message_reaction group by message_id, reaction
		)
		select
			rc.message_id,
			rc.reaction,
			rn.user_ids,
			rc.count
		from reaction_counts rc
		join reaction_users_last_n rn on (rc.message_id, rc.reaction) = (rn.message_id, rn.reaction)
		order by rn.create_date_time
		`, additionalCondition)

	err := sqlscan.Select(ctx, co, &reactions, q, sqlArgs...)
	if err != nil {
		return res, fmt.Errorf("error during interacting with db: %w", err)
	}

	for i, de := range reactions {
		mapped := dto.ReactionDto{
			MessageId: de.MessageId,
			Reaction:  de.Reaction,
			Count:     de.Count,
		}
		err = de.UserIds.AssignTo(&mapped.UserIds)
		if err != nil {
			return res, fmt.Errorf("error during mapping on index %d: %w", i, err)
		}
		res = append(res, mapped)
	}

	return res, nil
}

func (m *EnrichingProjection) getReactions(ctx context.Context, co db.CommonOperations, chatId int64, messageIds []int64) (map[int64][]dto.ReactionDto, error) {
	ret := map[int64][]dto.ReactionDto{} // messageId:reactionList

	reactions, err := getReactionsCommon(ctx, co, chatId, messageIds, nil, m.cfg.Message.MaxDisplayableReactionUsers)
	if err != nil {
		return ret, fmt.Errorf("error during interacting with db: %w", err)
	}

	for _, reaction := range reactions {
		if _, found := ret[reaction.MessageId]; !found {
			ret[reaction.MessageId] = []dto.ReactionDto{}
		}

		ret[reaction.MessageId] = append(ret[reaction.MessageId], reaction)
	}
	return ret, nil
}

func (m *CommonProjection) GetReaction(ctx context.Context, co db.CommonOperations, chatId, messageId int64, reaction string) (dto.ReactionDto, error) {
	reactions, err := getReactionsCommon(ctx, co, chatId, []int64{messageId}, &reaction, m.cfg.Message.MaxDisplayableReactionUsers)
	if err != nil {
		return dto.ReactionDto{}, fmt.Errorf("error during interacting with db: %w", err)
	}

	if len(reactions) == 0 {
		return dto.ReactionDto{
			MessageId: messageId,
			UserIds:   []int64{},
			Reaction:  reaction,
			Count:     0,
		}, nil
	}

	if len(reactions) > 1 {
		return dto.ReactionDto{}, fmt.Errorf("wrong invarint: more than 1 reaction: %w", err)
	}

	r := reactions[0]

	return r, nil
}

func (m *CommonProjection) HasMyReaction(ctx context.Context, co db.CommonOperations, chatId, messageId, behalfUserId int64, reaction string) (bool, error) {
	var reactionExists bool
	errInner := sqlscan.Get(ctx, co, &reactionExists, "SELECT EXISTS(SELECT 1 FROM message_reaction WHERE chat_id = $1 AND message_id = $2 AND user_id = $3 AND reaction = $4)", chatId, messageId, behalfUserId, reaction)
	if errInner != nil {
		return false, errInner
	}

	return reactionExists, nil
}

func takeOnAccountReactions(messageId int64, ownersSet map[int64]bool, messageReactions map[int64][]dto.ReactionDto) {
	rl, ok := messageReactions[messageId]
	if ok {
		for _, r := range rl {
			for _, u := range r.UserIds {
				ownersSet[u] = true
			}
		}
	}
}

func makeReactions(users map[int64]*dto.User, reactionsList []dto.ReactionDto) []dto.Reaction {
	var convertedReactionsOfMessageToReturn = make([]dto.Reaction, 0, len(reactionsList))
	for _, dbReaction := range reactionsList {

		reactionUsers := []*dto.User{}
		for _, u := range dbReaction.UserIds {
			ru := users[u]
			if ru == nil {
				ru = getDeletedUser(u)
			}
			reactionUsers = append(reactionUsers, ru)
		}

		convertedReactionsOfMessageToReturn = append(convertedReactionsOfMessageToReturn, dto.Reaction{
			Count:    dbReaction.Count,
			Users:    reactionUsers,
			Reaction: dbReaction.Reaction,
		})
	}

	return convertedReactionsOfMessageToReturn
}

func (m *CommonProjection) IsReactionExists(ctx context.Context, chatId, messageId int64, reaction string) (bool, error) {
	var exists bool
	err := sqlscan.Get(ctx, m.db, &exists, "select exists (select * from message_reaction where chat_id = $1 and message_id = $2 and reaction = $3)", chatId, messageId, reaction)
	if err != nil {
		return false, err
	}
	return exists, err
}

func (m *CommonProjection) GetReactionsOnMessage(ctx context.Context, co db.CommonOperations, chatId, messageId int64) ([]string, error) {
	res := []string{}

	err := sqlscan.Select(ctx, co, &res, `
		select distinct on (reaction) reaction from message_reaction where chat_id = $1 and message_id = $2
	`, chatId, messageId)
	if err != nil {
		return nil, err
	}
	return res, nil
}
