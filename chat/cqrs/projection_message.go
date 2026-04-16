package cqrs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"nkonev.name/chat/config"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/preview"
	"nkonev.name/chat/sanitizer"
	"nkonev.name/chat/utils"

	"github.com/PuerkitoBio/goquery"
	"github.com/georgysavva/scany/v2/sqlscan"
	"github.com/jackc/pgtype"
)

func (m *CommonProjection) OnMessageCreatedBatch(ctx context.Context, co db.CommonOperations, events []MessageCreated) error {
	chatIds := []int64{} // actually it's 1 chat id
	for _, e := range events {
		chatIds = append(chatIds, e.MessageCommoned.ChatId)
	}

	chatsExists, err := m.checkAreChatsExist(ctx, co, chatIds)
	if err != nil {
		return err
	}

	validMessageCreateds := []MessageCreated{}

	for _, event := range events {
		if chatsExists[event.MessageCommoned.ChatId] {
			validMessageCreateds = append(validMessageCreateds, event)
		} else {
			m.lgr.InfoContext(ctx, "Skipping MessageCreated because there is no chat", logger.AttributeChatId, event.MessageCommoned.ChatId)
		}
	}

	var messageIds = []int64{}
	var ownerIds = []int64{}
	var contents = []string{}
	var embeds = []pgtype.JSONB{}
	var fileItemUuids = []*string{}
	var dbChatIds = []int64{}
	var createdAts = []time.Time{}

	for _, event := range validMessageCreateds {
		messageIds = append(messageIds, event.MessageCommoned.Id)
		dbChatIds = append(dbChatIds, event.MessageCommoned.ChatId)
		ownerIds = append(ownerIds, event.AdditionalData.BehalfUserId)
		contents = append(contents, event.MessageCommoned.Content)

		var embed pgtype.JSONB
		if event.MessageCommoned.Embed != nil {
			err = embed.Set(event.MessageCommoned.Embed)
			if err != nil {
				return err
			}
		} else {
			embed.Status = pgtype.Null
		}
		embeds = append(embeds, embed)

		fileItemUuids = append(fileItemUuids, event.MessageCommoned.FileItemUuid)
		createdAts = append(createdAts, event.AdditionalData.CreatedAt)
	}
	_, err = co.ExecContext(ctx, `
		with input_data as (
			select * from unnest(
				 cast($1 as bigint[])
				,cast($2 as bigint[])
				,cast($3 as bigint[])
				,cast($4 as text[])
				,cast($5 as jsonb[])
				,cast($6 as varchar(36)[])
				,cast($7 as timestamp[])
			) as t (
				 message_id
				,chat_id
				,owner_id
				,content
				,embed
				,file_item_uuid
				,create_date_time
			)
		)
		insert into message(
			 id
			,chat_id
			,owner_id
			,content
			,embed
			,file_item_uuid
			,create_date_time
		) 
		select
			 idt.message_id
			,idt.chat_id
			,idt.owner_id
			,idt.content
			,idt.embed
			,idt.file_item_uuid
			,idt.create_date_time
		from input_data idt
		on conflict(chat_id, id) do update set 
		     owner_id = excluded.owner_id
		    ,content = excluded.content
			,embed = excluded.embed
			,file_item_uuid = excluded.file_item_uuid
		    ,create_date_time = excluded.create_date_time
	`, messageIds, dbChatIds, ownerIds, contents, embeds, fileItemUuids, createdAts)
	if err != nil {
		return err
	}
	m.lgr.InfoContext(ctx,
		"Handling message added",
		"message_ids", messageIds,
		"chat_ids", dbChatIds,
	)
	return nil
}

type MessageEditDto struct {
	isPinned, isPublished       bool
	pinnedCount, publishedCount int64
}

func (m *CommonProjection) OnMessageEdited(ctx context.Context, co db.CommonOperations, event *MessageEdited) (*MessageEditDto, error) {

	var pinnedCount, publishedCount int64

	chatExists, err := m.checkChatExists(ctx, co, event.MessageCommoned.ChatId)
	if err != nil {
		return nil, err
	}
	if !chatExists {
		m.lgr.InfoContext(ctx, "Skipping MessageEdited because there is no chat", logger.AttributeChatId, event.MessageCommoned.ChatId)
		return nil, nil
	}

	messageBlogPost, err := m.isMessageBlogPost(ctx, co, event.MessageCommoned.ChatId, event.MessageCommoned.Id)
	if err != nil {
		return nil, err
	}

	isMessagePinned, err := m.isMessagePinned(ctx, co, event.MessageCommoned.ChatId, event.MessageCommoned.Id)
	if err != nil {
		return nil, err
	}

	isMessagePublished, err := m.isMessagePublished(ctx, co, event.MessageCommoned.ChatId, event.MessageCommoned.Id)
	if err != nil {
		return nil, err
	}

	var embed pgtype.JSONB
	if event.MessageCommoned.Embed != nil {
		err = embed.Set(event.MessageCommoned.Embed)
		if err != nil {
			return nil, err
		}
	} else {
		embed.Status = pgtype.Null
	}

	_, err = co.ExecContext(ctx, `
			update message
			set	
			    content = $3
				, embed = $4
				, update_date_time = $5
				, file_item_uuid = $6
			where chat_id = $2 and id = $1 
		`, event.MessageCommoned.Id, event.MessageCommoned.ChatId, event.MessageCommoned.Content, embed, event.AdditionalData.CreatedAt, event.MessageCommoned.FileItemUuid)
	if err != nil {
		return nil, err
	}

	if messageBlogPost {
		_, err = m.refreshBlog(ctx, co, event.MessageCommoned.ChatId, event.AdditionalData.CreatedAt, nil)
		if err != nil {
			return nil, err
		}
	}

	if isMessagePinned {
		previewTxt := m.createMessagePinnedText(event.MessageCommoned.Content)

		_, err = co.ExecContext(ctx, `
				update message_pinned
				set	
					preview = $3
					, update_date_time = $4
				where chat_id = $2 and message_id = $1 
			`, event.MessageCommoned.Id, event.MessageCommoned.ChatId, previewTxt, event.AdditionalData.CreatedAt)
		if err != nil {
			return nil, err
		}

		pinnedCount, err = m.GetPinnedMessageCount(ctx, m.db, event.MessageCommoned.ChatId)
		if err != nil {
			return nil, err
		}
	}

	if isMessagePublished {
		previewTxt := m.createMessagePublishedText(event.MessageCommoned.Content)

		_, err = co.ExecContext(ctx, `
				update message_published
				set	
					preview = $3
					, update_date_time = $4
				where chat_id = $2 and message_id = $1 
			`, event.MessageCommoned.Id, event.MessageCommoned.ChatId, previewTxt, event.AdditionalData.CreatedAt)
		if err != nil {
			return nil, err
		}

		publishedCount, err = m.GetPublishedMessageCount(ctx, co, event.MessageCommoned.ChatId)
		if err != nil {
			return nil, err
		}
	}

	m.lgr.InfoContext(ctx,
		"Handling message edited",
		logger.AttributeMessageId, event.MessageCommoned.Id,
		logger.AttributeChatId, event.MessageCommoned.ChatId,
		logger.AttributeMessageId, event.MessageCommoned.Id,
	)
	return &MessageEditDto{
		isPinned:       isMessagePinned,
		isPublished:    isMessagePublished,
		pinnedCount:    pinnedCount,
		publishedCount: publishedCount,
	}, nil

}

