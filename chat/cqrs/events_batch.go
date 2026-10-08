package cqrs

import "context"

const (
	BatchMessagesCreated     = "batchMessagesCreated"
	BatchChatsCreated        = "batchChatsCreated"
	BatchParticipantsAdded   = "batchParticipantsAdded"
	BatchEventUserChatAdded  = "batchUserChatAdded"
	BatchEventUserChatEdited = "batchUserChatEdited"

	defaultOrder = 100000
)

func (p *EventHolder) MakeBatchItem() (BatchEvent, context.Context, error) {
	switch typed := p.event.(type) {
	case *MessageCreated:
		return &MessageCreatedEventBatch{
			ChatId: typed.MessageCommoned.ChatId,
			MessageCreateds: []MessageCreated{
				*typed,
			},
			FirstElementContext: p.ctx,
		}, p.ctx, nil
	case *ChatCreated:
		return &ChatCreatedEventBatch{
			ChatCreateds: []ChatCreated{
				*typed,
			},
			FirstElementContext: p.ctx,
		}, p.ctx, nil
	case *ParticipantsAdded:
		return &ParticipantsAddedEventBatch{
			ParticipantsAddeds: []ParticipantsAdded{
				*typed,
			},
			FirstElementContext: p.ctx,
		}, p.ctx, nil
	case *UserChatParticipantAdded:
		return &UserChatCreatedBatch{
			UserId: typed.UserId,
			UserChatAddeds: []UserChatParticipantAdded{
				*typed,
			},
			FirstElementContext: p.ctx,
		}, p.ctx, nil
	case *UserChatEdited:
		return &UserChatEditedBatch{
			UserId: typed.UserId,
			UserChatEditeds: []UserChatEdited{
				*typed,
			},
			FirstElementContext: p.ctx,
		}, p.ctx, nil
	default:
		return &SingleEventBatch{
			*p,
		}, p.ctx, nil
	}
}

type BatchEvent interface {
	TryAppend(event EventHolder) bool
	GetBatchType() string
	GetContext() context.Context
	GetOrder() int
}
type SingleEventBatch struct {
	EventHolder
}

func (p *SingleEventBatch) TryAppend(event EventHolder) bool {
	return false
}
func (p *SingleEventBatch) GetBatchType() string {
	return p.EventHolder.metadata.EventType
}
func (p *SingleEventBatch) GetContext() context.Context {
	return p.ctx
}
func (p *SingleEventBatch) GetOrder() int {
	return defaultOrder
}

type batchCommonPart struct {
	// Closed implies that we cannot add any event to the batch
	closedForAppendingNew bool
}

type MessageCreatedEventBatch struct {
	batchCommonPart

	ChatId              int64
	FirstElementContext context.Context
	MessageCreateds     []MessageCreated
}

type ChatCreatedEventBatch struct {
	batchCommonPart

	FirstElementContext context.Context
	ChatCreateds        []ChatCreated
}

type ParticipantsAddedEventBatch struct {
	batchCommonPart

	FirstElementContext context.Context
	ParticipantsAddeds  []ParticipantsAdded
}

type UserChatCreatedBatch struct {
	batchCommonPart

	UserId int64

	FirstElementContext context.Context
	UserChatAddeds      []UserChatParticipantAdded
}

type UserChatEditedBatch struct {
	batchCommonPart

	UserId int64

	FirstElementContext context.Context
	UserChatEditeds     []UserChatEdited
}

