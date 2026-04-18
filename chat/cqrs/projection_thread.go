package cqrs

func CanCreateThread(chatCanCreateThread, isParticipant bool) bool {
	if !isParticipant {
		return false
	}

	return chatCanCreateThread
}
