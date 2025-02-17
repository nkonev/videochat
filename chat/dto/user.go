package dto

import "github.com/guregu/null"

type User struct {
	Id               int64       `json:"id"`
	Login            string      `json:"login"`
	Avatar           null.String `json:"avatar"`
	ShortInfo        null.String `json:"shortInfo"`
	LoginColor       null.String `json:"loginColor"`
	LastSeenDateTime null.Time   `json:"lastSeenDateTime"`
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
