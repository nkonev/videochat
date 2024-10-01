package db

import (
	"context"
	"database/sql"
	"errors"
	"github.com/rotisserie/eris"
	"nkonev.name/notification/dto"
)

func (db *DB) InitGlobalNotificationSettings(ctx context.Context, userId int64) error {
	if _, err := db.ExecContext(ctx, `insert into notification_settings(user_id) values($1) on conflict(user_id) do nothing`, userId); err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func (db *DB) InitPerChatNotificationSettings(ctx context.Context, userId, chatId int64) error {
	if _, err := db.ExecContext(ctx, `insert into notification_settings_chat(user_id, chat_id) values($1, $2) on conflict(user_id, chat_id) do nothing`, userId, chatId); err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func (db *DB) GetNotificationGlobalSettings(ctx context.Context, userId int64) (*dto.NotificationGlobalSettings, error) {
	row := db.QueryRowContext(ctx, `select mentions_enabled, missed_calls_enabled, answers_enabled, reactions_enabled from notification_settings where user_id = $1`, userId)
	if row.Err() != nil {
		return nil, eris.Wrap(row.Err(), "error during interacting with db")
	}
	var result = new(dto.NotificationGlobalSettings)
	err := row.Scan(&result.MentionsEnabled, &result.MissedCallsEnabled, &result.AnswersEnabled, &result.ReactionsEnabled)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// if there is no rows then return default
			return &dto.NotificationGlobalSettings{ // should match to defaults
				MentionsEnabled:    true,
				MissedCallsEnabled: true,
				AnswersEnabled:     true,
				ReactionsEnabled:   true,
			}, nil
		}
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	return result, nil
}

func (db *DB) PutNotificationGlobalSettings(ctx context.Context, userId int64, to *dto.NotificationGlobalSettings) error {
	if _, err := db.ExecContext(ctx, `update notification_settings set mentions_enabled = $2, missed_calls_enabled = $3, answers_enabled = $4, reactions_enabled = $5 where user_id = $1`, userId, to.MentionsEnabled, to.MissedCallsEnabled, to.AnswersEnabled, to.ReactionsEnabled); err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func (db *DB) GetNotificationPerChatSettings(ctx context.Context, userId, chatId int64) (*dto.NotificationPerChatSettings, error) {
	row := db.QueryRowContext(ctx, `select mentions_enabled, missed_calls_enabled, answers_enabled, reactions_enabled from notification_settings_chat where user_id = $1 and chat_id = $2`, userId, chatId)
	if row.Err() != nil {
		return nil, eris.Wrap(row.Err(), "error during interacting with db")
	}
	var result = new(dto.NotificationPerChatSettings)
	err := row.Scan(&result.MentionsEnabled, &result.MissedCallsEnabled, &result.AnswersEnabled, &result.ReactionsEnabled)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// if there is no rows then return default
			return &dto.NotificationPerChatSettings{ // should match to defaults
				MentionsEnabled:    nil,
				MissedCallsEnabled: nil,
				AnswersEnabled:     nil,
				ReactionsEnabled:   nil,
			}, nil
		}
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	return result, nil
}

func (db *DB) PutNotificationPerChatSettings(ctx context.Context, userId, chatId int64, to *dto.NotificationPerChatSettings) error {
	if _, err := db.ExecContext(ctx, `update notification_settings_chat set mentions_enabled = $3, missed_calls_enabled = $4, answers_enabled = $5, reactions_enabled = $6 where user_id = $1 and chat_id = $2`, userId, chatId, to.MentionsEnabled, to.MissedCallsEnabled, to.AnswersEnabled, to.ReactionsEnabled); err != nil { // TODO
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}
