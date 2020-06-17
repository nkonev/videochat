package db

import (
	 "time"
	. "nkonev.name/chat/logger"
)


type Message struct {
	Id    int64
	Text string
	ChatId int64
	OwnerId int64
	CreateDateTime time.Time
	EditDateTime time.Time
}

func (db *DB) GetMessages(chatId int64, userId int64, limit int, offset int) ([]*Message, error) {
	if rows, err := db.Query(`SELECT * FROM message WHERE chat_id IN ( SELECT chat_id FROM chat_participant WHERE user_id = $1 AND chat_id = $4 ) ORDER BY id LIMIT $2 OFFSET $3`, userId, limit, offset, chatId); err != nil {
		Logger.Errorf("Error during get chat rows %v", err)
		return nil, err
	} else {
		defer rows.Close()
		list := make([]*Message, 0)
		for rows.Next() {
			message := Message{}
			if err := rows.Scan(&message.Id, &message.Text); err != nil {
				Logger.Errorf("Error during scan message rows %v", err)
				return nil, err
			} else {
				list = append(list, &message)
			}
		}
		return list, nil
	}
}