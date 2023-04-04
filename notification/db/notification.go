package db

import (
	"github.com/spf13/viper"
	"nkonev.name/notification/dto"
	. "nkonev.name/notification/logger"
	"time"
)

func (db *DB) DeleteNotification(id int64, userId int64) error {
	if res, err := db.Exec(`delete from notification where id = $1 and user_id = $2`, id, userId); err != nil {
		Logger.Errorf("Error during deleting notification id %v", err)
		return err
	} else {
		affected, err := res.RowsAffected()
		if err != nil {
			Logger.Errorf("Error during checking rows affected %v", err)
			return err
		}
		if affected == 0 {
			Logger.Infof("No rows affected")
		}
	}
	return nil
}

func (db *DB) DeleteNotificationByMessageId(messageId int64, notificationType string, userId int64) (int64, error) {
	res := db.QueryRow(`delete from notification where message_id = $1 and user_id = $2 and notification_type = $3 returning id`, messageId, userId, notificationType)
	if res.Err() != nil {
		return 0, res.Err()
	}
	var id int64
	err := res.Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (db *DB) PutNotification(messageId *int64, userId int64, chatId int64, notificationType string, description string, byUserId int64, byLogin string, chatTitle string) (int64, time.Time, error) {

	res := db.QueryRow(
		`insert into notification(notification_type, description, message_id, user_id, chat_id, by_user_id, by_login, chat_title) 
			values ($1, $2, $3, $4, $5, $6, $7, $8) 
			on conflict(notification_type, message_id, user_id, chat_id) 
			do update set description = excluded.description
			RETURNING id, create_date_time`,
		notificationType, description, messageId, userId, chatId, byUserId, byLogin, chatTitle)
	if res.Err() != nil {
		return 0, time.Now(), res.Err()
	}
	var id int64
	var createDatetime time.Time
	if err := res.Scan(&id, &createDatetime); err != nil {
		Logger.Errorf("Error during putting notification %v", err)
		return 0, time.Now(), err
	}
	return id, createDatetime, nil
}

func (db *DB) GetNotifications(userId int64) ([]dto.NotificationDto, error) {
	maxNotifications := viper.GetInt("maxNotifications")
	rows, err := db.Query("select id, notification_type, description, chat_id, message_id, create_date_time, by_user_id, by_login, chat_title from notification where user_id = $1 order by id desc limit $2", userId, maxNotifications)
	if err != nil {
		Logger.Errorf("Error during getting notifications %v", err)
		return nil, err
	}
	defer rows.Close()

	list := make([]dto.NotificationDto, 0)
	for rows.Next() {
		notificationDto := dto.NotificationDto{}
		if err := rows.Scan(&notificationDto.Id, &notificationDto.NotificationType, &notificationDto.Description, &notificationDto.ChatId, &notificationDto.MessageId, &notificationDto.CreateDateTime, &notificationDto.ByUserId, &notificationDto.ByLogin, &notificationDto.ChatTitle); err != nil {
			Logger.Errorf("Error during scan notification rows %v", err)
			return nil, err
		} else {
			list = append(list, notificationDto)
		}
	}
	return list, nil

}

func (db *DB) GetNotificationCount(userId int64) (int64, error) {
	row := db.QueryRow("select count(*) from notification where user_id = $1", userId)
	if row.Err() != nil {
		Logger.Errorf("Error during getting notification count %v", row.Err())
		return 0, row.Err()
	}
	var count int64
	err := row.Scan(&count)
	if err != nil {
		Logger.Errorf("Error during scanning notification count %v", err)
		return 0, err
	}

	return count, nil
}

func (db *DB) GetExcessUserNotificationIds(userId int64, numToDelete int64) ([]int64, error) {
	rows, err := db.Query("select id from notification where user_id = $1 order by id asc limit $2", userId, numToDelete)
	if err != nil {
		Logger.Errorf("Error during remiving excess notificautions %v", err)
		return nil, err
	}
	defer rows.Close()

	list := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			Logger.Errorf("Error during scan notification rows %v", err)
			return nil, err
		} else {
			list = append(list, id)
		}
	}
	return list, nil
}
