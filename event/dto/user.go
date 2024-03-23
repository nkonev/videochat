package dto

import "github.com/guregu/null"

type User struct {
	Id        int64        `json:"id"`
	Login     string       `json:"login"`
	Avatar    null.String  `json:"avatar"`
	ShortInfo null.String  `json:"shortInfo"`
	LoginColor null.String `json:"loginColor"`
}

type UserWithAdmin struct {
	User
	Admin bool `json:"admin"`
}
