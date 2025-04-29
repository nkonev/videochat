package dto

type User struct {
	Id             int64           `json:"id"`
	Login          string          `json:"login"`
	Avatar         *string         `json:"avatar"`
	ShortInfo      *string         `json:"shortInfo"`
	LoginColor     *string         `json:"loginColor"`
	AdditionalData *AdditionalData `json:"additionalData"`
}

type AdditionalData struct {
	Enabled   bool     `json:"enabled"`
	Expired   bool     `json:"expired"`
	Locked    bool     `json:"locked"`
	Confirmed bool     `json:"confirmed"`
	Roles     []string `json:"roles"`
}
