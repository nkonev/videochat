package cqrs

// In general, to avoid race conditions, we should avoid relying on database here, in command_handler.
// Invoking db here, we can get an old data and make wrong decisions.
// The best place to perform checks against database is the projection side.
// In sake optimization here we have as an exception a few db calls.
// See comments about it in TestUnreads()
// Also, in order to keep these command's response times fast we should avoid iterations over db rows here. The best place for it is event_handler, projection.

// To have some happens-before relationship garantees, it's strongly not recommended to send user events (EventPartitioningByUserId) depending on chat data, from here (command_handler).
// It's recommended to (re)send them from event_handler

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"nkonev.name/chat/config"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/producer"
	"nkonev.name/chat/sanitizer"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/qdm12/reprint"
)

const minChatNameLen = 1
const maxChatNameLen = 256

const maxMessageLen = 1024 * 1024
const minMessageLen = 1

type UnauthorizedError struct {
	info string
}

func NewUnauthorizedError(info string) *UnauthorizedError {
	return &UnauthorizedError{info: info}
}

func (u *UnauthorizedError) Error() string {
	return u.info
}

type ValidationError struct {
	info string
}

func (u *ValidationError) Error() string {
	return u.info
}

func NewValidationError(info string) *ValidationError {
	return &ValidationError{info: info}
}

type ChatStillNotExistsError struct {
	info string
}

func (u *ChatStillNotExistsError) Error() string {
	return u.info
}

func NewChatStillNotExistsError(info string) *ChatStillNotExistsError {
	return &ChatStillNotExistsError{info: info}
}

type MessageStillNotExistsError struct {
	info string
}

func (u *MessageStillNotExistsError) Error() string {
	return u.info
}

func NewMessageStillNotExistsError(info string) *MessageStillNotExistsError {
	return &MessageStillNotExistsError{info: info}
}

type ParticipantsError struct {
	info string
}

func (u *ParticipantsError) Error() string {
	return u.info
}

func NewParticipantsError(info string) *ParticipantsError {
	return &ParticipantsError{info: info}
}

type ChatCreate struct {
	AdditionalData                      *AdditionalData
	Title                               string
	ParticipantIds                      []int64
	TetATet                             bool
	Blog                                bool
	BlogAbout                           bool
	Avatar                              *string
	AvatarBig                           *string
	CanResend                           bool
	CanReact                            bool
	AvailableToSearch                   bool
	RegularParticipantCanPublishMessage bool
	RegularParticipantCanPinMessage     bool
	RegularParticipantCanWriteMessage   bool
	RegularParticipantCanAddParticipant bool
}

type ChatEdit struct {
	AdditionalData                      *AdditionalData
	ChatId                              int64
	Title                               string
	ParticipantIdsToAdd                 []int64
	Blog                                bool // desired state
	BlogAbout                           bool // desired state
	Avatar                              *string
	AvatarBig                           *string
	CanResend                           bool
	CanReact                            bool
	AvailableToSearch                   bool
	RegularParticipantCanPublishMessage bool
	RegularParticipantCanPinMessage     bool
	RegularParticipantCanWriteMessage   bool
	RegularParticipantCanAddParticipant bool
}

func (cc *ChatEdit) IsValidatabale() bool {
	return true
}

func (a *ChatEdit) Validate() error {
	return validation.ValidateStruct(a,
		validation.Field(&a.Title, validation.Required, validation.Length(minChatNameLen, maxChatNameLen), validation.NotIn(dto.ReservedPublicallyAvailableForSearchChats)),
		validation.Field(&a.ChatId, validation.Required),
	)
}

func (cc *ChatCreate) IsValidatabale() bool {
	return true
}

func (a *ChatCreate) Validate() error {
	return validation.ValidateStruct(a,
		validation.Field(&a.Title, validation.Required, validation.Length(minChatNameLen, maxChatNameLen), validation.NotIn(dto.ReservedPublicallyAvailableForSearchChats)),
	)
}

type ChatDelete struct {
	ChatId         int64
	AdditionalData *AdditionalData
}

type ParticipantAdd struct {
	AdditionalData *AdditionalData
	ChatId         int64
	ParticipantIds []int64
	IsJoining      bool
}

type ParticipantDelete struct {
	AdditionalData *AdditionalData
	ChatId         int64
	ParticipantIds []int64
	IsLeaving      bool
}

type Truncate struct {
}

type ParticipantChange struct {
	AdditionalData *AdditionalData
	ChatId         int64
	ParticipantId  int64
	NewAdmin       bool
}

type EmbedMessage struct {
	Id        int64 // message id
	ChatId    int64
	EmbedType string
}

type MessageCreate struct {
	AdditionalData *AdditionalData
	ChatId         int64
	Content        string
	EmbedMessage   *EmbedMessage
	FileItemUuid   *string
}

type MessageEdit struct {
	AdditionalData *AdditionalData
	ChatId         int64
	MessageId      int64
	Content        string
	EmbedMessage   *EmbedMessage
	FileItemUuid   *string
}

type MessageSetFileItemUuid struct {
	AdditionalData *AdditionalData
	ChatId         int64
	MessageId      int64
	FileItemUuid   *string
}

type MessageSyncEmbed struct {
	AdditionalData *AdditionalData
	ChatId         int64
	MessageId      int64
}

func (a *MessageCreate) Validate() error {
	return validation.ValidateStruct(a,
		validation.Field(&a.Content, validation.Required, validation.Length(minMessageLen, maxMessageLen)),
	)
}

func (mcd *MessageCreate) IsValidatabale() bool {
	return mcd.EmbedMessage == nil || (mcd.EmbedMessage != nil && mcd.EmbedMessage.EmbedType == dto.EmbedMessageTypeReply)
}

func (a *MessageEdit) Validate() error {
	return validation.ValidateStruct(a,
		validation.Field(&a.Content, validation.Required, validation.Length(minMessageLen, maxMessageLen)),
		validation.Field(&a.MessageId, validation.Required),
	)
}

func (mcd *MessageEdit) IsValidatabale() bool {
	return true
}

type MessageDelete struct {
	AdditionalData *AdditionalData
	ChatId         int64
	MessageId      int64
}

type MessagePin struct {
	AdditionalData *AdditionalData
	ChatId         int64
	MessageId      int64
	Pin            bool
}

type MessagePublish struct {
	AdditionalData *AdditionalData
	ChatId         int64
	MessageId      int64
	Publish        bool
}

type ChatPin struct {
	AdditionalData *AdditionalData
	ChatId         int64
	Pin            bool
}

type ChatNotificationSettingsSet struct {
	AdditionalData *AdditionalData
	ChatId         int64
	Set            bool
}

