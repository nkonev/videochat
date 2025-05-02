package client

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"nkonev.name/chat/dto"
	"testing"
)

func TestParsingWoMillis(t *testing.T) {
	s := `
	[
	  {
		"id": 1007,
		"login": "forgot-password-user",
		"avatar": "/api/storage/public/user/avatar/1007_AVATAR_200x200.jpg?time=1746128796",
		"shortInfo": "biba",
		"loginColor": null,
		"lastSeenDateTime": "2024-04-29T19:55:56Z",
		"additionalData": {
		  "enabled": true,
		  "expired": false,
		  "locked": true,
		  "confirmed": true,
		  "roles": [
			"ROLE_USER"
		  ]
		}
	  }
	]
	`

	users := &[]*dto.User{}
	err := json.Unmarshal([]byte(s), users)
	assert.Nil(t, err)
}

func TestParsingWSomeMillis(t *testing.T) {
	s := `
	[
	  {
		"id": 1007,
		"login": "forgot-password-user",
		"avatar": "/api/storage/public/user/avatar/1007_AVATAR_200x200.jpg?time=1746128796",
		"shortInfo": "biba",
		"loginColor": null,
		"lastSeenDateTime": "2024-04-29T19:55:56.1Z",
		"additionalData": {
		  "enabled": true,
		  "expired": false,
		  "locked": true,
		  "confirmed": true,
		  "roles": [
			"ROLE_USER"
		  ]
		}
	  }
	]
	`

	users := &[]*dto.User{}
	err := json.Unmarshal([]byte(s), users)
	assert.Nil(t, err)
}

func TestParsingWAllMillis(t *testing.T) {
	s := `
	[
	  {
		"id": 1007,
		"login": "forgot-password-user",
		"avatar": "/api/storage/public/user/avatar/1007_AVATAR_200x200.jpg?time=1746128796",
		"shortInfo": "biba",
		"loginColor": null,
		"lastSeenDateTime": "2024-04-29T19:55:56.123Z",
		"additionalData": {
		  "enabled": true,
		  "expired": false,
		  "locked": true,
		  "confirmed": true,
		  "roles": [
			"ROLE_USER"
		  ]
		}
	  }
	]
	`

	users := &[]*dto.User{}
	err := json.Unmarshal([]byte(s), users)
	assert.Nil(t, err)
}
