package dto

import (
	"github.com/guregu/null"
	"time"
)

type ChatDto struct {
	Id                 int64     `json:"id"`
	Name               string    `json:"name"`
	LastUpdateDateTime time.Time `json:"lastUpdateDateTime"`
	ParticipantIds     []int64   `json:"participantIds"`
	Participants       []*User   `json:"participants"`
	CanEdit            null.Bool `json:"canEdit"`
}
