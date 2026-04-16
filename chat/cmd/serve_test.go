package cmd

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"nkonev.name/chat/client"
	"nkonev.name/chat/config"
	"nkonev.name/chat/cqrs"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/kafka"
	"nkonev.name/chat/listener"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/producer"
	"nkonev.name/chat/tasks"
	"nkonev.name/chat/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kadm"
	"go.uber.org/fx"
)

func TestUnreads(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		dba *db.DB,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user3 int64 = 3
		const user1Login = "admin1"
		const user2Login = "admin2"
		const user3Login = "admin3"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser3 := dto.User{
			Id:               user3,
			Login:            user3Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		const chat1Name = "new chat 1"

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2, &mockUser3}, nil)

		ctx := context.Background()

		avatar := "http://example.com/avatar.jpg"
		avatarBig := "http://example.com/avatar-big.jpg"

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name, client.NewChatOptionAvatar(&avatar, &avatarBig))
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		waitForChatExists(lgr, m, dba, chat1Id, user1, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		const message1Text = "new message 1"

		message1Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message1Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		user1Chats, _, err := testRestClient.GetChats(ctx, user1)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user1Chats))
		chat1OfUser1 := user1Chats[0]
		assert.Equal(t, chat1Name, chat1OfUser1.Title)
		assert.Equal(t, int64(0), chat1OfUser1.UnreadMessages)
		assert.Equal(t, avatar, *chat1OfUser1.Avatar)
		assert.Equal(t, avatarBig, *chat1OfUser1.AvatarBig)

		user1HasUnreadMessages, err := testRestClient.GetHasUnreadMessages(ctx, user1)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, false, user1HasUnreadMessages)

		user2Chats, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 0, len(user2Chats))

		user2HasUnreadMessages, err := testRestClient.GetHasUnreadMessages(ctx, user2)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, false, user2HasUnreadMessages)

		user3Chats, _, err := testRestClient.GetChats(ctx, user3)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 0, len(user3Chats))

		chat1Messages, _, err := testRestClient.GetMessages(ctx, user1, chat1Id)
		require.NoError(t, err, "error in getting messages")
		assert.Equal(t, 1, len(chat1Messages))
		message1 := chat1Messages[0]
		assert.Equal(t, message1Id, message1.Id)
		assert.Equal(t, message1Text, message1.Content)

		testOutputEventsAccumulator.Clean()

		// 2 separate calls to guarantee order
		err = testRestClient.AddChatParticipants(ctx, user1, chat1Id, []int64{user2})
		require.NoError(t, err, "error in adding participants")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatCreated &&
					e.UserId == user2 &&
					e.ChatNotification.ChatViewDto.Id == chat1Id &&
					e.ChatNotification.ChatViewDto.Title == chat1Name &&
					len(e.ChatNotification.Participants) == 2 &&
					e.ChatNotification.Participants[0].Id == user2 &&
					e.ChatNotification.Participants[0].Login == user2Login &&
					e.ChatNotification.Participants[1].Id == user1 &&
					e.ChatNotification.Participants[1].Login == user1Login &&
					e.ChatNotification.UnreadMessages == 1
			},

			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeHasUnreadMessagesChanged &&
					e.UserId == user2 &&
					e.HasUnreadMessagesChanged.HasUnreadMessages == true
			},
		}))

		err = testRestClient.AddChatParticipants(ctx, user1, chat1Id, []int64{user3})
		require.NoError(t, err, "error in adding participants")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		chat1Participants, _, err := testRestClient.GetChatParticipants(ctx, user1, chat1Id)
		require.NoError(t, err, "error in chat participants")
		require.Equal(t, 3, len(chat1Participants))
		assert.Equal(t, user3, chat1Participants[0].Id)
		assert.Equal(t, user3Login, chat1Participants[0].Login)
		assert.Equal(t, user2, chat1Participants[1].Id)
		assert.Equal(t, user2Login, chat1Participants[1].Login)
		assert.Equal(t, user1, chat1Participants[2].Id)
		assert.Equal(t, user1Login, chat1Participants[2].Login)

		user2ChatsNew, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user2ChatsNew))
		chat1OfUser2 := user2ChatsNew[0]
		assert.Equal(t, chat1Name, chat1OfUser2.Title)
		assert.Equal(t, int64(1), chat1OfUser2.UnreadMessages)

		user2HasUnreadMessagesNew, err := testRestClient.GetHasUnreadMessages(ctx, user2)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, true, user2HasUnreadMessagesNew)

		user3ChatsNew, _, err := testRestClient.GetChats(ctx, user3)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user3ChatsNew))
		chat1OfUser3 := user3ChatsNew[0]
		assert.Equal(t, chat1Name, chat1OfUser3.Title)
		assert.Equal(t, int64(1), chat1OfUser3.UnreadMessages)

		user3HasUnreadMessagesNew, err := testRestClient.GetHasUnreadMessages(ctx, user3)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, true, user3HasUnreadMessagesNew)

		testOutputEventsAccumulator.Clean()

		err = testRestClient.ReadMessage(ctx, user2, chat1Id, message1.Id)
		require.NoError(t, err, "error in reading message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatUnreadMessagesChanged &&
					e.UserId == user2 &&
					e.UnreadMessagesNotification.ChatId == chat1Id &&
					e.UnreadMessagesNotification.UnreadMessages == 0
			},

			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeHasUnreadMessagesChanged &&
					e.UserId == user2 &&
					e.HasUnreadMessagesChanged.HasUnreadMessages == false
			},
		}))

		user2ChatsNew2, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user2ChatsNew2))
		chat1OfUser22 := user2ChatsNew2[0]
		assert.Equal(t, int64(0), chat1OfUser22.UnreadMessages)

		user2HasUnreadMessagesNew2, err := testRestClient.GetHasUnreadMessages(ctx, user2)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, false, user2HasUnreadMessagesNew2)

		user1WhoReadedNew2, err := testRestClient.GetReadMessageUsers(ctx, user1, chat1Id, message1Id)
		require.NoError(t, err, "error in getting who read the message")
		assert.Equal(t, user2, user1WhoReadedNew2.Data[0].Id)
		assert.Equal(t, user1, user1WhoReadedNew2.Data[1].Id)

		user3ChatsNew2, _, err := testRestClient.GetChats(ctx, user3)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user3ChatsNew2))
		chat1OfUser32 := user3ChatsNew2[0]
		assert.Equal(t, int64(1), chat1OfUser32.UnreadMessages)

		user3HasUnreadMessagesNew2, err := testRestClient.GetHasUnreadMessages(ctx, user3)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, true, user3HasUnreadMessagesNew2)

		testOutputEventsAccumulator.Clean()

		const message2Text = "new message 2"
		messageId2, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message2Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatEdited &&
					e.UserId == user2 &&
					e.ChatNotification.ChatViewDto.Id == chat1Id &&
					e.ChatNotification.ChatViewDto.Title == chat1Name &&
					len(e.ChatNotification.Participants) == 3 && // in case race condition it's going to fail
					e.ChatNotification.Participants[0].Id == user3 &&
					e.ChatNotification.Participants[0].Login == user3Login &&
					e.ChatNotification.Participants[1].Id == user2 &&
					e.ChatNotification.Participants[1].Login == user2Login &&
					e.ChatNotification.Participants[2].Id == user1 &&
					e.ChatNotification.Participants[2].Login == user1Login &&
					e.ChatNotification.UnreadMessages == 1
			},

			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeHasUnreadMessagesChanged &&
					e.UserId == user2 &&
					e.HasUnreadMessagesChanged.HasUnreadMessages == true
			},
		}))

		const message3Text = "new message 3"
		messageId3, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message3Text)
		require.NoError(t, err, "error in creating message")
		assert.True(t, messageId3 > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		user2ChatsNew3, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user2ChatsNew3))
		chat1OfUser23 := user2ChatsNew3[0]
		assert.Equal(t, int64(2), chat1OfUser23.UnreadMessages)

		user2HasUnreadMessagesNew3, err := testRestClient.GetHasUnreadMessages(ctx, user2)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, true, user2HasUnreadMessagesNew3)

		user3ChatsNew3, _, err := testRestClient.GetChats(ctx, user3)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user3ChatsNew3))
		chat1OfUser33 := user3ChatsNew3[0]
		assert.Equal(t, int64(3), chat1OfUser33.UnreadMessages)

		user3HasUnreadMessagesNew3, err := testRestClient.GetHasUnreadMessages(ctx, user3)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, true, user3HasUnreadMessagesNew3)

		testOutputEventsAccumulator.Clean()

		err = testRestClient.DeleteMessage(ctx, user1, chat1Id, messageId3)
		require.NoError(t, err, "error in delete message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeMessageDeleted &&
					e.UserId == user1 &&
					e.ChatId == chat1Id &&
					e.MessageDeletedNotification.Id == messageId3 &&
					e.MessageDeletedNotification.ChatId == chat1Id
			},
			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeMessageDeleted &&
					e.UserId == user2 &&
					e.ChatId == chat1Id &&
					e.MessageDeletedNotification.Id == messageId3 &&
					e.MessageDeletedNotification.ChatId == chat1Id
			},
			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeMessageDeleted &&
					e.UserId == user3 &&
					e.ChatId == chat1Id &&
					e.MessageDeletedNotification.Id == messageId3 &&
					e.MessageDeletedNotification.ChatId == chat1Id
			},
		}))

		user2ChatsNew4, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user2ChatsNew4))
		chat1OfUser24 := user2ChatsNew4[0]
		assert.Equal(t, int64(1), chat1OfUser24.UnreadMessages)

		user2HasUnreadMessagesNew4, err := testRestClient.GetHasUnreadMessages(ctx, user2)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, true, user2HasUnreadMessagesNew4)

		user3ChatsNew4, _, err := testRestClient.GetChats(ctx, user3)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user3ChatsNew4))
		chat1OfUser34 := user3ChatsNew4[0]
		assert.Equal(t, int64(2), chat1OfUser34.UnreadMessages)

		user3HasUnreadMessagesNew4, err := testRestClient.GetHasUnreadMessages(ctx, user3)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, true, user3HasUnreadMessagesNew4)

		err = testRestClient.PutUserChatNotificationSettings(ctx, user2, chat1Id, false)
		require.NoError(t, err, "error in setting contribute into has new messages")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		user2ChatsNew40, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user2ChatsNew40))
		chat1OfUser240 := user2ChatsNew40[0]
		assert.Equal(t, int64(1), chat1OfUser240.UnreadMessages)
		assert.Equal(t, false, chat1OfUser240.ConsiderMessagesAsUnread)

		// this message should not contribute into user 2's new messages because user 2 disabled them for chat 1
		messageId4, err := testRestClient.CreateMessage(ctx, user1, chat1Id, "msg 4")
		require.NoError(t, err, "error in creating message")
		assert.True(t, messageId4 > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		user2ChatsNew41, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user2ChatsNew41))
		chat1OfUser241 := user2ChatsNew41[0]
		assert.Equal(t, int64(2), chat1OfUser241.UnreadMessages)
		assert.Equal(t, messageId4, *chat1OfUser241.LastMessageId)
		assert.Equal(t, false, chat1OfUser241.ConsiderMessagesAsUnread)

		user2HasUnreadMessagesNew41, err := testRestClient.GetHasUnreadMessages(ctx, user2)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, false, user2HasUnreadMessagesNew41)

		// assert that one more message won't erase existing status
		user3HasUnreadMessagesNew41, err := testRestClient.GetHasUnreadMessages(ctx, user3)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, true, user3HasUnreadMessagesNew41)

		err = testRestClient.PutUserChatNotificationSettings(ctx, user2, chat1Id, true) // restore
		require.NoError(t, err, "error in setting contribute into has new messages")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		user2ChatsNew42, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user2ChatsNew42))
		chat1OfUser242 := user2ChatsNew42[0]
		assert.Equal(t, int64(2), chat1OfUser242.UnreadMessages)
		assert.Equal(t, messageId4, *chat1OfUser242.LastMessageId)
		assert.Equal(t, true, chat1OfUser242.ConsiderMessagesAsUnread)

		user2HasUnreadMessagesNew42, err := testRestClient.GetHasUnreadMessages(ctx, user2)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, true, user2HasUnreadMessagesNew42)

		testOutputEventsAccumulator.Clean()

		err = testRestClient.DeleteMessage(ctx, user1, chat1Id, messageId4)
		require.NoError(t, err, "error in delete message")
		err = testRestClient.DeleteMessage(ctx, user1, chat1Id, messageId2)
		require.NoError(t, err, "error in delete message")
		err = testRestClient.DeleteMessage(ctx, user1, chat1Id, message1Id)
		require.NoError(t, err, "error in delete message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeHasUnreadMessagesChanged &&
					e.UserId == user1 &&
					e.HasUnreadMessagesChanged.HasUnreadMessages == false
			},
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeHasUnreadMessagesChanged &&
					e.UserId == user2 &&
					e.HasUnreadMessagesChanged.HasUnreadMessages == false
			},
		}))

		user1HasUnreadMessagesNew5, err := testRestClient.GetHasUnreadMessages(ctx, user1)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, false, user1HasUnreadMessagesNew5)

		user2HasUnreadMessagesNew5, err := testRestClient.GetHasUnreadMessages(ctx, user2)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, false, user2HasUnreadMessagesNew5)

		user2ChatsNew50, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user2ChatsNew50))
		chat1OfUser250 := user2ChatsNew50[0]
		assert.Equal(t, int64(0), chat1OfUser250.UnreadMessages)
		assert.Nil(t, chat1OfUser250.LastMessageId)
		assert.Equal(t, true, chat1OfUser250.ConsiderMessagesAsUnread)

		user3HasUnreadMessagesNew5, err := testRestClient.GetHasUnreadMessages(ctx, user3)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, false, user3HasUnreadMessagesNew5)

		messageId5, err := testRestClient.CreateMessage(ctx, user1, chat1Id, "msg 5")
		require.NoError(t, err, "error in creating message")
		assert.True(t, messageId5 > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		user2ChatsNew52, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user2ChatsNew52))
		chat1OfUser252 := user2ChatsNew52[0]
		assert.Equal(t, int64(1), chat1OfUser252.UnreadMessages)
		assert.Equal(t, messageId5, *chat1OfUser252.LastMessageId)
		assert.Equal(t, true, chat1OfUser252.ConsiderMessagesAsUnread)

		user2HasUnreadMessagesNew52, err := testRestClient.GetHasUnreadMessages(ctx, user2)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, true, user2HasUnreadMessagesNew52)

		err = testRestClient.ReadMessage(ctx, user2, chat1Id, messageId5)
		require.NoError(t, err, "error in reading message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		user2ChatsNew53, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user2ChatsNew53))
		chat1OfUser253 := user2ChatsNew53[0]
		assert.Equal(t, int64(0), chat1OfUser253.UnreadMessages)
		assert.Equal(t, messageId5, *chat1OfUser253.LastMessageId)
		assert.Equal(t, true, chat1OfUser253.ConsiderMessagesAsUnread)

		user2HasUnreadMessagesNew53, err := testRestClient.GetHasUnreadMessages(ctx, user2)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, false, user2HasUnreadMessagesNew53)

		err = testRestClient.DeleteMessage(ctx, user1, chat1Id, messageId5)
		require.NoError(t, err, "error in delete message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		user2ChatsNew54, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user2ChatsNew54))
		chat1OfUser254 := user2ChatsNew54[0]
		assert.Equal(t, int64(0), chat1OfUser254.UnreadMessages)
		assert.Nil(t, chat1OfUser254.LastMessageId)
		assert.Equal(t, true, chat1OfUser254.ConsiderMessagesAsUnread)

		user2HasUnreadMessagesNew54, err := testRestClient.GetHasUnreadMessages(ctx, user2)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, false, user2HasUnreadMessagesNew54)

		// now test that counter will count good after deletion
		messageId6, err := testRestClient.CreateMessage(ctx, user1, chat1Id, "msg 6")
		require.NoError(t, err, "error in creating message")
		assert.True(t, messageId6 > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		user2ChatsNew61, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user2ChatsNew61))
		chat1OfUser261 := user2ChatsNew61[0]
		assert.Equal(t, int64(1), chat1OfUser261.UnreadMessages)
		assert.Equal(t, messageId6, *chat1OfUser261.LastMessageId)
		assert.Equal(t, true, chat1OfUser261.ConsiderMessagesAsUnread)

		user2HasUnreadMessagesNew61, err := testRestClient.GetHasUnreadMessages(ctx, user2)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, true, user2HasUnreadMessagesNew61)
	})
}