type ThreadCreate struct {
	AdditionalData *AdditionalData
	ChatId         int64
	MessageId      int64
}

type ThreadDelete struct {
	AdditionalData *AdditionalData
	ChatId         int64
	MessageId      int64
}

type MessageRead struct {
	AdditionalData     *AdditionalData
	ChatId             int64
	MessageId          int64
	ReadMessagesAction ReadMessagesAction
}

type MakeMessageBlogPost struct {
	AdditionalData *AdditionalData
	ChatId         int64
	MessageId      int64
	BlogPost       bool
}

type MessageReactionFlip struct {
	AdditionalData *AdditionalData
	ChatId         int64
	MessageId      int64
	Reaction       string
}

type TechnicalRemoveContentOfDeletedUser struct {
	UserId int64
	ChatId int64 // only to make partition key
}

type TechnicalRemoveAbandonedChat struct {
	ChatId int64
}

func (sp *ChatCreate) Handle(ctx context.Context, eventBus *KafkaProducer, dba *db.DB, commonProjection *CommonProjection, stripTagsPolicy *sanitizer.StripTagsPolicy, cfg *config.AppConfig, rabbitmqOutputEventPublisher *producer.RabbitOutputEventsPublisher, lgr *logger.LoggerWrapper) (int64, error) {
	var copyCommand *ChatCreate
	err := reprint.FromTo(&sp, &copyCommand)
	if err != nil {
		return 0, err
	}

	if !slices.Contains(copyCommand.ParticipantIds, copyCommand.AdditionalData.BehalfUserId) {
		copyCommand.ParticipantIds = append(copyCommand.ParticipantIds, copyCommand.AdditionalData.BehalfUserId)
	}

	if int32(len(copyCommand.ParticipantIds)) > cfg.Cqrs.Commands.MaxParticipantsPerSingleCommand {
		return 0, fmt.Errorf("Max allowed participants %d, got %d", cfg.Cqrs.Commands.MaxParticipantsPerSingleCommand, copyCommand.ParticipantIds)
	}

	copyCommand.Title = sanitizer.TrimAmdSanitizeChatTitle(stripTagsPolicy, copyCommand.Title)

	if copyCommand.IsValidatabale() {
		if err = copyCommand.Validate(); err != nil {
			return 0, NewValidationError(fmt.Sprintf("Error during validation: %v", err))
		}
	}

	var tetATetOppositeUserId *int64
	if copyCommand.TetATet {
		if len(copyCommand.ParticipantIds) != 2 && len(copyCommand.ParticipantIds) != 1 {
			return 0, NewValidationError("Error during validation: tet-a-tet chat doesn't have 2 or 1 participants")
		}

		if len(copyCommand.ParticipantIds) == 2 {
			if copyCommand.ParticipantIds[0] == copyCommand.ParticipantIds[1] {
				return 0, NewValidationError("Error during validation: tet-a-tet should have different participants")
			}
		}
		if copyCommand.Blog {
			return 0, NewValidationError("Error during validation: tet-a-tet cannot be blog")
		}

		tetATetOppositeUserId = tetATetOpposite(copyCommand.ParticipantIds, copyCommand.AdditionalData.BehalfUserId)
		if tetATetOppositeUserId != nil {
			tetATetTwoExists, tetATetExistingTwoChatId, err := commonProjection.IsExistsTetATetTwo(ctx, dba, copyCommand.AdditionalData.BehalfUserId, *tetATetOppositeUserId)
			if err != nil {
				return 0, err
			}

			if tetATetTwoExists {
				// send upsert event
				err = rabbitmqOutputEventPublisher.Publish(ctx, copyCommand.AdditionalData.GetCorrelationId(), dto.GlobalUserEvent{
					UserId:    copyCommand.AdditionalData.BehalfUserId,
					EventType: dto.EventTypeChatTetATetUpserted,
					ChatTetATetUpsertedDto: &dto.ChatTetATetUpsertedDto{
						ChatId: tetATetExistingTwoChatId,
					},
				})
				if err != nil {
					lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
				}

				return tetATetExistingTwoChatId, nil
			}
		} else {
			tetATetOneExists, tetATetExistingOneChatId, err := commonProjection.IsExistsTetATetOne(ctx, dba, copyCommand.AdditionalData.BehalfUserId)
			if err != nil {
				return 0, err
			}

			if tetATetOneExists {
				// send upsert event
				err = rabbitmqOutputEventPublisher.Publish(ctx, copyCommand.AdditionalData.GetCorrelationId(), dto.GlobalUserEvent{
					UserId:    copyCommand.AdditionalData.BehalfUserId,
					EventType: dto.EventTypeChatTetATetUpserted,
					ChatTetATetUpsertedDto: &dto.ChatTetATetUpsertedDto{
						ChatId: tetATetExistingOneChatId,
					},
				})
				if err != nil {
					lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
				}

				return tetATetExistingOneChatId, nil
			}
		}
	}

	chatId, err := commonProjection.GetNextChatId(ctx, dba)
	if err != nil {
		return 0, err
	}

	cc := &ChatCreated{
		AdditionalData:        copyCommand.AdditionalData,
		TetATet:               copyCommand.TetATet,
		TetATetOppositeUserId: tetATetOppositeUserId,
		ChatCommoned: ChatCommoned{
			ChatId:                              chatId,
			Title:                               copyCommand.Title,
			Blog:                                copyCommand.Blog,
			BlogAbout:                           copyCommand.BlogAbout,
			Avatar:                              copyCommand.Avatar,
			AvatarBig:                           copyCommand.AvatarBig,
			CanResend:                           copyCommand.CanResend,
			CanReact:                            copyCommand.CanReact,
			AvailableToSearch:                   copyCommand.AvailableToSearch,
			RegularParticipantCanPublishMessage: copyCommand.RegularParticipantCanPublishMessage,
			RegularParticipantCanPinMessage:     copyCommand.RegularParticipantCanPinMessage,
			RegularParticipantCanWriteMessage:   copyCommand.RegularParticipantCanWriteMessage,
			RegularParticipantCanAddParticipant: copyCommand.RegularParticipantCanAddParticipant,
		},
	}
	err = eventBus.Publish(ctx, cc)
	if err != nil {
		return 0, err
	}

	pa := &ParticipantsAdded{
		AdditionalData: copyCommand.AdditionalData,
		ChatId:         chatId,
		Participants:   make([]ParticipantWithAdmin, 0),
		IsChatCreating: true,
	}
	for _, participantId := range copyCommand.ParticipantIds {
		pa.Participants = append(pa.Participants, ParticipantWithAdmin{
			ParticipantId: participantId,
			ChatAdmin:     participantId == copyCommand.AdditionalData.BehalfUserId || copyCommand.TetATet,
		})
	}

	if len(pa.Participants) == 0 {
		return dto.NoId, NewParticipantsError("Cannot add 0 participants")
	}

	err = eventBus.Publish(ctx, pa)
	if err != nil {
		return 0, err
	}

	return chatId, nil
}

