package listener

import (
	. "nkonev.name/chat/logger"
)

type AaaUserProfileUpdateListener func(channel string, data []byte) error

func CreateAaaUserProfileUpdateListener() AaaUserProfileUpdateListener {
	return func(channel string, data []byte) error {
		s := string(data)
		Logger.Infof("Received %v", s)
		return nil
	}
}