func TestUnreadsInitFromEmptyChatOfBothUsers(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		const chat1Name = "new chat 1"

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name, client.NewChatOptionParticipants(user2))
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		const message1Text = "new message 1"

		_, err = testRestClient.CreateMessage(ctx, user1, chat1Id, message1Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		user1HasUnreadMessages1, err := testRestClient.GetHasUnreadMessages(ctx, user1)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, false, user1HasUnreadMessages1)

		user2HasUnreadMessages1, err := testRestClient.GetHasUnreadMessages(ctx, user2)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, true, user2HasUnreadMessages1)

		const message2Text = "new message 2"

		_, err = testRestClient.CreateMessage(ctx, user2, chat1Id, message2Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		user1HasUnreadMessages2, err := testRestClient.GetHasUnreadMessages(ctx, user1)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, true, user1HasUnreadMessages2)

		const message3Text = "new message 3"

		_, err = testRestClient.CreateMessage(ctx, user1, chat1Id, message3Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		user1HasUnreadMessages3, err := testRestClient.GetHasUnreadMessages(ctx, user1)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, false, user1HasUnreadMessages3)
	})
}

func TestReadAllChats(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		const chat1Name = "new chat 1"
		const chat2Name = "new chat 2"

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name)
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		const message11Text = "new message 11"
		const message12Text = "new message 12"

		message1Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message11Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")
		assert.True(t, message1Id > 0)

		chat2Id, err := testRestClient.CreateChat(ctx, user1, chat2Name)
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat2Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		message2Id, err := testRestClient.CreateMessage(ctx, user1, chat2Id, message12Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")
		assert.True(t, message2Id > 0)

		err = testRestClient.AddChatParticipants(ctx, user1, chat1Id, []int64{user2})
		require.NoError(t, err, "error in adding participants")
		err = testRestClient.AddChatParticipants(ctx, user1, chat2Id, []int64{user2})
		require.NoError(t, err, "error in adding participants")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		user2Chats, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 2, len(user2Chats))
		chat1OfUser2 := user2Chats[0]
		chat2OfUser2 := user2Chats[1]
		assert.Equal(t, int64(1), chat1OfUser2.UnreadMessages)
		assert.Equal(t, int64(1), chat2OfUser2.UnreadMessages)

		user2HasUnreadMessages, err := testRestClient.GetHasUnreadMessages(ctx, user2)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, true, user2HasUnreadMessages)

		err = testRestClient.MarkAllChatsAsRead(ctx, user2)
		require.NoError(t, err, "error in read all messages")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatUnreadMessagesChanged &&
					e.UserId == user2 &&
					e.UnreadMessagesNotification.ChatId == chat1Id &&
					e.UnreadMessagesNotification.UnreadMessages == 0
			},
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatUnreadMessagesChanged &&
					e.UserId == user2 &&
					e.UnreadMessagesNotification.ChatId == chat2Id &&
					e.UnreadMessagesNotification.UnreadMessages == 0
			},

			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeHasUnreadMessagesChanged &&
					e.UserId == user2 &&
					e.HasUnreadMessagesChanged.HasUnreadMessages == false
			},
		}))

		user2ChatsNew, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 2, len(user2ChatsNew))
		chat1OfUser2New := user2ChatsNew[0]
		chat2OfUser2New := user2ChatsNew[1]
		assert.Equal(t, int64(0), chat1OfUser2New.UnreadMessages)
		assert.Equal(t, int64(0), chat2OfUser2New.UnreadMessages)

		user2HasUnreadMessagesNew, err := testRestClient.GetHasUnreadMessages(ctx, user2)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, false, user2HasUnreadMessagesNew)
	})
}

func TestReadOneChat(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		const chat1Name = "new chat 1"
		const chat2Name = "new chat 2"

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name)
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		const message11Text = "new message 11"
		const message12Text = "new message 12"

		message1Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message11Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")
		assert.True(t, message1Id > 0)

		chat2Id, err := testRestClient.CreateChat(ctx, user1, chat2Name)
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat2Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		message2Id, err := testRestClient.CreateMessage(ctx, user1, chat2Id, message12Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")
		assert.True(t, message2Id > 0)

		err = testRestClient.AddChatParticipants(ctx, user1, chat1Id, []int64{user2})
		require.NoError(t, err, "error in adding participants")
		err = testRestClient.AddChatParticipants(ctx, user1, chat2Id, []int64{user2})
		require.NoError(t, err, "error in adding participants")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		user2Chats, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 2, len(user2Chats))
		chat1OfUser2 := user2Chats[0]
		chat2OfUser2 := user2Chats[1]
		assert.Equal(t, int64(1), chat1OfUser2.UnreadMessages)
		assert.Equal(t, int64(1), chat2OfUser2.UnreadMessages)

		user2HasUnreadMessages, err := testRestClient.GetHasUnreadMessages(ctx, user2)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, true, user2HasUnreadMessages)

		err = testRestClient.MarkChatAsRead(ctx, user2, chat1Id)
		require.NoError(t, err, "error in read all messages")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatUnreadMessagesChanged &&
					e.UserId == user2 &&
					e.UnreadMessagesNotification.ChatId == chat1Id &&
					e.UnreadMessagesNotification.UnreadMessages == 0
			},

			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeHasUnreadMessagesChanged &&
					e.UserId == user2 &&
					e.HasUnreadMessagesChanged.HasUnreadMessages == true
			},
		}))

		user2ChatsNew, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 2, len(user2ChatsNew))
		chat1OfUser2New := user2ChatsNew[0]
		chat2OfUser2New := user2ChatsNew[1]
		assert.Equal(t, chat1Id, chat2OfUser2New.Id)
		assert.Equal(t, int64(0), chat2OfUser2New.UnreadMessages)
		assert.Equal(t, int64(1), chat1OfUser2New.UnreadMessages)

		user2HasUnreadMessagesNew, err := testRestClient.GetHasUnreadMessages(ctx, user2)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, true, user2HasUnreadMessagesNew)

		err = testRestClient.MarkChatAsRead(ctx, user2, chat2Id)
		require.NoError(t, err, "error in read all messages")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatUnreadMessagesChanged &&
					e.UserId == user2 &&
					e.UnreadMessagesNotification.ChatId == chat2Id &&
					e.UnreadMessagesNotification.UnreadMessages == 0
			},

			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeHasUnreadMessagesChanged &&
					e.UserId == user2 &&
					e.HasUnreadMessagesChanged.HasUnreadMessages == false
			},
		}))

		user2ChatsNew2, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 2, len(user2ChatsNew2))
		chat1OfUser2New2 := user2ChatsNew2[0]
		chat2OfUser2New2 := user2ChatsNew2[1]
		assert.Equal(t, int64(0), chat2OfUser2New2.UnreadMessages)
		assert.Equal(t, int64(0), chat1OfUser2New2.UnreadMessages)

		user2HasUnreadMessagesNew2, err := testRestClient.GetHasUnreadMessages(ctx, user2)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, false, user2HasUnreadMessagesNew2)
	})
}

func TestReaction(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		const chat1Name = "new chat 1"

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name)
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		err = testRestClient.AddChatParticipants(ctx, user1, chat1Id, []int64{user2})
		require.NoError(t, err, "error in adding participants")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		const message11Text = "new message 11"

		message1Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message11Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")
		assert.True(t, message1Id > 0)

		const reaction = "😀"

		// both users add the reaction
		err = testRestClient.Reaction(ctx, user1, chat1Id, message1Id, reaction)
		require.NoError(t, err, "error in reacting on message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeReactionChanged &&
					e.UserId == user1 &&
					e.ReactionChangedEvent.MessageId == message1Id &&
					len(e.ReactionChangedEvent.Reaction.Users) == 1 &&
					e.ReactionChangedEvent.Reaction.Users[0].Id == user1 &&
					e.ReactionChangedEvent.Reaction.Users[0].Login == user1Login &&
					e.ReactionChangedEvent.Reaction.Count == 1 &&
					e.ReactionChangedEvent.Reaction.Reaction == reaction
			},

			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeReactionChanged &&
					e.UserId == user2 &&
					e.ReactionChangedEvent.MessageId == message1Id &&
					len(e.ReactionChangedEvent.Reaction.Users) == 1 &&
					e.ReactionChangedEvent.Reaction.Users[0].Id == user1 &&
					e.ReactionChangedEvent.Reaction.Users[0].Login == user1Login &&
					e.ReactionChangedEvent.Reaction.Count == 1 &&
					e.ReactionChangedEvent.Reaction.Reaction == reaction
			},
		}))

		testOutputEventsAccumulator.Clean()

		err = testRestClient.Reaction(ctx, user2, chat1Id, message1Id, reaction)
		require.NoError(t, err, "error in reacting on message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeReactionChanged &&
					e.UserId == user1 &&
					e.ReactionChangedEvent.MessageId == message1Id &&
					len(e.ReactionChangedEvent.Reaction.Users) == 2 &&
					e.ReactionChangedEvent.Reaction.Users[0].Id == user1 &&
					e.ReactionChangedEvent.Reaction.Users[0].Login == user1Login &&
					e.ReactionChangedEvent.Reaction.Users[1].Id == user2 &&
					e.ReactionChangedEvent.Reaction.Users[1].Login == user2Login &&
					e.ReactionChangedEvent.Reaction.Count == 2 &&
					e.ReactionChangedEvent.Reaction.Reaction == reaction
			},

			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeReactionChanged &&
					e.UserId == user2 &&
					e.ReactionChangedEvent.MessageId == message1Id &&
					len(e.ReactionChangedEvent.Reaction.Users) == 2 &&
					e.ReactionChangedEvent.Reaction.Users[0].Id == user1 &&
					e.ReactionChangedEvent.Reaction.Users[0].Login == user1Login &&
					e.ReactionChangedEvent.Reaction.Users[1].Id == user2 &&
					e.ReactionChangedEvent.Reaction.Users[1].Login == user2Login &&
					e.ReactionChangedEvent.Reaction.Count == 2 &&
					e.ReactionChangedEvent.Reaction.Reaction == reaction
			},
		}))

		chat1Messages, _, err := testRestClient.GetMessages(ctx, user1, chat1Id)
		require.NoError(t, err, "error in getting messages")
		require.Equal(t, 1, len(chat1Messages))
		message := chat1Messages[0]
		assert.Equal(t, message11Text, message.Content)
		assert.Equal(t, 1, len(message.Reactions))
		assert.Equal(t, int64(2), message.Reactions[0].Count)
		assert.Equal(t, reaction, message.Reactions[0].Reaction)
		assert.Equal(t, 2, len(message.Reactions[0].Users))
		assert.Equal(t, user1, message.Reactions[0].Users[0].Id)
		assert.Equal(t, user2, message.Reactions[0].Users[1].Id)

		// user 2 flips - decreases reaction's count
		err = testRestClient.Reaction(ctx, user2, chat1Id, message1Id, reaction)
		require.NoError(t, err, "error in reacting on message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		chat1MessagesNew, _, err := testRestClient.GetMessages(ctx, user1, chat1Id)
		require.NoError(t, err, "error in getting messages")
		require.Equal(t, 1, len(chat1MessagesNew))
		messageNew := chat1MessagesNew[0]
		assert.Equal(t, message11Text, messageNew.Content)
		assert.Equal(t, 1, len(messageNew.Reactions))
		assert.Equal(t, int64(1), messageNew.Reactions[0].Count)
		assert.Equal(t, reaction, messageNew.Reactions[0].Reaction)
		assert.Equal(t, 1, len(messageNew.Reactions[0].Users))
		assert.Equal(t, user1, messageNew.Reactions[0].Users[0].Id)

		testOutputEventsAccumulator.Clean()

		// user 1 flips - removes the reaction
		err = testRestClient.Reaction(ctx, user1, chat1Id, message1Id, reaction)
		require.NoError(t, err, "error in reacting on message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeReactionRemoved &&
					e.UserId == user1 &&
					e.ReactionChangedEvent.MessageId == message1Id &&
					e.ReactionChangedEvent.Reaction.Count == 0 &&
					e.ReactionChangedEvent.Reaction.Reaction == reaction
			},

			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeReactionRemoved &&
					e.UserId == user2 &&
					e.ReactionChangedEvent.MessageId == message1Id &&
					e.ReactionChangedEvent.Reaction.Count == 0 &&
					e.ReactionChangedEvent.Reaction.Reaction == reaction
			},
		}))
	})
}

func TestCreateTetATetChat(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"
		var user2Short = "admin2 short info"

		avatar2 := "http://example.com/avatar-admin2.jpg"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           &avatar2,
			ShortInfo:        &user2Short,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)
		mockAaaClient.EXPECT().SearchGetUsers(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]*dto.User{&mockUser2}, 1, nil)
		mockAaaClient.EXPECT().GetOnlines(mock.Anything, mock.Anything).Return([]*dto.UserOnline{{
			Id:     user1,
			Online: true,
		}, {
			Id:     user2,
			Online: true,
		}}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateTetATetChat(ctx, user1, user2)
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		user1Chats, _, err := testRestClient.GetChats(ctx, user1)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user1Chats))
		chat1OfUser1 := user1Chats[0]
		assert.Equal(t, user2Login, chat1OfUser1.Title)
		assert.Equal(t, avatar2, *chat1OfUser1.Avatar)
		assert.Nil(t, chat1OfUser1.LastSeenDateTime)
		assert.Equal(t, user2Short, *chat1OfUser1.ShortInfo)

		assert.Equal(t, []int64{2, 1}, chat1OfUser1.ParticipantIds)
		assert.Equal(t, user2Login, chat1OfUser1.Participants[0].Login)
		assert.Equal(t, user1Login, chat1OfUser1.Participants[1].Login)

		searchString := user2Login
		resp2Search, _, err := testRestClient.GetChats(ctx, user1, client.NewChatGetOptionWithSearch(searchString))
		require.NoError(t, err)
		require.Equal(t, 1, len(resp2Search))
		chat1OfUser1New := resp2Search[0]
		assert.Equal(t, user2Login, chat1OfUser1New.Title)
		assert.Equal(t, avatar2, *chat1OfUser1New.Avatar)
	})
}

