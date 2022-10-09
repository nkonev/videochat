package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"nkonev.name/chat/graph/generated"
	"nkonev.name/chat/graph/model"
)

// ChatMessageEvents is the resolver for the chatMessageEvents field.
func (r *subscriptionResolver) ChatMessageEvents(ctx context.Context, chatID int64) (<-chan *model.MessageNotify, error) {
	return nil, nil
}

// Subscription returns generated.SubscriptionResolver implementation.
func (r *Resolver) Subscription() generated.SubscriptionResolver { return &subscriptionResolver{r} }

type subscriptionResolver struct{ *Resolver }
