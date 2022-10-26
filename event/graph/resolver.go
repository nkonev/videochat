package graph

import (
	"github.com/montag451/go-eventbus"
	"nkonev.name/event/client"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	Bus        *eventbus.Bus
	HttpClient *client.RestClient
}
