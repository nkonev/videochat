package dto

type User struct {
	Id     int64       `json:"id"`
	Login  string      `json:"login"`
	Avatar *string `json:"avatar"`
}