func TestResendMessage(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		const chat1Name = "new chat 1 src"
		const chat2Name = "new chat 1 dst"

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name, client.NewChatOptionResend(true))
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)

		chat2Id, err := testRestClient.CreateChat(ctx, user2, chat2Name)
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat2Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		// user 1 adds user 2 to chat 1
		err = testRestClient.AddChatParticipants(ctx, user1, chat1Id, []int64{user2})
		require.NoError(t, err, "error in adding participant")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		// assert user 2 sees both chats
		user2ChatsNew, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 2, len(user2ChatsNew))
		chat1OfUser2 := user2ChatsNew[0]
		chat2OfUser2 := user2ChatsNew[1]
		assert.Equal(t, chat2Name, chat2OfUser2.Title)
		assert.Equal(t, chat1Name, chat1OfUser2.Title)

		const message1Text = "message 1 from chat 1"

		// user 1 creates a message
		message1Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message1Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		// user 2 resends the message from chat 1 to chat 2
		message1ResentId, err := testRestClient.CreateMessage(ctx, user2, chat2Id, dto.NoMessageContent, client.NewMessageCreateOptionResend(chat1Id, message1Id))
		require.NoError(t, err, "error in resending message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		// assert that chat 2 contains the embed message
		chat2Messages, _, err := testRestClient.GetMessages(ctx, user2, chat2Id)
		require.NoError(t, err, "error in getting messages")
		assert.Equal(t, 1, len(chat2Messages))
		resentMessage1 := chat2Messages[0]
		assert.Equal(t, message1ResentId, resentMessage1.Id)
		assert.Equal(t, user2, resentMessage1.OwnerId)
		require.NotNil(t, resentMessage1.EmbedMessage)
		assert.Equal(t, dto.EmbedMessageTypeResend, resentMessage1.EmbedMessage.EmbedType)
		assert.Equal(t, message1Text, resentMessage1.EmbedMessage.Text)
		assert.Equal(t, message1Id, resentMessage1.EmbedMessage.Id)
		assert.Equal(t, chat1Id, *resentMessage1.EmbedMessage.ChatId)
		assert.Equal(t, chat1Name, *resentMessage1.EmbedMessage.ChatName)
		assert.Equal(t, user1, resentMessage1.EmbedMessage.Owner.Id)
		assert.Equal(t, user1Login, resentMessage1.EmbedMessage.Owner.Login)

		const message1TextNew = "message 1 from chat 1 new"

		// user 1 changes original message
		err = testRestClient.EditMessage(ctx, user1, chat1Id, message1Id, message1TextNew)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		testOutputEventsAccumulator.Clean()

		// user 2 synchronizes the resend message
		err = testRestClient.SyncMessage(ctx, user2, chat2Id, message1ResentId)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		// assert there is message edit with the new embed content
		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, true, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeMessageEdited &&
					e.UserId == user2 &&
					e.ChatId == chat2Id &&
					e.MessageNotification.Id == message1ResentId &&
					e.MessageNotification.Content == dto.NoMessageContent &&
					*e.MessageNotification.EmbedMessage.ChatId == chat1Id &&
					*e.MessageNotification.EmbedMessage.ChatName == chat1Name &&
					e.MessageNotification.EmbedMessage.Text == message1TextNew &&
					e.MessageNotification.EmbedMessage.Id == message1Id &&
					e.MessageNotification.EmbedMessage.EmbedType == dto.EmbedMessageTypeResend &&
					e.MessageNotification.EmbedMessage.Owner.Id == user1 &&
					e.MessageNotification.EmbedMessage.Owner.Login == user1Login &&
					e.MessageNotification.Owner.Id == user2 &&
					e.MessageNotification.Owner.Login == user2Login
			},

			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatEdited &&
					e.UserId == user2 &&
					e.ChatNotification.ChatViewDto.Id == chat2Id &&
					e.ChatNotification.ChatViewDto.Title == chat2Name &&
					*e.ChatNotification.ChatViewDto.LastMessageId == message1ResentId &&
					*e.ChatNotification.ChatViewDto.LastMessageOwnerId == user2 &&
					*e.ChatNotification.ChatViewDto.LastMessageContent == message1TextNew &&
					len(e.ChatNotification.Participants) == 1 &&
					e.ChatNotification.Participants[0].Id == user2 &&
					e.ChatNotification.Participants[0].Login == user2Login &&
					e.ChatNotification.UnreadMessages == 0
			},
		}))

		// assert that chat 2 contains the embed message
		chat2MessagesNew, _, err := testRestClient.GetMessages(ctx, user2, chat2Id)
		require.NoError(t, err, "error in getting messages")
		assert.Equal(t, 1, len(chat2MessagesNew))
		resentMessage1New := chat2MessagesNew[0]
		assert.Equal(t, message1ResentId, resentMessage1New.Id)
		assert.Equal(t, user2, resentMessage1New.OwnerId)
		require.NotNil(t, resentMessage1New.EmbedMessage)
		assert.Equal(t, dto.EmbedMessageTypeResend, resentMessage1New.EmbedMessage.EmbedType)
		assert.Equal(t, message1TextNew, resentMessage1New.EmbedMessage.Text)
		assert.Equal(t, message1Id, resentMessage1New.EmbedMessage.Id)
		assert.Equal(t, chat1Id, *resentMessage1New.EmbedMessage.ChatId)
		assert.Equal(t, chat1Name, *resentMessage1New.EmbedMessage.ChatName)
		assert.Equal(t, user1, resentMessage1New.EmbedMessage.Owner.Id)
		assert.Equal(t, user1Login, resentMessage1New.EmbedMessage.Owner.Login)

	})
}

func TestReplyMessage(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		const chat1Name = "new chat 1"

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name)
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		// user 1 adds user 2 to chat 1
		err = testRestClient.AddChatParticipants(ctx, user1, chat1Id, []int64{user2})
		require.NoError(t, err, "error in adding participant")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		// assert user 2 sees chat 1
		user2ChatsNew, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user2ChatsNew))
		chat1OfUser2 := user2ChatsNew[0]
		assert.Equal(t, chat1Name, chat1OfUser2.Title)

		const message1Text = "new message 1"

		message1Id, err := testRestClient.CreateMessage(ctx, user2, chat1Id, message1Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		const message2Text = "It is a reply"

		// user 1 replies on the message of user 2
		message2ResentId, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message2Text, client.NewMessageCreateOptionReply(message1Id))
		require.NoError(t, err, "error in resending message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		// assert that chat 1 contains the embed message
		{
			chat1Messages, _, err := testRestClient.GetMessages(ctx, user1, chat1Id)
			require.NoError(t, err, "error in getting messages")
			assert.Equal(t, 2, len(chat1Messages))
			repliedMessage2 := chat1Messages[1]
			assert.Equal(t, message2ResentId, repliedMessage2.Id)
			assert.Equal(t, message2Text, repliedMessage2.Content)
			require.NotNil(t, repliedMessage2.EmbedMessage)
			assert.Equal(t, dto.EmbedMessageTypeReply, repliedMessage2.EmbedMessage.EmbedType)
			assert.Equal(t, message1Text, repliedMessage2.EmbedMessage.Text)
			assert.Equal(t, message1Id, repliedMessage2.EmbedMessage.Id)
			assert.Equal(t, user2, repliedMessage2.EmbedMessage.Owner.Id)
			assert.Equal(t, user2Login, repliedMessage2.EmbedMessage.Owner.Login)
		}

		const message2TextNew = "It is a reply new"
		err = testRestClient.EditMessage(ctx, user1, chat1Id, message2ResentId, message2TextNew, client.NewMessageCreateOptionReply(message1Id))
		require.NoError(t, err, "error in resending message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		// assert that chat 1 contains the embed message
		{
			chat1Messages, _, err := testRestClient.GetMessages(ctx, user1, chat1Id)
			require.NoError(t, err, "error in getting messages")
			assert.Equal(t, 2, len(chat1Messages))
			repliedMessage2 := chat1Messages[1]
			assert.Equal(t, message2ResentId, repliedMessage2.Id)
			assert.Equal(t, message2TextNew, repliedMessage2.Content)
			require.NotNil(t, repliedMessage2.EmbedMessage)
			assert.Equal(t, dto.EmbedMessageTypeReply, repliedMessage2.EmbedMessage.EmbedType)
			assert.Equal(t, message1Text, repliedMessage2.EmbedMessage.Text)
			assert.Equal(t, message1Id, repliedMessage2.EmbedMessage.Id)
			assert.Equal(t, user2, repliedMessage2.EmbedMessage.Owner.Id)
			assert.Equal(t, user2Login, repliedMessage2.EmbedMessage.Owner.Login)
		}

		// user 2 edits the original
		const message1TextNew = "It is a new original"
		err = testRestClient.EditMessage(ctx, user2, chat1Id, message1Id, message1TextNew)
		require.NoError(t, err, "error in resending message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		// user 1 synchronizes the resend message
		err = testRestClient.SyncMessage(ctx, user1, chat1Id, message2ResentId)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		// assert
		{
			chat1Messages, _, err := testRestClient.GetMessages(ctx, user2, chat1Id)
			require.NoError(t, err, "error in getting messages")
			assert.Equal(t, 2, len(chat1Messages))
			repliedMessage2 := chat1Messages[1]
			assert.Equal(t, message2ResentId, repliedMessage2.Id)
			assert.Equal(t, message2TextNew, repliedMessage2.Content)
			require.NotNil(t, repliedMessage2.EmbedMessage)
			assert.Equal(t, dto.EmbedMessageTypeReply, repliedMessage2.EmbedMessage.EmbedType)
			assert.Equal(t, message1TextNew, repliedMessage2.EmbedMessage.Text)
			assert.Equal(t, message1Id, repliedMessage2.EmbedMessage.Id)
			assert.Equal(t, user2, repliedMessage2.EmbedMessage.Owner.Id)
			assert.Equal(t, user2Login, repliedMessage2.EmbedMessage.Owner.Login)
		}

		// remove reply
		const message2TextNewest = "It is a view without reply"
		err = testRestClient.EditMessage(ctx, user1, chat1Id, message2ResentId, message2TextNewest)
		require.NoError(t, err, "error in resending message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		// assert that chat 1 contains the embed message
		{
			chat1Messages, _, err := testRestClient.GetMessages(ctx, user1, chat1Id)
			require.NoError(t, err, "error in getting messages")
			assert.Equal(t, 2, len(chat1Messages))
			repliedMessage2 := chat1Messages[1]
			assert.Equal(t, message2ResentId, repliedMessage2.Id)
			assert.Nil(t, repliedMessage2.EmbedMessage)
			assert.Equal(t, message2TextNewest, repliedMessage2.Content)
		}

	})
}

func TestMentionNotification(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		testNotificationEventsAccumulator *listener.TestNotificationEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		const chat1Name = "new chat 1"

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name, client.NewChatOptionParticipants(user2))
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		// assert user 2 sees chat 1
		user2ChatsNew, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user2ChatsNew))
		chat1OfUser2 := user2ChatsNew[0]
		assert.Equal(t, chat1Name, chat1OfUser2.Title)

		t.Run("edit_message", func(t *testing.T) {
			const message1Text = "Just say hello"

			message1Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message1Text)
			require.NoError(t, err, "error in creating message")
			require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
			require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

			t.Run("add_mention", func(t *testing.T) {
				testOutputEventsAccumulator.Clean()
				testNotificationEventsAccumulator.Clean()

				// edit and add the mention
				var message1TextNew = fmt.Sprintf(`<p>Hello <a href="/user/%d" data-type="mention" class="mention" data-id="%d" data-label="%s" data-mention-suggestion-char="@">@%s</a> </p>`, user2, user2, user2Login, user2Login)

				err = testRestClient.EditMessage(ctx, user1, chat1Id, message1Id, message1TextNew)
				require.NoError(t, err, "error in resending message")
				require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
				require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

				require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, true, []func(e any) bool{
					func(ee any) bool {
						e, ok := ee.(*dto.ChatEvent)
						return ok && e.EventType == dto.EventTypeMessageEdited &&
							e.UserId == user2 &&
							e.ChatId == chat1Id &&
							e.MessageNotification.Id == message1Id &&
							strings.Contains(e.MessageNotification.Content, "Hello") && strings.Contains(e.MessageNotification.Content, "@"+user2Login) &&
							e.MessageNotification.Owner.Id == user1 &&
							e.MessageNotification.Owner.Login == user1Login
					},
				}))

				require.NoError(t, testNotificationEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, true, []func(e any) bool{
					func(ee any) bool {
						e, ok := ee.(*dto.NotificationEvent)
						return ok && e.EventType == dto.EventTypeMentionAdded &&
							e.UserId == user2 &&
							e.ChatId == chat1Id &&
							e.MentionNotification.Id == message1Id &&
							strings.Contains(e.MentionNotification.Text, "Hello") && strings.Contains(e.MentionNotification.Text, "@"+user2Login)
					},
				}))
			})

			t.Run("remove_mention", func(t *testing.T) {
				testOutputEventsAccumulator.Clean()
				testNotificationEventsAccumulator.Clean()

				// edit and remove the mention
				err = testRestClient.EditMessage(ctx, user1, chat1Id, message1Id, message1Text)
				require.NoError(t, err, "error in resending message")
				require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
				require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

				require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, true, []func(e any) bool{
					func(ee any) bool {
						e, ok := ee.(*dto.ChatEvent)
						return ok && e.EventType == dto.EventTypeMessageEdited &&
							e.UserId == user2 &&
							e.ChatId == chat1Id &&
							e.MessageNotification.Id == message1Id &&
							e.MessageNotification.Content == message1Text &&
							e.MessageNotification.Owner.Id == user1 &&
							e.MessageNotification.Owner.Login == user1Login
					},
				}))

				require.NoError(t, testNotificationEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, true, []func(e any) bool{
					func(ee any) bool {
						e, ok := ee.(*dto.NotificationEvent)
						return ok && e.EventType == dto.EventTypeMentionDeleted &&
							e.UserId == user2 &&
							e.ChatId == chat1Id &&
							e.MentionNotification.Id == message1Id
					},
				}))
			})
		})

		t.Run("read_message", func(t *testing.T) {
			testOutputEventsAccumulator.Clean()
			testNotificationEventsAccumulator.Clean()

			var message2Text = fmt.Sprintf(`<p>Hello <a href="/user/%d" data-type="mention" class="mention" data-id="%d" data-label="%s" data-mention-suggestion-char="@">@%s</a> </p>`, user2, user2, user2Login, user2Login)

			message2Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message2Text)
			require.NoError(t, err, "error in creating message")
			require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
			require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

			require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, true, []func(e any) bool{
				func(ee any) bool {
					e, ok := ee.(*dto.ChatEvent)
					return ok && e.EventType == dto.EventTypeMessageCreated &&
						e.UserId == user2 &&
						e.ChatId == chat1Id &&
						e.MessageNotification.Id == message2Id &&
						strings.Contains(e.MessageNotification.Content, "Hello") && strings.Contains(e.MessageNotification.Content, "@"+user2Login) &&
						e.MessageNotification.Owner.Id == user1 &&
						e.MessageNotification.Owner.Login == user1Login
				},
			}))

			require.NoError(t, testNotificationEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, true, []func(e any) bool{
				func(ee any) bool {
					e, ok := ee.(*dto.NotificationEvent)
					return ok && e.EventType == dto.EventTypeMentionAdded &&
						e.UserId == user2 &&
						e.ChatId == chat1Id &&
						e.MentionNotification.Id == message2Id &&
						strings.Contains(e.MentionNotification.Text, "Hello") && strings.Contains(e.MentionNotification.Text, "@"+user2Login)
				},
			}))

			testOutputEventsAccumulator.Clean()
			testNotificationEventsAccumulator.Clean()

			// read the message
			err = testRestClient.ReadMessage(ctx, user2, chat1Id, message2Id)
			require.NoError(t, err, "error in reading message")
			require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
			require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

			require.NoError(t, testNotificationEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, true, []func(e any) bool{
				func(ee any) bool {
					e, ok := ee.(*dto.NotificationEvent)
					return ok && e.EventType == dto.EventTypeMentionDeleted &&
						e.UserId == user2 &&
						e.ChatId == chat1Id &&
						e.MentionNotification.Id == message2Id
				},
			}))
		})
	})
}

func TestReplyNotification(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		testNotificationEventsAccumulator *listener.TestNotificationEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		const chat1Name = "new chat 1"

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name, client.NewChatOptionParticipants(user2))
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		const message1Text = "hi there1"
		const message2Text = "hi there2"

		message1Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message1Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		message2Id, err := testRestClient.CreateMessage(ctx, user2, chat1Id, message2Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		t.Run("edit_message", func(t *testing.T) {
			testOutputEventsAccumulator.Clean()
			testNotificationEventsAccumulator.Clean()

			const message2TextNew = "It is a reply new2"
			err = testRestClient.EditMessage(ctx, user2, chat1Id, message2Id, message2TextNew, client.NewMessageCreateOptionReply(message1Id))
			require.NoError(t, err, "error in resending message")
			require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
			require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

			require.NoError(t, testNotificationEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, true, []func(e any) bool{
				func(ee any) bool {
					e, ok := ee.(*dto.NotificationEvent)
					return ok && e.EventType == dto.EventTypeReplyAdded &&
						e.UserId == user1 &&
						e.ChatId == chat1Id &&
						e.ReplyNotification.MessageId == message2Id &&
						e.ReplyNotification.ReplyableMessage == message2TextNew
				},
			}))
		})

		t.Run("read_message", func(t *testing.T) {
			testOutputEventsAccumulator.Clean()
			testNotificationEventsAccumulator.Clean()

			err = testRestClient.ReadMessage(ctx, user1, chat1Id, message2Id)
			require.NoError(t, err, "error in reading message")
			require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
			require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

			require.NoError(t, testNotificationEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, true, []func(e any) bool{
				func(ee any) bool {
					e, ok := ee.(*dto.NotificationEvent)
					return ok && e.EventType == dto.EventTypeReplyDeleted &&
						e.UserId == user1 &&
						e.ChatId == chat1Id &&
						e.ReplyNotification.MessageId == message2Id
				},
			}))
		})
	})
}

func TestReactionNotification(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		testNotificationEventsAccumulator *listener.TestNotificationEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		const chat1Name = "new chat 1"

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name, client.NewChatOptionParticipants(user2))
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		const message1Text = "hi there1"

		message1Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message1Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		const reaction = "😀"

		t.Run("reaction_add", func(t *testing.T) {
			testOutputEventsAccumulator.Clean()
			testNotificationEventsAccumulator.Clean()

			// user adds the reaction
			err = testRestClient.Reaction(ctx, user2, chat1Id, message1Id, reaction)
			require.NoError(t, err, "error in reacting on message")
			require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
			require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

			require.NoError(t, testNotificationEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, true, []func(e any) bool{
				func(ee any) bool {
					e, ok := ee.(*dto.NotificationEvent)
					return ok && e.EventType == dto.EventTypeReactionAdded &&
						e.UserId == user1 &&
						e.ChatId == chat1Id &&
						e.ReactionEvent.Reaction == reaction &&
						e.ReactionEvent.MessageId == message1Id &&
						e.ReactionEvent.UserId == user2
				},
			}))
		})

		t.Run("read_message", func(t *testing.T) {
			testOutputEventsAccumulator.Clean()
			testNotificationEventsAccumulator.Clean()

			// read the message
			err = testRestClient.ReadMessage(ctx, user1, chat1Id, message1Id)
			require.NoError(t, err, "error in reading message")
			require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
			require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

			require.NoError(t, testNotificationEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, true, []func(e any) bool{
				func(ee any) bool {
					e, ok := ee.(*dto.NotificationEvent)
					return ok && e.EventType == dto.EventTypeReactionDeleted &&
						e.UserId == user1 &&
						e.ChatId == chat1Id &&
						e.ReactionEvent.Reaction == reaction &&
						e.ReactionEvent.MessageId == message1Id
				},
			}))
		})
	})
}