func (m *CommonProjection) initializeMessageUnreadMultipleParticipants(ctx context.Context, tx *db.Tx, participantId int64, chatId int64) error {
	err := m.setUnreadMessages(ctx, tx, participantId, chatId, dto.NoId, SetUnreadedMessagesActionInitialize)
	if err != nil {
		return err
	}
	return nil
}

type MessageRemovedDto struct {
	promotedMessageId   *int64
	pinnedCount         int64
	publishedCount      int64
	wasMessagePinned    bool
	wasMessagePublished bool
}

func (m *CommonProjection) OnMessageRemoved(ctx context.Context, co db.CommonOperations, event *MessageDeleted) (*MessageRemovedDto, error) {
	var pinnedCount int64
	var promotedMessageId *int64

	var publishedCount int64

	messageBlogPost, err := m.isMessageBlogPost(ctx, co, event.ChatId, event.MessageId)
	if err != nil {
		return nil, err
	}

	wasMessagePublished, err := m.isMessagePublished(ctx, co, event.ChatId, event.MessageId)
	if err != nil {
		return nil, err
	}

	wasMessagePinned, err := m.isMessagePinned(ctx, co, event.ChatId, event.MessageId)
	if err != nil {
		return nil, err
	}

	var wasPromoted bool
	if wasMessagePinned {
		wasPromoted, err = m.isMessagePromoted(ctx, co, event.ChatId, event.MessageId)
		if err != nil {
			return nil, err
		}
	}

	_, err = co.ExecContext(ctx, `
			delete from message where (id, chat_id) = ($1, $2)
		`, event.MessageId, event.ChatId)
	if err != nil {
		return nil, err
	}

	if messageBlogPost {
		_, err = m.refreshBlog(ctx, co, event.ChatId, event.AdditionalData.CreatedAt, nil)
		if err != nil {
			return nil, err
		}
	}

	if wasMessagePinned {
		if wasPromoted {
			promotedMessageId, err = m.tryNominatePreviousToPromote(ctx, co, event.ChatId)
			if err != nil {
				return nil, err
			}
		}

		var errc error
		pinnedCount, errc = m.GetPinnedMessageCount(ctx, co, event.ChatId)
		if errc != nil {
			return nil, errc
		}
	}

	if wasMessagePublished {
		var errc error
		publishedCount, errc = m.GetPublishedMessageCount(ctx, co, event.ChatId)
		if errc != nil {
			return nil, errc
		}
	}

	return &MessageRemovedDto{
		pinnedCount:         pinnedCount,
		publishedCount:      publishedCount,
		promotedMessageId:   promotedMessageId,
		wasMessagePinned:    wasMessagePinned,
		wasMessagePublished: wasMessagePublished,
	}, nil
}

func (m *CommonProjection) setLastMessage(ctx context.Context, tx *db.Tx, chatId int64) error {
	_, err := tx.ExecContext(ctx, `
		with last_message as (
			select 
				m.id,
				m.owner_id, 
				nullif(trim(left(strip_tags(m.content), $2)), '') as content,
				nullif(trim(left(strip_tags(embed ->> 'embedMessageContent'), $2)), '') as embed_content
			from message m 
			where m.chat_id = $1 and m.id = (select max(mm.id) from message mm where mm.chat_id = $1)
		)
		UPDATE chat_common 
		SET 
			last_message_id = (select id from last_message),
			last_message_content = (select coalesce(content, embed_content) from last_message),
			last_message_owner_id = (select owner_id from last_message)
		WHERE id = $1;
	`, chatId, m.cfg.Cqrs.Projections.ChatUserView.LastMessageMaxTextDbPreviewSize)
	if err != nil {
		return fmt.Errorf("error during setting last message: %w", err)
	}
	return nil
}

func CanReadMessage(isParticipant bool) bool {
	return isParticipant
}

