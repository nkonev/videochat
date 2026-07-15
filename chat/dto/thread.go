package dto

const RootThreadId = 0

type ThreadAuthorizationData struct {
	IsChatFound         bool `db:"is_chat_found"`
	IsParticipant       bool `db:"is_chat_participant"`
	ChatCanCreateThread bool `db:"chat_can_create_thread"`
}
