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
}

type UserAccountEvent struct {
	ForWho string `json:"forWho"`
	UserId *int64 `json:"userId"` // nullable
	ForWhoRoles *[]string `json:"forWhoRoles"` // nullable
	EventType string `json:"eventType"`
	UserAccount *UserAccount `json:"userAccount"`
}

func (UserAccountEvent) Name() eventbus.EventName {
	return AAA
}
