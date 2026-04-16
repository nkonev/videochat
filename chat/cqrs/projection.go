package cqrs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"nkonev.name/chat/client"
	"nkonev.name/chat/config"
	"nkonev.name/chat/db"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/sanitizer"

	"github.com/georgysavva/scany/v2/sqlscan"
)

type CommonProjection struct {
	db        *db.DB
	lgr       *logger.LoggerWrapper
	cfg       *config.AppConfig
	stripTags *sanitizer.StripTagsPolicy
}

type EnrichingProjection struct {
	cp                 *CommonProjection
	lgr                *logger.LoggerWrapper
	aaaRestClient      client.AaaRestClient
	policy             *sanitizer.SanitizerPolicy
	stripAllTags       *sanitizer.StripTagsPolicy
	stripSourceContent *sanitizer.StripSourcePolicy
	cfg                *config.AppConfig
}

func NewCommonProjection(db *db.DB, lgr *logger.LoggerWrapper, cfg *config.AppConfig, stripTags *sanitizer.StripTagsPolicy) *CommonProjection {
	return &CommonProjection{
		db:        db,
		lgr:       lgr,
		cfg:       cfg,
		stripTags: stripTags,
	}
}

func NewEnrichingProjection(cp *CommonProjection, lgr *logger.LoggerWrapper, aaaRestClient client.AaaRestClient, cfg *config.AppConfig, policy *sanitizer.SanitizerPolicy, stripAllTags *sanitizer.StripTagsPolicy, stripSourceContent *sanitizer.StripSourcePolicy) *EnrichingProjection {
	return &EnrichingProjection{
		cp:                 cp,
		lgr:                lgr,
		aaaRestClient:      aaaRestClient,
		cfg:                cfg,
		stripAllTags:       stripAllTags,
		policy:             policy,
		stripSourceContent: stripSourceContent,
	}
}

func (m *CommonProjection) GetNextChatId(ctx context.Context, tx *db.Tx) (int64, error) {
	var nid int64
	err := sqlscan.Get(ctx, tx, &nid, "select nextval('chat_id_sequence')")
	if err != nil {
		return 0, err
	}
	return nid, nil
}

