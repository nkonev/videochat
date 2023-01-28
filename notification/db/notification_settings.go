package db

import (
	"nkonev.name/notification/dto"
	. "nkonev.name/notification/logger"
)

func (db *DB) InitNotificationSettings(userId int64) error {
	if _, err := db.Exec(`insert into notification_settings(user_id) values($1) on conflict(user_id) do nothing`, userId); err != nil {
		Logger.Errorf("Error during initializing notification settings %v", err)
		return err
	}
	return nil
}

func (db *DB) GetNotificationSettings(userId int64) (*dto.NotificationSettings, error) {
	row := db.QueryRow(`select mentions_enabled, missed_calls_enabled, answers_enabled from notification_settings where user_id = $1`, userId)
	if row.Err() != nil {
		return nil, row.Err()
	}
	var result = new(dto.NotificationSettings)
	err := row.Scan(&result.MentionsEnabled, &result.MissedCallsEnabled, &result.AnswersEnabled)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (db *DB) PutNotificationSettings(userId int64, to *dto.NotificationSettings) error {
	if _, err := db.Exec(`update notification_settings set mentions_enabled = $2, missed_calls_enabled = $3, answers_enabled = $4 where user_id = $1`, userId, to.MentionsEnabled, to.MissedCallsEnabled, to.AnswersEnabled); err != nil {
		Logger.Errorf("Error during updating notification settings %v", err)
		return err
	}
	return nil
}
