package dto

import (
	"github.com/google/uuid"
	"time"
)

type UserCallStateId struct {
	TokenId uuid.UUID
	UserId int64
}

type UserCallState struct {
	TokenId uuid.UUID
	UserId int64

	ChatId int64

	TokenTaken bool

	// Owner* fields are set only when Status == CallStatusBeingInvited, CallStatusCancelling, CallStatusRemoving
	OwnerTokenId *uuid.UUID
	OwnerUserId *int64

	Status string

	// just cached chat is tet-a-tet and owner's avatar
	ChatTetATet bool
	OwnerAvatar *string

	// time of marking for removal (soft removal)
	// set only when IsTemporary(Status) == true
	MarkedForRemoveAt *time.Time
	// orphan means "not existed in Livekit"
	MarkedForOrphanRemoveAttempt int

	CreateDateTime time.Time
}
