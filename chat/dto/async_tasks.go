package dto

type PublishBroadcastMessage struct {
	MessageText string `json:"messageText"`
	ChatId      int64  `json:"chatId"`
	UserId      int64  `json:"userId"`
	UserLogin   string `json:"userLogin"`
}

type PublishUserTyping struct {
	ChatId    int64  `json:"chatId"`
	UserId    int64  `json:"userId"`
	UserLogin string `json:"userLogin"`
}