// see also updateParticipantMessageReadIdBatch()
func (m *CommonProjection) OnUserMessagesCreated(ctx context.Context, co db.CommonOperations, event *UserMessagesCreatedEvent) error {
	if len(event.MessageCreateds) == 0 {
		return nil
	}

	var myHighestMessageId int64
	var myDelta int

	for _, msg := range event.MessageCreateds {
		if msg.AdditionalData.BehalfUserId == event.UserId {
			if msg.Id > myHighestMessageId {
				myHighestMessageId = msg.Id
			}
		}
	}

	for _, msg := range event.MessageCreateds {
		if msg.AdditionalData.BehalfUserId != event.UserId {
			if msg.Id > myHighestMessageId {
				myDelta += 1
			}
		}
	}

	if myHighestMessageId > 0 {
		err := m.setUnreadMessages(ctx, co, event.UserId, event.ChatId, myHighestMessageId, SetUnreadedMessagesActionCalculateUnreadsFromTheProvidedMessage) // includes updateHasUnreads()
		if err != nil {
			return err
		}
	}

	if myDelta > 0 {
		// we don't really increase because we should be tolerant to duplicated processing
		err := m.setUnreadMessages(ctx, co, event.UserId, event.ChatId, dto.NoId, SetUnreadedMessagesActionCalculateUnreadsFromTheUsersLastSavedReadedMessage)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *CommonProjection) OnUserMessageDeleted(ctx context.Context, co db.CommonOperations, event *UserMessageDeletedEvent) error {
	err := m.setUnreadMessages(ctx, co, event.UserId, event.ChatId, dto.NoId, SetUnreadedMessagesActionCalculateUnreadsFromTheUsersLastSavedReadedMessage)
	if err != nil {
		return err
	}

	return nil
}

func (m *CommonProjection) checkMessageExists(ctx context.Context, co db.CommonOperations, chatId, messageId int64) (bool, error) {
	var messageExists bool
	err := sqlscan.Get(ctx, co, &messageExists, "select exists (select * from message where chat_id = $1 and id = $2)", chatId, messageId)
	if err != nil {
		return false, err
	}
	return messageExists, nil
}

func (m *CommonProjection) GetMessageOwner(ctx context.Context, chatId, messageId int64) (int64, error) {
	var ownerId int64
	err := sqlscan.Get(ctx, m.db, &ownerId, "select owner_id from message where (chat_id, id) = ($1, $2)", chatId, messageId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// there were no rows, but otherwise no error occurred
			return dto.NoOwner, nil
		} else {
			return 0, err
		}
	}
	return ownerId, nil
}

func (m *CommonProjection) GetLastMessageId(ctx context.Context, co db.CommonOperations, chatId int64) (int64, error) {
	var maxMessageId int64
	err := sqlscan.Get(ctx, co, &maxMessageId, `
		select coalesce(inn.max_id, 0) 
		from (select max(id) as max_id from message m where m.chat_id = $1) inn
		`, chatId)
	if err != nil {
		return 0, err
	}
	return maxMessageId, nil
}

func (m *EnrichingProjection) GetMessagesEnriched(ctx context.Context, behalfUserIds []int64, needCheckAuth, isForPublic bool, authForUserId *int64, chatId int64, size int32, startingFromItemId *int64, includeStartingFrom, reverse bool, searchString string, requestedMessageIds []int64, additionalUserIdToFetch []int64) ([]dto.MessageViewEnrichedDto, bool, []*dto.User, error) {
	type resDto struct {
		items           []dto.MessageViewEnrichedDto
		notAparticipant bool
		users           []*dto.User
	}

	if isForPublic && len(behalfUserIds) > 0 {
		return nil, false, nil, errors.New("Wrong invariant - isForPublic and more than 0 behalfUserIds")
	}

	if isForPublic && len(requestedMessageIds) > 1 {
		return nil, false, nil, errors.New("Unknown invariant - and more than 1 messageIds")
	}

	res, errOuter := db.TransactWithResult(ctx, m.cp.db, func(tx *db.Tx) (*resDto, error) {
		if needCheckAuth {
			if authForUserId != nil {
				participant, err := m.cp.IsParticipant(ctx, m.cp.db, *authForUserId, chatId)
				if err != nil {
					return nil, err
				}
				if !participant {
					return &resDto{
						items:           nil,
						notAparticipant: true,
					}, nil
				}
			} else {
				return nil, errors.New("Unknown invariant")
			}
		}

		searchString = sanitizer.TrimAmdSanitize(m.policy, searchString)

		const fakeUserId = dto.NonExistentUser
		if isForPublic {
			// to use below for getting GetChatsBasicExtended() and then get this chat by fakeUserId in enrichMessage()
			behalfUserIds = []int64{fakeUserId}
		}

		messages, err := m.cp.GetMessages(ctx, tx, chatId, size, startingFromItemId, includeStartingFrom, reverse, searchString, requestedMessageIds, behalfUserIds)
		if err != nil {
			m.lgr.ErrorContext(ctx, "Error getting messages", logger.AttributeError, err)
			return nil, err
		}

		if isForPublic && len(messages) != 0 {
			var messagesTmp = []dto.MessageDto{}
			msg := messages[0]
			if msg.Published { // here we check if the message published, if no - we gonna respond the empty slice
				messagesTmp = append(messagesTmp, msg)
			}
			messages = messagesTmp
		}

		messageIds := make([]int64, 0)
		messageIdMap := make(map[int64]struct{})
		for _, message := range messages {
			messageIdMap[message.Id] = struct{}{}
		}
		for messageId := range messageIdMap {
			messageIds = append(messageIds, messageId)
		}

		reactions, err := m.getReactions(ctx, tx, chatId, messageIds)
		if err != nil {
			return nil, fmt.Errorf("Got error during enriching messages with reactions: %v", err)
		}

		var usersSet = map[int64]bool{}
		var chatsPreSet = map[int64]bool{}
		for _, message := range messages {
			err = populateSets(message.Id, message.OwnerId, additionalUserIdToFetch, message.Embed, usersSet, chatsPreSet, chatId, reactions)
			if err != nil {
				return nil, err
			}
		}

		var chatsByUserIdByChatId map[int64]map[int64]*dto.BasicChatDtoExtended = map[int64]map[int64]*dto.BasicChatDtoExtended{}
		notAparticipant := false
		if isForPublic {
			notAparticipant = true
			chatsByUserIdByChatId, err = m.cp.GetChatsBasicExtended(ctx, tx, utils.SetMapIdBoolToSlice(chatsPreSet), []int64{fakeUserId})
			if err != nil {
				m.lgr.ErrorContext(ctx, "Error getting chat basic", logger.AttributeError, err)
				return nil, err
			}
		} else {
			chatsByUserIdByChatId, err = m.cp.GetChatsBasicExtended(ctx, tx, utils.SetMapIdBoolToSlice(chatsPreSet), behalfUserIds)
			if err != nil {
				m.lgr.ErrorContext(ctx, "Error getting chat basic", logger.AttributeError, err)
				return nil, err
			}
		}

		// it's ok because we have 1 chat in the both cases
		areAdmins, err := m.cp.getAreAdminsOfUserIds(ctx, tx, behalfUserIds, chatId)
		if err != nil {
			return nil, err
		}

		users, err := m.aaaRestClient.GetUsers(ctx, utils.SetMapIdBoolToSlice(usersSet))
		if err != nil {
			m.lgr.WarnContext(ctx, "unable to get users", logger.AttributeError, err)
		}

		usersMap := utils.ToMap(users)

		messagesEnriched := make([]dto.MessageViewEnrichedDto, 0, len(messages))
		for _, mm := range messages {
			bloggingAllowed := IsBloggingAllowed(m.cfg, getUserPermissions(usersMap, mm.BehalfUserId))

			me, err := enrichMessage(
				ctx, m.lgr,
				m.cfg,
				mm,
				chatId,
				usersMap,
				chatsByUserIdByChatId,
				reactions,
				mm.BehalfUserId,
				areAdmins,
				!notAparticipant,
				bloggingAllowed,
			)
			if err != nil {
				return nil, err
			}
			messagesEnriched = append(messagesEnriched, *me)
		}
		return &resDto{
			items:           messagesEnriched,
			notAparticipant: notAparticipant,
			users:           users,
		}, nil
	})

	if errOuter != nil {
		return nil, false, nil, errOuter
	}

	return res.items, res.notAparticipant, res.users, nil
}

