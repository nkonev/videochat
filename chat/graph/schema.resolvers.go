package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/montag451/go-eventbus"
	"nkonev.name/chat/auth"
	"nkonev.name/chat/graph/generated"
	"nkonev.name/chat/graph/model"
	"nkonev.name/chat/handlers/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
)

// Ping is the resolver for the ping field.
func (r *queryResolver) Ping(ctx context.Context) (*bool, error) {
	panic(fmt.Errorf("not implemented: Ping - ping"))
}

// ChatMessageEvents is the resolver for the chatMessageEvents field.
func (r *subscriptionResolver) ChatMessageEvents(ctx context.Context, chatID int64) (<-chan *model.MessageNotify, error) {
	authResult, ok := ctx.Value(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		return nil, errors.New("Unable to get auth context")
	}

	var correctChatId int64 = int64(chatID)
	isParticipant, err := r.Db.IsParticipant(authResult.UserId, correctChatId)
	if err != nil {
		return nil, err
	}
	if !isParticipant {
		logger.GetLogEntry(ctx).Infof("User %v is not participant of chat %v", authResult.UserId, correctChatId)
		return nil, errors.New("Unauthorized")
	}

	var cam = make(chan *model.MessageNotify)
	subscribeHandler, err := r.Bus.Subscribe(dto.MESSAGE_NOTIFY_COMMON, func(e eventbus.Event, t time.Time) {
		switch e := e.(type) {
		case dto.MessageNotify:
			cam <- convertMessageNotify(&e, authResult.UserId)
		}
	})

	go func() {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("Channel closed.")
				close(cam)
				r.Bus.Unsubscribe(subscribeHandler)
				return
			}
		}
	}()

	return cam, nil
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

// Subscription returns generated.SubscriptionResolver implementation.
func (r *Resolver) Subscription() generated.SubscriptionResolver { return &subscriptionResolver{r} }

type queryResolver struct{ *Resolver }
type subscriptionResolver struct{ *Resolver }

// !!! WARNING !!!
// The code below was going to be deleted when updating resolvers. It has been copied here so you have
// one last chance to move it out of harms way if you want. There are two reasons this happens:
//   - When renaming or deleting a resolver the old code will be put in here. You can safely delete
//     it when you're done.
//   - You have helper methods in this file. Move them out to keep these resolver files clean.
func convertMessageNotify(e *dto.MessageNotify, participantId int64) *model.MessageNotify {
	displayMessageDto := e.MessageNotification
	// TODO move to better place
	var canEdit = displayMessageDto.OwnerId == participantId
	return &model.MessageNotify{
		Type: &e.Type,
		MessageNotification: &model.DisplayMessageDto{
			ID:             displayMessageDto.Id,
			Text:           displayMessageDto.Text,
			ChatID:         displayMessageDto.ChatId,
			OwnerID:        displayMessageDto.OwnerId,
			CreateDateTime: displayMessageDto.CreateDateTime,
			EditDateTime:   displayMessageDto.EditDateTime.Ptr(),
			Owner:          convertOwner(displayMessageDto.Owner),
			CanEdit:        canEdit,
			FileItemUUID:   displayMessageDto.FileItemUuid,
		},
	}
}
func convertOwner(owner *dto.User) *model.User {
	if owner == nil {
		return nil
	}
	return &model.User{
		ID:     owner.Id,
		Login:  owner.Login,
		Avatar: owner.Avatar.Ptr(),
	}
}
