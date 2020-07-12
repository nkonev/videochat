package dto

import "github.com/guregu/null"

type ChatDto struct {
	Id             int64     `json:"id"`
	Name           string    `json:"name"`
	ParticipantIds []int64   `json:"participantIds"`
	Participants   []*User   `json:"participants"`
	CanEdit        null.Bool `json:"canEdit"`
}
