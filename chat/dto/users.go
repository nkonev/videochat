package dto

import "time"

const NoId = -1
const NoSize = -1
const NonExistentUser = -65000
const DeletedUser = -1
const AllUsers = -2
const HereUsers = -3

const AllUsersLogin = "all"
const HereUsersLogin = "here"

const ROLE_ADMIN = "ROLE_ADMIN"

const CAN_CREATE_BLOG = "CAN_CREATE_BLOG"

const SystemUserCleaner = -1000

type User struct {
	Id               int64           `json:"id"`
	Login            string          `json:"login"`
	Avatar           *string         `json:"avatar"`
	ShortInfo        *string         `json:"shortInfo"`
	LoginColor       *string         `json:"loginColor"`
	LastSeenDateTime *time.Time      `json:"lastSeenDateTime"`
	AdditionalData   *AdditionalData `json:"additionalData"`
	Permissions      []string        `json:"permissions"`
}

func (u *User) GetId() int64 {
	if u != nil {
		return u.Id
	} else {
		return NoId
	}
}

type UserWithAdmin struct {
	User
	ChatAdmin bool `json:"admin"`
}

type UserViewEnrichedDto struct {
	UserWithAdmin
	BehalfUserId int64 `json:"-"` // behalf userId
	CanChange    bool  `json:"canChange"`
	CanDelete    bool  `json:"canDelete"`
}

type AdditionalData struct {
	Enabled   bool     `json:"enabled"`
	Expired   bool     `json:"expired"`
	Locked    bool     `json:"locked"`
	Confirmed bool     `json:"confirmed"`
	Roles     []string `json:"roles"`
}

type UserOnline struct {
	Id     int64 `json:"userId"`
	Online bool  `json:"online"`
}

func (u UserOnline) GetId() int64 {
	return u.Id
}

type ParticipantAddDto struct {
	ParticipantIds []int64 `json:"addParticipantIds"`
}

type ParticipantsWithAdminWrapper struct {
	Data  []*UserViewEnrichedDto `json:"items"`
	Count int64                  `json:"count"` // for paginating purposes
}

type CountRequestDto struct {
	SearchString string `json:"searchString"`
}

type FilteredParticipantsRequestDto struct {
	SearchString string  `json:"searchString"`
	UserId       []int64 `json:"userId"`
}

type FilteredParticipantItemResponse struct {
	Id int64 `json:"id"`
}

type UserAccountEventChanged struct {
	User      *User  `json:"user"`
	EventType string `json:"eventType"`
}

type UserExists struct {
	Exists bool  `json:"exists"`
	UserId int64 `json:"userId"`
}

func (u UserExists) GetId() int64 {
	return u.UserId
}
