package handlers

import (
	"context"
	"errors"
	"fmt"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/guregu/null"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"net/http"
	"nkonev.name/chat/auth"
	"nkonev.name/chat/client"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/services"
	"nkonev.name/chat/utils"
	"strings"
)

const minChatNameLen = 1
const maxChatNameLen = 256

type ParticipantsWithAdminWrapper struct {
	Data  []*dto.UserWithAdmin `json:"items"`
	Count int                  `json:"count"` // for paginating purposes
}

type ParticipantsWrapper struct {
	Data  []*dto.User `json:"participants"`
	Count int         `json:"participantsCount"` // for paginating purposes
}

type MessageReadResponse struct {
	ParticipantsWrapper
	Text string `json:"text"`
}

type EditChatDto struct {
	Id int64 `json:"id"`
	CreateChatDto
}

type CreateChatDto struct {
	Name                                string      `json:"name"`
	ParticipantIds                      *[]int64    `json:"participantIds"`
	Avatar                              null.String `json:"avatar"`
	AvatarBig                           null.String `json:"avatarBig"`
	CanResend                           bool        `json:"canResend"`
	AvailableToSearch                   bool        `json:"availableToSearch"`
	Blog                                *bool       `json:"blog"`
	RegularParticipantCanPublishMessage bool        `json:"regularParticipantCanPublishMessage"`
	RegularParticipantCanPinMessage     bool        `json:"regularParticipantCanPinMessage"`
}

type ChatHandler struct {
	db                     *db.DB
	notificator            *services.Events
	restClient             *client.RestClient
	policy                 *services.SanitizerPolicy
	stripTagsPolicy        *services.StripTagsPolicy
	onlyAdminCanCreateBlog bool
}

func NewChatHandler(dbR *db.DB, notificator *services.Events, restClient *client.RestClient, policy *services.SanitizerPolicy, cleanTagsPolicy *services.StripTagsPolicy) *ChatHandler {
	return &ChatHandler{
		db:                     dbR,
		notificator:            notificator,
		restClient:             restClient,
		policy:                 policy,
		stripTagsPolicy:        cleanTagsPolicy,
		onlyAdminCanCreateBlog: viper.GetBool("onlyAdminCanCreateBlog"),
	}
}

func (a *CreateChatDto) Validate() error {
	return validation.ValidateStruct(a,
		validation.Field(&a.Name, validation.Required, validation.Length(minChatNameLen, maxChatNameLen), validation.NotIn(db.ReservedPublicallyAvailableForSearchChats)),
	)
}

func (a *EditChatDto) Validate() error {
	return validation.ValidateStruct(a,
		validation.Field(&a.Name, validation.Required, validation.Length(minChatNameLen, maxChatNameLen), validation.NotIn(db.ReservedPublicallyAvailableForSearchChats)),
		validation.Field(&a.Id, validation.Required),
	)
}

func (ch *ChatHandler) GetChats(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	var startingFromItemId *int64
	startingFromItemIdString := c.QueryParam("startingFromItemId")
	if startingFromItemIdString == "" {
		startingFromItemId = nil
	} else {
		startingFromItemId2, err := utils.ParseInt64(startingFromItemIdString) // exclusive
		if err != nil {
			return err
		}
		startingFromItemId = &startingFromItemId2
	}

	size := utils.FixSizeString(c.QueryParam("size"))
	reverse := utils.GetBoolean(c.QueryParam("reverse"))
	searchString := c.QueryParam("searchString")
	searchString = strings.TrimSpace(searchString)
	searchString = TrimAmdSanitize(ch.policy, searchString)

	hasHash := utils.GetBoolean(c.QueryParam("hasHash"))

	var additionalFoundUserIds = []int64{}

	if searchString != "" && searchString != db.ReservedPublicallyAvailableForSearchChats {
		users, _, err := ch.restClient.SearchGetUsers(c.Request().Context(), searchString, true, []int64{}, 0, 0)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error get users from aaa %v", err)
		}
		for _, u := range users {
			additionalFoundUserIds = append(additionalFoundUserIds, u.Id)
		}
	}

	return db.Transact(c.Request().Context(), ch.db, func(tx *db.Tx) error {

		dbChats, err := tx.GetChatsWithParticipants(c.Request().Context(), userPrincipalDto.UserId, size, startingFromItemId, reverse, hasHash, searchString, additionalFoundUserIds, 0, 0)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error get chats from db %v", err)
			return err
		}

		var chatIds []int64 = make([]int64, 0)
		for _, cc := range dbChats {
			chatIds = append(chatIds, cc.Id)
		}
		unreadMessageBatch, err := tx.GetUnreadMessagesCountBatch(c.Request().Context(), chatIds, userPrincipalDto.UserId)
		if err != nil {
			return err
		}

		membership, err := tx.GetAmIParticipantBatch(c.Request().Context(), chatIds, userPrincipalDto.UserId) // need to setting isResultFromSearch correctly
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error get chats with me from db %v", err)
			return err
		}

		chatDtos := make([]*dto.ChatDto, 0)
		for _, cc := range dbChats {
			messages := unreadMessageBatch[cc.Id]
			isParticipant := membership[cc.Id]

			cd := convertToDto(cc, []*dto.User{}, messages, isParticipant)

			chatDtos = append(chatDtos, cd)
		}

		var participantIdSet = map[int64]bool{}
		for _, chatDto := range chatDtos {
			for _, participantId := range chatDto.ParticipantIds {
				participantIdSet[participantId] = true
			}
		}
		var users = getUsersRemotelyOrEmpty(c.Request().Context(), participantIdSet, ch.restClient)
		for _, chatDto := range chatDtos {
			for _, participantId := range chatDto.ParticipantIds {
				user := users[participantId]
				if user != nil {
					chatDto.Participants = append(chatDto.Participants, user)
					utils.ReplaceChatNameToLoginForTetATet(chatDto, user, userPrincipalDto.UserId, len(chatDto.ParticipantIds) == 1)
				}
			}
		}

		GetLogEntry(c.Request().Context()).Infof("Successfully returning %v chats", len(chatDtos))
		return c.JSON(http.StatusOK, chatDtos)
	})
}

func (ch *ChatHandler) HasNewMessages(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	return db.Transact(c.Request().Context(), ch.db, func(tx *db.Tx) error {
		has, err := tx.HasUnreadMessages(c.Request().Context(), userPrincipalDto.UserId)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, &utils.H{"hasUnreadMessages": has})
	})
}

type ChatFilterDto struct {
	SearchString string `json:"searchString"`
	ChatId       int64  `json:"chatId"`
}

func (ch *ChatHandler) Filter(c echo.Context) error {
	var _, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	var bindTo = new(ChatFilterDto)
	if err := c.Bind(bindTo); err != nil {
		GetLogEntry(c.Request().Context()).Warnf("Error during binding to dto %v", err)
		return err
	}

	searchString := strings.TrimSpace(bindTo.SearchString)
	searchString = TrimAmdSanitize(ch.policy, searchString)

	return db.Transact(c.Request().Context(), ch.db, func(tx *db.Tx) error {
		found, err := tx.ChatFilter(c.Request().Context(), searchString, bindTo.ChatId)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, &utils.H{"found": found})
	})
}