func getUserPermissions(usersMap map[int64]*dto.User, behalfUserId int64) []string {
	user := usersMap[behalfUserId]
	if user == nil || user.AdditionalData == nil {
		return []string{}
	}

	return user.Permissions
}

func IsBloggingAllowed(cfg *config.AppConfig, userPermissions []string) bool {
	if !cfg.Blog.RestrictCreateBlog {
		return true
	}

	return slices.Contains(userPermissions, dto.CAN_CREATE_BLOG)
}

func populateSets(messageId, messageOwnerId int64, additionalUserIdToFetch []int64, embed dto.Embeddable, usersSet map[int64]bool, chatsPreSet map[int64]bool, currentChatId int64, reactions map[int64][]dto.ReactionDto) error {
	usersSet[messageOwnerId] = true

	for _, au := range additionalUserIdToFetch {
		usersSet[au] = true
	}

	chatsPreSet[currentChatId] = true

	if embed != nil {
		switch typed := embed.(type) {
		case *dto.EmbedReply:
			var embeddedMessageReplyOwnerId = typed.OwnerId
			usersSet[embeddedMessageReplyOwnerId] = true
		case *dto.EmbedResend:
			var embeddedMessageResendOwnerId = typed.OwnerId
			usersSet[embeddedMessageResendOwnerId] = true
			var embeddedMessageResendChatId = typed.ChatId
			chatsPreSet[embeddedMessageResendChatId] = true
		default:
			return fmt.Errorf("Unknown type in populateSets: %T", typed)
		}
	}

	takeOnAccountReactions(messageId, usersSet, reactions)

	return nil
}

func enrichMessage(
	ctx context.Context, lgr *logger.LoggerWrapper, cfg *config.AppConfig,
	m dto.MessageDto,
	chatId int64,
	users map[int64]*dto.User,
	chatsByUserIdByChatId map[int64]map[int64]*dto.BasicChatDtoExtended,
	reactions map[int64][]dto.ReactionDto,
	behalfUserId int64,
	areAdmins map[int64]bool,
	isParticipant bool,
	bloggingIsAllowed bool,
) (*dto.MessageViewEnrichedDto, error) {
	me := dto.MessageViewEnrichedDto{
		Id:      m.Id,
		ChatId:  chatId,
		OwnerId: m.OwnerId,
		// no need to patchStorageUrlToPreventCachingVideo because there is no video html tags
		Content:        m.Content,
		BlogPost:       m.BlogPost,
		UpdateDateTime: m.UpdateDateTime,
		CreateDateTime: m.CreateDateTime,
		Owner:          users[m.OwnerId],
		BehalfUserId:   behalfUserId,
		FileItemUuid:   m.FileItemUuid,
		Pinned:         m.Pinned,
		Published:      m.Published,
	}

	chatsBehalfUser := chatsByUserIdByChatId[behalfUserId]
	embed, err := makeEmbed(m.Embed, users, chatsBehalfUser)
	if err != nil {
		return nil, err
	}
	me.EmbedMessage = embed

	rl := reactions[m.Id]
	me.Reactions = makeReactions(users, rl)

	chat := chatsBehalfUser[chatId]
	if chat == nil {
		return nil, fmt.Errorf("Logical error during enriching messages not found chat by chatId = %v, userId = %v", chatId, behalfUserId)
	}

	setMessagePersonalizedFields(&me, chat.TetATet, chat.IsBlog, chat.RegularParticipantCanPublishMessage, chat.RegularParticipantCanPinMessage, chat.RegularParticipantCanWriteMessage, areAdmins[behalfUserId], behalfUserId, isParticipant, bloggingIsAllowed)

	return &me, nil
}

