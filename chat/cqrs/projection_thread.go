package cqrs

func CanCreateThread(chatCanCreateThread, cfgCanCreateThread, isParticipant bool) bool {
	return isParticipant && chatCanCreateThread && cfgCanCreateThread
}
