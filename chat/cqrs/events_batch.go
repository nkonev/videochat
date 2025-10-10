package cqrs

import "context"

const (
	BatchMessagesCreated = "batchMessagesCreated"
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
}
type SingleEventBatch struct {
	EventHolder
}

func (p *SingleEventBatch) TryAppend(event EventHolder) bool {
	return false
}
func (p *SingleEventBatch) GetBatchType() string {
	return p.EventHolder.event.GetMetadata().EventType
}
func (p *SingleEventBatch) GetContext() context.Context {
	return p.ctx
}

type MessageCreatedEventBatch struct {
	ChatId              int64
	FirstElementContext context.Context
	MessageCreateds     []MessageCreated

	// Closed implies that we cannot add any event to the batch
	closedForAppendingNew bool
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
