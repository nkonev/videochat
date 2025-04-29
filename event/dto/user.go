package dto

import "github.com/guregu/null"

type User struct {
	Id             int64           `json:"id"`
	Login          string          `json:"login"`
	Avatar         null.String     `json:"avatar"`
	ShortInfo      null.String     `json:"shortInfo"`
	LoginColor     null.String     `json:"loginColor"`
	AdditionalData *AdditionalData `json:"additionalData"`
}

type AdditionalData struct {
	Enabled   bool     `json:"enabled"`
	Expired   bool     `json:"expired"`
	Locked    bool     `json:"locked"`
	Confirmed bool     `json:"confirmed"`
	Roles     []string `json:"roles"`
}

type UserAccountEvent struct {
	Id                            int64           `json:"id"`
	Login                         string          `json:"login"`
	Email                         null.String     `json:"email"`
	AwaitingForConfirmEmailChange bool            `json:"awaitingForConfirmEmailChange"`
	Avatar                        null.String     `json:"avatar"`
	ShortInfo                     null.String     `json:"shortInfo"`
	LoginColor                    null.String     `json:"loginColor"`
	AdditionalData                *AdditionalData `json:"additionalData"`
}

type UserWithAdmin struct {
	User
	Admin bool `json:"admin"`
}