func TestBrowserNotification(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		testNotificationEventsAccumulator *listener.TestNotificationEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		const chat1Name = "new chat 1"

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name, client.NewChatOptionParticipants(user2))
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		var message1Id int64

		t.Run("message_create", func(t *testing.T) {
			const message1Text = "Hi there"

			testOutputEventsAccumulator.Clean()
			testNotificationEventsAccumulator.Clean()

			message1Id, err = testRestClient.CreateMessage(ctx, user1, chat1Id, message1Text)
			require.NoError(t, err, "error in creating message")
			require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
			require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

			require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, true, []func(e any) bool{
				func(ee any) bool {
					e, ok := ee.(*dto.ChatEvent)
					return ok && e.EventType == dto.EventTypeMessageCreated &&
						e.UserId == user2 &&
						e.ChatId == chat1Id &&
						e.MessageNotification.Id == message1Id &&
						e.MessageNotification.Content == message1Text &&
						e.MessageNotification.Owner.Id == user1 &&
						e.MessageNotification.Owner.Login == user1Login
				},

				func(ee any) bool {
					e, ok := ee.(*dto.GlobalUserEvent)
					return ok && e.EventType == dto.EventTypeMessageBrowserNotificationAdd &&
						e.UserId == user2 &&
						e.BrowserNotification.ChatId == chat1Id &&
						e.BrowserNotification.ChatName == chat1Name &&
						e.BrowserNotification.MessageId == message1Id &&
						e.BrowserNotification.MessageText == message1Text &&
						e.BrowserNotification.OwnerId == user1 &&
						e.BrowserNotification.OwnerLogin == user1Login
				},
			}))
		})

		t.Run("message_read", func(t *testing.T) {
			testOutputEventsAccumulator.Clean()
			testNotificationEventsAccumulator.Clean()

			// read the message
			err = testRestClient.ReadMessage(ctx, user2, chat1Id, message1Id)
			require.NoError(t, err, "error in reading message")
			require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
			require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

			require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, true, []func(e any) bool{
				func(ee any) bool {
					e, ok := ee.(*dto.GlobalUserEvent)
					return ok && e.EventType == dto.EventTypeMessageBrowserNotificationDelete &&
						e.UserId == user2 &&
						e.BrowserNotification.ChatId == chat1Id &&
						e.BrowserNotification.MessageId == message1Id
				},
			}))
		})
	})
}

func TestPinChat(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		dba *db.DB,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		const chat1Name = "new chat 1"

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name)
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		waitForChatExists(lgr, m, dba, chat1Id, user1, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		const message1Text = "new message 1"

		message1Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message1Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		user1Chats, _, err := testRestClient.GetChats(ctx, user1)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user1Chats))
		chat1OfUser1 := user1Chats[0]
		assert.Equal(t, false, chat1OfUser1.Pinned)
		assert.Equal(t, chat1Name, chat1OfUser1.Title)
		assert.Equal(t, int64(0), chat1OfUser1.UnreadMessages)

		user2Chats, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 0, len(user2Chats))

		chat1Messages, _, err := testRestClient.GetMessages(ctx, user1, chat1Id)
		require.NoError(t, err, "error in getting messages")
		assert.Equal(t, 1, len(chat1Messages))
		message1 := chat1Messages[0]
		assert.Equal(t, message1Id, message1.Id)
		assert.Equal(t, message1Text, message1.Content)

		err = testRestClient.AddChatParticipants(ctx, user1, chat1Id, []int64{user2})
		require.NoError(t, err, "error in adding participants")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		chat1Participants, _, err := testRestClient.GetChatParticipants(ctx, user1, chat1Id)
		require.NoError(t, err, "error in chat participants")
		require.Equal(t, 2, len(chat1Participants))
		assert.Equal(t, user2, chat1Participants[0].Id)
		assert.Equal(t, user1, chat1Participants[1].Id)

		user2ChatsNew, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user2ChatsNew))
		chat1OfUser2 := user2ChatsNew[0]
		assert.Equal(t, false, chat1OfUser2.Pinned)
		assert.Equal(t, chat1Name, chat1OfUser2.Title)
		assert.Equal(t, int64(1), chat1OfUser2.UnreadMessages)

		testOutputEventsAccumulator.Clean()

		err = testRestClient.PinChat(ctx, user1, chat1Id, true)
		require.NoError(t, err, "error in pinning chats")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatEdited &&
					e.UserId == user1 &&
					e.ChatNotification.ChatViewDto.Id == chat1Id &&
					e.ChatNotification.ChatViewDto.Title == chat1Name &&
					len(e.ChatNotification.Participants) == 2 &&
					e.ChatNotification.Participants[0].Id == user2 &&
					e.ChatNotification.Participants[0].Login == user2Login &&
					e.ChatNotification.Participants[1].Id == user1 &&
					e.ChatNotification.Participants[1].Login == user1Login &&
					e.ChatNotification.Pinned
			},
		}))

		user1ChatsNew2, _, err := testRestClient.GetChats(ctx, user1)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user1ChatsNew2))
		chat1OfUser1New2 := user1ChatsNew2[0]
		assert.Equal(t, true, chat1OfUser1New2.Pinned)
		assert.Equal(t, chat1Name, chat1OfUser1New2.Title)
		assert.Equal(t, int64(0), chat1OfUser1New2.UnreadMessages)
		assert.Equal(t, message1.Id, *chat1OfUser1New2.LastMessageId)
		assert.Equal(t, message1.Content, *chat1OfUser1New2.LastMessageContent)

		user2ChatsNew2, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user2ChatsNew2))
		chat1OfUser2New2 := user2ChatsNew2[0]
		assert.Equal(t, false, chat1OfUser2New2.Pinned)
		assert.Equal(t, chat1Name, chat1OfUser2New2.Title)
		assert.Equal(t, int64(1), chat1OfUser2New2.UnreadMessages)
		assert.Equal(t, message1.Id, *chat1OfUser2New2.LastMessageId)
		assert.Equal(t, message1.Content, *chat1OfUser2New2.LastMessageContent)
	})

}

func TestCreateChat(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		const chat1Name = "new chat 1"

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name)
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, true, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatCreated &&
					e.UserId == user1 &&
					e.ChatNotification.ChatViewDto.Id == chat1Id &&
					e.ChatNotification.ChatViewDto.Title == chat1Name &&
					len(e.ChatNotification.Participants) == 1 &&
					e.ChatNotification.Participants[0].Id == user1 &&
					e.ChatNotification.Participants[0].Login == user1Login
			},
		}))
	})
}

func TestCreateChatWithMultipleParticipants(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		const chat1Name = "new chat 1 with multiple participants"

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name, client.NewChatOptionParticipants(user2))
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatCreated &&
					e.UserId == user1 &&
					e.ChatNotification.ChatViewDto.Id == chat1Id &&
					e.ChatNotification.ChatViewDto.Title == chat1Name &&
					len(e.ChatNotification.Participants) == 2 &&
					e.ChatNotification.Participants[0].Id == user2 &&
					e.ChatNotification.Participants[0].Login == user2Login &&
					e.ChatNotification.Participants[1].Id == user1 &&
					e.ChatNotification.Participants[1].Login == user1Login
			},
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatCreated &&
					e.UserId == user2 &&
					e.ChatNotification.ChatViewDto.Id == chat1Id &&
					e.ChatNotification.ChatViewDto.Title == chat1Name &&
					len(e.ChatNotification.Participants) == 2 &&
					e.ChatNotification.Participants[0].Id == user2 &&
					e.ChatNotification.Participants[0].Login == user2Login &&
					e.ChatNotification.Participants[1].Id == user1 &&
					e.ChatNotification.Participants[1].Login == user1Login
			},
		}))
	})
}

func TestEditChatWithAddingParticipants(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		const chat1Name = "new chat 1"
		const chat1NewName = "new chat 1 with adding participants"

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name)
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		err = testRestClient.EditChat(ctx, user1, chat1Id, chat1NewName, client.NewChatOptionParticipants(user2))
		require.NoError(t, err, "error in changing chat")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, true, []func(e any) bool{
			// caused by CreateChat()
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatCreated &&
					e.UserId == user1 &&
					e.ChatNotification.ChatViewDto.Id == chat1Id &&
					e.ChatNotification.ChatViewDto.Title == chat1Name &&
					len(e.ChatNotification.Participants) == 1 &&
					e.ChatNotification.Participants[0].Id == user1 &&
					e.ChatNotification.Participants[0].Login == user1Login
			},

			// caused by EditChat
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatCreated &&
					e.UserId == user2 &&
					e.ChatNotification.ChatViewDto.Id == chat1Id &&
					e.ChatNotification.ChatViewDto.Title == chat1NewName &&
					len(e.ChatNotification.Participants) == 2 &&
					e.ChatNotification.Participants[0].Id == user2 &&
					e.ChatNotification.Participants[0].Login == user2Login &&
					e.ChatNotification.Participants[1].Id == user1 &&
					e.ChatNotification.Participants[1].Login == user1Login
			},
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatEdited &&
					e.UserId == user1 &&
					e.ChatNotification.ChatViewDto.Id == chat1Id &&
					e.ChatNotification.ChatViewDto.Title == chat1NewName &&
					len(e.ChatNotification.Participants) == 2 &&
					e.ChatNotification.Participants[0].Id == user2 &&
					e.ChatNotification.Participants[0].Login == user2Login &&
					e.ChatNotification.Participants[1].Id == user1 &&
					e.ChatNotification.Participants[1].Login == user1Login
			},
		}))
	})
}

func TestDeleteChat(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		dba *db.DB,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		const chat1Name = "new chat 1"

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name, client.NewChatOptionParticipants(user2))
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatCreated &&
					e.UserId == user1 &&
					e.ChatNotification.ChatViewDto.Id == chat1Id &&
					e.ChatNotification.ChatViewDto.Title == chat1Name &&
					len(e.ChatNotification.Participants) == 2 &&
					e.ChatNotification.Participants[0].Id == user2 &&
					e.ChatNotification.Participants[0].Login == user2Login &&
					e.ChatNotification.Participants[1].Id == user1 &&
					e.ChatNotification.Participants[1].Login == user1Login
			},
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatCreated &&
					e.UserId == user2 &&
					e.ChatNotification.ChatViewDto.Id == chat1Id &&
					e.ChatNotification.ChatViewDto.Title == chat1Name &&
					len(e.ChatNotification.Participants) == 2 &&
					e.ChatNotification.Participants[0].Id == user2 &&
					e.ChatNotification.Participants[0].Login == user2Login &&
					e.ChatNotification.Participants[1].Id == user1 &&
					e.ChatNotification.Participants[1].Login == user1Login
			},
		}))

		testOutputEventsAccumulator.Clean()

		waitForChatExists(lgr, m, dba, chat1Id, user1, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		const message1Text = "new message 1"

		message1Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message1Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		user1Chats, _, err := testRestClient.GetChats(ctx, user1)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user1Chats))
		chat1OfUser1 := user1Chats[0]
		assert.Equal(t, false, chat1OfUser1.Pinned)
		assert.Equal(t, chat1Name, chat1OfUser1.Title)
		assert.Equal(t, int64(0), chat1OfUser1.UnreadMessages)

		chat1Messages, _, err := testRestClient.GetMessages(ctx, user1, chat1Id)
		require.NoError(t, err, "error in getting messages")
		assert.Equal(t, 1, len(chat1Messages))
		message1 := chat1Messages[0]
		assert.Equal(t, message1Id, message1.Id)
		assert.Equal(t, message1Text, message1.Content)

		chat1Participants, _, err := testRestClient.GetChatParticipants(ctx, user1, chat1Id)
		require.NoError(t, err, "error in chat participants")
		require.Equal(t, 2, len(chat1Participants))
		assert.Equal(t, user2, chat1Participants[1].Id)
		assert.Equal(t, user1, chat1Participants[0].Id)

		user2ChatsNew, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user2ChatsNew))
		chat1OfUser2 := user2ChatsNew[0]
		assert.Equal(t, false, chat1OfUser2.Pinned)
		assert.Equal(t, chat1Name, chat1OfUser2.Title)
		assert.Equal(t, int64(1), chat1OfUser2.UnreadMessages)

		testOutputEventsAccumulator.Clean()

		err = testRestClient.DeleteChat(ctx, user1, chat1Id)
		require.NoError(t, err, "error in removing chats")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		user1ChatsNew2, _, err := testRestClient.GetChats(ctx, user1)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 0, len(user1ChatsNew2))

		user2ChatsNew2, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 0, len(user2ChatsNew2))

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatDeleted &&
					e.UserId == user1 &&
					e.ChatDeletedDto.Id == chat1Id
			},
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatDeleted &&
					e.UserId == user2 &&
					e.ChatDeletedDto.Id == chat1Id
			},
		}))

		ch, err := m.GetChatBasic(ctx, dba, chat1Id)
		require.NoError(t, err, "error in getting chat")
		require.Nil(t, ch) // assert that the chat was physically removed
	})

}

