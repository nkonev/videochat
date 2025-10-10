package services

import (
	"context"
	"net/http"
	"nkonev.name/chat/cqrs"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
	"strings"
)

type AuthorizationService struct {
	lgr              *logger.LoggerWrapper
	dbWrapper        *db.DB
	commonProjection *cqrs.CommonProjection
}

func NewAuthorizationService(
	lgr *logger.LoggerWrapper,
	dbWrapper *db.DB,
	commonProjection *cqrs.CommonProjection,
) *AuthorizationService {
	return &AuthorizationService{
		lgr:              lgr,
		dbWrapper:        dbWrapper,
		commonProjection: commonProjection,
	}
}

func (ch *AuthorizationService) CheckAccess(ctx context.Context, params map[string]string) int {
	chatId, err := utils.ParseInt64(params["chatId"])
	if err != nil {
		ch.lgr.ErrorContext(ctx, "Error checking access", logger.AttributeError, err)
		return http.StatusInternalServerError
	}
	originalChat, err := ch.commonProjection.GetChatBasic(ctx, ch.dbWrapper, chatId) // chat where the file is stored
	if err != nil {
		ch.lgr.ErrorContext(ctx, "Error checking access", logger.AttributeError, err)
		return http.StatusInternalServerError
	}
	if originalChat == nil {
		return http.StatusUnauthorized
	}

	// this branch is for "public"
	// overrideChatId and overrideMessageId come together
	// in general, they can be crafted by an intruder ...
	overrideMessageId, _ := utils.ParseInt64(params[dto.OverrideMessageId])
	if overrideMessageId > 0 {
		overrideChatId, err := utils.ParseInt64(params[dto.OverrideChatId])
		if err != nil {
			ch.lgr.ErrorContext(ctx, "Error checking access", logger.AttributeError, err)
			return http.StatusInternalServerError
		}

		overrideMessage, err := ch.commonProjection.GetMessageBasic(ctx, ch.dbWrapper, overrideChatId, overrideMessageId)
		if err != nil {
			ch.lgr.ErrorContext(ctx, "Error checking access", logger.AttributeError, err)
			return http.StatusInternalServerError
		}
		overrideChat, err := ch.commonProjection.GetChatBasic(ctx, ch.dbWrapper, overrideChatId) // chat where the embedded message is stored
		if err != nil {
			ch.lgr.ErrorContext(ctx, "Error checking access", logger.AttributeError, err)
			return http.StatusInternalServerError
		}
		if overrideChat == nil {
			return http.StatusUnauthorized
		}

		fileItemUuid := params["fileItemUuid"]
		if overrideMessage != nil && (overrideChat.IsBlog || overrideMessage.Published || overrideMessage.BlogPost) {

			// ... here we check that the message which we found by potentially crafted overrideMessageId / overrideChatId with malicious intent
			// really contains this fileItemUuid
			encodedFileItemUuid := utils.UrlEncode(fileItemUuid)
			if len(fileItemUuid) != 0 {
				if strings.Contains(overrideMessage.Content, encodedFileItemUuid) {
					return http.StatusOK
				} else if overrideMessage.FileItemUuid != nil && *overrideMessage.FileItemUuid == fileItemUuid {
					return http.StatusOK
				}
			}
		}
		return http.StatusUnauthorized
	}

	// this branch is for "regular" and resent
	userId, err := utils.ParseInt64(params["userId"])
	if err != nil {
		ch.lgr.InfoContext(ctx, "Unable to get userId", logger.AttributeError, err) // it can be error when overrideChatId and overrideMessageId are missed
		return http.StatusUnauthorized
	}
	useCanResend := utils.GetBoolean(params["considerCanResend"])
	participant, err := ch.commonProjection.IsParticipant(ctx, ch.dbWrapper, userId, chatId)
	if err != nil {
		ch.lgr.ErrorContext(ctx, "Error checking access", logger.AttributeError, err)
		return http.StatusInternalServerError
	}

	if participant {
		return http.StatusOK
	} else {
		if useCanResend {
			if originalChat.CanResend {
				return http.StatusOK
			} else {
				return http.StatusUnauthorized
			}
		}
	}
	return http.StatusUnauthorized
}