func getChat(
	ctx context.Context,
	dbR db.CommonOperations,
	restClient *client.RestClient,
	chatId int64,
	behalfParticipantId int64,
	participantsSize, participantsOffset int,
) (*dto.ChatDto, error) {
	fixedParticipantsSize := utils.FixSize(participantsSize)

	var users []*dto.User = []*dto.User{}

	cc, err := dbR.GetChatWithParticipants(ctx, behalfParticipantId, chatId, fixedParticipantsSize, participantsOffset)
	if err != nil {
		return nil, err
	}
	if cc == nil {
		return nil, nil
	}

	users, err = restClient.GetUsers(ctx, cc.ParticipantsIds)
	if err != nil {
		users = []*dto.User{}
		GetLogEntry(ctx).Warn("Error during getting users from aaa")
	}

	unreadMessages, err := dbR.GetUnreadMessagesCount(ctx, cc.Id, behalfParticipantId)
	if err != nil {
		return nil, err
	}

	isParticipant, err := dbR.IsParticipant(ctx, behalfParticipantId, chatId)
	if err != nil {
		return nil, err
	}

	chatDto := convertToDto(cc, users, unreadMessages, isParticipant)

	if chatDto.IsTetATet {
		for _, participant := range users {

			isSingleParticipant := len(cc.ParticipantsIds) == 1

			utils.ReplaceChatNameToLoginForTetATet(chatDto, participant, behalfParticipantId, isSingleParticipant)

			// leave LastLoginDateTime not null only if the opposite user isn't online
			if participant.Id != behalfParticipantId {
				if participant.Id != behalfParticipantId && !isSingleParticipant {
					chatDto.SetLastLoginDateTime(participant.LastLoginDateTime)
				}

				onlines, err := restClient.GetOnlines(ctx, []int64{participant.Id}) // get online for opposite user
				if err != nil {
					GetLogEntry(ctx).Errorf("Unable to get online for the opposite user %v: %v", participant.Id, err)
					// nothing
				} else {
					if len(onlines) == 1 {
						if onlines[0].Online { // if the opposite user is online we don't need to show last login
							chatDto.SetLastLoginDateTime(null.TimeFromPtr(nil))
						}
					}
				}
			}
		}
	}

	return chatDto, nil
}

func (ch *ChatHandler) GetChat(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	participantsPage := utils.DefaultPage
	participantsSize := utils.DefaultSize
	participantsOffset := utils.GetOffset(participantsPage, participantsSize)

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	chat, err := getChat(c.Request().Context(), ch.db, ch.restClient, chatId, userPrincipalDto.UserId, participantsSize, participantsOffset)
	if err != nil {
		return err
	}
	if chat == nil {
		basic, err := ch.db.GetChatBasic(c.Request().Context(), chatId)
		if err != nil {
			return err
		}
		if basic != nil && (basic.AvailableToSearch || basic.IsBlog) {
			return c.NoContent(http.StatusResetContent)
		} else {
			return c.NoContent(http.StatusNoContent)
		}
	} else {

		GetLogEntry(c.Request().Context()).Infof("Successfully returning %v chat", chat)
		return c.JSON(http.StatusOK, chat)
	}
}

func convertToDto(c *db.ChatWithParticipants, users []*dto.User, unreadMessages int64, participant bool) *dto.ChatDto {
	b := dto.BaseChatDto{
		Id:                c.Id,
		Name:              c.Title,
		ParticipantIds:    c.ParticipantsIds,
		Avatar:            c.Avatar,
		AvatarBig:         c.AvatarBig,
		IsTetATet:         c.TetATet,
		CanResend:         c.CanResend,
		AvailableToSearch: c.AvailableToSearch,
		Pinned:            c.Pinned,
		// see also services/events.go:75 chatNotifyCommon()

		ParticipantsCount:                   c.ParticipantsCount,
		LastUpdateDateTime:                  c.LastUpdateDateTime,
		Blog:                                c.Blog,
		RegularParticipantCanPublishMessage: c.RegularParticipantCanPublishMessage,
		RegularParticipantCanPinMessage:     c.RegularParticipantCanPinMessage,
	}

	b.SetPersonalizedFields(c.IsAdmin, unreadMessages, participant)

	// set participant order as in c.ParticipantsIds
	orderedParticipants := make([]*dto.User, 0)
	for _, participantId := range c.ParticipantsIds {
		for _, u := range users {
			if u.Id == participantId {
				orderedParticipants = append(orderedParticipants, u)
				break
			}
		}
	}

	return &dto.ChatDto{
		BaseChatDto:  b,
		Participants: orderedParticipants,
	}
}