func TestAddParticipant(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		dba *db.DB,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		const chat1Name = "new chat 1"

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)
		mockAaaClient.EXPECT().SearchGetUsers(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]*dto.User{&mockUser2}, 2, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name)
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		waitForChatExists(lgr, m, dba, chat1Id, user1, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		const message1Text = "new message 1"

		message1Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message1Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		user1Chats, _, err := testRestClient.GetChats(ctx, user1)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user1Chats))
		chat1OfUser1 := user1Chats[0]
		assert.Equal(t, chat1Name, chat1OfUser1.Title)
		assert.Equal(t, int64(0), chat1OfUser1.UnreadMessages)
		assert.Equal(t, int64(1), chat1OfUser1.ParticipantsCount)
		assert.Equal(t, []int64{1}, chat1OfUser1.ParticipantIds)

		user2Chats, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 0, len(user2Chats))

		chat1Messages, _, err := testRestClient.GetMessages(ctx, user1, chat1Id)
		require.NoError(t, err, "error in getting messages")
		assert.Equal(t, 1, len(chat1Messages))
		message1 := chat1Messages[0]
		assert.Equal(t, message1Id, message1.Id)
		assert.Equal(t, message1Text, message1.Content)

		err = testRestClient.AddChatParticipants(ctx, user1, chat1Id, []int64{user2})
		require.NoError(t, err, "error in adding participants")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		chat1Participants, _, err := testRestClient.GetChatParticipants(ctx, user1, chat1Id)
		require.NoError(t, err, "error in chat participants")
		require.Equal(t, 2, len(chat1Participants))
		assert.Equal(t, user2, chat1Participants[0].Id)
		assert.Equal(t, user1, chat1Participants[1].Id)

		const searchString2 = user2Login
		chat1ParticipantsSearch2, _, err := testRestClient.GetChatParticipants(ctx, chat1Id, user1, client.NewParticipantGetOptionWithSearch(searchString2))
		require.NoError(t, err, "error in chat participants")
		require.Equal(t, 1, len(chat1ParticipantsSearch2))
		assert.Equal(t, user2, chat1ParticipantsSearch2[0].Id)

		user2ChatsNew, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user2ChatsNew))
		chat1OfUser2 := user2ChatsNew[0]
		assert.Equal(t, chat1Name, chat1OfUser2.Title)
		assert.Equal(t, int64(1), chat1OfUser2.UnreadMessages)
		assert.Equal(t, message1.Id, *chat1OfUser2.LastMessageId)
		assert.Equal(t, message1.Content, *chat1OfUser2.LastMessageContent)
		assert.Equal(t, int64(2), chat1OfUser2.ParticipantsCount)
		assert.Equal(t, []int64{2, 1}, chat1OfUser2.ParticipantIds)

		const chat1NewName = "new chat 1 renamed"
		avatar := "http://example.com/avatar.jpg"
		avatarBig := "http://example.com/avatar-big.jpg"

		// test CHatEdited on rename
		testOutputEventsAccumulator.Clean()

		err = testRestClient.EditChat(ctx, user1, chat1Id, chat1NewName, client.NewChatOptionAvatar(&avatar, &avatarBig))
		require.NoError(t, err, "error in changing chat")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			// caused by EditChat
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatEdited &&
					e.UserId == user2 &&
					e.ChatNotification.ChatViewDto.Id == chat1Id &&
					e.ChatNotification.ChatViewDto.Title == chat1NewName &&
					len(e.ChatNotification.Participants) == 2 &&
					e.ChatNotification.Participants[0].Id == user2 &&
					e.ChatNotification.Participants[0].Login == user2Login &&
					e.ChatNotification.Participants[1].Id == user1 &&
					e.ChatNotification.Participants[1].Login == user1Login
			},
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatEdited &&
					e.UserId == user1 &&
					e.ChatNotification.ChatViewDto.Id == chat1Id &&
					e.ChatNotification.ChatViewDto.Title == chat1NewName &&
					len(e.ChatNotification.Participants) == 2 &&
					e.ChatNotification.Participants[0].Id == user2 &&
					e.ChatNotification.Participants[0].Login == user2Login &&
					e.ChatNotification.Participants[1].Id == user1 &&
					e.ChatNotification.Participants[1].Login == user1Login
			},
		}))

		user1ChatsNew2, _, err := testRestClient.GetChats(ctx, user1)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user1ChatsNew2))
		chat1OfUser1New2 := user1ChatsNew2[0]
		assert.Equal(t, chat1NewName, chat1OfUser1New2.Title)
		assert.Equal(t, avatar, *chat1OfUser1New2.Avatar)
		assert.Equal(t, avatarBig, *chat1OfUser1New2.AvatarBig)
		assert.Equal(t, int64(0), chat1OfUser1New2.UnreadMessages)
		assert.Equal(t, message1.Id, *chat1OfUser1New2.LastMessageId)
		assert.Equal(t, message1.Content, *chat1OfUser1New2.LastMessageContent)
		assert.Equal(t, int64(2), chat1OfUser1New2.ParticipantsCount)
		assert.Equal(t, []int64{2, 1}, chat1OfUser1New2.ParticipantIds)

		user2ChatsNew2, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user2ChatsNew2))
		chat1OfUser2New2 := user2ChatsNew2[0]
		assert.Equal(t, chat1NewName, chat1OfUser2New2.Title)
		assert.Equal(t, int64(1), chat1OfUser2New2.UnreadMessages)
		assert.Equal(t, message1.Id, *chat1OfUser2New2.LastMessageId)
		assert.Equal(t, message1.Content, *chat1OfUser2New2.LastMessageContent)
		assert.Equal(t, int64(2), chat1OfUser2New2.ParticipantsCount)
		assert.Equal(t, []int64{2, 1}, chat1OfUser2New2.ParticipantIds)
	})
}

func TestAddParticipantChatStillNotExists(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		dba *db.DB,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2

		ctx := context.Background()

		const chat1Id = 234

		err := testRestClient.AddChatParticipants(ctx, user1, chat1Id, []int64{user2})
		require.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), fmt.Sprintf("code: %v", http.StatusTeapot)))
	})
}

func TestDeleteParticipant(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		dba *db.DB,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		const chat1Name = "new chat 1"

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name)
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		waitForChatExists(lgr, m, dba, chat1Id, user1, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		const message1Text = "new message 1"

		message1Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message1Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		user1Chats, _, err := testRestClient.GetChats(ctx, user1)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user1Chats))
		chat1OfUser1 := user1Chats[0]
		assert.Equal(t, chat1Name, chat1OfUser1.Title)
		assert.Equal(t, int64(0), chat1OfUser1.UnreadMessages)
		assert.Equal(t, int64(1), chat1OfUser1.ParticipantsCount)
		assert.Equal(t, []int64{1}, chat1OfUser1.ParticipantIds)

		user2Chats, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 0, len(user2Chats))

		chat1Messages, _, err := testRestClient.GetMessages(ctx, user1, chat1Id)
		require.NoError(t, err, "error in getting messages")
		assert.Equal(t, 1, len(chat1Messages))
		message1 := chat1Messages[0]
		assert.Equal(t, message1Id, message1.Id)
		assert.Equal(t, message1Text, message1.Content)

		err = testRestClient.AddChatParticipants(ctx, user1, chat1Id, []int64{user2})
		require.NoError(t, err, "error in adding participants")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		chat1Participants, _, err := testRestClient.GetChatParticipants(ctx, user1, chat1Id)
		require.NoError(t, err, "error in chat participants")
		require.Equal(t, 2, len(chat1Participants))
		assert.Equal(t, user2, chat1Participants[0].Id)
		assert.Equal(t, user1, chat1Participants[1].Id)

		user2ChatsNew, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user2ChatsNew))
		chat1OfUser2 := user2ChatsNew[0]
		assert.Equal(t, chat1Name, chat1OfUser2.Title)
		assert.Equal(t, int64(1), chat1OfUser2.UnreadMessages)
		assert.Equal(t, message1.Id, *chat1OfUser2.LastMessageId)
		assert.Equal(t, message1.Content, *chat1OfUser2.LastMessageContent)
		assert.Equal(t, int64(2), chat1OfUser2.ParticipantsCount)
		assert.Equal(t, []int64{2, 1}, chat1OfUser2.ParticipantIds)

		user2HasUnreadMessages, err := testRestClient.GetHasUnreadMessages(ctx, user2)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, true, user2HasUnreadMessages)

		err = testRestClient.DeleteChatParticipants(ctx, user1, chat1Id, user2)
		require.NoError(t, err, "error in removing chat participants")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		user2ChatsNew2, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 0, len(user2ChatsNew2))

		// after removing from chat user 2 got no unread messages
		user2HasUnreadMessagesNew2, err := testRestClient.GetHasUnreadMessages(ctx, user2)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, false, user2HasUnreadMessagesNew2)

		chat1Participants2, _, err := testRestClient.GetChatParticipants(ctx, user1, chat1Id)
		require.NoError(t, err, "error in chat participants")
		require.Equal(t, 1, len(chat1Participants2))
		assert.Equal(t, user1, chat1Participants2[0].Id)

		user1ChatsNew2, _, err := testRestClient.GetChats(ctx, user1)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user1ChatsNew2))
		chat1OfUser1New2 := user1ChatsNew2[0]
		assert.Equal(t, int64(1), chat1OfUser1New2.ParticipantsCount)
		assert.Equal(t, []int64{1}, chat1OfUser1New2.ParticipantIds)
	})
}

func TestLeaveFromChat(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		dba *db.DB,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		const chat1Name = "new chat 1"

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name)
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		waitForChatExists(lgr, m, dba, chat1Id, user1, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		const message1Text = "new message 1"

		message1Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message1Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		user1Chats, _, err := testRestClient.GetChats(ctx, user1)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user1Chats))
		chat1OfUser1 := user1Chats[0]
		assert.Equal(t, chat1Name, chat1OfUser1.Title)
		assert.Equal(t, int64(0), chat1OfUser1.UnreadMessages)
		assert.Equal(t, int64(1), chat1OfUser1.ParticipantsCount)
		assert.Equal(t, []int64{1}, chat1OfUser1.ParticipantIds)

		user2Chats, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 0, len(user2Chats))

		chat1Messages, _, err := testRestClient.GetMessages(ctx, user1, chat1Id)
		require.NoError(t, err, "error in getting messages")
		assert.Equal(t, 1, len(chat1Messages))
		message1 := chat1Messages[0]
		assert.Equal(t, message1Id, message1.Id)
		assert.Equal(t, message1Text, message1.Content)

		err = testRestClient.AddChatParticipants(ctx, user1, chat1Id, []int64{user2})
		require.NoError(t, err, "error in adding participants")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		chat1Participants, _, err := testRestClient.GetChatParticipants(ctx, user1, chat1Id)
		require.NoError(t, err, "error in chat participants")
		require.Equal(t, 2, len(chat1Participants))
		assert.Equal(t, user2, chat1Participants[0].Id)
		assert.Equal(t, user1, chat1Participants[1].Id)

		user2ChatsNew, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user2ChatsNew))
		chat1OfUser2 := user2ChatsNew[0]
		assert.Equal(t, chat1Name, chat1OfUser2.Title)
		assert.Equal(t, int64(1), chat1OfUser2.UnreadMessages)
		assert.Equal(t, message1.Id, *chat1OfUser2.LastMessageId)
		assert.Equal(t, message1.Content, *chat1OfUser2.LastMessageContent)
		assert.Equal(t, int64(2), chat1OfUser2.ParticipantsCount)
		assert.Equal(t, []int64{2, 1}, chat1OfUser2.ParticipantIds)

		user2HasUnreadMessages, err := testRestClient.GetHasUnreadMessages(ctx, user2)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, true, user2HasUnreadMessages)

		testOutputEventsAccumulator.Clean()

		err = testRestClient.LeaveChat(ctx, user2, chat1Id)
		require.NoError(t, err, "error in removing chat participants")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, true, []func(e any) bool{

			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeParticipantDeleted &&
					e.UserId == user1 &&
					e.ChatId == chat1Id &&
					len(*e.Participants) == 1 &&
					(*e.Participants)[0].Id == user2
			},
			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeParticipantDeleted &&
					e.UserId == user2 &&
					e.ChatId == chat1Id &&
					len(*e.Participants) == 1 &&
					(*e.Participants)[0].Id == user2
			},

			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatDeleted &&
					e.UserId == user2 &&
					e.ChatDeletedDto.Id == chat1Id
			},

			// caused by LeaveChat
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatEdited &&
					e.UserId == user1 &&
					e.ChatNotification.ChatViewDto.Id == chat1Id &&
					e.ChatNotification.ChatViewDto.Title == chat1Name &&
					len(e.ChatNotification.Participants) == 1 &&
					e.ChatNotification.Participants[0].Id == user1 &&
					e.ChatNotification.Participants[0].Login == user1Login
			},
		}))

		user2ChatsNew2, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 0, len(user2ChatsNew2))

		// after removing from chat user 2 got no unread messages
		user2HasUnreadMessagesNew2, err := testRestClient.GetHasUnreadMessages(ctx, user2)
		require.NoError(t, err, "error in getting has unread messages")
		assert.Equal(t, false, user2HasUnreadMessagesNew2)

		chat1Participants2, _, err := testRestClient.GetChatParticipants(ctx, user1, chat1Id)
		require.NoError(t, err, "error in chat participants")
		require.Equal(t, 1, len(chat1Participants2))
		assert.Equal(t, user1, chat1Participants2[0].Id)

		user1ChatsNew2, _, err := testRestClient.GetChats(ctx, user1)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user1ChatsNew2))
		chat1OfUser1New2 := user1ChatsNew2[0]
		assert.Equal(t, int64(1), chat1OfUser1New2.ParticipantsCount)
		assert.Equal(t, []int64{1}, chat1OfUser1New2.ParticipantIds)
	})
}

func TestAddChangeAndDeleteParticipant(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "tobeAdmin2"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		const chat1Name = "new chat 1"

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name)
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatCreated &&
					e.UserId == user1 &&
					e.ChatNotification.ChatViewDto.Id == chat1Id &&
					e.ChatNotification.ChatViewDto.Title == chat1Name &&
					len(e.ChatNotification.Participants) == 1 &&
					e.ChatNotification.Participants[0].Id == user1 &&
					e.ChatNotification.Participants[0].Login == user1Login
			},
		}))

		testOutputEventsAccumulator.Clean()

		err = testRestClient.AddChatParticipants(ctx, user1, chat1Id, []int64{user2})
		require.NoError(t, err, "error in adding participants")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, true, []func(e any) bool{
			// caused by AddChatParticipants
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatCreated &&
					e.UserId == user2 &&
					e.ChatNotification.ChatViewDto.Id == chat1Id &&
					e.ChatNotification.ChatViewDto.Title == chat1Name &&
					len(e.ChatNotification.Participants) == 2 &&
					e.ChatNotification.Participants[0].Id == user2 &&
					e.ChatNotification.Participants[0].Login == user2Login &&
					e.ChatNotification.Participants[1].Id == user1 &&
					e.ChatNotification.Participants[1].Login == user1Login
			},
			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeParticipantAdded &&
					e.UserId == user1 &&
					e.ChatId == chat1Id &&
					len(*e.Participants) == 1 &&
					(*e.Participants)[0].Id == user2 &&
					(*e.Participants)[0].Login == user2Login
			},
			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeParticipantAdded &&
					e.UserId == user2 &&
					e.ChatId == chat1Id &&
					len(*e.Participants) == 1 &&
					(*e.Participants)[0].Id == user2 &&
					(*e.Participants)[0].Login == user2Login
			},
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatEdited &&
					e.UserId == user1 &&
					e.ChatNotification.ChatViewDto.Id == chat1Id &&
					e.ChatNotification.ChatViewDto.Title == chat1Name &&
					len(e.ChatNotification.Participants) == 2 &&
					e.ChatNotification.Participants[0].Id == user2 &&
					e.ChatNotification.Participants[0].Login == user2Login &&
					e.ChatNotification.Participants[1].Id == user1 &&
					e.ChatNotification.Participants[1].Login == user1Login
			},
		}))

		testOutputEventsAccumulator.Clean()

		// negative test
		err = testRestClient.ChangeChatParticipant(ctx, user2, chat1Id, user1, false)
		require.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), fmt.Sprintf("code: %v", http.StatusUnauthorized)))

		chat1Participants, _, err := testRestClient.GetChatParticipants(ctx, user1, chat1Id)
		require.NoError(t, err, "error in chat participants")
		require.Equal(t, 2, len(chat1Participants))
		assert.Equal(t, user2, chat1Participants[0].Id)
		assert.Equal(t, false, chat1Participants[0].ChatAdmin)
		assert.Equal(t, user1, chat1Participants[1].Id)
		assert.Equal(t, true, chat1Participants[1].ChatAdmin)

		user2ChatsNew, _, err := testRestClient.GetChats(ctx, user2)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user2ChatsNew))
		chat1OfUser2 := user2ChatsNew[0]
		assert.Equal(t, chat1Name, chat1OfUser2.Title)
		assert.Equal(t, int64(2), chat1OfUser2.ParticipantsCount)
		assert.Equal(t, []int64{2, 1}, chat1OfUser2.ParticipantIds)

		testOutputEventsAccumulator.Clean()

		err = testRestClient.ChangeChatParticipant(ctx, user1, chat1Id, user2, true)
		require.NoError(t, err, "error in changing chat participants")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		chat1Participants2, _, err := testRestClient.GetChatParticipants(ctx, user1, chat1Id)
		require.NoError(t, err, "error in chat participants")
		require.Equal(t, 2, len(chat1Participants2))
		assert.Equal(t, user2, chat1Participants2[0].Id)
		assert.Equal(t, true, chat1Participants2[0].ChatAdmin)
		assert.Equal(t, user1, chat1Participants2[1].Id)
		assert.Equal(t, true, chat1Participants2[1].ChatAdmin)

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, true, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeParticipantEdited &&
					e.UserId == user1 &&
					e.ChatId == chat1Id &&
					len(*e.Participants) == 1 &&
					(*e.Participants)[0].Id == user2 &&
					(*e.Participants)[0].Login == user2Login &&
					(*e.Participants)[0].ChatAdmin == true
			},
			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeParticipantEdited &&
					e.UserId == user2 &&
					e.ChatId == chat1Id &&
					len(*e.Participants) == 1 &&
					(*e.Participants)[0].Id == user2 &&
					(*e.Participants)[0].Login == user2Login &&
					(*e.Participants)[0].ChatAdmin == true
			},

			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeParticipantsReload &&
					e.UserId == user2 &&
					e.ChatId == chat1Id
			},

			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatEdited &&
					e.UserId == user2 &&
					e.ChatNotification.ChatViewDto.Id == chat1Id &&
					e.ChatNotification.ChatViewDto.Title == chat1Name &&
					len(e.ChatNotification.Participants) == 2 &&
					e.ChatNotification.Participants[0].Id == user2 &&
					e.ChatNotification.Participants[0].Login == user2Login &&
					e.ChatNotification.Participants[1].Id == user1 &&
					e.ChatNotification.Participants[1].Login == user1Login &&
					e.ChatNotification.CanAddParticipant == true // we test that user2 now can do it because he became a chat admin
			},
		}))

		testOutputEventsAccumulator.Clean()

		err = testRestClient.DeleteChatParticipants(ctx, user1, chat1Id, user2)
		require.NoError(t, err, "error in removing chat participants")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, true, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeParticipantDeleted &&
					e.UserId == user1 &&
					e.ChatId == chat1Id &&
					len(*e.Participants) == 1 &&
					(*e.Participants)[0].Id == user2
			},
			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeParticipantDeleted &&
					e.UserId == user2 &&
					e.ChatId == chat1Id &&
					len(*e.Participants) == 1 &&
					(*e.Participants)[0].Id == user2
			},

			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatDeleted &&
					e.UserId == user2 &&
					e.ChatDeletedDto.Id == chat1Id
			},
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatEdited &&
					e.UserId == user1 &&
					e.ChatNotification.ChatViewDto.Id == chat1Id &&
					e.ChatNotification.ChatViewDto.Title == chat1Name &&
					len(e.ChatNotification.Participants) == 1 &&
					e.ChatNotification.Participants[0].Id == user1 &&
					e.ChatNotification.Participants[0].Login == user1Login
			},
		}))
	})
}

