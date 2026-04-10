package cqrs

import (
	"context"
	"errors"
	"fmt"
	"github.com/georgysavva/scany/v2/pgxscan"
	"nkonev.name/chat/config"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
	"time"

	sqlscanv2 "github.com/georgysavva/scany/v2/sqlscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RunMigrateFromOldDb(cfg *config.AppConfig, eventBus *KafkaProducer, lgr *logger.LoggerWrapper, dba *db.DB, commonProjection *CommonProjection) error {
	ctx := context.Background()

	isNeedToSkipMigrate, err := commonProjection.GetIsNeedToSkipMigrate(ctx, dba)
	if err != nil {
		return err
	}

	if isNeedToSkipMigrate {
		lgr.InfoContext(ctx, "Skipping old db migration because already migrated")
		return nil
	}

	lgr.InfoContext(ctx, "Starting old db migration from the old db")
	config, err := pgxpool.ParseConfig(cfg.PostgreSQLOld.Url)
	if err != nil {
		return err
	}

	connOldDb, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return err
	}

	defer connOldDb.Close()

	// * migrate chats
	chatOffset := 0
	for {
		lgr.InfoContext(ctx, "Starting migrating chats bunch on the offset", "chat_offset", chatOffset)

		type oldChat struct {
			Id                                  int64     `db:"id"`
			Title                               string    `db:"title"`
			CreateDateTime                      time.Time `db:"create_date_time"`
			TetATet                             bool      `db:"tet_a_tet"`
			Avatar                              *string   `db:"avatar"`
			AvatarBig                           *string   `db:"avatar_big"`
			CanResend                           bool      `db:"can_resend"`
			AvailableToSearch                   bool      `db:"available_to_search"`
			Blog                                bool      `db:"blog"`
			RegularParticipantCanPublishMessage bool      `db:"regular_participant_can_publish_message"`
			RegularParticipantCanPinMessage     bool      `db:"regular_participant_can_pin_message"`
			BlogAbout                           bool      `db:"blog_about"`
			RegularParticipantCanWriteMessage   bool      `db:"regular_participant_can_write_message"`
			CanReact                            bool      `db:"can_react"`
		}
		oldChats := []oldChat{}
		err = pgxscan.Select(ctx, connOldDb, &oldChats, `
			select
				id
				,title
				,create_date_time
				,tet_a_tet
				,avatar
				,avatar_big
				,can_resend
				,available_to_search
				,blog
				,regular_participant_can_publish_message
				,regular_participant_can_pin_message
				,blog_about
				,regular_participant_can_write_message
				,can_react
			from chat
			order by id
			limit $1 offset $2
		`, utils.DefaultSize, chatOffset)
		if err != nil {
			return fmt.Errorf("error during get old chats: %w", err)
		}

		for _, oldChat := range oldChats {
			lgr.InfoContext(ctx, "Starting migrating chat on the offset", "chat_offset", chatOffset, logger.AttributeChatId, oldChat.Id)

			var behalfUserId int64
			err = pgxscan.Get(ctx, connOldDb, &behalfUserId, `
				select user_id from chat_participant where chat_id = $1 order by create_date_time limit 1
			`, oldChat.Id)
			if err != nil {
				return fmt.Errorf("error during get behalfUserId: %w", err)
			}

			var tetATetOppositeUserId *int64
			if oldChat.TetATet {
				err = pgxscan.Get(ctx, connOldDb, &tetATetOppositeUserId, `
					select user_id from chat_participant where chat_id = $1 and user_id != $2 order by create_date_time limit 1
				`, oldChat.Id, behalfUserId)
				if errors.Is(err, pgx.ErrNoRows) {
					// nothing
				} else if err != nil {
					return fmt.Errorf("error during get tetATetOppositeUserId: %w", err)
				}
			}

			err = eventBus.Publish(ctx, &ChatCreated{
				AdditionalData: &AdditionalData{
					CreatedAt:    oldChat.CreateDateTime,
					BehalfUserId: behalfUserId,
				},
				TetATet:               oldChat.TetATet,
				TetATetOppositeUserId: tetATetOppositeUserId,
				ChatCommoned: ChatCommoned{
					ChatId:                              oldChat.Id,
					Title:                               oldChat.Title,
					Blog:                                oldChat.Blog,
					BlogAbout:                           oldChat.BlogAbout,
					Avatar:                              oldChat.Avatar,
					AvatarBig:                           oldChat.AvatarBig,
					CanResend:                           oldChat.CanResend,
					CanReact:                            oldChat.CanReact,
					AvailableToSearch:                   oldChat.AvailableToSearch,
					RegularParticipantCanPublishMessage: oldChat.RegularParticipantCanPublishMessage,
					RegularParticipantCanPinMessage:     oldChat.RegularParticipantCanPinMessage,
					RegularParticipantCanWriteMessage:   oldChat.RegularParticipantCanWriteMessage,
					RegularParticipantCanAddParticipant: false,
				},
			})
			if err != nil {
				return err
			}

			// * * migrate participants
			participantOffset := 0
			for {
				lgr.InfoContext(ctx, "Starting migrating participants bunch on the offset", "chat_offset", chatOffset, "participant_offset", participantOffset, logger.AttributeChatId, oldChat.Id)
				type oldParticipant struct {
					ChatId         int64     `db:"chat_id"`
					UserId         int64     `db:"user_id"`
					Admin          bool      `db:"admin"`
					CreateDateTime time.Time `db:"create_date_time"`
				}
				oldParticipants := []oldParticipant{}
				err = pgxscan.Select(ctx, connOldDb, &oldParticipants, `
					select
						chat_id
						,user_id
						,admin
						,create_date_time
					from chat_participant
					where chat_id = $1
					order by create_date_time
					limit $2 offset $3
				`, oldChat.Id, utils.DefaultSize, participantOffset)
				if err != nil {
					return fmt.Errorf("error during get old participants: %w", err)
				}

				pa := &ParticipantsAdded{
					AdditionalData: GenerateMessageAdditionalData(nil, behalfUserId),
					ChatId:         oldChat.Id,
					IsChatCreating: true,
				}

				for _, oldParticipant := range oldParticipants {
					lgr.InfoContext(ctx, "Migrating participant on the offset", "chat_offset", chatOffset, "participant_offset", participantOffset, logger.AttributeChatId, oldChat.Id, logger.AttributeUserId, oldParticipant.UserId)

					pa.Participants = append(pa.Participants, ParticipantWithAdmin{
						ParticipantId: oldParticipant.UserId,
						ChatAdmin:     oldParticipant.Admin,
					})
				}
				err = eventBus.Publish(ctx, pa)
				if err != nil {
					return err
				}

				lgr.InfoContext(ctx, "Finishing migrating participants bunch on the offset", "chat_offset", chatOffset, "participant_offset", participantOffset, logger.AttributeChatId, oldChat.Id)
				if len(oldParticipants) < utils.DefaultSize {
					break
				}
				participantOffset += utils.DefaultSize
			}

			var pinnedRromotedMessageId, pinnedRromotedMessageOwnerId *int64

			// * * migrate messages
			messageOffset := 0
			for {
				lgr.InfoContext(ctx, "Starting migrating messages bunch on the offset", "chat_offset", chatOffset, "message_offset", messageOffset)

				type oldMessage struct {
					Id               int64      `db:"id"`
					Text             string     `db:"text"`
					OwnerId          int64      `db:"owner_id"`
					CreateDateTime   time.Time  `db:"create_date_time"`
					EditDateTime     *time.Time `db:"edit_date_time"`
					FileItemUuid     *string    `db:"file_item_uuid"`
					EmbedMessageId   *int64     `db:"embed_message_id"`
					EmbedChatId      *int64     `db:"embed_chat_id"`
					EmbedOwnerId     *int64     `db:"embed_owner_id"`
					EmbedMessageType *string    `db:"embed_message_type"`
					Pinned           bool       `db:"pinned"`
					PinPromoted      bool       `db:"pin_promoted"`
					BlogPost         bool       `db:"blog_post"`
					Published        bool       `db:"published"`
					ChatId           int64      `db:"chat_id"`
				}
				oldMessages := []oldMessage{}
				err = pgxscan.Select(ctx, connOldDb, &oldMessages, `
					select
						id
						,text
						,owner_id
						,create_date_time
						,edit_date_time
						,file_item_uuid
						,embed_message_id
						,embed_chat_id
						,embed_owner_id
						,embed_message_type
						,pinned
						,pin_promoted
						,blog_post
						,published
						,chat_id
					from message
					where chat_id = $1
					order by id
					limit $2 offset $3
				`, oldChat.Id, utils.DefaultSize, messageOffset)
				if err != nil {
					return fmt.Errorf("error during get old messages: %w", err)
				}

				for _, oldMessage := range oldMessages {
					lgr.InfoContext(ctx, "Starting migrating message on the offset", "chat_offset", chatOffset, "message_offset", messageOffset, logger.AttributeChatId, oldChat.Id, logger.AttributeMessageId, oldMessage.Id)

					if oldMessage.PinPromoted {
						pinnedRromotedMessageId = &oldMessage.Id
						pinnedRromotedMessageOwnerId = &oldMessage.OwnerId
					}

					// send to the event
					mc := &MessageCreated{
						MessageCommoned: MessageCommoned{
							Id:           oldMessage.Id,
							ChatId:       oldChat.Id,
							Content:      oldMessage.Text,
							FileItemUuid: oldMessage.FileItemUuid,
						},
						AdditionalData: &AdditionalData{
							CreatedAt:    oldMessage.CreateDateTime,
							BehalfUserId: oldMessage.OwnerId,
						},
					}

					getEmbedContent := func(chatId, messageId int64) (*string, error) {
						var c string
						err = pgxscan.Get(ctx, connOldDb, &c, `
							select 
								text
							from message
							where chat_id = $1 and id = $2
						`, chatId, messageId)
						if errors.Is(err, pgx.ErrNoRows) {
							// there were no rows, but otherwise no error occurred
							return nil, nil
						} else if err != nil {
							return nil, fmt.Errorf("error during getEmbedContent: %w", err)
						}

						return &c, nil
					}

					getEmbedOwner := func(chatId, messageId int64) (*int64, error) {
						var c int64
						err = pgxscan.Get(ctx, connOldDb, &c, `
							select 
								owner_id
							from message
							where chat_id = $1 and id = $2
						`, chatId, messageId)
						if errors.Is(err, pgx.ErrNoRows) {
							// there were no rows, but otherwise no error occurred
							return nil, nil
						} else if err != nil {
							return nil, fmt.Errorf("error during getEmbedOwner: %w", err)
						}

						return &c, nil
					}

					if oldMessage.EmbedMessageType != nil {
						if *oldMessage.EmbedMessageType == "reply" {
							ec, err := getEmbedContent(oldChat.Id, *oldMessage.EmbedMessageId)
							if err != nil {
								return err
							}

							eo, err := getEmbedOwner(oldChat.Id, *oldMessage.EmbedMessageId)
							if err != nil {
								return err
							}

							if ec != nil && eo != nil {
								mc.MessageCommoned.Embed = dto.NewEmbedReply(
									*oldMessage.EmbedMessageId,
									*ec,
									*eo,
								)
							}
						} else if *oldMessage.EmbedMessageType == "resend" {
							ec, err := getEmbedContent(*oldMessage.EmbedChatId, *oldMessage.EmbedMessageId)
							if err != nil {
								return err
							}
							if ec != nil {
								mc.MessageCommoned.Embed = dto.NewEmbedResend(
									*oldMessage.EmbedMessageId,
									*ec,
									*oldMessage.EmbedOwnerId,
									*oldMessage.EmbedChatId,
								)
							}
						}
					}

					err = eventBus.Publish(ctx, mc)
					if err != nil {
						return err
					}

					// * * * migrate pinned
					if oldMessage.Pinned {
						cpin := &MessagePinned{
							AdditionalData: GenerateMessageAdditionalData(nil, oldMessage.OwnerId),
							ChatId:         oldChat.Id,
							MessageId:      oldMessage.Id,
							Pinned:         oldMessage.Pinned,
						}
						err = eventBus.Publish(ctx, cpin)
						if err != nil {
							return err
						}
					}

					// * * * migrate published
					if oldMessage.Published {
						cpub := &MessagePublished{
							AdditionalData: GenerateMessageAdditionalData(nil, oldMessage.OwnerId),
							ChatId:         oldChat.Id,
							MessageId:      oldMessage.Id,
							Published:      oldMessage.Published,
						}
						err = eventBus.Publish(ctx, cpub)
						if err != nil {
							return err
						}
					}

					// message blog post
					if oldChat.Blog && oldMessage.BlogPost {
						ev := MessageBlogPostMade{
							AdditionalData: GenerateMessageAdditionalData(nil, oldMessage.OwnerId),
							ChatId:         oldChat.Id,
							MessageId:      oldMessage.Id,
							BlogPost:       true,
						}

						err = eventBus.Publish(ctx, &ev)
						if err != nil {
							return err
						}
					}

					// reactions
					lgr.InfoContext(ctx, "Starting migrating reactions on the offset", "chat_offset", chatOffset, "message_offset", messageOffset, logger.AttributeChatId, oldChat.Id, logger.AttributeMessageId, oldMessage.Id)
					type oldReaction struct {
						UserId    int64  `db:"user_id"`
						Reaction  string `db:"reaction"`
						MessageId int64  `db:"message_id"`
						ChatId    int64  `db:"chat_id"`
					}
					oldReactions := []oldReaction{}
					err = pgxscan.Select(ctx, connOldDb, &oldReactions, `
						select
							user_id
							,reaction
							,message_id
							,chat_id
						from message_reaction
						where chat_id = $1 and message_id = $2
						order by user_id
					`, oldChat.Id, oldMessage.Id)
					if err != nil {
						return fmt.Errorf("error during get old reactions: %w", err)
					}

					for _, oldReaction := range oldReactions {
						fl := &MessageReactionCreated{
							AdditionalData: GenerateMessageAdditionalData(nil, oldReaction.UserId),
							MessageReactionCommoned: MessageReactionCommoned{
								ChatId:    oldChat.Id,
								MessageId: oldMessage.Id,
								Reaction:  oldReaction.Reaction,
							},
						}

						err = eventBus.Publish(ctx, fl)
						if err != nil {
							return err
						}
					}

					lgr.InfoContext(ctx, "Finishing migrating reactions on the offset", "chat_offset", chatOffset, "message_offset", messageOffset, logger.AttributeChatId, oldChat.Id, logger.AttributeMessageId, oldMessage.Id)

					lgr.InfoContext(ctx, "Finishing migrating message on the offset", "chat_offset", chatOffset, "message_offset", messageOffset, logger.AttributeChatId, oldChat.Id, logger.AttributeMessageId, oldMessage.Id)
				}

				lgr.InfoContext(ctx, "Finishing migrating messages bunch on the offset", "chat_offset", chatOffset, "message_offset", messageOffset, logger.AttributeChatId, oldChat.Id)
				if len(oldMessages) < utils.DefaultSize {
					break
				}
				messageOffset += utils.DefaultSize
			}

			if pinnedRromotedMessageId != nil && pinnedRromotedMessageOwnerId != nil {
				cpin := &MessagePinned{
					AdditionalData: GenerateMessageAdditionalData(nil, *pinnedRromotedMessageOwnerId),
					ChatId:         oldChat.Id,
					MessageId:      *pinnedRromotedMessageId,
					Pinned:         true,
				}
				err = eventBus.Publish(ctx, cpin)
				if err != nil {
					return err
				}
			}

			lgr.InfoContext(ctx, "Finishing migrating chat on the offset", "chat_offset", chatOffset, logger.AttributeChatId, oldChat.Id)
		}

		lgr.InfoContext(ctx, "Finishing migrating chats bunch on the offset", "chat_offset", chatOffset)
		if len(oldChats) < utils.DefaultSize {
			break
		}
		chatOffset += utils.DefaultSize
	}

	// * * migrate chat pinneds
	chatPinnedOffset := 0
	for {
		lgr.InfoContext(ctx, "Starting migrating chat pinned bunch on the offset", "chat_pinned_offset", chatPinnedOffset)
		type oldChatPinned struct {
			ChatId int64 `db:"chat_id"`
			UserId int64 `db:"user_id"`
		}
		oldChatPinneds := []oldChatPinned{}
		err = pgxscan.Select(ctx, connOldDb, &oldChatPinneds, `
					select
						chat_id
						,user_id
					from chat_pinned
					order by chat_id
					limit $1 offset $2
				`, utils.DefaultSize, chatPinnedOffset)
		if err != nil {
			return fmt.Errorf("error during get old chat pinneds: %w", err)
		}

		for _, oldChatPinned := range oldChatPinneds {
			chpin := &ChatPinned{
				AdditionalData: GenerateMessageAdditionalData(nil, oldChatPinned.UserId),
				ChatId:         oldChatPinned.ChatId,
				Pinned:         true,
			}
			err := eventBus.Publish(ctx, chpin)
			if err != nil {
				return err
			}
		}

		lgr.InfoContext(ctx, "Finishing migrating chat pinned bunch on the offset", "chat_pinned_offset", chatPinnedOffset)
		if len(oldChatPinneds) < utils.DefaultSize {
			break
		}
		chatPinnedOffset += utils.DefaultSize
	}

	// * * migrate user chat message readeds
	userChatMessageReadedOffset := 0
	for {
		lgr.InfoContext(ctx, "Starting migrating user chat message readed bunch on the offset", "user_chat_message_readed_offset", userChatMessageReadedOffset)
		type userChatMessageReaded struct {
			ChatId        int64 `db:"chat_id"`
			UserId        int64 `db:"user_id"`
			LastMessageId int64 `db:"last_message_id"`
		}
		oldUserChatMessageReadeds := []userChatMessageReaded{}
		err = pgxscan.Select(ctx, connOldDb, &oldUserChatMessageReadeds, `
					select
						chat_id
						,user_id
						,last_message_id
					from message_read
					order by chat_id, user_id
					limit $1 offset $2
				`, utils.DefaultSize, userChatMessageReadedOffset)
		if err != nil {
			return fmt.Errorf("error during get old user chat message readeds: %w", err)
		}

		for _, oldUSerChatMessageReaded := range oldUserChatMessageReadeds {
			chpin := &MessageReaded{
				AdditionalData:     GenerateMessageAdditionalData(nil, oldUSerChatMessageReaded.UserId),
				ChatId:             oldUSerChatMessageReaded.ChatId,
				MessageId:          oldUSerChatMessageReaded.LastMessageId,
				ReadMessagesAction: ReadMessagesActionOneMessage,
			}
			err := eventBus.Publish(ctx, chpin)
			if err != nil {
				return err
			}
		}

		lgr.InfoContext(ctx, "Finishing migrating user chat message readed bunch on the offset", "user_chat_message_readed_offset", userChatMessageReadedOffset)
		if len(oldUserChatMessageReadeds) < utils.DefaultSize {
			break
		}
		userChatMessageReadedOffset += utils.DefaultSize
	}

	err = commonProjection.SetIsNeedToSkipMigrate(ctx)
	if err != nil {
		return err
	}

	err = commonProjection.SetIsNeedToFastForwardSequences(ctx)
	if err != nil {
		return err
	}

	lgr.InfoContext(ctx, "Finishing old db migration from the old db")

	return nil
}

const need_to_skip_migrate_key = "need_to_skip_old_db_migration"
const need_to_skip_migrate_value = "true"

func (m *CommonProjection) SetIsNeedToSkipMigrate(ctx context.Context) error {
	_, err := m.db.ExecContext(ctx, "insert into technical(the_key, the_value) values ($1, $2) on conflict (the_key) do update set the_value = excluded.the_value", need_to_skip_migrate_key, need_to_skip_migrate_value)
	return err
}

func (m *CommonProjection) UnsetIsNeedToSkipMigrate(ctx context.Context, co db.CommonOperations) error {
	_, err := co.ExecContext(ctx, "delete from technical where the_key = $1", need_to_skip_migrate_key)
	return err
}

func (m *CommonProjection) GetIsNeedToSkipMigrate(ctx context.Context, co db.CommonOperations) (bool, error) {
	var e bool
	err := sqlscanv2.Get(ctx, co, &e, "select exists(select * from technical where the_key = $1 and the_value = $2)", need_to_skip_migrate_key, need_to_skip_migrate_value)
	if err != nil {
		return false, err
	}
	return e, err
}