func (sp *ChatEdit) Handle(ctx context.Context, eventBus *KafkaProducer, dba *db.DB, commonProjection *CommonProjection, stripTagsPolicy *sanitizer.StripTagsPolicy, cfg *config.AppConfig) error {
	var copyCommand *ChatEdit
	err := reprint.FromTo(&sp, &copyCommand)
	if err != nil {
		return err
	}

	adt, err := commonProjection.GetChatDataForAuthorization(ctx, dba, copyCommand.AdditionalData.BehalfUserId, copyCommand.ChatId)
	if err != nil {
		return err
	}

	if !CanEditChat(adt.IsChatAdmin, adt.ChatIsTetATet) {
		return NewUnauthorizedError(fmt.Sprintf("user %v is not authorized to edit the chat %v", sp.AdditionalData.BehalfUserId, sp.ChatId))
	}

	if int32(len(copyCommand.ParticipantIdsToAdd)) > cfg.Cqrs.Commands.MaxParticipantsPerSingleCommand {
		return fmt.Errorf("Max allowed participants %d, got %d", cfg.Cqrs.Commands.MaxParticipantsPerSingleCommand, copyCommand.ParticipantIdsToAdd)
	}

	copyCommand.Title = sanitizer.TrimAmdSanitizeChatTitle(stripTagsPolicy, copyCommand.Title)

	if copyCommand.IsValidatabale() {
		if err = copyCommand.Validate(); err != nil {
			return NewValidationError(fmt.Sprintf("Error during validation: %v", err))
		}
	}

	cc := &ChatEdited{
		AdditionalData: copyCommand.AdditionalData,
		ChatCommoned: ChatCommoned{
			ChatId:                              copyCommand.ChatId,
			Title:                               copyCommand.Title,
			Blog:                                copyCommand.Blog,
			BlogAbout:                           copyCommand.BlogAbout,
			Avatar:                              copyCommand.Avatar,
			AvatarBig:                           copyCommand.AvatarBig,
			CanResend:                           copyCommand.CanResend,
			CanReact:                            copyCommand.CanReact,
			AvailableToSearch:                   copyCommand.AvailableToSearch,
			RegularParticipantCanPublishMessage: copyCommand.RegularParticipantCanPublishMessage,
			RegularParticipantCanPinMessage:     copyCommand.RegularParticipantCanPinMessage,
			RegularParticipantCanWriteMessage:   copyCommand.RegularParticipantCanWriteMessage,
			RegularParticipantCanAddParticipant: copyCommand.RegularParticipantCanAddParticipant,
		},
	}
	err = eventBus.Publish(ctx, cc)
	if err != nil {
		return err
	}

	if len(copyCommand.ParticipantIdsToAdd) > 0 {
		pa := &ParticipantsAdded{
			AdditionalData: copyCommand.AdditionalData,
			ChatId:         copyCommand.ChatId,
		}
		for _, participantId := range copyCommand.ParticipantIdsToAdd {
			pa.Participants = append(pa.Participants, ParticipantWithAdmin{
				ParticipantId: participantId,
				ChatAdmin:     false,
			})
		}
		if len(pa.Participants) == 0 {
			return NewParticipantsError("Cannot add 0 participants")
		}

		err = eventBus.Publish(ctx, pa)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *ChatDelete) Handle(ctx context.Context, eventBus *KafkaProducer, dba *db.DB, commonProjection *CommonProjection) error {
	adt, err := commonProjection.GetChatDataForAuthorization(ctx, dba, s.AdditionalData.BehalfUserId, s.ChatId)
	if err != nil {
		return err
	}

	if !CanDeleteChat(adt.IsChatAdmin) {
		return NewUnauthorizedError(fmt.Sprintf("user %v is not authorized to delete the chat %v", s.AdditionalData.BehalfUserId, s.ChatId))
	}

	pa := &ParticipantDeleted{
		AdditionalData:             s.AdditionalData,
		GetParticipantsType:        GetParticipantsTypeAllInChatExcepting,
		AllParticipantIdsExcepting: []int64{},
		ChatId:                     s.ChatId,
		IsChatRemoving:             true,
	}
	errInner := eventBus.Publish(ctx, pa)
	if errInner != nil {
		return errInner
	}

	cc := &ChatDeleted{
		AdditionalData: s.AdditionalData,
		ChatId:         s.ChatId,
	}
	err = eventBus.Publish(ctx, cc)
	if err != nil {
		return err
	}
	return nil
}

func (s *ParticipantAdd) Handle(ctx context.Context, eventBus *KafkaProducer, dba *db.DB, commonProjection *CommonProjection, cfg *config.AppConfig) error {
	adt, err := commonProjection.GetChatDataForAuthorization(ctx, dba, s.AdditionalData.BehalfUserId, s.ChatId)
	if err != nil {
		return err
	}

	if !adt.IsChatFound {
		return NewChatStillNotExistsError(fmt.Sprintf("chat %d still does not exist", s.ChatId))
	}

	if !CanAddParticipant(adt.IsChatAdmin, adt.ChatIsTetATet, s.IsJoining, adt.AvailableToSearch, adt.IsBlog, false, adt.IsParticipant, adt.RegularParticipantCanAddParticipants) {
		return NewUnauthorizedError(fmt.Sprintf("user %v cannot add into chat %v", s.AdditionalData.BehalfUserId, s.ChatId))
	}

	if int32(len(s.ParticipantIds)) > cfg.Cqrs.Commands.MaxParticipantsPerSingleCommand {
		return fmt.Errorf("Max allowed participants %d, got %d", cfg.Cqrs.Commands.MaxParticipantsPerSingleCommand, s.ParticipantIds)
	}

	pa := &ParticipantsAdded{
		AdditionalData: s.AdditionalData,
		ChatId:         s.ChatId,
		IsJoining:      s.IsJoining,
	}
	for _, participantId := range s.ParticipantIds {
		pa.Participants = append(pa.Participants, ParticipantWithAdmin{
			ParticipantId: participantId,
			ChatAdmin:     false,
		})
	}
	if len(pa.Participants) == 0 {
		return NewParticipantsError("Cannot add 0 participants")
	}

	err = eventBus.Publish(ctx, pa)
	if err != nil {
		return err
	}

	return nil
}

func (s *ParticipantDelete) Handle(ctx context.Context, eventBus *KafkaProducer, dba *db.DB, commonProjection *CommonProjection, cfg *config.AppConfig) error {
	adt, err := commonProjection.GetChatDataForAuthorization(ctx, dba, s.AdditionalData.BehalfUserId, s.ChatId)
	if err != nil {
		return err
	}

	for _, participantId := range s.ParticipantIds {
		if !CanRemoveParticipant(s.AdditionalData.BehalfUserId, adt.IsChatAdmin, adt.ChatIsTetATet, s.IsLeaving, adt.IsParticipant, participantId, false) {
			return NewUnauthorizedError(fmt.Sprintf("user %v is not authorized to delete the chat %v participant %v", s.AdditionalData.BehalfUserId, s.ChatId, s.ParticipantIds))
		}
	}

	if int32(len(s.ParticipantIds)) > cfg.Cqrs.Commands.MaxParticipantsPerSingleCommand {
		return fmt.Errorf("Max allowed participants %d, got %d", cfg.Cqrs.Commands.MaxParticipantsPerSingleCommand, s.ParticipantIds)
	}

	pa := &ParticipantDeleted{
		AdditionalData:      s.AdditionalData,
		ParticipantIds:      s.ParticipantIds,
		GetParticipantsType: GetParticipantsTypeNormal,
		ChatId:              s.ChatId,
		IsLeaving:           s.IsLeaving,
	}
	err = eventBus.Publish(ctx, pa)
	if err != nil {
		return err
	}

	return nil
}

func (s *Truncate) Handle(ctx context.Context, eventBus *KafkaProducer, dba *db.DB, commonProjection *CommonProjection, lgr *logger.LoggerWrapper, cfg *config.AppConfig) error {
	pa := &ProjectionsTruncated{}
	err := eventBus.Publish(ctx, pa)
	if err != nil {
		return err
	}

	i := 0
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			if err != nil {
				lgr.ErrorContext(ctx, "error from context", logger.AttributeError, err)
			}
			break
		default:
		}

		completed, err := commonProjection.GetIsTruncatingCompleted(ctx, dba)
		if err != nil {
			lgr.InfoContext(ctx, "error during GetIsTruncatingCompleted", logger.AttributeError, err)
		}
		if completed {
			err = commonProjection.UnsetIsTruncatingCompleted(ctx, dba)
			if err != nil {
				lgr.ErrorContext(ctx, "error during UnsetIsTruncatingCompleted", logger.AttributeError, err)
			}
			break
		}

		time.Sleep(cfg.Cqrs.SleepBeforePolling)

		i++
		if i > cfg.Cqrs.PollingMaxTimes {
			return fmt.Errorf("Exceed max %d poll times", cfg.Cqrs.PollingMaxTimes)
		}
	}
	return nil
}