func TestCreateMessage(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		const chat1Name = "new chat 1"
		const message1Text = "message text 1"

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name, client.NewChatOptionParticipants(user2))
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		testOutputEventsAccumulator.Clean()

		message1Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message1Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")
		assert.True(t, message1Id > 0)

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, true, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeMessageCreated &&
					e.UserId == user1 &&
					e.ChatId == chat1Id &&
					e.MessageNotification.Id == message1Id &&
					e.MessageNotification.Content == message1Text &&
					e.MessageNotification.Owner.Id == user1 &&
					e.MessageNotification.Owner.Login == user1Login
			},
			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeMessageCreated &&
					e.UserId == user2 &&
					e.ChatId == chat1Id &&
					e.MessageNotification.Id == message1Id &&
					e.MessageNotification.Content == message1Text &&
					e.MessageNotification.Owner.Id == user1 &&
					e.MessageNotification.Owner.Login == user1Login
			},
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeHasUnreadMessagesChanged &&
					e.UserId == user1 &&
					e.HasUnreadMessagesChanged.HasUnreadMessages == false // it's not being change for himself
			},
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatEdited &&
					e.UserId == user1 &&
					e.ChatNotification.ChatViewDto.Id == chat1Id &&
					e.ChatNotification.ChatViewDto.Title == chat1Name &&
					*e.ChatNotification.ChatViewDto.LastMessageId == message1Id &&
					*e.ChatNotification.ChatViewDto.LastMessageOwnerId == user1 &&
					*e.ChatNotification.ChatViewDto.LastMessageContent == message1Text &&
					len(e.ChatNotification.Participants) == 2 &&
					e.ChatNotification.Participants[0].Id == user2 &&
					e.ChatNotification.Participants[0].Login == user2Login &&
					e.ChatNotification.Participants[1].Id == user1 &&
					e.ChatNotification.Participants[1].Login == user1Login &&
					e.ChatNotification.UnreadMessages == 0
			},
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeHasUnreadMessagesChanged &&
					e.UserId == user2 &&
					e.HasUnreadMessagesChanged.HasUnreadMessages == true // a cumulative indicator representing unreads in all the chats, user dor the red dot
			},
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatEdited &&
					e.UserId == user2 &&
					e.ChatNotification.ChatViewDto.Id == chat1Id &&
					e.ChatNotification.ChatViewDto.Title == chat1Name &&
					*e.ChatNotification.ChatViewDto.LastMessageId == message1Id &&
					*e.ChatNotification.ChatViewDto.LastMessageOwnerId == user1 &&
					*e.ChatNotification.ChatViewDto.LastMessageContent == message1Text &&
					len(e.ChatNotification.Participants) == 2 &&
					e.ChatNotification.Participants[0].Id == user2 &&
					e.ChatNotification.Participants[0].Login == user2Login &&
					e.ChatNotification.Participants[1].Id == user1 &&
					e.ChatNotification.Participants[1].Login == user1Login &&
					e.ChatNotification.UnreadMessages == 1
			},
		}))
	})
}

func TestCreateMessageChatStillNotExists(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1

		const message1Text = "message text 1"

		ctx := context.Background()

		const chat1Id = 12345

		_, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message1Text)
		require.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), fmt.Sprintf("code: %v", http.StatusTeapot)))
	})
}

func TestEditMessage(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		dba *db.DB,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"
		const chat1Name = "new chat 1"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name, client.NewChatOptionParticipants(user2))
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		waitForChatExists(lgr, m, dba, chat1Id, user1, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		const message1Text = "new message 1"
		message1Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message1Text)
		require.NoError(t, err, "error in creating message")

		const message2Text = "new message 2"
		message2Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message2Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		user1Chats, _, err := testRestClient.GetChats(ctx, user1)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user1Chats))
		chat1OfUser1 := user1Chats[0]
		assert.Equal(t, chat1Name, chat1OfUser1.Title)
		assert.Equal(t, int64(0), chat1OfUser1.UnreadMessages)
		assert.Equal(t, message2Text, *chat1OfUser1.LastMessageContent)

		chat1Messages, _, err := testRestClient.GetMessages(ctx, user1, chat1Id)
		require.NoError(t, err, "error in getting messages")
		assert.Equal(t, 2, len(chat1Messages))
		message1 := chat1Messages[0]
		message2 := chat1Messages[1]
		assert.Equal(t, message1Id, message1.Id)
		assert.Equal(t, message1Text, message1.Content)
		assert.Equal(t, message2Id, message2.Id)
		assert.Equal(t, message2Text, message2.Content)

		const message1TextNew = "new message 1 edited"
		err = testRestClient.EditMessage(ctx, user1, chat1Id, message1.Id, message1TextNew)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		user1ChatsNew, _, err := testRestClient.GetChats(ctx, user1)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user1ChatsNew))
		chat1OfUser1New := user1ChatsNew[0]
		assert.Equal(t, chat1Name, chat1OfUser1New.Title)
		assert.Equal(t, int64(0), chat1OfUser1New.UnreadMessages)
		assert.Equal(t, message2Text, *chat1OfUser1New.LastMessageContent)

		chat1MessagesNew, _, err := testRestClient.GetMessages(ctx, user1, chat1Id)
		require.NoError(t, err, "error in getting messages")
		assert.Equal(t, 2, len(chat1MessagesNew))
		message1New := chat1MessagesNew[0]
		message2New := chat1MessagesNew[1]
		assert.Equal(t, message1Id, message1New.Id)
		assert.Equal(t, message1TextNew, message1New.Content)
		assert.Equal(t, message2Id, message2New.Id)
		assert.Equal(t, message2Text, message2New.Content)

		testOutputEventsAccumulator.Clean()
		const message2TextNew = "new message 2 edited"
		err = testRestClient.EditMessage(ctx, user1, chat1Id, message2.Id, message2TextNew)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, true, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeMessageEdited &&
					e.UserId == user1 &&
					e.ChatId == chat1Id &&
					e.MessageNotification.Id == message2Id &&
					e.MessageNotification.Content == message2TextNew &&
					e.MessageNotification.Owner.Id == user1 &&
					e.MessageNotification.Owner.Login == user1Login
			},
			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeMessageEdited &&
					e.UserId == user2 &&
					e.ChatId == chat1Id &&
					e.MessageNotification.Id == message2Id &&
					e.MessageNotification.Content == message2TextNew &&
					e.MessageNotification.Owner.Id == user1 &&
					e.MessageNotification.Owner.Login == user1Login
			},

			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatEdited &&
					e.UserId == user1 &&
					e.ChatNotification.ChatViewDto.Id == chat1Id &&
					e.ChatNotification.ChatViewDto.Title == chat1Name &&
					*e.ChatNotification.ChatViewDto.LastMessageId == message2Id &&
					*e.ChatNotification.ChatViewDto.LastMessageOwnerId == user1 &&
					*e.ChatNotification.ChatViewDto.LastMessageContent == message2TextNew &&
					len(e.ChatNotification.Participants) == 2 &&
					e.ChatNotification.Participants[0].Id == user2 &&
					e.ChatNotification.Participants[0].Login == user2Login &&
					e.ChatNotification.Participants[1].Id == user1 &&
					e.ChatNotification.Participants[1].Login == user1Login &&
					e.ChatNotification.UnreadMessages == 0
			},
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatEdited &&
					e.UserId == user2 &&
					e.ChatNotification.ChatViewDto.Id == chat1Id &&
					e.ChatNotification.ChatViewDto.Title == chat1Name &&
					*e.ChatNotification.ChatViewDto.LastMessageId == message2Id &&
					*e.ChatNotification.ChatViewDto.LastMessageOwnerId == user1 &&
					*e.ChatNotification.ChatViewDto.LastMessageContent == message2TextNew &&
					len(e.ChatNotification.Participants) == 2 &&
					e.ChatNotification.Participants[0].Id == user2 &&
					e.ChatNotification.Participants[0].Login == user2Login &&
					e.ChatNotification.Participants[1].Id == user1 &&
					e.ChatNotification.Participants[1].Login == user1Login &&
					e.ChatNotification.UnreadMessages == 2
			},
		}))

		user1ChatsNew2, _, err := testRestClient.GetChats(ctx, user1)
		require.NoError(t, err, "error in getting chats")
		assert.Equal(t, 1, len(user1ChatsNew2))
		chat1OfUser1New2 := user1ChatsNew2[0]
		assert.Equal(t, chat1Name, chat1OfUser1New2.Title)
		assert.Equal(t, int64(0), chat1OfUser1New2.UnreadMessages)
		assert.Equal(t, message2TextNew, *chat1OfUser1New2.LastMessageContent)

		chat1MessagesNew2, _, err := testRestClient.GetMessages(ctx, user1, chat1Id)
		require.NoError(t, err, "error in getting messages")
		assert.Equal(t, 2, len(chat1MessagesNew2))
		message1New2 := chat1MessagesNew2[0]
		message2New2 := chat1MessagesNew2[1]
		assert.Equal(t, message1Id, message1New2.Id)
		assert.Equal(t, message1TextNew, message1New2.Content)
		assert.Equal(t, message2Id, message2New2.Id)
		assert.Equal(t, message2TextNew, message2New2.Content)
	})
}

func TestPinMessage(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		dba *db.DB,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"
		const chat1Name = "new chat 1"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name, client.NewChatOptionParticipants(user2))
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		waitForChatExists(lgr, m, dba, chat1Id, user1, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		const message1Text = "new message 1"
		message1Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message1Text)
		require.NoError(t, err, "error in creating message")

		const message2Text = "new message 2"
		message2Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message2Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		t.Run("pin_unpin", func(t *testing.T) {
			testOutputEventsAccumulator.Clean()

			err = testRestClient.PinMessage(ctx, user1, chat1Id, message1Id, true)
			require.NoError(t, err, "error in pinning message")

			require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
				func(ee any) bool {
					e, ok := ee.(*dto.ChatEvent)
					return ok && e.EventType == dto.EventTypePinnedMessagePromote &&
						e.UserId == user1 &&
						e.ChatId == chat1Id &&
						e.PromoteMessageNotification.Message.Id == message1Id &&
						e.PromoteMessageNotification.Message.Text == message1Text
				},
			}))

			testOutputEventsAccumulator.Clean()

			err = testRestClient.PinMessage(ctx, user1, chat1Id, message1Id, false)
			require.NoError(t, err, "error in pinning message")

			require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
				func(ee any) bool {
					e, ok := ee.(*dto.ChatEvent)
					return ok && e.EventType == dto.EventTypePinnedMessageUnpromote &&
						e.UserId == user1 &&
						e.ChatId == chat1Id &&
						e.PromoteMessageNotification.Message.Id == message1Id
				},
			}))

			testOutputEventsAccumulator.Clean()

			err = testRestClient.PinMessage(ctx, user1, chat1Id, message1Id, true)

			require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
				func(ee any) bool {
					e, ok := ee.(*dto.ChatEvent)
					return ok && e.EventType == dto.EventTypePinnedMessagePromote &&
						e.UserId == user1 &&
						e.ChatId == chat1Id &&
						e.PromoteMessageNotification.Message.Id == message1Id &&
						e.PromoteMessageNotification.Message.Text == message1Text
				},
			}))

			testOutputEventsAccumulator.Clean()
		})

		t.Run("pin_and_promote_new", func(t *testing.T) {
			err = testRestClient.PinMessage(ctx, user1, chat1Id, message2Id, true)
			require.NoError(t, err, "error in pinning message")

			require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
				// here we not get unpromote, because we don't send unpromote, only promote new, unpromoting is made on the frontend
				func(ee any) bool {
					e, ok := ee.(*dto.ChatEvent)
					return ok && e.EventType == dto.EventTypePinnedMessagePromote &&
						e.UserId == user1 &&
						e.ChatId == chat1Id &&
						e.PromoteMessageNotification.Message.Id == message2Id &&
						e.PromoteMessageNotification.Message.Text == message2Text
				},
			}))

			testOutputEventsAccumulator.Clean()
		})

		const message2TextNew = "new message 2 edited"

		t.Run("edit_message", func(t *testing.T) {
			err = testRestClient.EditMessage(ctx, user1, chat1Id, message2Id, message2TextNew)
			require.NoError(t, err, "error in creating message")
			require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
			require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

			require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
				func(ee any) bool {
					e, ok := ee.(*dto.ChatEvent)
					return ok && e.EventType == dto.EventTypePinnedMessageEdit &&
						e.UserId == user1 &&
						e.ChatId == chat1Id &&
						e.PromoteMessageNotification.Message.Id == message2Id &&
						e.PromoteMessageNotification.Message.Text == message2TextNew
				},
			}))

			testOutputEventsAccumulator.Clean()
		})

		t.Run("unpin_makes_other_pinned", func(t *testing.T) {
			err = testRestClient.PinMessage(ctx, user1, chat1Id, message2Id, false)
			require.NoError(t, err, "error in pinning message")

			require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
				func(ee any) bool {
					e, ok := ee.(*dto.ChatEvent)
					return ok && e.EventType == dto.EventTypePinnedMessageUnpromote &&
						e.UserId == user1 &&
						e.ChatId == chat1Id &&
						e.PromoteMessageNotification.Message.Id == message2Id
				},

				func(ee any) bool {
					e, ok := ee.(*dto.ChatEvent)
					return ok && e.EventType == dto.EventTypePinnedMessagePromote &&
						e.UserId == user1 &&
						e.ChatId == chat1Id &&
						e.PromoteMessageNotification.Message.Id == message1Id &&
						e.PromoteMessageNotification.Message.Text == message1Text
				},
			}))

			testOutputEventsAccumulator.Clean()
		})

		// restore pinned of 2 and check web handles
		err = testRestClient.PinMessage(ctx, user1, chat1Id, message2Id, true)
		require.NoError(t, err, "error in pinning message")
		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypePinnedMessagePromote &&
					e.UserId == user1 &&
					e.ChatId == chat1Id &&
					e.PromoteMessageNotification.Message.Id == message2Id &&
					e.PromoteMessageNotification.Message.Text == message2TextNew
			},
		}))
		testOutputEventsAccumulator.Clean()

		pinneds, err := testRestClient.GetPinnedMessages(ctx, user1, chat1Id)
		require.NoError(t, err, "error in get pinned messages")
		require.Equal(t, 2, len(pinneds))
		assert.Equal(t, message2Id, pinneds[0].Id)
		assert.Equal(t, message2TextNew, pinneds[0].Text)
		assert.Equal(t, message1Id, pinneds[1].Id)
		assert.Equal(t, message1Text, pinneds[1].Text)

		pinnedPromoted, err := testRestClient.GetPinnedPromotedMessage(ctx, user1, chat1Id)
		require.NoError(t, err, "error in get pinned promoted message")
		require.NotNil(t, pinnedPromoted)
		assert.Equal(t, message2Id, pinnedPromoted.Id)
		assert.Equal(t, message2TextNew, pinnedPromoted.Text)

		t.Run("delete_makes_other_pinned", func(t *testing.T) {
			err = testRestClient.DeleteMessage(ctx, user1, chat1Id, message2Id)
			require.NoError(t, err, "error in pinning message")

			require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
				func(ee any) bool {
					e, ok := ee.(*dto.ChatEvent)
					return ok && e.EventType == dto.EventTypePinnedMessageUnpromote &&
						e.UserId == user1 &&
						e.ChatId == chat1Id &&
						e.PromoteMessageNotification.Message.Id == message2Id
				},

				func(ee any) bool {
					e, ok := ee.(*dto.ChatEvent)
					return ok && e.EventType == dto.EventTypePinnedMessagePromote &&
						e.UserId == user1 &&
						e.ChatId == chat1Id &&
						e.PromoteMessageNotification.Message.Id == message1Id &&
						e.PromoteMessageNotification.Message.Text == message1Text
				},
			}))

			testOutputEventsAccumulator.Clean()
		})

	})
}