func setMessagePersonalizedFields(copied *dto.MessageViewEnrichedDto, chatTetATet, chatIsBlog, chatRegularParticipantCanPublishMessage, chatRegularParticipantCanPinMessage, chatCanWriteMessage, chatIsAdmin bool, participantId int64, isParticipant bool, bloggingIsAllowed bool) {
	canWriteMessage := CanWriteMessage(isParticipant, chatIsAdmin, chatCanWriteMessage)

	copied.CanEdit = CanEditMessage(participantId, copied.OwnerId, copied.EmbedMessage != nil, copied.GetEmbedTypeSafe(), canWriteMessage)
	copied.CanSyncEmbed = CanSyncEmbedMessage(participantId, copied.OwnerId, copied.EmbedMessage != nil, canWriteMessage)
	copied.CanDelete = CanDeleteMessage(participantId, copied.OwnerId, canWriteMessage)
	copied.CanPublish = CanPublishMessage(chatRegularParticipantCanPublishMessage, chatIsAdmin, copied.OwnerId, participantId)
	copied.CanPin = CanPinMessage(chatRegularParticipantCanPinMessage, chatIsAdmin)

	copied.CanMakeBlogPost = CanMakeMessageBlogPost(chatIsAdmin, chatTetATet, copied.BlogPost, chatIsBlog, bloggingIsAllowed)
}

// We use pure functions for authorization, for sake simplicity and composability
func CanWriteMessage(isParticipant, chatIsAdmin, chatCanWriteMessage bool) bool {
	return isParticipant && (isChatAdminInternal(chatIsAdmin) || canWriteMessageInternal(chatCanWriteMessage))
}

func isChatAdminInternal(a bool) bool {
	return a
}

func canWriteMessageInternal(chatCanWriteMessage bool) bool {
	return chatCanWriteMessage
}

func CanEditMessage(behalfParticipantId int64, messageOwnerId int64, hasEmbed bool, embedTypeSafe string, canWriteMessage bool) bool {
	return ((messageOwnerId == behalfParticipantId) && (!hasEmbed || embedTypeSafe != dto.EmbedMessageTypeResend)) && canWriteMessage
}

func CanSyncEmbedMessage(behalfParticipantId int64, messageOwnerId int64, hasEmbed bool, canWriteMessage bool) bool {
	return messageOwnerId == behalfParticipantId && hasEmbed && canWriteMessage
}

func CanDeleteMessage(behalfParticipantId int64, messageOwnerId int64, canWriteMessage bool) bool {
	return messageOwnerId == behalfParticipantId && canWriteMessage
}

func CanPublishMessage(chatRegularParticipantCanPublishMessage, chatIsAdmin bool, messageOwnerId, behalfUserId int64) bool {
	return isChatAdminInternal(chatIsAdmin) || (canPublishMessageInternal(chatRegularParticipantCanPublishMessage) && messageOwnerId == behalfUserId)
}

func CanPinMessage(chatRegularParticipantCanPinMessage, chatIsAdmin bool) bool {
	return isChatAdminInternal(chatIsAdmin) || canPinMessageInternal(chatRegularParticipantCanPinMessage)
}

func canPublishMessageInternal(chatRegularParticipantCanPublishMessage bool) bool {
	return chatRegularParticipantCanPublishMessage
}

func canPinMessageInternal(chatRegularParticipantCanPinMessage bool) bool {
	return chatRegularParticipantCanPinMessage
}

func (m *CommonProjection) ChatHasMessages(ctx context.Context, co db.CommonOperations, chatId int64) (bool, error) {
	var has bool
	err := sqlscan.Get(ctx, co, &has, "select exists(select * from message m where chat_id = $1 limit 1)", chatId)
	if err != nil {
		return false, err
	}
	return has, nil
}

func (m *CommonProjection) GetMessageDataForAuthorization(ctx context.Context, co db.CommonOperations, userId, chatId, messageId int64) (dto.MessageAuthorizationData, error) {
	d := dto.MessageAuthorizationData{}
	// it's ok if message is not found - sql handles it
	err := sqlscan.Get(ctx, co, &d, `
		with
		provided as (
			select 
				 cast($2 as bigint) as chat_id
				,cast($3 as bigint) as message_id
		),
		chat_participant_row as (
			SELECT user_id, chat_id, chat_admin FROM chat_participant WHERE user_id = $1 AND chat_id = $2 LIMIT 1
		),
		chat_info as (
			select * from chat_common where id = $2
		),
		message_info as (
			select * from message m where chat_id = $2 and id = $3
		)
		SELECT
			 cc.id is not null as is_chat_found
			,mm.id is not null as is_message_found
			,exists(SELECT * FROM chat_participant_row) as is_chat_participant
			,exists(SELECT * FROM chat_participant_row WHERE chat_admin) as is_chat_admin
			,coalesce(cc.regular_participant_can_write_message, false) as chat_can_write_message
			,coalesce(cc.tet_a_tet, false) as chat_is_tet_a_tet
			,(mm.id is not null) and (mm.embed is not null) as message_has_embed
			,coalesce(mm.owner_id, $4) as message_owner_id
			,coalesce(mm.embed ->> 'embedMessageType', $5) as message_embed_type
			,coalesce(mm.blog_post, false) as is_message_blog_post
			,coalesce(cc.regular_participant_can_pin_message, false) as chat_can_pin_message
			,coalesce(cc.regular_participant_can_publish_message, false) as chat_can_publish_message
			,b.id is not null as chat_is_blog
		FROM provided pr
		LEFT JOIN chat_info cc on pr.chat_id = cc.id
		LEFT JOIN message_info mm ON pr.message_id = mm.id
		left join blog b on cc.id = b.id
	`, userId, chatId, messageId, dto.NoOwner, dto.EmbedMessageTypeNone)
	if err != nil {
		return d, err
	}
	return d, nil
}