func (s *ParticipantChange) Handle(ctx context.Context, eventBus *KafkaProducer, dba *db.DB, commonProjection *CommonProjection) error {
	adt, err := commonProjection.GetChatDataForAuthorization(ctx, dba, s.AdditionalData.BehalfUserId, s.ChatId)
	if err != nil {
		return err
	}

	if !adt.IsChatFound {
		return NewChatStillNotExistsError(fmt.Sprintf("chat %d still does not exist", s.ChatId))
	}

	if !CanChangeParticipant(s.AdditionalData.BehalfUserId, adt.IsChatAdmin, adt.ChatIsTetATet, s.ParticipantId) {
		return NewUnauthorizedError(fmt.Sprintf("user %v is not authorized to change the chat %v participants", s.AdditionalData.BehalfUserId, s.ChatId))
	}

	pa := &ParticipantChanged{
		AdditionalData: s.AdditionalData,
		ParticipantId:  s.ParticipantId,
		ChatId:         s.ChatId,
		NewAdmin:       s.NewAdmin,
	}
	err = eventBus.Publish(ctx, pa)
	if err != nil {
		return err
	}

	return nil
}

func (s *ChatPin) Handle(ctx context.Context, eventBus *KafkaProducer) error {
	cp := &ChatPinned{
		AdditionalData: s.AdditionalData,
		ChatId:         s.ChatId,
		Pinned:         s.Pin,
	}
	err := eventBus.Publish(ctx, cp)
	if err != nil {
		return err
	}

	return nil
}

func (s *ChatNotificationSettingsSet) Handle(ctx context.Context, eventBus *KafkaProducer) error {
	cp := &ChatNotificationSettingsSetted{
		AdditionalData: s.AdditionalData,
		ChatId:         s.ChatId,
		Setted:         s.Set,
	}
	return eventBus.Publish(ctx, cp)
}

func (sp *MessageCreate) Handle(ctx context.Context, eventBus *KafkaProducer, dba *db.DB, commonProjection *CommonProjection, cfg *config.AppConfig, lgr *logger.LoggerWrapper, policy *sanitizer.SanitizerPolicy, userPermissions []string) (int64, error) {
	var copyCommand *MessageCreate
	err := reprint.FromTo(&sp, &copyCommand)
	if err != nil {
		return 0, err
	}

	type authDto struct {
		adt             dto.MessageAuthorizationData
		chatHasMessages bool
	}

	ad, err := db.TransactWithResult(ctx, dba, func(tx *db.Tx) (*authDto, error) {
		adt, errInn := commonProjection.GetMessageDataForAuthorization(ctx, tx, copyCommand.AdditionalData.BehalfUserId, copyCommand.ChatId, dto.NoId)
		if errInn != nil {
			return nil, errInn
		}

		has, errInn := commonProjection.ChatHasMessages(ctx, tx, copyCommand.ChatId)
		if errInn != nil {
			return nil, errInn
		}

		return &authDto{
			adt:             adt,
			chatHasMessages: has,
		}, nil
	})
	if err != nil {
		return 0, err
	}

	adt := ad.adt

	if !adt.IsChatFound {
		return 0, NewChatStillNotExistsError(fmt.Sprintf("chat %d still does not exist", copyCommand.ChatId))
	}

	if !CanWriteMessage(adt.IsParticipant, adt.IsChatAdmin, adt.ChatCanWriteMessage) {
		return 0, NewUnauthorizedError(fmt.Sprintf("user %v is not authorized to write the message in chat %v", sp.AdditionalData.BehalfUserId, sp.ChatId))
	}

	bloggingAllowed := IsBloggingAllowed(cfg, userPermissions)
	canMakeMessageBlogPost := CanMakeMessageBlogPost(adt.IsChatAdmin, adt.ChatIsTetATet, adt.IsMessageBlogPost, adt.IsBlog, bloggingAllowed)

	trimmedAndSanitized, err := sanitizer.TrimAmdSanitizeMessage(ctx, cfg, lgr, policy, copyCommand.Content)
	if err != nil {
		return 0, err
	}
	copyCommand.Content = trimmedAndSanitized

	if copyCommand.IsValidatabale() {
		if err = copyCommand.Validate(); err != nil {
			return 0, NewValidationError(fmt.Sprintf("Error during validation: %v", err))
		}
	}

	mc := &MessageCreated{
		MessageCommoned: MessageCommoned{
			ChatId:       copyCommand.ChatId,
			Content:      copyCommand.Content,
			FileItemUuid: copyCommand.FileItemUuid,
		},
		AdditionalData: copyCommand.AdditionalData,
	}

	err = validateAndSetEmbedFieldsEmbedMessage(ctx, dba, commonProjection, copyCommand.ChatId, copyCommand.EmbedMessage, &mc.MessageCommoned, adt.IsParticipant)
	if err != nil {
		return 0, err
	}

	messageId, err := commonProjection.GetNextMessageId(ctx, dba, copyCommand.ChatId)
	if err != nil {
		return 0, err
	}

	if messageId == ChatStillNotExists {
		return 0, NewChatStillNotExistsError(fmt.Sprintf("chat %d still does not exist", copyCommand.ChatId))
	}

	mc.MessageCommoned.Id = messageId

	err = eventBus.Publish(ctx, mc)
	if err != nil {
		return 0, err
	}

	if adt.IsBlog && canMakeMessageBlogPost && !ad.chatHasMessages {
		ev := MessageBlogPostMade{
			AdditionalData: copyCommand.AdditionalData,
			ChatId:         copyCommand.ChatId,
			MessageId:      messageId,
			BlogPost:       true,
		}

		err = eventBus.Publish(ctx, &ev)
		if err != nil {
			return 0, err
		}
	}

	return messageId, nil
}

