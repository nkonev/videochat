package dto

import (
	"github.com/montag451/go-eventbus"
	"time"
)

type Oauth2Identifiers struct {
	FacebookId  *string `json:"facebookId"`
	VkontakteId *string `json:"vkontakteId"`
	GoogleId    *string `json:"googleId"`
	KeycloakId  *string `json:"keycloakId"`
}

type UserAccount struct {
	Id                int64              `json:"id"`
	Login             string             `json:"login"`
	Avatar            *string            `json:"avatar"`
	AvatarBig         *string            `json:"avatarBig"`
	ShortInfo         *string            `json:"shortInfo"`
	LastLoginDateTime *time.Time         `json:"lastLoginDateTime"`
	Oauth2Identifiers *Oauth2Identifiers `json:"oauth2Identifiers"`
	LoginColor        *string            `json:"loginColor"`
	Ldap              bool               `json:"ldap"`
}

type DataDTO struct {
	Enabled   bool     `json:"enabled"`
	Expired   bool     `json:"expired"`
	Locked    bool     `json:"locked"`
	Confirmed bool     `json:"confirmed"`
	Roles     []string `json:"roles"`
}

type UserAccountExtended struct {
	UserAccount
	AdditionalData    *DataDTO `json:"additionalData"`
	CanLock           bool     `json:"canLock"`
	CanDelete         bool     `json:"canDelete"`
	CanChangeRole     bool     `json:"canChangeRole"`
	CanConfirm        bool     `json:"canConfirm"`
	CanRemoveSessions bool     `json:"canRemoveSessions"`
}

type UserAccountEventChanged struct {
	TraceString string            `json:"-"`
	UserId      int64             `json:"userId"`
	EventType   string            `json:"eventType"`
	User        *UserAccountEvent `json:"user"`
}

func (UserAccountEventChanged) Name() eventbus.EventName {
	return AAA_CHANGE
}

type UserAccountEventCreated struct {
	TraceString string            `json:"-"`
	UserId      int64             `json:"userId"`
	User        *UserAccountEvent `json:"user"`
	EventType   string            `json:"eventType"`
}

func (UserAccountEventCreated) Name() eventbus.EventName {
	return AAA_CREATE
}

type UserAccountEventDeleted struct {
	TraceString string `json:"-"`
	UserId      int64  `json:"userId"`
	EventType   string `json:"eventType"`
}

func (UserAccountEventDeleted) Name() eventbus.EventName {
	return AAA_DELETE
}

type UserSessionsKilledEvent struct {
	TraceString string `json:"-"`
	UserId      int64  `json:"userId"`
	EventType   string `json:"eventType"`
	ReasonType  string `json:"reasonType"`
}

func (UserSessionsKilledEvent) Name() eventbus.EventName {
	return AAA_KILL_SESSIONS
}
