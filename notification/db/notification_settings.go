package db

import (
	"github.com/rotisserie/eris"
	"nkonev.name/notification/dto"
)

func (db *DB) InitGlobalNotificationSettings(userId int64) error {
	if _, err := db.Exec(`insert into notification_settings(user_id) values($1) on conflict(user_id) do nothing`, userId); err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func (db *DB) InitPerChatNotificationSettings(userId, chatId int64) error {
	if _, err := db.Exec(`insert into notification_settings_chat(user_id, chat_id) values($1, $2) on conflict(user_id, chat_id) do nothing`, userId, chatId); err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func (db *DB) GetNotificationGlobalSettings(userId int64) (*dto.NotificationGlobalSettings, error) {
	row := db.QueryRow(`select mentions_enabled, missed_calls_enabled, answers_enabled, reactions_enabled from notification_settings where user_id = $1`, userId)
	if row.Err() != nil {
		return nil, eris.Wrap(row.Err(), "error during interacting with db")
	}
	var result = new(dto.NotificationGlobalSettings)
	err := row.Scan(&result.MentionsEnabled, &result.MissedCallsEnabled, &result.AnswersEnabled, &result.ReactionsEnabled)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	return result, nil
}

func (db *DB) PutNotificationGlobalSettings(userId int64, to *dto.NotificationGlobalSettings) error {
	if _, err := db.Exec(`update notification_settings set mentions_enabled = $2, missed_calls_enabled = $3, answers_enabled = $4, reactions_enabled = $5 where user_id = $1`, userId, to.MentionsEnabled, to.MissedCallsEnabled, to.AnswersEnabled, to.ReactionsEnabled); err != nil {
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

func (db *DB) GetNotificationPerChatSettings(userId, chatId int64) (*dto.NotificationPerChatSettings, error) {
	row := db.QueryRow(`select mentions_enabled, missed_calls_enabled, answers_enabled, reactions_enabled from notification_settings_chat where user_id = $1 and chat_id = $2`, userId, chatId)
	if row.Err() != nil {
		return nil, eris.Wrap(row.Err(), "error during interacting with db")
	}
	var result = new(dto.NotificationPerChatSettings)
	err := row.Scan(&result.MentionsEnabled, &result.MissedCallsEnabled, &result.AnswersEnabled, &result.ReactionsEnabled)
	if err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	}
	return result, nil
}

func (db *DB) PutNotificationPerChatSettings(userId, chatId int64, to *dto.NotificationPerChatSettings) error {
	if _, err := db.Exec(`update notification_settings_chat set mentions_enabled = $3, missed_calls_enabled = $4, answers_enabled = $5, reactions_enabled = $6 where user_id = $1 and chat_id = $2`, userId, chatId, to.MentionsEnabled, to.MissedCallsEnabled, to.AnswersEnabled, to.ReactionsEnabled); err != nil { // TODO
		return eris.Wrap(err, "error during interacting with db")
	}
	return nil
}