func (s *MessageRead) Handle(ctx context.Context, lgr *logger.LoggerWrapper, eventBus *KafkaProducer, commonProjection *CommonProjection, dba *db.DB, rabbitmqOutputEventPublisher *producer.RabbitOutputEventsPublisher, rabbitmqNotificationEventsPublisher *producer.RabbitNotificationEventsPublisher) error {
	// seems it's not need to immediately respond error in case is no participant, so we skip authorization check here
	// the authorization is in event_handler
	if s.ReadMessagesAction == ReadMessagesActionAllMessagesInOneChat {
		participant, err := commonProjection.IsParticipant(ctx, dba, s.AdditionalData.BehalfUserId, s.ChatId)
		if err != nil {
			return err
		}

		if !participant {
			return NewUnauthorizedError(fmt.Sprintf("user %v is not a participant of chat %v", s.AdditionalData.BehalfUserId, s.ChatId))
		}

		cp := &MessageReaded{
			AdditionalData:     s.AdditionalData,
			ReadMessagesAction: ReadMessagesActionAllMessagesInOneChat,
			ChatId:             s.ChatId,
		}
		err = eventBus.Publish(ctx, cp)
		if err != nil {
			return err
		}
		return nil
	} else if s.ReadMessagesAction == ReadMessagesActionOneMessage {
		participant, err := commonProjection.IsParticipant(ctx, dba, s.AdditionalData.BehalfUserId, s.ChatId)
		if err != nil {
			return err
		}

		if !participant {
			return NewUnauthorizedError(fmt.Sprintf("user %v is not a participant of chat %v", s.AdditionalData.BehalfUserId, s.ChatId))
		}

		lastMessageReadedId, lastMessgeReadedExists, maxMessageId, err := commonProjection.getLastMessageUnreadReaded(ctx, s.ChatId, s.AdditionalData.BehalfUserId)
		if err != nil {
			return err
		}
		messageIdToMark := s.MessageId
		if s.MessageId > maxMessageId {
			messageIdToMark = maxMessageId
		}
		// Optimizations in order to not send useless messages in Kafka
		if (lastMessgeReadedExists && messageIdToMark > lastMessageReadedId) || (!lastMessgeReadedExists && lastMessageReadedId == 0) {
			cp := &MessageReaded{
				AdditionalData:     s.AdditionalData,
				ChatId:             s.ChatId,
				MessageId:          messageIdToMark,
				ReadMessagesAction: ReadMessagesActionOneMessage,
			}
			err = eventBus.Publish(ctx, cp)
			if err != nil {
				return err
			}
		}

		// notification deletes
		messageBasic, err := commonProjection.GetMessageBasic(ctx, dba, s.ChatId, s.MessageId)
		if err != nil {
			return err
		}

		err = rabbitmqNotificationEventsPublisher.Publish(ctx, s.AdditionalData.GetCorrelationId(), dto.NotificationEvent{
			EventType: dto.EventTypeMentionDeleted,
			UserId:    s.AdditionalData.BehalfUserId,
			ChatId:    s.ChatId,
			MentionNotification: &dto.MentionNotification{
				Id: s.MessageId,
			},
		})
		if err != nil {
			lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
		}

		err = rabbitmqNotificationEventsPublisher.Publish(ctx, s.AdditionalData.GetCorrelationId(), dto.NotificationEvent{
			EventType: dto.EventTypeReplyDeleted,
			UserId:    s.AdditionalData.BehalfUserId,
			ChatId:    s.ChatId,
			ReplyNotification: &dto.ReplyDto{
				MessageId: s.MessageId,
				ChatId:    s.ChatId,
			},
		})
		if err != nil {
			lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
		}

		var messageOwnerId = messageBasic.GetOwnerId()
		if messageOwnerId == s.AdditionalData.BehalfUserId { // only for myself
			var reactions []string
			reactions, err = commonProjection.GetReactionsOnMessage(ctx, dba, s.ChatId, s.MessageId)
			if err != nil {
				return err
			}

			for _, reaction := range reactions {
				if messageOwnerId == dto.NoOwner || messageOwnerId == dto.NoId {
					lgr.InfoContext(ctx, "Unable to get message owner for reaction notification")
				} else {
					err = rabbitmqNotificationEventsPublisher.Publish(ctx, s.AdditionalData.GetCorrelationId(), dto.NotificationEvent{
						EventType: dto.EventTypeReactionDeleted,
						ReactionEvent: &dto.ReactionEvent{
							Reaction:  reaction,
							MessageId: s.MessageId,
						},
						UserId: s.AdditionalData.BehalfUserId,
						ChatId: s.ChatId,
					})
					if err != nil {
						lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
					}
				}
			}
		}

		err = rabbitmqOutputEventPublisher.Publish(ctx, s.AdditionalData.GetCorrelationId(), dto.GlobalUserEvent{
			UserId:    s.AdditionalData.BehalfUserId,
			EventType: dto.EventTypeMessageBrowserNotificationDelete,
			BrowserNotification: &dto.BrowserNotification{
				ChatId:    s.ChatId,
				MessageId: s.MessageId,
				OwnerId:   dto.NonExistentUser,
			},
		})
		if err != nil {
			lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
		}

		return nil
	} else if s.ReadMessagesAction == ReadMessagesActionAllChats {
		cp := &MessageReaded{
			AdditionalData:     s.AdditionalData,
			ReadMessagesAction: ReadMessagesActionAllChats,
		}
		err := eventBus.Publish(ctx, cp)
		if err != nil {
			return err
		}
		return nil
	} else {
		return fmt.Errorf("Unknown action: %T", s.ReadMessagesAction)
	}
}