func TestPublishMessage(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		dba *db.DB,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"
		const chat1Name = "new chat 1"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name, client.NewChatOptionParticipants(user2))
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		waitForChatExists(lgr, m, dba, chat1Id, user1, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		const message1Text = "new message 1"
		message1Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message1Text)
		require.NoError(t, err, "error in creating message")

		err = testRestClient.PublishMessage(ctx, user1, chat1Id, message1Id, true)
		require.NoError(t, err, "error in publishing message")
		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypePublishedMessageAdd &&
					e.UserId == user1 &&
					e.ChatId == chat1Id &&
					e.PublishedMessageNotification.Message.Id == message1Id &&
					e.PublishedMessageNotification.Message.ChatId == chat1Id &&
					e.PublishedMessageNotification.Message.Text == message1Text
			},
		}))
		testOutputEventsAccumulator.Clean()

		publisheds, err := testRestClient.GetPublishedMessages(ctx, user1, chat1Id)
		require.NoError(t, err, "error in get published messages")
		require.Equal(t, 1, len(publisheds))
		assert.Equal(t, message1Id, publisheds[0].Id)
		assert.Equal(t, message1Text, publisheds[0].Text)

		published, err := testRestClient.GetPublishedMessageForPublic(ctx, chat1Id, message1Id)
		require.NoError(t, err, "error in get published message")
		require.NotNil(t, published)
		assert.Equal(t, message1Id, published.Id)
		assert.Equal(t, message1Text, published.Content)

		const message1TextNew = "new message 1 edited"
		err = testRestClient.EditMessage(ctx, user1, chat1Id, message1Id, message1TextNew)
		require.NoError(t, err, "error in editing message")
		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypePublishedMessageEdit &&
					e.UserId == user1 &&
					e.ChatId == chat1Id &&
					e.PublishedMessageNotification.Message.Id == message1Id &&
					e.PublishedMessageNotification.Message.ChatId == chat1Id &&
					e.PublishedMessageNotification.Message.Text == message1TextNew
			},
		}))
		testOutputEventsAccumulator.Clean()

		err = testRestClient.PublishMessage(ctx, user1, chat1Id, message1Id, false)
		require.NoError(t, err, "error in publishing message")
		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypePublishedMessageRemove &&
					e.UserId == user1 &&
					e.ChatId == chat1Id &&
					e.PublishedMessageNotification.Message.Id == message1Id &&
					e.PublishedMessageNotification.Message.ChatId == chat1Id
			},
		}))
		testOutputEventsAccumulator.Clean()

		publishedsNo, err := testRestClient.GetPublishedMessages(ctx, user1, chat1Id)
		require.NoError(t, err, "error in get published messages")
		require.Equal(t, 0, len(publishedsNo))

		publishedNo, err := testRestClient.GetPublishedMessageForPublic(ctx, chat1Id, message1Id)
		require.NoError(t, err, "error in get published message")
		require.Nil(t, publishedNo)
	})
}

func TestEditMessageStillNotExists(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		dba *db.DB,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"

		const chat10Id = 12345
		const message1Id = 54321

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		ctx := context.Background()

		// chat still not exists
		const message1TextNew = "new message 1 edited"
		err := testRestClient.EditMessage(ctx, user1, chat10Id, message1Id, message1TextNew)
		require.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), fmt.Sprintf("code: %v", http.StatusTeapot)))

		// chat exists but the message still not exists
		chat12Id, err := testRestClient.CreateChat(ctx, user1, "chat1Name")
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat12Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		err = testRestClient.EditMessage(ctx, user1, chat12Id, message1Id, message1TextNew)
		require.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), fmt.Sprintf("code: %v", http.StatusTeapot)))
	})
}

func TestBlog(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user1Login = "admin1"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		const chat1Name = "new chat 1"

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name)
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		const message1Text = "new message 1"
		message1Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message1Text)
		require.NoError(t, err, "error in creating message")
		// await before chat editing
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		// probably not needed
		// it's old behaviour. just check for backward compatibility
		// actually just marking message as blog should be enough
		err = testRestClient.EditChat(ctx, user1, chat1Id, chat1Name, client.NewChatOptionBlog(true))
		require.NoError(t, err)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeMessagesReload &&
					e.UserId == user1 &&
					e.ChatId == chat1Id
			},
		}))

		const message2Text = "new message 2"
		message2Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message2Text)
		require.NoError(t, err, "error in creating message")

		err = testRestClient.MakeMessageBlogPost(ctx, user1, chat1Id, message1Id)
		require.NoError(t, err, "error in making message blog post")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		blogsW, err := testRestClient.SearchBlogs(ctx)
		require.NoError(t, err, "error in searching blog posts")
		blogs := blogsW.Items
		assert.Equal(t, 1, len(blogs))
		assert.Equal(t, chat1Id, blogs[0].Id)
		assert.Equal(t, chat1Name, blogs[0].Title)

		commentsW, err := testRestClient.SearchBlogComments(ctx, chat1Id)
		require.NoError(t, err, "error in searching blog comments")
		comments := commentsW.Items
		assert.Equal(t, 1, len(comments))
		assert.Equal(t, message2Id, comments[0].Id)
		assert.Equal(t, message2Text, comments[0].Content)

		testOutputEventsAccumulator.Clean()

		err = testRestClient.MakeMessageBlogPost(ctx, user1, chat1Id, message2Id)
		require.NoError(t, err, "error in making message blog post")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, true, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeMessageEdited &&
					e.UserId == user1 &&
					e.ChatId == chat1Id &&
					e.MessageNotification.Id == message1Id &&
					e.MessageNotification.Content == message1Text &&
					e.MessageNotification.Owner.Id == user1 &&
					e.MessageNotification.Owner.Login == user1Login &&
					e.MessageNotification.BlogPost == false
			},

			func(ee any) bool {
				e, ok := ee.(*dto.ChatEvent)
				return ok && e.EventType == dto.EventTypeMessageEdited &&
					e.UserId == user1 &&
					e.ChatId == chat1Id &&
					e.MessageNotification.Id == message2Id &&
					e.MessageNotification.Content == message2Text &&
					e.MessageNotification.Owner.Id == user1 &&
					e.MessageNotification.Owner.Login == user1Login &&
					e.MessageNotification.BlogPost == true
			},
		}))

		err = testRestClient.EditChat(ctx, user1, chat1Id, chat1Name, client.NewChatOptionBlog(false))
		require.NoError(t, err, "error in unmaking message blog post")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")
		blogsNewW, err := testRestClient.SearchBlogs(ctx)
		require.NoError(t, err, "error in searching blog posts")
		blogsNew := blogsNewW.Items
		assert.Equal(t, 0, len(blogsNew))
	})
}

func TestChatPaginate(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		dba *db.DB,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		lc fx.Lifecycle,
	) {
		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{}, nil)
		mockAaaClient.EXPECT().SearchGetUsers(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]*dto.User{}, 0, nil)

		const user1 int64 = 1
		const num = 1000
		const chatPrefix = "generated_chat"

		ctx := context.Background()

		var lastChatId int64
		var err error
		for i := 1; i <= num; i++ {
			lastChatId, err = testRestClient.CreateChat(ctx, user1, chatPrefix+utils.ToString(i))
			require.NoError(t, err, "error in creating chat")
			assert.True(t, lastChatId > 0)
		}
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		waitForChatExists(lgr, m, dba, lastChatId, user1, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		// get initial page
		resp1, _, err := testRestClient.GetChats(ctx, user1, client.NewChatGetOptionWithSize(40))
		require.NoError(t, err)
		assert.Equal(t, 40, len(resp1))
		assert.Equal(t, "generated_chat1000", resp1[0].Title)
		assert.Equal(t, "generated_chat999", resp1[1].Title)
		assert.Equal(t, "generated_chat998", resp1[2].Title)
		assert.Equal(t, "generated_chat961", resp1[39].Title)

		lastPinned := resp1[len(resp1)-1].Pinned
		lastId := resp1[len(resp1)-1].Id
		lastLastUpdateDateTime := resp1[len(resp1)-1].UpdateDateTime

		// get second page
		resp2, _, err := testRestClient.GetChats(ctx, user1, client.NewChatGetOptionWithSize(40), client.NewChatGetOptionWithStartsFromChatPinned(lastPinned), client.NewChatGetOptionWithStartsFromChatLastUpdateDateTime(lastLastUpdateDateTime), client.NewChatGetOptionWithStartsFromChatId(lastId))
		require.NoError(t, err)
		assert.Equal(t, 40, len(resp2))
		assert.Equal(t, "generated_chat960", resp2[0].Title)
		assert.Equal(t, "generated_chat959", resp2[1].Title)
		assert.Equal(t, "generated_chat958", resp2[2].Title)
		assert.Equal(t, "generated_chat921", resp2[39].Title)

		// get second page with search
		const searchString = "generated_chat96"
		resp2Search, _, err := testRestClient.GetChats(ctx, user1, client.NewChatGetOptionWithSize(40), client.NewChatGetOptionWithStartsFromChatPinned(lastPinned), client.NewChatGetOptionWithStartsFromChatLastUpdateDateTime(lastLastUpdateDateTime), client.NewChatGetOptionWithStartsFromChatId(lastId), client.NewChatGetOptionWithSearch(searchString))
		require.NoError(t, err)
		assert.Equal(t, 40, len(resp2Search))
		assert.Equal(t, "generated_chat960", resp2Search[0].Title)
		assert.Equal(t, "generated_chat959", resp2Search[1].Title)
	})
}

func TestChatFuzzySearch(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		dba *db.DB,
		aaaRestClient client.AaaRestClient,
		lc fx.Lifecycle,
	) {
		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{}, nil)
		mockAaaClient.EXPECT().SearchGetUsers(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]*dto.User{}, 1, nil)

		const user1 int64 = 1
		const chat1Name = "чат Опубликована платформа Node.js 25.0.0"

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name)
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		const chat2Name = "samsung"
		chat2Id, err := testRestClient.CreateChat(ctx, user1, chat2Name)
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat2Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		const searchString1 = "Опубликованный"
		resp1Search, _, err := testRestClient.GetChats(ctx, user1, client.NewChatGetOptionWithSearch(searchString1))
		require.NoError(t, err)
		assert.Equal(t, 1, len(resp1Search))
		assert.Equal(t, resp1Search[0].Title, chat1Name)

		const searchString2 = "публик"
		resp2Search, _, err := testRestClient.GetChats(ctx, user1, client.NewChatGetOptionWithSearch(searchString2))
		require.NoError(t, err)
		assert.Equal(t, 1, len(resp2Search))
		assert.Equal(t, resp2Search[0].Title, chat1Name)

		const searchString3 = "самсунгу"

		resp3Search, _, err := testRestClient.GetChats(ctx, user1, client.NewChatGetOptionWithSearch(searchString3))
		require.NoError(t, err)
		assert.Equal(t, 1, len(resp3Search))
		assert.Equal(t, resp3Search[0].Title, chat2Name)
	})
}

func TestMessagePaginate(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		dba *db.DB,
		aaaRestClient client.AaaRestClient,
		lc fx.Lifecycle,
	) {
		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{}, nil)

		const user1 int64 = 1
		const chat1Name = "new chat 1"
		const num = 500

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name)
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		const messagePrefix = "generated_message"

		var lastMessageId int64
		for i := 1; i <= num; i++ {
			lastMessageId, err = testRestClient.CreateMessage(ctx, user1, chat1Id, messagePrefix+utils.ToString(i))
			require.NoError(t, err, "error in creating message")
		}
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")
		waitForMessageExists(lgr, m, dba, chat1Id, lastMessageId, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		// get first page
		resp1, _, err := testRestClient.GetMessages(ctx, user1, chat1Id, client.NewMessageGetOptionWithSize(3), client.NewMessageGetOptionWithStartsFromItemId(6))
		require.NoError(t, err)
		assert.Equal(t, 3, len(resp1))
		assert.True(t, strings.HasPrefix(resp1[0].Content, "generated_message7")) // different from chat because of different way of generating test data
		assert.True(t, strings.HasPrefix(resp1[1].Content, "generated_message8"))
		assert.True(t, strings.HasPrefix(resp1[2].Content, "generated_message9"))
		assert.Equal(t, int64(7), resp1[0].Id)
		assert.Equal(t, int64(8), resp1[1].Id)
		assert.Equal(t, int64(9), resp1[2].Id)

		lastId := resp1[len(resp1)-1].Id

		// get second page
		resp2, _, err := testRestClient.GetMessages(ctx, user1, chat1Id, client.NewMessageGetOptionWithSize(3), client.NewMessageGetOptionWithStartsFromItemId(lastId))
		require.NoError(t, err)
		assert.Equal(t, 3, len(resp2))
		assert.True(t, strings.HasPrefix(resp2[0].Content, "generated_message10"))
		assert.True(t, strings.HasPrefix(resp2[1].Content, "generated_message11"))
		assert.True(t, strings.HasPrefix(resp2[2].Content, "generated_message12"))
		assert.Equal(t, int64(10), resp2[0].Id)
		assert.Equal(t, int64(11), resp2[1].Id)
		assert.Equal(t, int64(12), resp2[2].Id)

		const searchString = "generated_message10"
		// get second page with search
		resp2Search, _, err := testRestClient.GetMessages(ctx, user1, chat1Id, client.NewMessageGetOptionWithSize(3), client.NewMessageGetOptionWithStartsFromItemId(lastId), client.NewMessageGetOptionWithSearch(searchString))
		require.NoError(t, err)
		assert.Equal(t, 3, len(resp2Search))
		assert.True(t, strings.HasPrefix(resp2Search[0].Content, "generated_message10"))
		assert.True(t, strings.HasPrefix(resp2Search[1].Content, "generated_message11"))
		assert.True(t, strings.HasPrefix(resp2Search[2].Content, "generated_message12"))
	})
}