func (m *CommonProjection) InitializeChatIdSequenceIfNeed(ctx context.Context, tx *db.Tx) error {
	var called bool
	err := sqlscan.Get(ctx, tx, &called, "SELECT is_called FROM chat_id_sequence")
	if err != nil {
		return err
	}

	if !called {
		var maxChatId int64
		err = sqlscan.Get(ctx, tx, &maxChatId, "SELECT coalesce(max(id), 0) from chat_common")
		if err != nil {
			return err
		}

		if maxChatId > 0 {
			m.lgr.Info("Fast-forwarding chatId sequence")
			_, err = tx.ExecContext(ctx, "SELECT setval('chat_id_sequence', $1, true)", maxChatId)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

const ChatStillNotExists = -1

func (m *CommonProjection) GetNextMessageId(ctx context.Context, co db.CommonOperations, chatId int64) (int64, error) {
	var messageId int64
	err := sqlscan.Get(ctx, co, &messageId, "UPDATE chat_common SET last_generated_message_id = last_generated_message_id + 1 WHERE id = $1 RETURNING last_generated_message_id;", chatId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// there were no rows, but otherwise no error occurred
			return ChatStillNotExists, nil
		}
		return 0, fmt.Errorf("error during generating message id: %w", err)
	}
	return messageId, nil
}

func (m *CommonProjection) InitializeMessageIdSequenceIfNeed(ctx context.Context, tx *db.Tx, chatId int64) error {
	var currentGeneratedMessageId int64

	err := sqlscan.Get(ctx, tx, &currentGeneratedMessageId, "SELECT coalesce(last_generated_message_id, 0) from chat_common where id = $1", chatId)
	if err != nil {
		return err
	}

	if currentGeneratedMessageId == 0 {
		var maxMessageId int64
		err = sqlscan.Get(ctx, tx, &maxMessageId, "SELECT coalesce(max(id), 0) from message where chat_id = $1", chatId)
		if err != nil {
			return err
		}

		if maxMessageId > 0 {
			m.lgr.Info("Fast-forwarding messageId sequence", logger.AttributeChatId, chatId)

			_, err = tx.ExecContext(ctx, "update chat_common set last_generated_message_id = $2 where id = $1", chatId, maxMessageId)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

const need_to_fast_forward_sequences_key = "need_to_fast_forward_sequences"
const need_to_fast_forward_sequences_value = "true"

func (m *CommonProjection) SetIsNeedToFastForwardSequences(ctx context.Context) error {
	_, err := m.db.ExecContext(ctx, "insert into technical(the_key, the_value) values ($1, $2) on conflict (the_key) do update set the_value = excluded.the_value", need_to_fast_forward_sequences_key, need_to_fast_forward_sequences_value)
	return err
}

func (m *CommonProjection) UnsetIsNeedToFastForwardSequences(ctx context.Context, co db.CommonOperations) error {
	_, err := co.ExecContext(ctx, "delete from technical where the_key = $1", need_to_fast_forward_sequences_key)
	return err
}

func (m *CommonProjection) GetIsNeedToFastForwardSequences(ctx context.Context, co db.CommonOperations) (bool, error) {
	var e bool
	err := sqlscan.Get(ctx, co, &e, "select exists(select * from technical where the_key = $1 and the_value = $2)", need_to_fast_forward_sequences_key, need_to_fast_forward_sequences_value)
	if err != nil {
		return false, err
	}
	return e, err
}

const need_to_skip_import_key = "need_to_skip_import"
const need_to_skip_import_value = "true"

func (m *CommonProjection) SetIsNeedToSkipImport(ctx context.Context) error {
	_, err := m.db.ExecContext(ctx, "insert into technical(the_key, the_value) values ($1, $2) on conflict (the_key) do update set the_value = excluded.the_value", need_to_skip_import_key, need_to_skip_import_value)
	return err
}

func (m *CommonProjection) UnsetIsNeedToSkipImport(ctx context.Context, co db.CommonOperations) error {
	_, err := co.ExecContext(ctx, "delete from technical where the_key = $1", need_to_skip_import_key)
	return err
}

func (m *CommonProjection) GetIsNeedToSkipImport(ctx context.Context, co db.CommonOperations) (bool, error) {
	var e bool
	err := sqlscan.Get(ctx, co, &e, "select exists(select * from technical where the_key = $1 and the_value = $2)", need_to_skip_import_key, need_to_skip_import_value)
	if err != nil {
		return false, err
	}
	return e, err
}

const truncating_completed_key = "truncating_completed"
const truncating_completed_value = "true"

func (m *CommonProjection) SetIsTruncatingCompleted(ctx context.Context) error {
	_, err := m.db.ExecContext(ctx, "insert into technical(the_key, the_value) values ($1, $2) on conflict (the_key) do update set the_value = excluded.the_value", truncating_completed_key, truncating_completed_value)
	return err
}

func (m *CommonProjection) UnsetIsTruncatingCompleted(ctx context.Context, co db.CommonOperations) error {
	_, err := co.ExecContext(ctx, "delete from technical where the_key = $1", truncating_completed_key)
	return err
}

func (m *CommonProjection) GetIsTruncatingCompleted(ctx context.Context, co db.CommonOperations) (bool, error) {
	var e bool
	err := sqlscan.Get(ctx, co, &e, "select exists(select * from technical where the_key = $1 and the_value = $2)", truncating_completed_key, truncating_completed_value)
	if err != nil {
		return false, err
	}
	return e, err
}

const lockIdKey1 = 1
const lockIdKey2 = 2

func (m *CommonProjection) SetXactFastForwardSequenceLock(ctx context.Context, tx *db.Tx) error {
	_, err := tx.ExecContext(ctx, "select pg_advisory_xact_lock($1, $2)", lockIdKey1, lockIdKey2)
	return err
}

func (m *EnrichingProjection) SanitizeSearchString(searchString string) string {
	return sanitizer.TrimAmdSanitize(m.policy, searchString)
}