func (ch *ChatHandler) CreateChat(c echo.Context) error {
	var bindTo = new(CreateChatDto)
	if err := c.Bind(bindTo); err != nil {
		GetLogEntry(c.Request().Context()).Warnf("Error during binding to dto %v", err)
		return err
	}
	if valid, err := ValidateAndRespondError(c, bindTo); err != nil || !valid {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	if !ch.checkCanCreateBlog(userPrincipalDto, bindTo.Blog) {
		GetLogEntry(c.Request().Context()).Infof("Blog is disabled for regular users")
		bindTo.Blog = nil
	}

	creatableChat := convertToCreatableChat(bindTo, ch.stripTagsPolicy)
	if err := lateValidateChatTitle(creatableChat.Title); err != nil {
		GetLogEntry(c.Request().Context()).Infof("Failed late validation: %v", err.Error())
		return c.JSON(http.StatusBadRequest, &utils.H{"error": err.Error()})
	}

	chatId, errOuter := db.TransactWithResult(c.Request().Context(), ch.db, func(tx *db.Tx) (int64, error) {
		id, _, err := tx.CreateChat(c.Request().Context(), creatableChat)
		if err != nil {
			return 0, err
		}
		// add admin
		if err = tx.AddParticipant(c.Request().Context(), userPrincipalDto.UserId, id, true); err != nil {
			return 0, err
		}

		if bindTo.ParticipantIds != nil {
			participantIds := *bindTo.ParticipantIds
			// add other participants except admin
			for _, participantId := range participantIds {
				if participantId == userPrincipalDto.UserId {
					continue
				}
				if err = tx.AddParticipant(c.Request().Context(), participantId, id, false); err != nil {
					return 0, err
				}
			}
		}
		return id, nil
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
		return errOuter
	}

	errOuter = db.Transact(c.Request().Context(), ch.db, func(tx *db.Tx) error {
		chatDto, err := getChat(c.Request().Context(), tx, ch.restClient, chatId, userPrincipalDto.UserId, 0, 0)
		if err != nil {
			return err
		}

		err = tx.IterateOverChatParticipantIds(c.Request().Context(), chatId, func(participantIds []int64) error {
			areAdmins, err := getAreAdminsOfUserIds(c.Request().Context(), tx, participantIds, chatId)
			if err != nil {
				return err
			}

			ch.notificator.NotifyAboutNewChat(c.Request().Context(), chatDto, participantIds, len(chatDto.ParticipantIds) == 1, true, tx, areAdmins)
			return nil
		})
		if err != nil {
			return err
		}
		return c.JSON(http.StatusCreated, chatDto)
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

func lateValidateChatTitle(title string) error {
	if len(title) == 0 {
		return fmt.Errorf("empty chat title")
	}
	return nil
}

func convertToCreatableChat(d *CreateChatDto, policy *services.StripTagsPolicy) *db.Chat {
	isBlog := utils.NullableToBoolean(d.Blog)
	return &db.Chat{
		Title:                               TrimAmdSanitizeChatTitle(policy, d.Name),
		CanResend:                           d.CanResend,
		AvailableToSearch:                   d.AvailableToSearch,
		Blog:                                isBlog,
		RegularParticipantCanPublishMessage: d.RegularParticipantCanPublishMessage,
		RegularParticipantCanPinMessage:     d.RegularParticipantCanPinMessage,
	}
}

func (ch *ChatHandler) DeleteChat(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	errOuter := db.Transact(c.Request().Context(), ch.db, func(tx *db.Tx) error {
		if admin, err := tx.IsAdmin(c.Request().Context(), userPrincipalDto.UserId, chatId); err != nil {
			return err
		} else if !admin {
			return errors.New(fmt.Sprintf("User %v is not admin of chat %v", userPrincipalDto.UserId, chatId))
		}

		err = tx.IterateOverChatParticipantIds(c.Request().Context(), chatId, func(participantIds []int64) error {
			ch.notificator.NotifyAboutDeleteChat(c.Request().Context(), chatId, participantIds, tx)
			return nil
		})
		if err != nil {
			return err
		}

		if err := tx.DeleteChat(c.Request().Context(), chatId); err != nil {
			return err
		}

		return c.JSON(http.StatusAccepted, &utils.H{"id": chatId})
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

func (ch *ChatHandler) EditChat(c echo.Context) error {
	var bindTo = new(EditChatDto)
	if err := c.Bind(bindTo); err != nil {
		GetLogEntry(c.Request().Context()).Warnf("Error during binding to dto %v", err)
		return err
	}

	if valid, err := ValidateAndRespondError(c, bindTo); err != nil || !valid {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	if !ch.checkCanCreateBlog(userPrincipalDto, bindTo.Blog) {
		GetLogEntry(c.Request().Context()).Infof("Blog is disabled for regular users")
		bindTo.Blog = nil
	}

	chatTitle := TrimAmdSanitizeChatTitle(ch.stripTagsPolicy, bindTo.Name)
	if err := lateValidateChatTitle(chatTitle); err != nil {
		GetLogEntry(c.Request().Context()).Infof("Failed late validation: %v", err.Error())
		return c.JSON(http.StatusBadRequest, &utils.H{"error": err.Error()})
	}

	errOuter := db.Transact(c.Request().Context(), ch.db, func(tx *db.Tx) error {
		if admin, err := tx.IsAdmin(c.Request().Context(), userPrincipalDto.UserId, bindTo.Id); err != nil {
			return err
		} else if !admin {
			return errors.New(fmt.Sprintf("User %v is not admin of chat %v", userPrincipalDto.UserId, bindTo.Id))
		}

		chatBasicBefore, err := tx.GetChatBasic(c.Request().Context(), bindTo.Id)
		if err != nil {
			return err
		}

		_, err = tx.EditChat(
			c.Request().Context(),
			bindTo.Id,
			chatTitle,
			TrimAmdSanitizeAvatar(c.Request().Context(), ch.policy, bindTo.Avatar),
			TrimAmdSanitizeAvatar(c.Request().Context(), ch.policy, bindTo.AvatarBig),
			bindTo.CanResend,
			bindTo.AvailableToSearch,
			bindTo.Blog,
			bindTo.RegularParticipantCanPublishMessage,
			bindTo.RegularParticipantCanPinMessage,
		)
		if err != nil {
			return err
		}

		chatDto, err := getChat(c.Request().Context(), tx, ch.restClient, bindTo.Id, userPrincipalDto.UserId, 0, 0)
		if err != nil {
			return err
		}

		err = tx.IterateOverChatParticipantIds(c.Request().Context(), bindTo.Id, func(participantIds []int64) error {
			areAdmins, err := getAreAdminsOfUserIds(c.Request().Context(), tx, participantIds, bindTo.Id)
			if err != nil {
				return err
			}

			ch.notificator.NotifyAboutChangeChat(c.Request().Context(), chatDto, participantIds, len(chatDto.ParticipantIds) == 1, true, tx, areAdmins)

			if chatBasicBefore.RegularParticipantCanPublishMessage != bindTo.RegularParticipantCanPublishMessage ||
				chatBasicBefore.RegularParticipantCanPinMessage != bindTo.RegularParticipantCanPinMessage {
				regularParticipants := make([]int64, 0)
				for userId, isAdmin := range areAdmins {
					if !isAdmin {
						regularParticipants = append(regularParticipants, userId)
					}
				}

				ch.notificator.NotifyMessagesReloadCommand(c.Request().Context(), bindTo.Id, regularParticipants)
			}
			return nil
		})
		if err != nil {
			return err
		}
		return c.JSON(http.StatusAccepted, chatDto)

	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

func (ch *ChatHandler) LeaveChat(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	errOuter := db.Transact(c.Request().Context(), ch.db, func(tx *db.Tx) error {
		if err = tx.DeleteParticipant(c.Request().Context(), userPrincipalDto.UserId, chatId); err != nil {
			return err
		}

		firstUser, err := tx.GetFirstParticipant(c.Request().Context(), chatId)
		if err != nil {
			return err
		}
		if chatDto, err := getChat(c.Request().Context(), tx, ch.restClient, chatId, firstUser, 0, 0); err != nil {
			return err
		} else {

			err = tx.IterateOverChatParticipantIds(c.Request().Context(), chatId, func(participantIds []int64) error {
				areAdmins, err := getAreAdminsOfUserIds(c.Request().Context(), tx, participantIds, chatId)
				if err != nil {
					return err
				}

				ch.notificator.NotifyAboutDeleteParticipants(c.Request().Context(), participantIds, chatId, []int64{userPrincipalDto.UserId})
				ch.notificator.NotifyAboutChangeChat(c.Request().Context(), chatDto, participantIds, len(chatDto.ParticipantIds) == 1, true, tx, areAdmins)
				return nil
			})
			if err != nil {
				GetLogEntry(c.Request().Context()).Errorf("Error during getting chat participants %v", err)
				return c.NoContent(http.StatusInternalServerError)
			}
			if chatDto.AvailableToSearch || chatDto.Blog {
				// send duplicated event to the former user to re-draw chat on their search results
				ch.notificator.NotifyAboutRedrawLeftChat(c.Request().Context(), chatDto, userPrincipalDto.UserId, len(chatDto.ParticipantIds) == 1, false, tx, map[int64]bool{userPrincipalDto.UserId: false}) // false because userPrincipalDto left the chat
			} else {
				ch.notificator.NotifyAboutDeleteChat(c.Request().Context(), chatDto.Id, []int64{userPrincipalDto.UserId}, tx)
			}
			return c.JSON(http.StatusAccepted, chatDto)
		}
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

func (ch *ChatHandler) JoinChat(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	isAdmin := false

	errOuter := db.Transact(c.Request().Context(), ch.db, func(tx *db.Tx) error {
		chat, err := tx.GetChatBasic(c.Request().Context(), chatId)
		if err != nil {
			return err
		}
		if !chat.AvailableToSearch && !chat.IsBlog {
			GetLogEntry(c.Request().Context()).Infof("User %d isn't allowed to loin to this chat beacuse chat isn't avaliable for search", userPrincipalDto.UserId)
			return c.NoContent(http.StatusUnauthorized)
		}

		if err := tx.AddParticipant(c.Request().Context(), userPrincipalDto.UserId, chatId, isAdmin); err != nil {
			return err
		}
		return nil
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
		return errOuter
	}

	errOuter = db.Transact(c.Request().Context(), ch.db, func(tx *db.Tx) error {
		firstUser, err := tx.GetFirstParticipant(c.Request().Context(), chatId)
		if err != nil {
			return err
		}
		chatDto, err := getChat(c.Request().Context(), tx, ch.restClient, chatId, firstUser, 0, 0)
		if err != nil {
			return err
		}

		err = tx.IterateOverChatParticipantIds(c.Request().Context(), chatId, func(participantIds []int64) error {
			areAdmins, err := getAreAdminsOfUserIds(c.Request().Context(), tx, participantIds, chatId)
			if err != nil {
				return err
			}

			ch.notificator.NotifyAboutNewParticipants(c.Request().Context(), participantIds, chatId, []*dto.UserWithAdmin{
				{
					User: dto.User{
						Id:    userPrincipalDto.UserId,
						Login: userPrincipalDto.UserLogin,
					},
					Admin: isAdmin,
				},
			})
			ch.notificator.NotifyAboutChangeChat(c.Request().Context(), chatDto, participantIds, len(chatDto.ParticipantIds) == 1, true, tx, areAdmins)

			// update chats at left for the new user who joined
			if utils.Contains(participantIds, userPrincipalDto.UserId) {
				ch.notificator.NotifyAboutNewChat(c.Request().Context(), chatDto, []int64{userPrincipalDto.UserId}, len(chatDto.ParticipantIds) == 1, true, tx, areAdmins)
			}

			return nil
		})
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during getting chat participants %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
		return c.JSON(http.StatusAccepted, chatDto)
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

type ChangeAdminResponseDto struct {
	Admin bool `json:"admin"`
}

func (ch *ChatHandler) ChangeParticipant(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	errOuter := db.Transact(c.Request().Context(), ch.db, func(tx *db.Tx) error {

		// check that I am admin
		admin, err := tx.IsAdmin(c.Request().Context(), userPrincipalDto.UserId, chatId)
		if err != nil {
			return err
		}
		if !admin {
			return c.JSON(http.StatusUnauthorized, &utils.H{"message": "You have no access to this chat"})
		}
		interestingUserId, err := GetPathParamAsInt64(c, "participantId")
		participant, err := tx.IsParticipant(c.Request().Context(), interestingUserId, chatId)
		if err != nil {
			return err
		}
		if !participant {
			return c.JSON(http.StatusBadRequest, &utils.H{"message": "User is not belong to chat"})
		}

		newAdmin, err := GetQueryParamAsBoolean(c, "admin")
		if err != nil {
			return err
		}
		err = tx.SetAdmin(c.Request().Context(), interestingUserId, chatId, newAdmin)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during changing chat admin in database %v", err)
			return err
		}

		newUsersWithAdmin, err := ch.getParticipantsWithAdmin(tx, []int64{interestingUserId}, chatId, c.Request().Context())
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during getting participants aith admin %v", err)
			return err
		}

		err = tx.IterateOverChatParticipantIds(c.Request().Context(), chatId, func(participantIds []int64) error {
			ch.notificator.NotifyAboutChangeParticipants(c.Request().Context(), participantIds, chatId, newUsersWithAdmin)
			return nil
		})
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during getting chat participants %v", err)
			return err
		}

		chatDto, err := getChat(c.Request().Context(), tx, ch.restClient, chatId, userPrincipalDto.UserId, 0, 0)
		if err != nil {
			return err
		}
		ch.notificator.NotifyAboutChangeChat(c.Request().Context(), chatDto, []int64{interestingUserId}, len(chatDto.ParticipantIds) == 1, true, tx, map[int64]bool{interestingUserId: newAdmin})

		ch.notificator.NotifyMessagesReloadCommand(c.Request().Context(), chatId, []int64{interestingUserId})

		return c.JSON(http.StatusAccepted, newUsersWithAdmin)
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

func (ch *ChatHandler) PinChat(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	pin, err := GetQueryParamAsBoolean(c, "pin")
	if err != nil {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	return db.Transact(c.Request().Context(), ch.db, func(tx *db.Tx) error {
		if isParticipant, err := tx.IsParticipant(c.Request().Context(), userPrincipalDto.UserId, chatId); err != nil {
			return err
		} else if !isParticipant {
			return errors.New(fmt.Sprintf("User %v is not isParticipant of chat %v", userPrincipalDto.UserId, chatId))
		}

		err = tx.PinChat(c.Request().Context(), chatId, userPrincipalDto.UserId, pin)
		if err != nil {
			return err
		}

		admin, err := tx.IsAdmin(c.Request().Context(), userPrincipalDto.UserId, chatId)
		if err != nil {
			return err
		}

		chatDto, err := getChat(c.Request().Context(), tx, ch.restClient, chatId, userPrincipalDto.UserId, 0, 0)
		if err != nil {
			return err
		}

		ch.notificator.NotifyAboutChangeChat(c.Request().Context(), chatDto, []int64{userPrincipalDto.UserId}, len(chatDto.ParticipantIds) == 1, true, tx, map[int64]bool{userPrincipalDto.UserId: admin})

		return c.JSON(http.StatusOK, chatDto)
	})
}

func (ch *ChatHandler) DeleteParticipant(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	interestingUserId, err := GetPathParamAsInt64(c, "participantId")
	if err != nil {
		return err
	}

	errOuter := db.Transact(c.Request().Context(), ch.db, func(tx *db.Tx) error {
		// check that I am admin
		admin, err := tx.IsAdmin(c.Request().Context(), userPrincipalDto.UserId, chatId)
		if err != nil {
			return err
		}
		if !admin {
			return c.JSON(http.StatusUnauthorized, &utils.H{"message": "You have no access to this chat"})
		}

		err = tx.DeleteParticipant(c.Request().Context(), interestingUserId, chatId)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during changing chat admin in database %v", err)
			return err
		}
		return nil
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
		return errOuter
	}
	errOuter = db.Transact(c.Request().Context(), ch.db, func(tx *db.Tx) error {
		chatDto, err := getChat(c.Request().Context(), tx, ch.restClient, chatId, userPrincipalDto.UserId, 0, 0)
		if err != nil {
			return err
		}
		err = tx.IterateOverChatParticipantIds(c.Request().Context(), chatId, func(participantIds []int64) error {
			areAdmins, err := getAreAdminsOfUserIds(c.Request().Context(), tx, participantIds, chatId)
			if err != nil {
				return err
			}

			ch.notificator.NotifyAboutDeleteParticipants(c.Request().Context(), participantIds, chatId, []int64{interestingUserId})
			ch.notificator.NotifyAboutChangeChat(c.Request().Context(), chatDto, participantIds, len(chatDto.ParticipantIds) == 1, true, tx, areAdmins)
			return nil
		})
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during getting chat participants %v", err)
			return err
		}

		if chatDto.AvailableToSearch || chatDto.Blog {
			// send duplicated event to the former user to re-draw chat on their search results
			ch.notificator.NotifyAboutRedrawLeftChat(c.Request().Context(), chatDto, interestingUserId, len(chatDto.ParticipantIds) == 1, false, tx, map[int64]bool{interestingUserId: false}) // // false because interestingUserId left the chat
		} else {
			ch.notificator.NotifyAboutDeleteChat(c.Request().Context(), chatId, []int64{interestingUserId}, tx)
		}
		return nil
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
		return errOuter
	}
	return c.NoContent(http.StatusAccepted)
}

func (ch *ChatHandler) getParticipantsWithAdmin(cdo db.CommonOperations, participantIds []int64, chatId int64, ctx context.Context) ([]*dto.UserWithAdmin, error) {
	newUsers, err := ch.restClient.GetUsers(ctx, participantIds)
	if err != nil {
		GetLogEntry(ctx).Errorf("Error during getting users %v", err)
		return nil, err
	}
	return ch.enrichWithAdmin(ctx, cdo, newUsers, chatId)
}

func (ch *ChatHandler) enrichWithAdmin(ctx context.Context, cdo db.CommonOperations, users []*dto.User, chatId int64) ([]*dto.UserWithAdmin, error) {
	newUsersWithAdmin := []*dto.UserWithAdmin{}

	userIds := []int64{}
	for _, anUser := range users {
		userIds = append(userIds, anUser.Id)
	}

	areAdmins, err := cdo.IsAdminBatchByParticipants(ctx, userIds, chatId)
	if err != nil {
		GetLogEntry(ctx).Errorf("Error during getting users %v", err)
		return nil, err
	}

	for _, anAdmin := range areAdmins { // keep order
		var anUser *dto.User
		for _, u := range users {
			if u.Id == anAdmin.UserId {
				anUser = u
				break
			}
		}
		if anUser == nil {
			GetLogEntry(ctx).Errorf("Unable to find an user")
			continue
		}

		newUsersWithAdmin = append(newUsersWithAdmin, &dto.UserWithAdmin{
			User:  *anUser,
			Admin: anAdmin.Admin,
		})
	}
	return newUsersWithAdmin, nil
}

type AddParticipantsDto struct {
	ParticipantIds []int64 `json:"addParticipantIds"`
}

func (ch *ChatHandler) AddParticipants(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	var bindTo = new(AddParticipantsDto)
	if err := c.Bind(bindTo); err != nil {
		GetLogEntry(c.Request().Context()).Warnf("Error during binding to dto %v", err)
		return err
	}

	errOuter := db.Transact(c.Request().Context(), ch.db, func(tx *db.Tx) error {

		// check that I am admin
		admin, err := tx.IsAdmin(c.Request().Context(), userPrincipalDto.UserId, chatId)
		if err != nil {
			return err
		}
		if !admin {
			return c.JSON(http.StatusUnauthorized, &utils.H{"message": "You have no access to this chat"})
		}

		for _, participantId := range bindTo.ParticipantIds {
			err = tx.AddParticipant(c.Request().Context(), participantId, chatId, false)
			if err != nil {
				GetLogEntry(c.Request().Context()).Errorf("Error during changing chat admin in database %v", err)
				return err
			}
		}
		return nil
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
		return errOuter
	}

	errOuter = db.Transact(c.Request().Context(), ch.db, func(tx *db.Tx) error {
		newUsersWithAdmin, err := ch.getParticipantsWithAdmin(tx, bindTo.ParticipantIds, chatId, c.Request().Context())
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during getting participants aith admin %v", err)
			return err
		}

		chatDto, err := getChat(c.Request().Context(), tx, ch.restClient, chatId, userPrincipalDto.UserId, 0, 0)
		if err != nil {
			return err
		}
		err = tx.IterateOverChatParticipantIds(c.Request().Context(), chatId, func(participantIds []int64) error {
			areAdmins, err := getAreAdminsOfUserIds(c.Request().Context(), tx, participantIds, chatId)
			if err != nil {
				return err
			}

			ch.notificator.NotifyAboutNewParticipants(c.Request().Context(), participantIds, chatId, newUsersWithAdmin)
			ch.notificator.NotifyAboutChangeChat(c.Request().Context(), chatDto, participantIds, len(chatDto.ParticipantIds) == 1, true, tx, areAdmins)
			return nil
		})
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during getting chat participants %v", err)
			return err
		}

		return c.NoContent(http.StatusAccepted)
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

func (ch *ChatHandler) SearchForUsersToAdd(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	admin, err := ch.db.IsAdmin(c.Request().Context(), userPrincipalDto.UserId, chatId)
	if err != nil {
		return err
	}
	if !admin {
		return c.JSON(http.StatusUnauthorized, &utils.H{"message": "You have no access to this chat"})
	}

	searchString := c.QueryParam("searchString")
	searchString = strings.TrimSpace(searchString)

	users, err := ch.searchUsersNotContaining(c, searchString, chatId, utils.DefaultSize)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, users)
}

func (ch *ChatHandler) searchUsersContaining(c echo.Context, searchString string, chatId int64, pageSize, requestOffset int) ([]*dto.User, int, error) {
	var users []*dto.User = make([]*dto.User, 0)

	shouldContinue := true

	processedItems := 0

	totalCountInChat := 0
	for page := 0; shouldContinue; page++ {
		offset := utils.GetOffset(page, pageSize)
		participantIds, err := ch.db.GetParticipantIds(c.Request().Context(), chatId, pageSize, offset)
		if len(participantIds) == 0 {
			break
		}
		if len(participantIds) < pageSize {
			shouldContinue = false
		}
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Got error during getting portion %v", err)
			break
		}

		usersPortion, _, err := ch.restClient.SearchGetUsers(c.Request().Context(), searchString, true, participantIds, 0, pageSize)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error get users from aaa %v", err)
			break
		}
		for _, u := range usersPortion {
			if len(users) < pageSize {
				if processedItems >= requestOffset {
					users = append(users, u)
				}
				processedItems++
			}
			totalCountInChat++
		}
	}

	return users, totalCountInChat, nil
}

func (ch *ChatHandler) searchUsersNotContaining(c echo.Context, searchString string, chatId int64, pageSize int) ([]*dto.User, error) {
	var notFoundUsers []*dto.User = make([]*dto.User, 0)
	shouldContinueSearch := true
	for page := 0; shouldContinueSearch; page++ {
		ignoredInAaa := false
		usersPortion, _, err := ch.restClient.SearchGetUsers(c.Request().Context(), searchString, ignoredInAaa, []int64{}, page, pageSize)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error get users from aaa %v", err)
		}
		if len(usersPortion) < pageSize {
			shouldContinueSearch = false
		}

		var portionUserIds = []int64{}
		for _, u := range usersPortion {
			portionUserIds = append(portionUserIds, u.Id)
		}

		foundParticipantIds, err := ch.db.ParticipantsExistence(c.Request().Context(), chatId, portionUserIds)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Got error during getting ParticipantsNonExistence %v", err)
			break
		}
		for _, u := range usersPortion {
			if len(notFoundUsers) < pageSize {
				if !utils.Contains(foundParticipantIds, u.Id) {
					notFoundUsers = append(notFoundUsers, u)
				}
			} else {
				shouldContinueSearch = false // break outer
				break                        // inner
			}
		}
	}

	return notFoundUsers, nil
}

func (ch *ChatHandler) GetParticipants(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	userSearchString := c.QueryParam("searchString")
	userSearchString = strings.TrimSpace(userSearchString)

	participantsPage := utils.FixPageString(c.QueryParam("page"))
	participantsSize := utils.FixSizeString(c.QueryParam("size"))
	participantsOffset := utils.GetOffset(participantsPage, participantsSize)

	var usersWithAdmin []*dto.UserWithAdmin = []*dto.UserWithAdmin{}

	totalFoundInChatUserCount := 0

	if userSearchString != "" {
		var users []*dto.User
		users, totalFoundInChatUserCount, err = ch.searchUsersContaining(c, userSearchString, chatId, participantsSize, participantsOffset)
		usersWithAdmin, err = ch.enrichWithAdmin(c.Request().Context(), ch.db, users, chatId)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during getting participants with admin %v", err)
			return err
		}
	} else {
		participantIds, err := ch.db.GetParticipantIds(c.Request().Context(), chatId, participantsSize, participantsOffset)
		if err != nil {
			return err
		}

		usersWithAdmin, err = ch.getParticipantsWithAdmin(ch.db, participantIds, chatId, c.Request().Context())
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during getting participants with admin %v", err)
			return err
		}

		count, err := ch.db.GetParticipantsCount(c.Request().Context(), chatId)
		if err != nil {
			return err
		}
		totalFoundInChatUserCount = count
	}

	return c.JSON(http.StatusOK, &ParticipantsWithAdminWrapper{
		Data:  usersWithAdmin,
		Count: totalFoundInChatUserCount,
	})
}

type CountRequestDto struct {
	SearchString string `json:"searchString"`
}

func (ch *ChatHandler) CountParticipants(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	bindTo := new(CountRequestDto)
	err = c.Bind(bindTo)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during unmarshalling %v", err)
		return err
	}
	userSearchString := strings.TrimSpace(bindTo.SearchString)

	totalFoundUserCount := 0

	if userSearchString != "" {
		_, aCount, err := ch.searchUsersContaining(c, userSearchString, chatId, utils.DefaultSize, utils.DefaultOffset)
		if err != nil {
			return err
		}
		totalFoundUserCount = aCount
	} else {
		count, err := ch.db.GetParticipantsCount(c.Request().Context(), chatId)
		if err != nil {
			return err
		}
		totalFoundUserCount = count
	}

	return c.JSON(http.StatusOK, &utils.H{"count": totalFoundUserCount})
}

type FilteredRequestDto struct {
	SearchString string  `json:"searchString"`
	UserId       []int64 `json:"userId"`
}

type FilteredParticipantItemResponse struct {
	Id int64 `json:"id"`
}

func (ch *ChatHandler) FilterParticipants(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	bindTo := new(FilteredRequestDto)
	err = c.Bind(bindTo)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during unmarshalling %v", err)
		return err
	}

	userSearchString := bindTo.SearchString
	userSearchString = strings.TrimSpace(userSearchString)

	requestedParticipantIds := bindTo.UserId

	var response = []*FilteredParticipantItemResponse{}

	if userSearchString != "" {
		var batches = [][]int64{}
		var batch = []int64{}
		for _, pid := range requestedParticipantIds {
			batch = append(batch, pid)
			if len(batch) == utils.DefaultSize {
				batches = append(batches, batch)
				batch = []int64{}
			}
		}
		for _, aBatch := range batches { // we already know that requestedParticipantIds belong to this chat, so our sole task is to pass them through aaa filter
			usersPortion, _, err := ch.restClient.SearchGetUsers(c.Request().Context(), userSearchString, true, aBatch, 0, utils.DefaultSize)
			if err != nil {
				GetLogEntry(c.Request().Context()).Errorf("Error get users from aaa %v", err)
			} else {
				for _, user := range usersPortion {
					response = append(response, &FilteredParticipantItemResponse{user.Id})
				}
			}
		}
	} else {
		foundParticipantIds, err := ch.db.ParticipantsExistence(c.Request().Context(), chatId, requestedParticipantIds)
		if err != nil {
			return err
		}

		for _, userId := range foundParticipantIds {
			response = append(response, &FilteredParticipantItemResponse{userId})
		}
	}

	return c.JSON(http.StatusOK, response)
}

func (ch *ChatHandler) SearchForUsersToMention(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	participant, err := ch.db.IsParticipant(c.Request().Context(), userPrincipalDto.UserId, chatId)
	if err != nil {
		return err
	}
	if !participant {
		return c.NoContent(http.StatusUnauthorized)
	}

	searchString := c.QueryParam("searchString")
	searchString = strings.TrimSpace(searchString)

	users, _, err := ch.searchUsersContaining(c, searchString, chatId, utils.DefaultSize, utils.DefaultOffset)
	if err != nil {
		return err
	}

	users = append(users, &dto.User{
		Id:    AllUsers, // -1 is reserved for 'deleted' in ./aaa/src/main/resources/db/migration/V1__init.sql
		Login: allUsers,
	})
	users = append(users, &dto.User{
		Id:    HereUsers, // -1 is reserved for 'deleted' in ./aaa/src/main/resources/db/migration/V1__init.sql
		Login: hereUsers,
	})

	return c.JSON(http.StatusOK, users)
}

func (ch *ChatHandler) CheckAccess(c echo.Context) error {
	chatId, err := GetQueryParamAsInt64(c, "chatId")
	if err != nil {
		return err
	}
	chat, err := ch.db.GetChatBasic(c.Request().Context(), chatId)
	if err != nil {
		return err
	}
	if chat == nil {
		return c.NoContent(http.StatusUnauthorized)
	}

	messageId, _ := GetQueryParamAsInt64(c, "messageId")
	if messageId > 0 {
		m, err := ch.db.GetMessageBasic(c.Request().Context(), chatId, messageId)
		if err != nil {
			return err
		}
		fileItemUuid := c.QueryParam("fileItemUuid")
		if m != nil && (chat.IsBlog || m.Published || m.BlogPost) {
			encodedFileItemUuid := utils.UrlEncode(fileItemUuid)
			if strings.Contains(m.Text, encodedFileItemUuid) {
				return c.NoContent(http.StatusOK)
			}
		}
		return c.NoContent(http.StatusUnauthorized)
	}

	userId, err := GetQueryParamAsInt64(c, "userId")
	if err != nil {
		return err
	}
	useCanResend, _ := GetQueryParamAsBoolean(c, "considerCanResend")
	participant, err := ch.db.IsParticipant(c.Request().Context(), userId, chatId)
	if err != nil {
		return err
	}

	if participant {
		return c.NoContent(http.StatusOK)
	} else {
		if useCanResend {
			if chat.CanResend {
				return c.NoContent(http.StatusOK)
			} else {
				return c.NoContent(http.StatusUnauthorized)
			}
		}
	}
	return c.NoContent(http.StatusUnauthorized)
}

func (ch *ChatHandler) IsAdmin(c echo.Context) error {
	chatId, err := GetQueryParamAsInt64(c, "chatId")
	if err != nil {
		return err
	}
	userId, err := GetQueryParamAsInt64(c, "userId")
	if err != nil {
		return err
	}
	isAdmin, err := ch.db.IsAdmin(c.Request().Context(), userId, chatId)
	if err != nil {
		return err
	}
	if isAdmin {
		return c.NoContent(http.StatusOK)
	} else {
		return c.NoContent(http.StatusUnauthorized)
	}
}

type TetATetResponse struct {
	Id int64 `json:"id"`
}

func (ch *ChatHandler) TetATet(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	toParticipantId, err := GetPathParamAsInt64(c, "participantId")
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during parsing participantId %v", err)
		return err
	}

	errOuter := db.Transact(c.Request().Context(), ch.db, func(tx *db.Tx) error {
		// check existing tet-a-tet chat
		exists, chatId, err := tx.IsExistsTetATet(c.Request().Context(), userPrincipalDto.UserId, toParticipantId)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during checking exists tet-a-tet chat %v", err)
			return err
		}
		if exists {
			return c.JSON(http.StatusAccepted, TetATetResponse{Id: chatId})
		}

		// create tet-a-tet chat
		chatId2, err := tx.CreateTetATetChat(c.Request().Context(), userPrincipalDto.UserId, toParticipantId)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during creating tet-a-tet chat %v", err)
			return err
		}

		if err := tx.AddParticipant(c.Request().Context(), userPrincipalDto.UserId, chatId2, true); err != nil {
			return err
		}
		if userPrincipalDto.UserId != toParticipantId {
			if err := tx.AddParticipant(c.Request().Context(), toParticipantId, chatId2, true); err != nil {
				return err
			}
		}

		chatDto, err := getChat(c.Request().Context(), tx, ch.restClient, chatId2, userPrincipalDto.UserId, 0, 0)
		if err != nil {
			return err
		}

		ch.notificator.NotifyAboutNewChat(c.Request().Context(), chatDto, chatDto.ParticipantIds, len(chatDto.ParticipantIds) == 1, true, tx, map[int64]bool{userPrincipalDto.UserId: true, toParticipantId: true}) // true because in tet-a-tet both are admins

		return c.JSON(http.StatusCreated, TetATetResponse{Id: chatId2})
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

type UserChatNotificationSettings struct {
	ConsiderMessagesOfThisChatAsUnread null.Bool `json:"considerMessagesOfThisChatAsUnread"`
}

func (ch *ChatHandler) PutUserChatNotificationSettings(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	bindTo := new(UserChatNotificationSettings)
	err = c.Bind(bindTo)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during unmarshalling %v", err)
		return err
	}

	err = ch.db.InitUserChatNotificationSettings(c.Request().Context(), userPrincipalDto.UserId, chatId)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during initializing notification settings %v", err)
		return err
	}

	err = ch.db.PutUserChatNotificationSettings(c.Request().Context(), bindTo.ConsiderMessagesOfThisChatAsUnread.Ptr(), userPrincipalDto.UserId, chatId)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, bindTo)
}

func (ch *ChatHandler) GetUserChatNotificationSettings(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	consider, err := ch.db.GetUserChatNotificationSettings(c.Request().Context(), userPrincipalDto.UserId, chatId)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, UserChatNotificationSettings{ConsiderMessagesOfThisChatAsUnread: null.BoolFromPtr(consider)})
}

type CleanHtmlTagsRequestDto struct {
	Text  string `json:"text"`
	Login string `json:"login"`
}

type CleanHtmlTagsResponseDto struct {
	Text string `json:"text"`
}

func (ch *ChatHandler) CreatePreview(c echo.Context) error {
	bindTo := new(CleanHtmlTagsRequestDto)
	err := c.Bind(bindTo)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during unmarshalling %v", err)
		return err
	}
	preview := createMessagePreview(ch.stripTagsPolicy, bindTo.Text, bindTo.Login)
	response := CleanHtmlTagsResponseDto{
		Text: preview,
	}
	return c.JSON(http.StatusOK, response)
}

type ChatExists struct {
	Exists bool  `json:"exists"`
	ChatId int64 `json:"chatId"`
}

func (ch *ChatHandler) IsExists(c echo.Context) error {
	chatIds, err := GetQueryParamsAsInt64Slice(c, "chatId")
	if err != nil {
		return err
	}

	responseList := make([]ChatExists, 0)
	if len(chatIds) == 0 {
		return c.JSON(http.StatusOK, responseList)
	}

	existsFromDb, err := ch.db.GetExistingChatIds(c.Request().Context(), chatIds)
	if err != nil {
		return err
	}

	for _, ce := range chatIds {
		responseList = append(responseList, ChatExists{
			ChatId: ce,
			Exists: utils.Contains(*existsFromDb, ce),
		})
	}

	return c.JSON(http.StatusOK, responseList)
}

type simpleChat struct {
	Id                int64
	Name              string
	IsTetATet         bool
	Avatar            null.String
	ShortInfo         null.String
	LoginColor        null.String
	LastLoginDateTime null.Time
}

func (r *simpleChat) GetId() int64 {
	return r.Id
}

func (r *simpleChat) GetName() string {
	return r.Name
}

func (r *simpleChat) GetAvatar() null.String {
	return r.Avatar
}

func (r *simpleChat) SetName(s string) {
	r.Name = s
}

func (r *simpleChat) SetAvatar(s null.String) {
	r.Avatar = s
}

func (r *simpleChat) SetShortInfo(s null.String) {
	r.ShortInfo = s
}

func (r *simpleChat) SetLoginColor(s null.String) {
	r.LoginColor = s
}

func (r *simpleChat) GetIsTetATet() bool {
	return r.IsTetATet
}

func (r *simpleChat) SetLastLoginDateTime(t null.Time) {
	r.LastLoginDateTime = t
}

func (ch *ChatHandler) GetNameForInvite(c echo.Context) error {
	chatId, err := GetQueryParamAsInt64(c, "chatId")
	if err != nil {
		return err
	}
	behalfUserId, err := GetQueryParamAsInt64(c, "behalfUserId")
	if err != nil {
		return err
	}
	participantIds, err := GetQueryParamsAsInt64Slice(c, "userIds")
	if err != nil {
		return err
	}

	chat, err := ch.db.GetChat(c.Request().Context(), behalfUserId, chatId)
	if err != nil {
		return err
	}

	ret := []dto.ChatName{}

	if chat == nil {
		return c.JSON(http.StatusOK, ret)
	}

	behalfUsers, err := ch.restClient.GetUsers(c.Request().Context(), []int64{behalfUserId})
	if err != nil {
		return err
	}
	var behalfUserLogin string
	if len(behalfUsers) != 1 {
		GetLogEntry(c.Request().Context()).Infof("Behalf user with id %v is not found", behalfUserId)
	} else {
		behalfUserLogin = behalfUsers[0].Login
	}

	users, err := ch.restClient.GetUsers(c.Request().Context(), participantIds)
	if err != nil {
		return err
	}

	count, err := ch.db.GetParticipantsCount(c.Request().Context(), chatId)
	if err != nil {
		return err
	}

	for _, user := range users {
		meAsUser := dto.User{Id: behalfUserId, Login: behalfUserLogin}
		var sch dto.ChatDtoWithTetATet = &simpleChat{
			Id:        chat.Id,
			Name:      chat.Title,
			IsTetATet: chat.TetATet,
			Avatar:    chat.Avatar,
		}
		utils.ReplaceChatNameToLoginForTetATet(
			sch,
			&meAsUser,
			user.Id,
			count == 1,
		)
		ret = append(ret, dto.ChatName{Name: sch.GetName(), Avatar: sch.GetAvatar(), UserId: user.Id})
	}
	return c.JSON(http.StatusOK, ret)
}

func (ch *ChatHandler) GetBasicInfo(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}
	behalfUserId, err := GetQueryParamAsInt64(c, "userId")
	if err != nil {
		return err
	}

	chat, err := ch.db.GetChatWithParticipants(c.Request().Context(), behalfUserId, chatId, utils.FixSize(0), utils.FixPage(0))
	if err != nil {
		return err
	}

	ret := dto.BasicChatDto{
		TetATet:        chat.TetATet,
		ParticipantIds: chat.ParticipantsIds,
	}
	return c.JSON(http.StatusOK, ret)
}

func (ch *ChatHandler) RemoveAllParticipants(c echo.Context) error {
	GetLogEntry(c.Request().Context()).Warnf("Removing ALL participants")
	return ch.db.DeleteAllParticipants(c.Request().Context())
}

func (ch *ChatHandler) GetChatParticipants(c echo.Context) error {
	chatId, err := GetQueryParamAsInt64(c, "chatId")
	if err != nil {
		return err
	}

	participantsPage := utils.FixPageString(c.QueryParam("page"))
	participantsSize := utils.FixSizeString(c.QueryParam("size"))
	participantsOffset := utils.GetOffset(participantsPage, participantsSize)

	ids, err := ch.db.GetParticipantIds(c.Request().Context(), chatId, participantsSize, participantsOffset)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, ids)
}

type ParticipantBelongsToChat struct {
	UserId  int64 `json:"userId"`
	Belongs bool  `json:"belongs"`
}

type ParticipantsBelongToChat struct {
	Users []*ParticipantBelongsToChat `json:"users"`
}

func (ch *ChatHandler) DoesParticipantBelongToChat(c echo.Context) error {
	GetLogEntry(c.Request().Context()).Infof("Checking if participant belongs")
	chatId, err := GetQueryParamAsInt64(c, "chatId")
	if err != nil {
		return err
	}
	userIds, err := GetQueryParamsAsInt64Slice(c, "userId")
	if err != nil {
		return err
	}

	var users = []*ParticipantBelongsToChat{}
	for _, userId := range userIds {
		var belongs = &ParticipantBelongsToChat{
			UserId:  userId,
			Belongs: false,
		}
		users = append(users, belongs)
	}

	err = ch.db.IterateOverChatParticipantIds(c.Request().Context(), chatId, func(participantIds []int64) error {
		for _, user := range users {
			if utils.Contains(participantIds, user.UserId) {
				user.Belongs = true
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &ParticipantsBelongToChat{Users: users})
}

func (ch *ChatHandler) checkCanCreateBlog(userPrincipalDto *auth.AuthResult, blog *bool) bool {
	if !ch.onlyAdminCanCreateBlog {
		return true
	}
	if blog != nil && userPrincipalDto != nil && userPrincipalDto.HasRole("ROLE_ADMIN") {
		return true
	} else {
		return false
	}
}

func (ch *ChatHandler) CanCreateBlog(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	isBlog := true
	return c.JSON(http.StatusOK, &utils.H{"canCreateBlog": ch.checkCanCreateBlog(userPrincipalDto, &isBlog)})
}

func (ch *ChatHandler) MarkAsRead(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	return db.Transact(c.Request().Context(), ch.db, func(tx *db.Tx) error {
		if isParticipant, err := tx.IsParticipant(c.Request().Context(), userPrincipalDto.UserId, chatId); err != nil {
			return err
		} else if !isParticipant {
			return errors.New(fmt.Sprintf("User %v is not isParticipant of chat %v", userPrincipalDto.UserId, chatId))
		}

		err = tx.MarkAllMessagesAsRead(c.Request().Context(), chatId, userPrincipalDto.UserId)
		if err != nil {
			return err
		}

		lastUpdated, err := tx.GetChatLastDatetimeChat(c.Request().Context(), chatId)
		if err != nil {
			return err
		}
		ch.notificator.NotifyAboutUnreadMessage(c.Request().Context(), chatId, userPrincipalDto.UserId, 0, lastUpdated)

		hasUnreadMessages, err := tx.HasUnreadMessages(c.Request().Context(), userPrincipalDto.UserId)
		if err != nil {
			return err
		}
		ch.notificator.NotifyAboutHasNewMessagesChanged(c.Request().Context(), userPrincipalDto.UserId, hasUnreadMessages)

		return c.NoContent(http.StatusOK)
	})
}

func (ch *ChatHandler) MarkAsReadAll(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	return db.Transact(c.Request().Context(), ch.db, func(tx *db.Tx) error {
		err := tx.IterateOverAllMyChatIds(c.Request().Context(), userPrincipalDto.UserId, func(chatIds []int64) error {
			chatsHasUnreadMessages, err := tx.HasUnreadMessagesByChatIdsBatch(c.Request().Context(), chatIds, userPrincipalDto.UserId)
			if err != nil {
				return err
			}

			for chatId, has := range chatsHasUnreadMessages {
				if has {
					err := tx.MarkAllMessagesAsRead(c.Request().Context(), chatId, userPrincipalDto.UserId)
					if err != nil {
						GetLogEntry(c.Request().Context()).Errorf("Error during marking chat %v as read: %v", chatId, err)
						continue
					}

					lastUpdated, err := tx.GetChatLastDatetimeChat(c.Request().Context(), chatId)
					if err != nil {
						GetLogEntry(c.Request().Context()).Errorf("Error during GetChatLastDatetimeChat chat %v: %v", chatId, err)
						continue
					}
					ch.notificator.NotifyAboutUnreadMessage(c.Request().Context(), chatId, userPrincipalDto.UserId, 0, lastUpdated)
				}
			}
			return nil
		})

		hasUnreadMessages, err := tx.HasUnreadMessages(c.Request().Context(), userPrincipalDto.UserId)
		if err != nil {
			return err
		}
		ch.notificator.NotifyAboutHasNewMessagesChanged(c.Request().Context(), userPrincipalDto.UserId, hasUnreadMessages)

		if err != nil {
			return err
		}
		return c.NoContent(http.StatusOK)
	})
}