func TestMessageFuzzySearch(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		dba *db.DB,
		aaaRestClient client.AaaRestClient,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		const chat1Name = "new chat 1 src"
		const chat2Name = "new chat 1 dst"

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name, client.NewChatOptionResend(true))
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		chat2Id, err := testRestClient.CreateChat(ctx, user2, chat2Name)
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat2Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		const messageText1 = "сообщение Опубликована платформа Node.js 25.0.0"

		messageId1, err := testRestClient.CreateMessage(ctx, user1, chat1Id, messageText1)
		require.NoError(t, err, "error in creating message")
		waitForMessageExists(lgr, m, dba, chat1Id, messageId1, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		const message2Text = "samsung"

		messageId2, err := testRestClient.CreateMessage(ctx, user1, chat1Id, message2Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")
		waitForMessageExists(lgr, m, dba, chat1Id, messageId2, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		const message3Text = `Рабочей силы еще больше!Иран отвлекает от внутренней повестки. Меж тем, индекс hh.ru (соотношение резюме к вакансиям) в марте резко вырос – 11,4 против 9,8 в феврале. Кажется, вообще впервые такой резкий рост. Напомню, минимум был в июне 2024 - 3,1. Значение от 2 до 3,9 - дефицит соискателей,  4-7,9 - умеренный уровень конкуренции за рабочие места, 8,0–11,9 — высокий уровень конкуренции соискателей за рабочие места.В Москве как и по стране 11,4 (в феврале – 10). В Питере 12,1 (в феврале – 10,5). По профобластям дефицит по-прежнему только в розничной торговле (3,9), и то на грани.Еще в ноябре разбирал (тут и тут) как это все вяжется с данными Росстата по безработице (2,1%).@NewGosplanhttps://stats.hh.ru/---Работа Госплан 2.0`

		_, err = testRestClient.CreateMessage(ctx, user1, chat1Id, message3Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")
		waitForMessageExists(lgr, m, dba, chat1Id, messageId2, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		const message4Text = `BASSANOVA - GAUNTLET DOCS LEAKED:$ATOM 2.0WILL CRUSH INFLATION — THE ENDGAME IS HERE #ATOM #cosmosTHE SHOCK DROPCosmos Labs handed Gauntlet (Coinbase/Uniswap tier) the keys to REINVENT $ATOM.16 weeks. Results July. They found the broken model. #Tokenomics #DeFiTHE DIAGNOSIS IS BRUTALPhase 1 rips apart:•Who’s actually holding ATOM?•Why staking dropped off a cliff•How past inflation changes BROKE demand #Crypto #InflationATOM 2.0: ZERO INFLATION ENGINEFees > Security = 0% issuance. First L1 that PAYS YOU TO EXIST.ATOM becomes THE reserve asset for:•Gas - IBC settlement - Interchain Security #Web3 #L1THE KILLER MECHANICS•3-YEAR MELTDOWN: Gradual issuance death•Osmosis cash machine: DEX fees → ATOM buybacks (2.5% supply cap)•Rollup yield: Stake ATOM, secure L2s, collect fees #Staking #YieldTIMING IS NUCLEAR•Gauntlet drops truth bombs July 2026•CometBFT 10k+ TPS live Q2•Bithumb whales positioning NOW #Bullish #WhalesTHE DIRTY SECRETForum buried this RFP for months. Normies sleeping.Maxis knew. Institutions smelled blood. #Insider #FOMO$ATOM isn’t “just another L1.”It’s THE coordination layer eating Solana’s lunch. #CosmosHub #IBCStack or get rekt.RT if you’re loading $ATOM 🚀---crypto`
		_, err = testRestClient.CreateMessage(ctx, user1, chat1Id, message4Text)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")
		waitForMessageExists(lgr, m, dba, chat1Id, messageId2, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		const searchString1 = "Опубликованный"
		resp1Search, _, err := testRestClient.GetMessages(ctx, user1, chat1Id, client.NewMessageGetOptionWithSearch(searchString1))
		require.NoError(t, err)
		assert.Equal(t, 1, len(resp1Search))
		assert.Equal(t, resp1Search[0].Content, messageText1)

		const searchString2 = "публик"
		resp2Search, _, err := testRestClient.GetMessages(ctx, user1, chat1Id, client.NewMessageGetOptionWithSearch(searchString2))
		require.NoError(t, err)
		assert.Equal(t, 1, len(resp2Search))
		assert.Equal(t, resp2Search[0].Content, messageText1)

		const searchString3 = "самсунгу"
		resp3Search, _, err := testRestClient.GetMessages(ctx, user1, chat1Id, client.NewMessageGetOptionWithSearch(searchString3))
		require.NoError(t, err)
		assert.Equal(t, 1, len(resp3Search))
		assert.Equal(t, resp3Search[0].Content, message2Text)

		const searchString4 = "пастер"
		resp4Search, _, err := testRestClient.GetMessages(ctx, user1, chat1Id, client.NewMessageGetOptionWithSearch(searchString4))
		require.NoError(t, err)
		assert.Equal(t, 0, len(resp4Search))

		// user 1 adds user 2 to chat 1
		err = testRestClient.AddChatParticipants(ctx, user1, chat1Id, []int64{user2})
		require.NoError(t, err, "error in adding participant")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		// user 2 resends the message from chat 1 to chat 2
		message1ResentId, err := testRestClient.CreateMessage(ctx, user2, chat2Id, dto.NoMessageContent, client.NewMessageCreateOptionResend(chat1Id, messageId1))
		require.NoError(t, err, "error in resending message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")
		waitForMessageExists(lgr, m, dba, chat1Id, message1ResentId, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		// user 2 searches for the message
		resp22Search, _, err := testRestClient.GetMessages(ctx, user2, chat2Id, client.NewMessageGetOptionWithSearch(searchString1))
		require.NoError(t, err)
		assert.Equal(t, 1, len(resp22Search))
		assert.Equal(t, resp22Search[0].EmbedMessage.Text, messageText1)
		assert.Equal(t, resp22Search[0].Id, message1ResentId)
	})
}

func TestEventSendingOnUserProfileChange(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		aaaRestClient client.AaaRestClient,
		testEventsPublisher *producer.RabbitTestInputEventsPublisher,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user1LoginNew = "admin1New"
		const user1Avatar = "http://example.com/ava.jpg"
		const user2Login = "admin2"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		const chat1Name = "new chat 1"

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name, client.NewChatOptionParticipants(user2))
		require.NoError(t, err, "error in creating chat")
		assert.True(t, chat1Id > 0)
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		a := user1Avatar
		err = testEventsPublisher.Publish(ctx, dto.UserAccountEventChanged{
			User: &dto.User{
				Id:     user1,
				Login:  user1LoginNew,
				Avatar: &a,
			},
			EventType: dto.EventTypeUserAccountChanged,
		})
		require.NoError(t, err, "error in sending test event")

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeParticipantChanged &&
					e.UserId == user1 &&
					e.CoChattedParticipantNotification.Id == user1 &&
					e.CoChattedParticipantNotification.Login == user1LoginNew &&
					*e.CoChattedParticipantNotification.Avatar == a
			},
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeParticipantChanged &&
					e.UserId == user2 &&
					e.CoChattedParticipantNotification.Id == user1 &&
					e.CoChattedParticipantNotification.Login == user1LoginNew &&
					*e.CoChattedParticipantNotification.Avatar == a
			},
		}))
	})
}

func TestDeleteLeftoversFromDb(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		dba *db.DB,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)

		const chat1Name = "new chat 1"
		const chat2Name = "new chat 2"
		const chat3Name = "new chat 3"

		ctx := context.Background()

		// checking message deletion causes reaction deletion
		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name, client.NewChatOptionParticipants(user2))
		require.NoError(t, err, "error in creating chat")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		waitForChatExists(lgr, m, dba, chat1Id, user1, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		const messageText1 = "message 1"

		message1Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, messageText1)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")
		waitForMessageExists(lgr, m, dba, chat1Id, message1Id, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		const reaction = "😀"
		err = testRestClient.Reaction(ctx, user1, chat1Id, message1Id, reaction)
		require.NoError(t, err, "error in reacting on message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		err = testRestClient.DeleteMessage(ctx, user1, chat1Id, message1Id)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		// assert that the reaction is deleted along with message
		reactionExists, err := m.IsReactionExists(ctx, chat1Id, message1Id, reaction)
		require.NoError(t, err, "error in checking reaction")
		assert.False(t, reactionExists)

		// checking chat deletion causes message deletion
		chat2Id, err := testRestClient.CreateChat(ctx, user1, chat2Name, client.NewChatOptionParticipants(user2))
		require.NoError(t, err, "error in creating chat")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		waitForChatExists(lgr, m, dba, chat2Id, user1, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		existsUv21, err := m.IsChatUserViewExists(ctx, dba, chat2Id, user1)
		require.NoError(t, err, "error in checking chat user view")
		assert.True(t, existsUv21)

		existsC21, err := m.IsChatExists(ctx, dba, chat2Id)
		require.NoError(t, err, "error in checking chat common")
		assert.True(t, existsC21)

		const messageText2 = "message 2"

		message2Id, err := testRestClient.CreateMessage(ctx, user1, chat2Id, messageText2)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")
		waitForMessageExists(lgr, m, dba, chat2Id, message2Id, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		messageExists, err := m.IsMessageExists(ctx, dba, chat2Id, message2Id)
		require.NoError(t, err, "error in checking message")
		assert.True(t, messageExists)

		testOutputEventsAccumulator.Clean()

		err = testRestClient.DeleteChat(ctx, user1, chat2Id)
		require.NoError(t, err, "error in deleting chat")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatDeleted &&
					e.UserId == user1 &&
					e.ChatDeletedDto.Id == chat2Id
			},
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatDeleted &&
					e.UserId == user2 &&
					e.ChatDeletedDto.Id == chat2Id
			},
		}))

		testOutputEventsAccumulator.Clean()

		// assert that the message is deleted along with chat
		messageExists2, err := m.IsMessageExists(ctx, dba, chat2Id, message2Id)
		require.NoError(t, err, "error in checking message")
		assert.False(t, messageExists2)

		existsUv22, err := m.IsChatUserViewExists(ctx, dba, chat2Id, user1)
		require.NoError(t, err, "error in checking chat user view")
		assert.False(t, existsUv22)

		existsC22, err := m.IsChatExists(ctx, dba, chat2Id)
		require.NoError(t, err, "error in checking chat common")
		assert.False(t, existsC22)

		// checking blog's chat deletion causes blog deletion
		chat3Id, err := testRestClient.CreateChat(ctx, user1, chat3Name, client.NewChatOptionParticipants(user2), client.NewChatOptionBlog(true))
		require.NoError(t, err, "error in creating chat")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		waitForChatExists(lgr, m, dba, chat3Id, user1, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		const messageText3 = "message 3"

		message3Id, err := testRestClient.CreateMessage(ctx, user1, chat3Id, messageText3)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")
		waitForMessageExists(lgr, m, dba, chat3Id, message3Id, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		blogsNewW, err := testRestClient.SearchBlogs(ctx)
		require.NoError(t, err, "error in searching blog posts")
		blogsNew := blogsNewW.Items
		assert.Equal(t, 1, len(blogsNew))

		testOutputEventsAccumulator.Clean()

		err = testRestClient.DeleteChat(ctx, user1, chat3Id)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")

		// for sake waiting on all the events were applied
		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatDeleted &&
					e.UserId == user1 &&
					e.ChatDeletedDto.Id == chat3Id
			},
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatDeleted &&
					e.UserId == user2 &&
					e.ChatDeletedDto.Id == chat3Id
			},
		}))

		blogsNewW2, err := testRestClient.SearchBlogs(ctx)
		require.NoError(t, err, "error in searching blog posts")
		blogsNew2 := blogsNewW2.Items
		assert.Equal(t, 0, len(blogsNew2))

		messageExists33, err := m.IsMessageExists(ctx, dba, chat3Id, message3Id)
		require.NoError(t, err, "error in checking message")
		assert.False(t, messageExists33)
	})
}

func TestCleanDeletedUsersData(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		dba *db.DB,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		cleanService *tasks.CleanDeletedUserDataService,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user2 int64 = 2
		const user1Login = "admin1"
		const user2Login = "admin2"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockUser2 := dto.User{
			Id:               user2,
			Login:            user2Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1, &mockUser2}, nil)
		mockAaaClient.EXPECT().CheckAreUsersExists(mock.Anything, mock.Anything).Return([]dto.UserExists{{
			Exists: true,
			UserId: user1,
		}, {
			Exists: false,
			UserId: user2,
		}}, nil) // not exists

		const chat1Name = "new chat 1"
		const chat2Name = "new chat 2"

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name, client.NewChatOptionParticipants(user2))
		require.NoError(t, err, "error in creating chat")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")
		waitForChatExists(lgr, m, dba, chat1Id, user1, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		chat2Id, err := testRestClient.CreateChat(ctx, user2, chat2Name)
		require.NoError(t, err, "error in creating chat")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")
		waitForChatExists(lgr, m, dba, chat2Id, user2, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		const messageText1 = "message 1"
		const messageText2 = "message 2"

		message1Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, messageText1)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")
		waitForMessageExists(lgr, m, dba, chat1Id, message1Id, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		message2Id, err := testRestClient.CreateMessage(ctx, user2, chat2Id, messageText2)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")
		waitForMessageExists(lgr, m, dba, chat1Id, message2Id, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		testOutputEventsAccumulator.Clean()

		cuv11before, err := m.IsChatUserViewExists(ctx, dba, chat1Id, user1)
		require.NoError(t, err, "error in checking chat user view")
		require.True(t, cuv11before)

		cuv21before, err := m.IsChatUserViewExists(ctx, dba, chat1Id, user2)
		require.NoError(t, err, "error in checking chat user view")
		require.True(t, cuv21before)

		cuv22before, err := m.IsChatUserViewExists(ctx, dba, chat2Id, user2)
		require.NoError(t, err, "error in checking chat user view")
		require.True(t, cuv22before)

		cp21before, err := m.IsParticipantExists(ctx, dba, chat1Id, user2)
		require.NoError(t, err, "error in checking chat participant")
		require.True(t, cp21before)

		cp22before, err := m.IsParticipantExists(ctx, dba, chat2Id, user2)
		require.NoError(t, err, "error in checking chat participant")
		require.True(t, cp22before)

		urm2before, err := m.AreHasUnreadMessagesExists(ctx, dba, user2)
		require.NoError(t, err, "error in checking unread messages")
		require.True(t, urm2before)

		// do cleanup
		cleanService.DoJob(ctx)

		require.NoError(t, testOutputEventsAccumulator.AwaitForBufferContainsSpecifiedEvents(cfg.RabbitMQ.MaxWaitForEvents, false, []func(e any) bool{
			func(ee any) bool {
				e, ok := ee.(*dto.GlobalUserEvent)
				return ok && e.EventType == dto.EventTypeChatEdited &&
					e.UserId == user1 &&
					e.ChatNotification.ChatViewDto.Id == chat1Id &&
					e.ChatNotification.ChatViewDto.Title == chat1Name &&
					len(e.ChatNotification.Participants) == 1 && // in case race condition it's going to fail
					e.ChatNotification.Participants[0].Id == user1 &&
					e.ChatNotification.Participants[0].Login == user1Login
			},
		}))

		cuv11after, err := m.IsChatUserViewExists(ctx, dba, chat1Id, user1)
		require.NoError(t, err, "error in checking chat user view")
		require.True(t, cuv11after)

		cuv21after, err := m.IsChatUserViewExists(ctx, dba, chat1Id, user2)
		require.NoError(t, err, "error in checking chat user view")
		require.False(t, cuv21after)

		cuv22after, err := m.IsChatUserViewExists(ctx, dba, chat2Id, user2)
		require.NoError(t, err, "error in checking chat user view")
		require.False(t, cuv22after)

		cp21after, err := m.IsParticipantExists(ctx, dba, chat1Id, user2)
		require.NoError(t, err, "error in checking chat participant")
		require.False(t, cp21after)

		cp22after, err := m.IsParticipantExists(ctx, dba, chat2Id, user2)
		require.NoError(t, err, "error in checking chat participant")
		require.False(t, cp22after)

		urm2after, err := m.AreHasUnreadMessagesExists(ctx, dba, user2)
		require.NoError(t, err, "error in checking unread messages")
		require.False(t, urm2after)
	})
}

func TestCleanAbandonedChats(t *testing.T) {
	startAppFull(t, func(
		lgr *logger.LoggerWrapper,
		cfg *config.AppConfig,
		testRestClient *client.TestRestClient,
		admCl *kadm.Client,
		m *cqrs.CommonProjection,
		dba *db.DB,
		aaaRestClient client.AaaRestClient,
		testOutputEventsAccumulator *listener.TestOutputEventAccumulator,
		cleanService *tasks.CleanAnandonedChatsService,
		lc fx.Lifecycle,
	) {
		const user1 int64 = 1
		const user1Login = "admin1"

		mockUser1 := dto.User{
			Id:               user1,
			Login:            user1Login,
			Avatar:           nil,
			ShortInfo:        nil,
			LoginColor:       nil,
			LastSeenDateTime: nil,
			AdditionalData:   nil,
		}

		mockAaaClient := aaaRestClient.(*client.MockAaaRestClient)
		mockAaaClient.EXPECT().GetUsers(mock.Anything, mock.Anything).Return([]*dto.User{&mockUser1}, nil)

		const chat1Name = "new chat 1"

		ctx := context.Background()

		chat1Id, err := testRestClient.CreateChat(ctx, user1, chat1Name)
		require.NoError(t, err, "error in creating chat")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")
		waitForChatExists(lgr, m, dba, chat1Id, user1, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		const messageText1 = "message 1"

		message1Id, err := testRestClient.CreateMessage(ctx, user1, chat1Id, messageText1)
		require.NoError(t, err, "error in creating message")
		require.NoError(t, kafka.WaitForAllEventsProcessedChat(lgr, cfg, admCl, lc), "error in waiting for processing events")
		require.NoError(t, kafka.WaitForAllEventsProcessedUser(lgr, cfg, admCl, lc), "error in waiting for processing events")
		waitForMessageExists(lgr, m, dba, chat1Id, message1Id, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		err = m.UnsafeDeleteParticipantForTest(ctx, dba, chat1Id, user1)
		require.NoError(t, err, "error in deleting chat")

		existsC1before, err := m.IsChatExists(ctx, dba, chat1Id)
		require.NoError(t, err, "error in checking chat common")
		assert.True(t, existsC1before)

		testOutputEventsAccumulator.Clean()

		// do cleanup
		cleanService.DoJob(ctx)

		waitForChatNotExists(lgr, m, dba, chat1Id, cfg.Cqrs.SleepBeforePolling, cfg.Cqrs.PollingMaxTimes)

		existsC1after, err := m.IsChatExists(ctx, dba, chat1Id)
		require.NoError(t, err, "error in checking existence")
		require.False(t, existsC1after)
	})
}
