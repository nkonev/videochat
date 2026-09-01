package cqrs

import "context"

const (
	BatchMessagesCreated = "batchMessagesCreated"
	BatchChatsCreated    = "batchChatsCreated"
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
	return 10000
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
	// those events make gotten authorization (canReadMessage) invalid
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
	return 200
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
