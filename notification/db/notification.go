package db

import (
	"errors"
	"github.com/spf13/viper"
	"nkonev.name/notification/dto"
	. "nkonev.name/notification/logger"
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
			return errors.New("No rows affected")
		}
	}
	return nil
}

func (db *DB) DeleteNotificationByMessageId(messageId int64, userId int64) error {
	if res, err := db.Exec(`delete from notification where message_id = $1 and user_id = $2`, messageId, userId); err != nil {
		Logger.Errorf("Error during deleting notification id %v", err)
		return err
	} else {
		affected, err := res.RowsAffected()
		if err != nil {
			Logger.Errorf("Error during checking rows affected %v", err)
			return err
		}
		if affected == 0 {
			return errors.New("No rows affected")
		}
	}
	return nil
}

func (db *DB) PutNotification(messageId *int64, userId int64, chatId int64, notificationType string, description *string) error {

	if res, err := db.Exec(
		`insert into notification(notification_type, description, message_id, user_id, chat_id) 
			values ($1, $2, $3, $4, $5) 
			on conflict(notification_type, message_id, user_id, chat_id) 
			do update set description = excluded.description;`,
		notificationType, description, messageId, userId, chatId); err != nil {

		Logger.Errorf("Error during putting notification id %v", err)
		return err
	} else {
		affected, err := res.RowsAffected()
		if err != nil {
			Logger.Errorf("Error during checking rows affected %v", err)
			return err
		}
		if affected == 0 {
			return errors.New("No rows affected")
		}
	}
	return nil
}

func (db *DB) GetNotifications(userId int64) ([]dto.NotificationDto, error) {
	maxNotifications := viper.GetInt("maxNotifications")
	rows, err := db.Query("select id, notification_type, description, chat_id, message_id, create_date_time from notification where user_id = $1 order by id desc limit $2", userId, maxNotifications)
	if err != nil {
		Logger.Errorf("Error during getting notifications %v", err)
		return nil, err
	}
	defer rows.Close()

	list := make([]dto.NotificationDto, 0)
	for rows.Next() {
		notificationDto := dto.NotificationDto{}
		if err := rows.Scan(&notificationDto.Id, &notificationDto.Type, &notificationDto.Description, &notificationDto.ChatId, &notificationDto.MessageId, &notificationDto.CreateDateTime); err != nil {
			Logger.Errorf("Error during scan notification rows %v", err)
			return nil, err
		} else {
			list = append(list, notificationDto)
		}
	}
	return list, nil

}
