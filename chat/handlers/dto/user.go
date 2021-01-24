package dto

import "github.com/guregu/null"

type User struct {
	Id     int64       `json:"id"`
	Login  string      `json:"login"`
	Avatar null.String `json:"avatar"`
	Admin  bool        `json:"admin"`
}