func (s *MakeMessageBlogPost) Handle(ctx context.Context, cfg *config.AppConfig, userPermissions []string, eventBus *KafkaProducer, dba *db.DB, commonProjection *CommonProjection) error {

	adt, err := commonProjection.GetMessageDataForAuthorization(ctx, dba, s.AdditionalData.BehalfUserId, s.ChatId, s.MessageId)
	if err != nil {
		return err
	}

	bloggingAllowed := IsBloggingAllowed(cfg, userPermissions)

	if !CanMakeMessageBlogPost(adt.IsChatAdmin, adt.ChatIsTetATet, adt.IsMessageBlogPost, adt.IsBlog, bloggingAllowed) {
		return NewUnauthorizedError(fmt.Sprintf("user %v is not authorized to make the message blog post in the chat %v", s.AdditionalData.BehalfUserId, s.ChatId))
	}

	ev := MessageBlogPostMade{
		AdditionalData: s.AdditionalData,
		ChatId:         s.ChatId,
		MessageId:      s.MessageId,
		BlogPost:       s.BlogPost,
	}

	return eventBus.Publish(ctx, &ev)
}

func (s *MessageDelete) Handle(ctx context.Context, eventBus *KafkaProducer, dba *db.DB, commonProjection *CommonProjection) error {
	adt, err := commonProjection.GetMessageDataForAuthorization(ctx, dba, s.AdditionalData.BehalfUserId, s.ChatId, s.MessageId)
	if err != nil {
		return err
	}

	canWriteMessage := CanWriteMessage(adt.IsParticipant, adt.IsChatAdmin, adt.ChatCanWriteMessage)

	if !CanDeleteMessage(s.AdditionalData.BehalfUserId, adt.MessageOwnerId, canWriteMessage) {
		return NewUnauthorizedError(fmt.Sprintf("user %v is not authorized to delete the message in chat %v", s.AdditionalData.BehalfUserId, s.ChatId))
	}

	cp := &MessageDeleted{
		AdditionalData: s.AdditionalData,
		ChatId:         s.ChatId,
		MessageId:      s.MessageId,
	}
	err = eventBus.Publish(ctx, cp)
	if err != nil {
		return err
	}

	return nil
}

func (sp *MessageEdit) Handle(ctx context.Context, eventBus *KafkaProducer, dba *db.DB, commonProjection *CommonProjection, cfg *config.AppConfig, lgr *logger.LoggerWrapper, policy *sanitizer.SanitizerPolicy) error {
	var copyCommand *MessageEdit
	err := reprint.FromTo(&sp, &copyCommand)
	if err != nil {
		return err
	}

	adt, err := commonProjection.GetMessageDataForAuthorization(ctx, dba, copyCommand.AdditionalData.BehalfUserId, copyCommand.ChatId, copyCommand.MessageId)
	if err != nil {
		return err
	}

	if !adt.IsChatFound {
		return NewChatStillNotExistsError(fmt.Sprintf("chat %d still does not exist", copyCommand.ChatId))
	}

	if !adt.IsMessageFound {
		return NewMessageStillNotExistsError(fmt.Sprintf("message %d still does not exist in chat %d", copyCommand.MessageId, copyCommand.ChatId))
	}

	canWriteMessage := CanWriteMessage(adt.IsParticipant, adt.IsChatAdmin, adt.ChatCanWriteMessage)

	if !CanEditMessage(copyCommand.AdditionalData.BehalfUserId, adt.MessageOwnerId, adt.HasEmbedMessage, adt.EmbedMessageTypeSafe, canWriteMessage) {
		return NewUnauthorizedError(fmt.Sprintf("user %v is not authorized to edit the message in chat %v", sp.AdditionalData.BehalfUserId, sp.ChatId))
	}

	trimmedAndSanitized, err := sanitizer.TrimAmdSanitizeMessage(ctx, cfg, lgr, policy, copyCommand.Content)
	if err != nil {
		return err
	}
	copyCommand.Content = trimmedAndSanitized

	if copyCommand.IsValidatabale() {
		if err = copyCommand.Validate(); err != nil {
			return NewValidationError(fmt.Sprintf("Error during validation: %v", err))
		}
	}

	cp := &MessageEdited{
		MessageCommoned: MessageCommoned{
			Id:           copyCommand.MessageId,
			ChatId:       copyCommand.ChatId,
			Content:      copyCommand.Content,
			FileItemUuid: copyCommand.FileItemUuid,
		},
		AdditionalData: copyCommand.AdditionalData,
	}

	err = validateAndSetEmbedFieldsEmbedMessage(ctx, dba, commonProjection, copyCommand.ChatId, copyCommand.EmbedMessage, &cp.MessageCommoned, adt.IsParticipant)
	if err != nil {
		return err
	}

	err = eventBus.Publish(ctx, cp)
	if err != nil {
		return err
	}

	return nil
}

