package db

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/rotisserie/eris"
	"nkonev.name/chat/logger"
	"time"
)

func MigrateMessages(ctx context.Context, lgr *logger.Logger, fromDbConnect, toDbConnect string) error {
	fromDdb, err := sql.Open(postgresDriverString, fromDbConnect)
	if err != nil {
		return eris.Wrap(err, "error during creating from db")
	}
	defer fromDdb.Close()
	lgr.Infof("Connected to from db")

	toDdb, err := sql.Open(postgresDriverString, toDbConnect)
	if err != nil {
		return eris.Wrap(err, "error during creating to db")
	}
	defer toDdb.Close()
	lgr.Infof("Connected to to db")

	// chat
	chatRows, err := fromDdb.QueryContext(ctx, `
	SELECT 
		id,
		title,
		create_date_time,
		last_update_date_time,
		tet_a_tet,
		avatar,
		avatar_big,
		can_resend,
		available_to_search,
		blog,
		regular_participant_can_publish_message,
		regular_participant_can_pin_message,
		blog_about,
		regular_participant_can_write_message
	FROM chat order by id`)
	if err != nil {
		return eris.Wrap(err, "error during querying chat")
	}
	defer chatRows.Close()
	for chatRows.Next() {
		var chatId int64
		var cTitle string
		var cCreateTs time.Time
		var cLastUpdateTs time.Time
		var cTetAtet bool
		var cAvatar *string
		var cAvatarBig *string
		var cCanResend bool
		var cAvailToSearch bool
		var cBlog bool
		var cRegularParticipantCanPublishMessage bool
		var cRegularParticipantCanPinMessage bool
		var cBlogAbout bool
		var cRegularParticipantCanWriteMessage bool
		err = chatRows.Scan(&chatId, &cTitle, &cCreateTs, &cLastUpdateTs, &cTetAtet, &cAvatar, &cAvatarBig, &cCanResend, &cAvailToSearch, &cBlog, &cRegularParticipantCanPublishMessage, &cRegularParticipantCanPinMessage, &cBlogAbout, &cRegularParticipantCanWriteMessage)
		if err != nil {
			return eris.Wrap(err, "error during scanning chat")
		}
		lgr.Infof("Processing chat %v", chatId)

		_, err = toDdb.ExecContext(ctx, `
				INSERT INTO chat(
									id,
									title,
									create_date_time,
									last_update_date_time,
									tet_a_tet,
									avatar,
									avatar_big,
									can_resend,
									available_to_search,
									blog,
									regular_participant_can_publish_message,
									regular_participant_can_pin_message,
									blog_about,
									regular_participant_can_write_message
				) values 
					($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
			`, chatId, cTitle, cCreateTs, cLastUpdateTs, cTetAtet, cAvatar, cAvatarBig, cCanResend, cAvailToSearch, cBlog, cRegularParticipantCanPublishMessage, cRegularParticipantCanPinMessage, cBlogAbout, cRegularParticipantCanWriteMessage)
		if err != nil {
			return eris.Wrap(err, "error during inserting chat")
		}

		lgr.Infof("For chat %v chat was copied", chatId)

		// chat participant
		chatParticipantRows, err := fromDdb.QueryContext(ctx, fmt.Sprintf(`
			select
    		chat_id, user_id, admin, create_date_time
			from chat_participant where chat_id = %v
		`, chatId))
		if err != nil {
			return eris.Wrap(err, "error during querying chat participant")
		}
		defer chatParticipantRows.Close()
		for chatParticipantRows.Next() {
			var cpChatId int64
			var cpUserId int64
			var cpAdmin bool
			var cpCreateTs time.Time
			err = chatParticipantRows.Scan(
				&cpChatId,
				&cpUserId,
				&cpAdmin,
				&cpCreateTs,
			)
			if err != nil {
				return eris.Wrap(err, "error during scanning chat participants")
			}

			_, err = toDdb.ExecContext(ctx, `
				INSERT INTO chat_participant(chat_id, user_id, admin, create_date_time) values 
					($1, $2, $3, $4)
			`, cpChatId, cpUserId, cpAdmin, cpCreateTs)
			if err != nil {
				return eris.Wrap(err, "error during inserting message reaction")
			}
		}
		lgr.Infof("For chat %v chat participants were copied", chatId)

		// chat pinned
		chatPinnedRows, err := fromDdb.QueryContext(ctx, fmt.Sprintf(`
			select
    		user_id, 
    		chat_id
			from chat_pinned where chat_id = %v
		`, chatId))
		if err != nil {
			return eris.Wrap(err, "error during querying chat pinned")
		}
		defer chatPinnedRows.Close()
		for chatPinnedRows.Next() {
			var cpiUserId int64
			var cpiChatId int64
			err = chatPinnedRows.Scan(
				&cpiUserId,
				&cpiChatId,
			)
			if err != nil {
				return eris.Wrap(err, "error during scanning chat pinned")
			}

			_, err = toDdb.ExecContext(ctx, `
				INSERT INTO chat_pinned(user_id, chat_id) values 
					($1, $2)
			`, cpiUserId, cpiChatId)
			if err != nil {
				return eris.Wrap(err, "error during inserting message reaction")
			}
		}
		lgr.Infof("For chat %v pinned were copied", chatId)

		// message
		messageRows, err := fromDdb.QueryContext(ctx, fmt.Sprintf(`
			select
    		m.id, 
    		m.text, 
    		m.owner_id,
    		m.create_date_time, 
    		m.edit_date_time, 
    		m.file_item_uuid,
			m.embed_message_type,
			m.embed_message_id,
			m.embed_chat_id,
			m.embed_owner_id,
			m.pinned,
			m.pin_promoted,
			m.blog_post,
			m.published
			from message_chat_%v m order by id
		`, chatId))
		if err != nil {
			return eris.Wrap(err, "error during querying message")
		}
		defer messageRows.Close()
		for messageRows.Next() {
			var mId int64
			var mText string
			var mOwnerId int64
			var mCreateTs time.Time
			var mEditTs *time.Time
			var mFileItemUuid *string
			var meMessageType *string
			var meMessageId *int64
			var meChatId *int64
			var meOwnerId *int64
			var mPinned bool
			var mPinPromoted bool
			var mBlogPost bool
			var mPublished bool
			err = messageRows.Scan(
				&mId,
				&mText,
				&mOwnerId,
				&mCreateTs,
				&mEditTs,
				&mFileItemUuid,
				&meMessageType,
				&meMessageId,
				&meChatId,
				&meOwnerId,
				&mPinned,
				&mPinPromoted,
				&mBlogPost,
				&mPublished,
			)
			if err != nil {
				return eris.Wrap(err, "error during scanning message")
			}

			_, err = toDdb.ExecContext(ctx, `
				INSERT INTO message(
				                    id, 
				                    chat_id,
				                    text, 
				                    owner_id,
				                    create_date_time,
				                    edit_date_time,
				                    file_item_uuid,
				                    embed_message_type,
				                    embed_message_id,
				                    embed_chat_id,
				                    embed_owner_id,
				                    pinned,
				                    pin_promoted,
				                    blog_post,
				                    published
				                    ) values 
					($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
			`, mId, chatId, mText, mOwnerId, mCreateTs, mEditTs, mFileItemUuid, meMessageType, meMessageId, meChatId, meOwnerId, mPinned, mPinPromoted, mBlogPost, mPublished)
			if err != nil {
				return eris.Wrap(err, "error during inserting message")
			}
		}
		lgr.Infof("For chat %v messages were copied", chatId)

		// message reactions
		messageReactionRows, err := fromDdb.QueryContext(ctx, fmt.Sprintf(`
			select
    		m.user_id, 
    		m.reaction, 
    		m.message_id
			from message_reaction_chat_%v m
		`, chatId))
		if err != nil {
			return eris.Wrap(err, "error during querying message reaction")
		}
		defer messageReactionRows.Close()
		for messageReactionRows.Next() {
			var mUserId int64
			var mMessageId int64
			var mReaction string
			err = messageReactionRows.Scan(
				&mUserId,
				&mReaction,
				&mMessageId,
			)
			if err != nil {
				return eris.Wrap(err, "error during scanning message reaction")
			}

			_, err = toDdb.ExecContext(ctx, `
				INSERT INTO message_reaction(user_id, reaction, message_id, chat_id) values 
					($1, $2, $3, $4)
			`, mUserId, mReaction, mMessageId, chatId)
			if err != nil {
				return eris.Wrap(err, "error during inserting message reaction")
			}
		}
		lgr.Infof("For chat %v message reactions were copied", chatId)
	}

	_, err = toDdb.ExecContext(ctx, `SELECT setval('chat_id_seq', (SELECT MAX(id) FROM chat));`)
	if err != nil {
		return eris.Wrap(err, "error during setting chat sequence")
	}

	lgr.Infof("Successfully fin")
	return nil
}
