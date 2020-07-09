package dto

import "github.com/guregu/null"

type ChatDto struct {
	Id             int64         `json:"id"`
	Name           string        `json:"name"`
	ParticipantIds []int64       `json:"participantIds"`
	Participants   []Participant `json:"participants"`
}

type Participant struct {
	Id     int64       `json:"id"`
	Login  string      `json:"login"`
	Avatar null.String `json:"avatar"`
}