func (m *CommonProjection) GetMessageDataForAuthorizationMessageCreatedBatch(ctx context.Context, co db.CommonOperations, userIds []int64, chatId int64) (map[int64]dto.MessageAuthorizationDataBatch, error) {
	d := []dto.MessageAuthorizationDataBatch{}
	// it's ok if message is not found - sql handles it
	err := sqlscan.Select(ctx, co, &d, `
		with requested_participants as (
			select * from unnest(cast ($1 as bigint[])) as t(user_id)
		)	
		select 
			(cp.user_id is not null) as is_chat_participant,
			rp.user_id,
			coalesce(cc.tet_a_tet, false) as chat_is_tet_a_tet,
			coalesce(cp.chat_admin, false) as is_chat_admin,
			coalesce(cc.regular_participant_can_write_message, false) chat_can_write_message
		from requested_participants rp 
		left join chat_participant cp on (rp.user_id = cp.user_id and cp.chat_id = $2)
		cross join (select * from chat_common where id = $2) cc
	`, userIds, chatId)
	if err != nil {
		return nil, err
	}

	res := map[int64]dto.MessageAuthorizationDataBatch{}
	for _, itm := range d {
		res[itm.UserId] = itm
	}

	return res, nil
}

func getDeletedUser(id int64) *dto.User {
	return &dto.User{Login: fmt.Sprintf("deleted_user_%v", id), Id: id}
}

func makeEmbed(
	srcEmbed dto.Embeddable,
	users map[int64]*dto.User,
	chatsBehalfUserByChatId map[int64]*dto.BasicChatDtoExtended,
) (*dto.EmbedMessageResponse, error) {
	if srcEmbed != nil {
		switch typed := srcEmbed.(type) {
		case *dto.EmbedReply:
			embeddedUser := users[typed.OwnerId]
			return &dto.EmbedMessageResponse{
				Id:        typed.MessageId,
				Text:      typed.MessageContent,
				EmbedType: string(typed.GetType()),
				Owner:     embeddedUser,
			}, nil
		case *dto.EmbedResend:
			embeddedUser := users[typed.OwnerId]
			var embedChatName *string = nil
			var isParticipant bool

			basicEmbeddedChat := chatsBehalfUserByChatId[typed.ChatId]
			if basicEmbeddedChat != nil { // basicEmbeddedChat can be deleted
				if !basicEmbeddedChat.TetATet {
					embedChatName = &basicEmbeddedChat.Title
				}
				isParticipant = basicEmbeddedChat.BehalfUserIsParticipant
			}

			return &dto.EmbedMessageResponse{
				Id:            typed.MessageId,
				ChatId:        &typed.ChatId,
				ChatName:      embedChatName,
				Text:          typed.MessageContent,
				EmbedType:     string(typed.GetType()),
				Owner:         embeddedUser,
				IsParticipant: isParticipant,
			}, nil
		default:
			return nil, fmt.Errorf("Unknown type in setEmbed: %T", typed)
		}
	}

	return nil, nil
}

func (m *CommonProjection) GetMessages(ctx context.Context, co db.CommonOperations, chatId int64, size int32, startingFromItemId *int64, includeStartingFrom, reverse bool, searchString string, messageIds []int64, behaldUserIds []int64) ([]dto.MessageDto, error) {
	type messageDto struct {
		Id             int64        `db:"id"`
		OwnerId        int64        `db:"owner_id"`
		BehalfUserId   int64        `db:"behalf_user_id"`
		Content        string       `db:"content"`
		BlogPost       bool         `db:"blog_post"`
		Embed          pgtype.JSONB `db:"embed"`
		CreateDateTime time.Time    `db:"create_date_time"`
		UpdateDateTime *time.Time   `db:"update_date_time"`
		FileItemUuid   *string      `db:"file_item_uuid"`
		Pinned         bool         `db:"pinned"`
		Published      bool         `db:"published"`
	}

	if startingFromItemId != nil && len(messageIds) != 0 {
		return nil, fmt.Errorf("wrong invariant: both startingFromItemId and messageIds provided")
	}

	if size == dto.NoSize && len(messageIds) == 0 {
		return nil, fmt.Errorf("wrong invariant: NoSize requires enumerated message ids")
	}

	mar := []dto.MessageDto{}
	ma := []messageDto{}

	queryArgs := []any{chatId, behaldUserIds}

	limitClause := ""
	if size != dto.NoSize {
		queryArgs = append(queryArgs, size)
		limitClause = fmt.Sprintf("limit $%d", len(queryArgs))
	}

	order := ""
	nonEquality := ""
	if reverse {
		order = "desc"
		if includeStartingFrom {
			nonEquality = "<="
		} else {
			nonEquality = "<"
		}
	} else {
		order = "asc"
		if includeStartingFrom {
			nonEquality = ">="
		} else {
			nonEquality = ">"
		}
	}

	conditionClause := ""

	paginationKeyset := ""
	if startingFromItemId != nil {
		queryArgs = append(queryArgs, *startingFromItemId)
		paginationKeyset = fmt.Sprintf(` and m.id %s $%d `, nonEquality, len(queryArgs))

		conditionClause = paginationKeyset
	}

	var searchClause string
	if len(searchString) > 0 {
		searchClause = " and ("

		queryArgs = append(queryArgs, "%"+searchString+"%")
		searchClause += fmt.Sprintf(" m.fts_all_content::text ilike $%d ", len(queryArgs))
		searchClause += " or "

		queryArgs = append(queryArgs, searchString)
		searchClause += fmt.Sprintf(`
		exists (
			select 1 from (select * from (select unnest(tsvector_to_array(m.fts_all_content))) t(av)) inq
			where
				word_similarity( inq.av, plainto_tsquery('russian', $%d)::text ) > 0.8
				or word_similarity( cyrillic_transliterate(inq.av), cyrillic_transliterate(plainto_tsquery('russian', $%d)::text) ) > 0.8
		) `, len(queryArgs), len(queryArgs))

		searchClause += " ) "
	}

	orderClause := fmt.Sprintf(" order by m.id %s ", order)

	if len(messageIds) != 0 {
		messageIdV := messageIds
		queryArgs = append(queryArgs, messageIdV)
		messageIdClause := fmt.Sprintf(" and m.id = any($%d) ", len(queryArgs))

		conditionClause = messageIdClause
		orderClause += ", bh.behalf_user_id "
	}

	err := sqlscan.Select(ctx, co, &ma, fmt.Sprintf(`
			with requested_behalfs as (
				select * from unnest(cast ($2 as bigint[])) as t(behalf_user_id)
			)
			select 
			    m.id,
			    m.owner_id,
				bh.behalf_user_id,
			    m.content,
			    m.blog_post,
				m.embed,
				m.create_date_time,
			    m.update_date_time,
			    m.file_item_uuid,
				m.pinned,
				m.published
			from message m
			cross join requested_behalfs bh
			where m.chat_id = $1 %s 
			%s
			%s 
			%s
		`, conditionClause, searchClause, orderClause, limitClause),
		queryArgs...)

	if err != nil {
		return mar, err
	}

	for i, mm := range ma {
		mc := dto.MessageDto{
			Id:             mm.Id,
			OwnerId:        mm.OwnerId,
			BehalfUserId:   mm.BehalfUserId,
			Content:        mm.Content,
			BlogPost:       mm.BlogPost,
			CreateDateTime: mm.CreateDateTime,
			UpdateDateTime: mm.UpdateDateTime,
			FileItemUuid:   mm.FileItemUuid,
			Pinned:         mm.Pinned,
			Published:      mm.Published,
		}

		embeddable, err := makeEmbedddable(mm.Embed)
		if err != nil {
			return mar, fmt.Errorf("error during mapping on index %d: %w", i, err)
		}
		mc.Embed = embeddable

		mar = append(mar, mc)
	}

	return mar, nil
}

