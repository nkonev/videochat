package dto

type IdResponse struct {
	Id int64 `json:"id"`
}

type ErrorMessageDto struct {
	Message string `json:"message"`
}

type HasUnreadMessages struct {
	HasUnreadMessages bool `json:"hasUnreadMessages"`
}

type PutChatNotificationSettingsDto struct {
	ConsiderMessagesOfThisChatAsUnread bool `json:"considerMessagesOfThisChatAsUnread"`
}

type SearchUsersRequestDto struct {
	Page         int64   `json:"page"`
	Size         int32   `json:"size"`
	UserIds      []int64 `json:"userIds"`
	SearchString string  `json:"searchString"`
	Including    bool    `json:"including"`
}

type SearchUsersResponseDto struct {
	Users []*User `json:"users"`
	Count int64   `json:"count"`
}

type FreshDto struct {
	Ok bool `json:"ok"`
}

type FilterDto struct {
	Found bool `json:"found"`
}
