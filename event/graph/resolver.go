package graph

//go:generate go run github.com/99designs/gqlgen generate

import (
	"github.com/montag451/go-eventbus"
	"go.opentelemetry.io/otel/trace"
	"nkonev.name/event/client"
	"nkonev.name/event/logger"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	Bus        *eventbus.Bus
	HttpClient *client.RestClient
	Tr         trace.Tracer
	Lgr        *logger.Logger
}
