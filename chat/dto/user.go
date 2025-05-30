package dto

import (
	"time"
)

type User struct {
	Id               int64           `json:"id"`
	Login            string          `json:"login"`
	Avatar           *string         `json:"avatar"`
	ShortInfo        *string         `json:"shortInfo"`
	LoginColor       *string         `json:"loginColor"`
	LastSeenDateTime *time.Time      `json:"lastSeenDateTime"`
	AdditionalData   *AdditionalData `json:"additionalData"`
}

type AdditionalData struct {
	Enabled   bool     `json:"enabled"`
	Expired   bool     `json:"expired"`
	Locked    bool     `json:"locked"`
	Confirmed bool     `json:"confirmed"`
	Roles     []string `json:"roles"`
}

type UserAccountEventChanged struct {
	User      *User  `json:"user"`
	EventType string `json:"eventType"`
}

type UserWithAdmin struct {
	User
	Admin bool `json:"admin"`
}

type UserOnline struct {
	Id     int64 `json:"userId"`
	Online bool  `json:"online"`
}
