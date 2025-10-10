package cqrs

import (
	"context"
	"fmt"
	"nkonev.name/chat/config"
	"nkonev.name/chat/logger"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasic(t *testing.T) {
	cfg, err := config.CreateTestTypedConfig()
	if err != nil {
		panic(err)
	}
	lgr := logger.NewLogger(os.Stdout, cfg)
	defer lgr.CloseLogger()

	bo := NewBatchOptimizer(lgr)

	const chatId int64 = 1
	const userId1 int64 = 1
	const userId2 int64 = 2
	const messageId1 int64 = 1
	const messageId2 int64 = 2

	messageCreated1 := mockMessageCreated(messageId1, chatId, userId1)
	messageCreated2 := mockMessageCreated(messageId2, chatId, userId2)

	events := []EventHolder{
		{
			event: messageCreated1,
			ctx:   context.Background(),
		},
		{
			event: messageCreated2,
			ctx:   context.Background(),
		},
	}

	batchEvents, _, err := bo.Optimize(events)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(batchEvents))
	messageCreateBatch := batchEvents[0].(*MessageCreatedEventBatch)
	require.Equal(t, chatId, messageCreateBatch.ChatId)

	require.Equal(t, 2, len(messageCreateBatch.MessageCreateds))
	require.Equal(t, *messageCreated1, messageCreateBatch.MessageCreateds[0])
	require.Equal(t, *messageCreated2, messageCreateBatch.MessageCreateds[1])
}

func mockMessageCreated(
	messageId int64,
	chatId int64,
	behalfUserId int64,
) *MessageCreated {
	return &MessageCreated{
		MessageCommoned: MessageCommoned{
			Id:      messageId,
			ChatId:  chatId,
			Content: fmt.Sprintf("Message id %d, chatId %d", messageId, chatId),
		},
		AdditionalData: GenerateMessageAdditionalData(nil, behalfUserId),
	}
}