func makeEmbedddable(embedJsonb pgtype.JSONB) (dto.Embeddable, error) {
	if embedJsonb.Status == pgtype.Present {
		var typer dto.EmbedTyper
		err := embedJsonb.AssignTo(&typer)
		if err != nil {
			return nil, fmt.Errorf("error during mapping %w", err)
		}

		switch typer.Type {
		case dto.EmbedMessageTypeReply:
			var erpl dto.EmbedReply
			err = embedJsonb.AssignTo(&erpl)
			if err != nil {
				return nil, fmt.Errorf("error during mapping: %w", err)
			}
			return &erpl, nil
		case dto.EmbedMessageTypeResend:
			var eres dto.EmbedResend
			err = embedJsonb.AssignTo(&eres)
			if err != nil {
				return nil, fmt.Errorf("error during mapping: %w", err)
			}
			return &eres, nil
		default:
			return nil, fmt.Errorf("Unknown type in GetMessages: %v", typer.Type)
		}
	}
	return nil, nil
}

func (m *CommonProjection) GetMessageBasic(ctx context.Context, co db.CommonOperations, chatId, messageId int64) (*dto.MessageBasic, error) {
	var msg dto.MessageBasic
	err := sqlscan.Get(ctx, co, &msg, `
	select m.id, m.owner_id, m.content, m.blog_post, m.published, m.pinned, m.file_item_uuid
	from message m where m.chat_id = $1 and m.id = $2
	`, chatId, messageId)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (m *CommonProjection) GetMessageEmbed(ctx context.Context, co db.CommonOperations, chatId, messageId int64) (dto.Embeddable, error) {
	var embed pgtype.JSONB
	err := sqlscan.Get(ctx, co, &embed, `
	select m.embed
	from message m where m.chat_id = $1 and m.id = $2
	`, chatId, messageId)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	embeddable, err := makeEmbedddable(embed)
	if err != nil {
		return nil, fmt.Errorf("error during mapping: %w", err)
	}

	return embeddable, nil
}

func (m *CommonProjection) GetMessageWithEmbed(ctx context.Context, co db.CommonOperations, chatId, messageId int64) (*dto.MessageWithEmbed, error) {
	type messageDto struct {
		Id      int64        `db:"id"`
		OwnerId int64        `db:"owner_id"`
		Content string       `db:"content"`
		Embed   pgtype.JSONB `db:"embed"`
	}

	var msg messageDto
	err := sqlscan.Get(ctx, co, &msg, `
	select m.id, m.owner_id, m.content, m.embed
	from message m where m.chat_id = $1 and m.id = $2
	`, chatId, messageId)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	embeddable, err := makeEmbedddable(msg.Embed)
	if err != nil {
		return nil, fmt.Errorf("error during mapping: %w", err)
	}

	return &dto.MessageWithEmbed{
		Id:      msg.Id,
		OwnerId: msg.OwnerId,
		Content: msg.Content,
		Embed:   embeddable,
	}, nil
}

func (m *CommonProjection) IsMessageExists(ctx context.Context, co db.CommonOperations, chatId, messageId int64) (bool, error) {
	var exists bool
	err := sqlscan.Get(ctx, co, &exists, `
	select exists (select * from message m where m.chat_id = $1 and m.id = $2)
	`, chatId, messageId)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (m *CommonProjection) FindMessageByFileItemUuid(ctx context.Context, chatId, userId int64, fileItemUuid string) (*dto.MessageId, error) {
	participant, err := m.IsParticipant(ctx, m.db, userId, chatId)
	if err != nil {
		return nil, err
	}
	if !participant {
		return nil, NewUnauthorizedError(fmt.Sprintf("user %v is not a participant of chat %v", userId, chatId))
	}

	var messageId int64
	err = sqlscan.Get(ctx, m.db, &messageId, `
		select id from message where chat_id = $1 AND file_item_uuid = $2 or content ilike '%' || $2 || '%' order by id limit 1	
	`, chatId, fileItemUuid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// there were no rows, but otherwise no error occurred
			return &dto.MessageId{dto.FileItemUuidMessageNotFoundId}, nil
		}
		return nil, err
	}

	return &dto.MessageId{messageId}, nil
}

func (m *EnrichingProjection) MessageFilter(ctx context.Context, co db.CommonOperations, behalfUserId, chatId int64, searchString string, messageId int64) (bool, error) {
	participant, err := m.cp.IsParticipant(ctx, co, behalfUserId, chatId)
	if err != nil {
		return false, err
	}
	if !participant {
		return false, NewUnauthorizedError(fmt.Sprintf("user %v is not a participant of chat %v", behalfUserId, chatId))
	}

	searchString = sanitizer.TrimAmdSanitize(m.policy, searchString)

	searchStringWithPercents := "%" + searchString + "%"

	var found bool
	err = sqlscan.Get(ctx, co, &found, "SELECT EXISTS (SELECT * FROM message m WHERE m.chat_id = $1 AND m.id = $2 AND strip_tags(m.content) ILIKE $3)", chatId, messageId, searchStringWithPercents)
	if err != nil {
		return false, err
	}

	return found, nil
}

func (m *EnrichingProjection) GetReadMessageUsers(ctx context.Context, userId int64, chatId int64, messageId int64, size int32, offset int64) (*dto.MessageReadResponse, error) {
	type result struct {
		userIds []int64
		count   int64
		msg     *dto.MessageBasic
	}

	txRes, errOuter := db.TransactWithResult(ctx, m.cp.db, func(tx *db.Tx) (*result, error) {
		if participant, err := m.cp.IsParticipant(ctx, tx, userId, chatId); err != nil {
			m.lgr.ErrorContext(ctx, "Error during checking participant")
			return nil, err
		} else if !participant {
			return nil, NewUnauthorizedError(fmt.Sprintf("User %v is not participant of chat %v, skipping", userId, chatId))
		}

		userIds, err := m.getParticipantsRead(ctx, tx, chatId, messageId, size, offset)
		if err != nil {
			return nil, err
		}

		count, err := m.getParticipantsReadCount(ctx, tx, chatId, messageId)
		if err != nil {
			return nil, err
		}

		msg, err := m.cp.GetMessageBasic(ctx, tx, chatId, messageId)
		if err != nil {
			return nil, err
		}

		return &result{
			userIds: userIds,
			count:   count,
			msg:     msg,
		}, nil
	})
	if errOuter != nil {
		return nil, errOuter
	}

	usersToGet := map[int64]bool{}
	for _, u := range txRes.userIds {
		usersToGet[u] = true
	}
	if txRes.msg != nil {
		usersToGet[txRes.msg.OwnerId] = true
	}

	users, err := m.aaaRestClient.GetUsers(ctx, utils.SetMapIdBoolToSlice(usersToGet))
	if err != nil {
		return nil, err
	}
	userMap := utils.ToMap(users)

	usersToReturn := []*dto.User{}
	var anOwnerLogin string

	for _, usId := range txRes.userIds {
		us, ok := userMap[usId]
		if ok {
			usersToReturn = append(usersToReturn, us)
			if txRes.msg != nil && us.Id == txRes.msg.OwnerId {
				anOwnerLogin = us.Login
			}
		}
	}

	var text string
	if txRes.msg != nil {
		text = txRes.msg.Content
	}
	previewTxt := preview.CreateMessagePreview(m.stripAllTags, m.cfg.Message.PreviewMaxTextSize, text, anOwnerLogin)

	return &dto.MessageReadResponse{
		ParticipantsWrapper: dto.ParticipantsWrapper{
			Data:  usersToReturn,
			Count: txRes.count,
		},
		Text: previewTxt,
	}, nil
}

func (m *CommonProjection) AreHasUnreadMessagesExists(ctx context.Context, co db.CommonOperations, userId int64) (bool, error) {
	var t bool
	err := sqlscan.Get(ctx, co, &t, "select exists(select u.* from has_unread_messages u where u.user_id = $1)", userId)
	if err != nil {
		return false, err
	}
	return t, nil
}

// see also cqrs/event_handler.go
func (m *EnrichingProjection) parseMentionUserIdsFromMessageHtml(ctx context.Context, msg string) ([]int64, bool, bool) {
	ret := []int64{}

	var hasHere, hasAll bool

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(msg))
	if err != nil {
		m.lgr.WarnContext(ctx, "Unable to read html", logger.AttributeError, err)
		return ret, false, false
	}

	doc.Find("a, span").Each(func(i int, s *goquery.Selection) { // span is for @all, @here
		maybeA := s.First()

		if maybeA != nil && maybeA.HasClass("mention") {
			idS, ok := maybeA.Attr("data-id")
			if !ok {
				m.lgr.WarnContext(ctx, "a with class mention has no data-id")
			} else {
				id, errP := utils.ParseInt64(idS)
				if errP != nil {
					m.lgr.WarnContext(ctx, fmt.Sprintf("unable to parse user id from data-id: '%s'", idS))
				} else {
					switch id {
					case dto.AllUsers:
						hasAll = true
					case dto.HereUsers:
						hasHere = true
					default:
						ret = append(ret, id)
					}
				}
			}
		}
	})

	return ret, hasHere, hasAll
}