func (sp *MessageSetFileItemUuid) Handle(ctx context.Context, eventBus *KafkaProducer, dba *db.DB, commonProjection *CommonProjection, cfg *config.AppConfig, lgr *logger.LoggerWrapper, policy *sanitizer.SanitizerPolicy) error {
	var copyCommand *MessageEdit = &MessageEdit{
		AdditionalData: sp.AdditionalData,
		ChatId:         sp.ChatId,
		MessageId:      sp.MessageId,
		FileItemUuid:   sp.FileItemUuid,
		// content and embed are set below
	}

	adt, err := commonProjection.GetMessageDataForAuthorization(ctx, dba, copyCommand.AdditionalData.BehalfUserId, copyCommand.ChatId, copyCommand.MessageId)
	if err != nil {
		return err
	}

	if !adt.IsChatFound {
		return NewChatStillNotExistsError(fmt.Sprintf("chat %d still does not exist", copyCommand.ChatId))
	}

	if !adt.IsMessageFound {
		return NewMessageStillNotExistsError(fmt.Sprintf("message %d still does not exist in chat %d", copyCommand.MessageId, copyCommand.ChatId))
	}

	canWriteMessage := CanWriteMessage(adt.IsParticipant, adt.IsChatAdmin, adt.ChatCanWriteMessage)

	if !CanEditMessage(copyCommand.AdditionalData.BehalfUserId, adt.MessageOwnerId, adt.HasEmbedMessage, adt.EmbedMessageTypeSafe, canWriteMessage) {
		return NewUnauthorizedError(fmt.Sprintf("user %v is not authorized to edit the message in chat %v", sp.AdditionalData.BehalfUserId, sp.ChatId))
	}

	mb, err := commonProjection.GetMessageWithEmbed(ctx, dba, copyCommand.ChatId, copyCommand.MessageId)
	if err != nil {
		return err
	}

	copyCommand.Content = mb.Content

	if copyCommand.IsValidatabale() {
		if err = copyCommand.Validate(); err != nil {
			return NewValidationError(fmt.Sprintf("Error during validation: %v", err))
		}
	}

	cp := &MessageEdited{
		MessageCommoned: MessageCommoned{
			Id:           copyCommand.MessageId,
			ChatId:       copyCommand.ChatId,
			Content:      copyCommand.Content,
			FileItemUuid: copyCommand.FileItemUuid,
			Embed:        mb.Embed,
		},
		AdditionalData: copyCommand.AdditionalData,
	}

	err = eventBus.Publish(ctx, cp)
	if err != nil {
		return err
	}

	return nil
}

func (sp *MessageSyncEmbed) Handle(ctx context.Context, eventBus *KafkaProducer, dba *db.DB, commonProjection *CommonProjection, cfg *config.AppConfig, lgr *logger.LoggerWrapper, policy *sanitizer.SanitizerPolicy) error {
	var copyCommand *MessageSyncEmbed
	err := reprint.FromTo(&sp, &copyCommand)
	if err != nil {
		return err
	}

	adt, err := commonProjection.GetMessageDataForAuthorization(ctx, dba, copyCommand.AdditionalData.BehalfUserId, copyCommand.ChatId, copyCommand.MessageId)
	if err != nil {
		return err
	}

	canWriteMessage := CanWriteMessage(adt.IsParticipant, adt.IsChatAdmin, adt.ChatCanWriteMessage)

	if !CanSyncEmbedMessage(copyCommand.AdditionalData.BehalfUserId, adt.MessageOwnerId, adt.HasEmbedMessage, canWriteMessage) {
		return NewUnauthorizedError(fmt.Sprintf("user %v is not authorized to sync the embed message in chat %v", sp.AdditionalData.BehalfUserId, sp.ChatId))
	}

	me, err := commonProjection.GetMessageWithEmbed(ctx, dba, copyCommand.ChatId, copyCommand.MessageId)
	if err != nil {
		return err
	}

	if me == nil {
		return errors.New("wrong invariant - message is not exists. it should be avoided by checks above")
	}

	embedMessageRequest, shouldSkip, err := buildEmbedRequestFromMessage(ctx, dba, commonProjection, me.Embed, copyCommand.ChatId, copyCommand.MessageId)
	if err != nil {
		return err
	}

	if shouldSkip {
		lgr.InfoContext(ctx, "Skipping handling MessageSyncEmbed", logger.AttributeMessageId, copyCommand.MessageId, logger.AttributeChatId, copyCommand.ChatId)
		return nil
	}

	cp := &MessageEdited{
		MessageCommoned: MessageCommoned{
			Id:      copyCommand.MessageId,
			ChatId:  copyCommand.ChatId,
			Content: me.Content,
		},
		IsEmbedSync:    true,
		AdditionalData: copyCommand.AdditionalData,
	}

	err = validateAndSetEmbedFieldsEmbedMessage(ctx, dba, commonProjection, copyCommand.ChatId, embedMessageRequest, &cp.MessageCommoned, adt.IsParticipant)
	if err != nil {
		return err
	}

	err = eventBus.Publish(ctx, cp)
	if err != nil {
		return err
	}

	return nil
}

func (s *MessagePin) Handle(ctx context.Context, eventBus *KafkaProducer, dba *db.DB, commonProjection *CommonProjection) error {
	adt, err := commonProjection.GetMessageDataForAuthorization(ctx, dba, s.AdditionalData.BehalfUserId, s.ChatId, s.MessageId)
	if err != nil {
		return err
	}

	if !CanPinMessage(adt.ChatCanPinMessage, adt.IsChatAdmin) {
		return NewUnauthorizedError(fmt.Sprintf("user %v is not authorized to pin the message in chat %v", s.AdditionalData.BehalfUserId, s.ChatId))
	}

	cp := &MessagePinned{
		AdditionalData: s.AdditionalData,
		ChatId:         s.ChatId,
		MessageId:      s.MessageId,
		Pinned:         s.Pin,
	}
	err = eventBus.Publish(ctx, cp)
	if err != nil {
		return err
	}

	return nil
}

func (s *MessagePublish) Handle(ctx context.Context, eventBus *KafkaProducer, dba *db.DB, commonProjection *CommonProjection) error {
	adt, err := commonProjection.GetMessageDataForAuthorization(ctx, dba, s.AdditionalData.BehalfUserId, s.ChatId, s.MessageId)
	if err != nil {
		return err
	}

	if !CanPublishMessage(adt.ChatCanPublishMessage, adt.IsChatAdmin, adt.MessageOwnerId, s.AdditionalData.BehalfUserId) {
		return NewUnauthorizedError(fmt.Sprintf("user %v is not authorized to publish the message in chat %v", s.AdditionalData.BehalfUserId, s.ChatId))
	}

	cp := &MessagePublished{
		AdditionalData: s.AdditionalData,
		ChatId:         s.ChatId,
		MessageId:      s.MessageId,
		Published:      s.Publish,
	}
	err = eventBus.Publish(ctx, cp)
	if err != nil {
		return err
	}

	return nil
}