func (p *MessageCreatedEventBatch) TryAppend(event EventHolder) bool {
	if p.closedForAppendingNew {
		return false
	}

	switch typed := event.event.(type) {
	case *MessageCreated:
		if typed.MessageCommoned.ChatId != p.ChatId {
			return false
		}
		p.MessageCreateds = append(p.MessageCreateds, *typed)

		return true
	// those events make gotten authorization (canWriteMessage) invalid
	case *ChatEdited:
		p.closedForAppendingNew = true
		return false
	case *ParticipantDeleted:
		p.closedForAppendingNew = true
		return false
	case *ParticipantChanged:
		p.closedForAppendingNew = true
		return false
	}

	return false
}
func (p *MessageCreatedEventBatch) GetBatchType() string {
	return BatchMessagesCreated
}
func (p *MessageCreatedEventBatch) GetContext() context.Context {
	return p.FirstElementContext
}
func (p *MessageCreatedEventBatch) GetOrder() int {
	return 300
}

func (p *ChatCreatedEventBatch) TryAppend(event EventHolder) bool {
	if p.closedForAppendingNew {
		return false
	}

	switch typed := event.event.(type) {
	case *ChatCreated:
		p.ChatCreateds = append(p.ChatCreateds, *typed)

		return true
	}

	return false
}
func (p *ChatCreatedEventBatch) GetBatchType() string {
	return BatchChatsCreated
}
func (p *ChatCreatedEventBatch) GetContext() context.Context {
	return p.FirstElementContext
}
func (p *ChatCreatedEventBatch) GetOrder() int {
	return 100
}

func (p *ParticipantsAddedEventBatch) TryAppend(event EventHolder) bool {
	if p.closedForAppendingNew {
		return false
	}

	switch typed := event.event.(type) {
	case *ParticipantsAdded:
		p.ParticipantsAddeds = append(p.ParticipantsAddeds, *typed)

		return true
	}

	return false
}
func (p *ParticipantsAddedEventBatch) GetBatchType() string {
	return BatchParticipantsAdded
}
func (p *ParticipantsAddedEventBatch) GetContext() context.Context {
	return p.FirstElementContext
}
func (p *ParticipantsAddedEventBatch) GetOrder() int {
	return 200
}

func (p *UserChatCreatedBatch) TryAppend(event EventHolder) bool {
	if p.closedForAppendingNew {
		return false
	}

	switch typed := event.event.(type) {
	case *UserChatParticipantAdded:
		if typed.UserId != p.UserId {
			return false
		}

		p.UserChatAddeds = append(p.UserChatAddeds, *typed)

		return true
		// we don't need p.closedForAppendingNew = true because there is no authorization because this is a secondary topic which just does commands, w/o authorization logic
	}

	return false
}
func (p *UserChatCreatedBatch) GetBatchType() string {
	return BatchEventUserChatAdded
}
func (p *UserChatCreatedBatch) GetContext() context.Context {
	return p.FirstElementContext
}
func (p *UserChatCreatedBatch) GetOrder() int {
	return 100 // the different topic though
}

func (p *UserChatEditedBatch) TryAppend(event EventHolder) bool {
	if p.closedForAppendingNew {
		return false
	}

	switch typed := event.event.(type) {
	case *UserChatEdited:
		if typed.UserId != p.UserId {
			return false
		}

		merged := false
		for i := range p.UserChatEditeds {
			if p.UserChatEditeds[i].ChatId == typed.ChatId && p.UserChatEditeds[i].ChatAction == typed.ChatAction {

				p.UserChatEditeds[i].EventTime = typed.EventTime
				p.UserChatEditeds[i].CorrelationId = typed.CorrelationId

				merged = true
			}
		}

		if !merged {
			p.UserChatEditeds = append(p.UserChatEditeds, *typed)
		}

		return true
		// we don't need p.closedForAppendingNew = true because there is no authorization because this is a secondary topic which just does commands, w/o authorization logic
	}

	return false
}
func (p *UserChatEditedBatch) GetBatchType() string {
	return BatchEventUserChatEdited
}
func (p *UserChatEditedBatch) GetContext() context.Context {
	return p.FirstElementContext
}
func (p *UserChatEditedBatch) GetOrder() int {
	return defaultOrder + 100 // the different topic though, but should be bigger, because UserMessagesCreated insude SingleEventBatch has the defaultOrder
}
