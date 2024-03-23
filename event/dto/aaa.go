package dto

import (
	"github.com/montag451/go-eventbus"
	"time"
)

type Oauth2Identifiers struct {
	FacebookId *string `json:"facebookId"`
	VkontakteId *string `json:"vkontakteId"`
	GoogleId *string `json:"googleId"`
	KeycloakId *string `json:"keycloakId"`
}

type UserAccount struct {
	Id        int64       `json:"id"`
	Login     string      `json:"login"`
	Avatar    *string `json:"avatar"`
	AvatarBig *string `json:"avatarBig"`
	ShortInfo *string `json:"shortInfo"`
	LastLoginDateTime *time.Time `json:"lastLoginDateTime"`
	Oauth2Identifiers *Oauth2Identifiers `json:"oauth2Identifiers"`
	LoginColor    *string `json:"loginColor"`
}

type DataDTO struct {
	Enabled bool `json:"enabled"`
	Expired bool `json:"expired"`
	Locked bool `json:"locked"`
	Confirmed bool `json:"confirmed"`
	Roles []string `json:"roles"`
}

type UserAccountExtended struct {
	UserAccount
	AdditionalData DataDTO `json:"additionalData"`
	CanLock bool `json:"canLock"`
	CanDelete bool `json:"canDelete"`
	CanChangeRole bool `json:"canChangeRole"`
	CanConfirm bool `json:"canConfirm"`
}

type UserAccountEventGroup struct {
	UserId int64 `json:"userId"`
	EventType string `json:"eventType"`
	ForMyself *UserAccountExtended `json:"forMyself"`
	ForRoleAdmin *UserAccountExtended `json:"forRoleAdmin"`
	ForRoleUser  *UserAccount `json:"forRoleUser"`
}

func (UserAccountEventGroup) Name() eventbus.EventName {
	return AAA_CHANGE
}

type UserAccountCreatedEventGroup struct {
	UserId int64 `json:"userId"`
	EventType string `json:"eventType"`
	ForRoleAdmin *UserAccountExtended `json:"forRoleAdmin"`
	ForRoleUser  *UserAccount `json:"forRoleUser"`
}

func (UserAccountCreatedEventGroup) Name() eventbus.EventName {
	return AAA_CREATE
}


type UserAccountDeletedEvent struct {
	UserId int64 `json:"userId"`
	EventType string `json:"eventType"`
}

func (UserAccountDeletedEvent) Name() eventbus.EventName {
	return AAA_DELETE
}

type UserSessionsKilledEvent struct {
	UserId int64 `json:"userId"`
	EventType string `json:"eventType"`
	ReasonType string `json:"reasonType"`
}

func (UserSessionsKilledEvent) Name() eventbus.EventName {
	return AAA_KILL_SESSIONS
}