func (s *MessageReactionFlip) Handle(ctx context.Context, eventBus *KafkaProducer, dba *db.DB, commonProjection *CommonProjection, policy *sanitizer.SanitizerPolicy) error {
	adt, err := commonProjection.GetChatDataForAuthorization(ctx, dba, s.AdditionalData.BehalfUserId, s.ChatId)
	if err != nil {
		return err
	}

	if !CanReactOnMessage(adt.ChatCanReactOnMessage, adt.IsParticipant) {
		return NewUnauthorizedError(fmt.Sprintf("user %v is not authorized to react on a message in chat %v", s.AdditionalData.BehalfUserId, s.ChatId))
	}

	sanitizedReaction := sanitizer.TrimAmdSanitize(policy, s.Reaction)

	if len([]rune(sanitizedReaction)) > 4 || len([]rune(sanitizedReaction)) < 1 {
		return NewValidationError("Wrong length of reaction")
	}

	has, err := commonProjection.HasMyReaction(ctx, dba, s.ChatId, s.MessageId, s.AdditionalData.BehalfUserId, sanitizedReaction)
	if err != nil {
		return err
	}

	if !has {
		cp := &MessageReactionCreated{
			AdditionalData: s.AdditionalData,
			MessageReactionCommoned: MessageReactionCommoned{
				ChatId:    s.ChatId,
				MessageId: s.MessageId,
				Reaction:  sanitizedReaction,
			},
		}

		err = eventBus.Publish(ctx, cp)
		if err != nil {
			return err
		}
	} else {
		cp := &MessageReactionRemoved{
			AdditionalData: s.AdditionalData,
			MessageReactionCommoned: MessageReactionCommoned{
				ChatId:    s.ChatId,
				MessageId: s.MessageId,
				Reaction:  sanitizedReaction,
			},
		}

		err = eventBus.Publish(ctx, cp)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *TechnicalRemoveContentOfDeletedUser) Handle(ctx context.Context, eventBus *KafkaProducer) error {
	pa := &ParticipantDeleted{
		AdditionalData:          GenerateMessageAdditionalData(nil, dto.SystemUserCleaner),
		ParticipantIds:          []int64{s.UserId},
		GetParticipantsType:     GetParticipantsTypeNormal,
		ChatId:                  s.ChatId,
		WereRemovedUsersFromAaa: true,
	}
	err := eventBus.Publish(ctx, pa)
	if err != nil {
		return err
	}

	return nil
}

func (s *TechnicalRemoveAbandonedChat) Handle(ctx context.Context, eventBus *KafkaProducer) error {
	return eventBus.Publish(ctx, &TechnicalAbandonedChatRemoved{ChatId: s.ChatId})
}

func buildEmbedRequestFromMessage(ctx context.Context, dba *db.DB, commonProjection *CommonProjection, embed dto.Embeddable, chatId int64, messageId int64) (*EmbedMessage, bool, error) {
	if embed == nil {
		return nil, false, NewValidationError("logical error - not got the embed message in the target chat")
	}

	ret := &EmbedMessage{}
	switch typed := embed.(type) {
	case *dto.EmbedReply:
		ret.EmbedType = string(typed.EmbedTyper.Type)
		ret.Id = typed.MessageId
		ret.ChatId = chatId
	case *dto.EmbedResend:
		ret.EmbedType = string(typed.EmbedTyper.Type)
		ret.Id = typed.MessageId
		ret.ChatId = typed.ChatId
	}

	exists, err := commonProjection.IsMessageExists(ctx, dba, ret.ChatId, ret.Id)
	if err != nil {
		return nil, false, err
	}

	if !exists {
		return nil, true, nil
	}

	return ret, false, nil
}

func validateAndSetEmbedFieldsEmbedMessage(ctx context.Context, dba *db.DB, commonProjection *CommonProjection, currentChatId int64, embedMessageRequest *EmbedMessage, receiver *MessageCommoned, isParticipant bool) error {
	if embedMessageRequest != nil {
		if embedMessageRequest.Id == 0 {
			return errors.New("Missed embed message id")
		}
		if embedMessageRequest.EmbedType == "" {
			return errors.New("Missed embedMessageType")
		} else {
			if embedMessageRequest.EmbedType != dto.EmbedMessageTypeReply && embedMessageRequest.EmbedType != dto.EmbedMessageTypeResend {
				return errors.New("Wrong embedMessageType")
			}
			if embedMessageRequest.EmbedType == dto.EmbedMessageTypeResend && embedMessageRequest.ChatId == 0 {
				return errors.New("Missed embedChatId for EmbedMessageTypeResend")
			}
		}

		if embedMessageRequest.EmbedType == dto.EmbedMessageTypeReply {
			m, err := commonProjection.GetMessageBasic(ctx, dba, currentChatId, embedMessageRequest.Id)
			if err != nil {
				return err
			}
			if m == nil {
				return errors.New("Missing the message")
			}

			receiver.Embed = dto.NewEmbedReply(
				embedMessageRequest.Id,
				m.Content,
				m.OwnerId,
			)
			return nil
		} else if embedMessageRequest.EmbedType == dto.EmbedMessageTypeResend {
			m, err := commonProjection.GetMessageBasic(ctx, dba, embedMessageRequest.ChatId, embedMessageRequest.Id)
			if err != nil {
				return err
			}
			if m == nil {
				return errors.New("Missing the message")
			}

			chat, err := commonProjection.GetChatBasic(ctx, dba, embedMessageRequest.ChatId)
			if err != nil {
				return err
			}
			if chat == nil {
				return errors.New("Missing the chat")
			}

			if !CanResendMessage(chat.CanResend, isParticipant) {
				return errors.New("Resending is forbidden for this chat")
			}
			receiver.Embed = dto.NewEmbedResend(
				embedMessageRequest.Id,
				m.Content,
				m.OwnerId,
				embedMessageRequest.ChatId,
			)
			return nil
		}
		return fmt.Errorf("Unexpected embed type '%v'", embedMessageRequest.EmbedType)
	}

	return nil
}

func (s *ThreadCreate) Handle(ctx context.Context, eventBus *KafkaProducer, dba *db.DB, commonProjection *CommonProjection, cfg *config.AppConfig) error {
	adt, err := commonProjection.GetThreadDataForAuthorization(ctx, dba, s.AdditionalData.BehalfUserId, s.ChatId, s.MessageId)
	if err != nil {
		return err
	}

	if !adt.IsChatFound {
		return NewChatStillNotExistsError(fmt.Sprintf("chat %d still does not exist", s.ChatId))
	}

	if adt.FoundThreadId != nil {
		return nil
	}

	if !CanCreateThread(adt.ChatCanCreateThread, adt.IsParticipant) {
		return NewUnauthorizedError(fmt.Sprintf("user %v cannot create thread in chat %v", s.AdditionalData.BehalfUserId, s.ChatId))
	}

	pa := &ThreadCreated{
		AdditionalData: s.AdditionalData,
		ChatId:         s.ChatId,
	}

	err = eventBus.Publish(ctx, pa)
	if err != nil {
		return err
	}

	return nil
}
