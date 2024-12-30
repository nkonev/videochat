package handlers

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	lkauth "github.com/livekit/protocol/auth"
	log "github.com/sirupsen/logrus"
	"net/http"
	"nkonev.name/video/auth"
	"nkonev.name/video/client"
	"nkonev.name/video/config"
	"nkonev.name/video/db"
	"nkonev.name/video/dto"
	. "nkonev.name/video/logger"
	"nkonev.name/video/producer"
	"nkonev.name/video/services"
	"nkonev.name/video/utils"
	"strings"
)

type InviteHandler struct {
	database                 *db.DB
	chatClient               *client.RestClient
	dialStatusPublisher      *producer.RabbitDialStatusPublisher
	notificationPublisher    *producer.RabbitNotificationsPublisher
	invitePublisher          *producer.RabbitInvitePublisher
	userService              *services.UserService
	stateChangedEventService *services.StateChangedEventService
	config                   *config.ExtendedConfig
	lgr                      *log.Logger
}

const EventMissedCall = "missed_call"

func NewInviteHandler(
	database *db.DB,
	chatClient *client.RestClient,
	dialStatusPublisher *producer.RabbitDialStatusPublisher,
	notificationPublisher *producer.RabbitNotificationsPublisher,
	invitePublisher *producer.RabbitInvitePublisher,
	userService *services.UserService,
	stateChangedEventService *services.StateChangedEventService,
	config *config.ExtendedConfig,
	lgr *log.Logger,
) *InviteHandler {
	return &InviteHandler{
		database:                 database,
		chatClient:               chatClient,
		dialStatusPublisher:      dialStatusPublisher,
		notificationPublisher:    notificationPublisher,
		invitePublisher:          invitePublisher,
		userService:              userService,
		stateChangedEventService: stateChangedEventService,
		config:                   config,
		lgr:                      lgr,
	}
}

// used by owner to add or remove from dial list
func (vh *InviteHandler) ProcessCreatingOrDeletingInvite(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context(), vh.lgr).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	callee, err := utils.ParseInt64(c.QueryParam("userId"))
	if err != nil {
		return err
	}

	addToCall, err := utils.ParseBoolean(c.QueryParam("call"))
	if err != nil {
		return err
	}

	tokenIdStr := c.QueryParam("tokenId")
	var ownerTokenId uuid.UUID

	// in user is not in a call then he doesn't have tokenId, in this case we generate int here
	// we could await on user connected to a call and has token, but it's going to worsen the user experience
	if len(tokenIdStr) == 0 {
		ownerTokenId = uuid.New()
	} else {
		ownerTokenId, err = uuid.Parse(tokenIdStr)
		if err != nil {
			return err
		}
	}

	// check my access to chat
	if ok, err := vh.chatClient.CheckAccess(c.Request().Context(), userPrincipalDto.UserId, chatId); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	ok, code := vh.checkAccessOfAnotherUser(c.Request().Context(), callee, chatId, userPrincipalDto)
	if !ok {
		return c.NoContent(code)
	}

	basicChatInfo, err := vh.chatClient.GetBasicChatInfo(c.Request().Context(), chatId, userPrincipalDto.UserId) // tet-a-tet
	if err != nil {
		return err
	}

	code, err = db.TransactWithResult(c.Request().Context(), vh.database, func(tx *db.Tx) (int, error) {
		if addToCall {
			// here we generate the token and further we're gonna take it
			return vh.addAsCallee(c.Request().Context(), tx, callee, chatId, userPrincipalDto, ownerTokenId, basicChatInfo.TetATet), nil
		} else {
			return vh.ownerRemoveFromCalling(c.Request().Context(), tx, callee, chatId, userPrincipalDto, ownerTokenId, basicChatInfo.TetATet), nil
		}
	})
	if err != nil {
		return err
	}

	if code == http.StatusOK {
		return c.JSON(http.StatusOK, utils.H{"tokenId": ownerTokenId})
	} else {
		return c.NoContent(code)
	}
}

func (vh *InviteHandler) checkAccessOfAnotherUser(ctx context.Context, callee int64, chatId int64, userPrincipalDto *auth.AuthResult) (bool, int) {
	// check participant's access to chat
	if ok, err := vh.chatClient.CheckAccess(ctx, callee, chatId); err != nil {
		return false, http.StatusInternalServerError
	} else if !ok {
		return false, http.StatusUnauthorized
	}

	return true, http.StatusOK
}

// addToCalling
func (vh *InviteHandler) addAsCallee(c context.Context, tx *db.Tx, calleeUserId int64, chatId int64, userPrincipalDto *auth.AuthResult, ownerTokenId uuid.UUID, tetATet bool) int {
	// forbid to make calls to the different chats
	calleesOfOwner, err := tx.GetUserOwnedBeingInvitedCallees(c, dto.UserCallStateId{
		TokenId: ownerTokenId,
		UserId:  userPrincipalDto.UserId,
	})
	if err != nil {
		GetLogEntry(c, vh.lgr).Errorf("Error %v", err)
		return http.StatusInternalServerError
	}
	for _, callee := range calleesOfOwner {
		if callee.ChatId != chatId {
			GetLogEntry(c, vh.lgr).Infof("Calls to the different chats are prohibited")
			return http.StatusConflict
		}
	}
	// end of forbid to make calls to the different chats

	// remove states which can override
	// we don't want to send both "removing" event to old chat and "beingInvited" event to the new chat
	gotStatusesAllChats, err := tx.GetByCalleeUserIdFromAllChats(c, calleeUserId)
	if err != nil {
		GetLogEntry(c, vh.lgr).Errorf("Error %v", err)
		return http.StatusInternalServerError
	}

	var ucss []dto.UserCallState = make([]dto.UserCallState, 0)
	for _, gotStatus := range gotStatusesAllChats {
		if !db.CanOverrideCallStatus(gotStatus.Status) {
			GetLogEntry(c, vh.lgr).Infof("Unable to invite somebody with non-overridable status")
			return http.StatusConflict
		} else {
			ucss = append(ucss, gotStatus)
		}
	}
	// we remove callee's previous inviting - only after CanOverrideCallStatus() check
	vh.hardRemove(c, tx, ucss)
	// end of remove states which can override

	var newCalleeStatus = dto.UserCallState{
		TokenId:      uuid.New(),
		UserId:       calleeUserId,
		ChatId:       chatId,
		TokenTaken:   false,
		OwnerTokenId: &ownerTokenId,
		OwnerUserId:  &userPrincipalDto.UserId,
		Status:       db.CallStatusBeingInvited,
		ChatTetATet:  tetATet,
		OwnerAvatar:  &userPrincipalDto.Avatar,
	}

	err = tx.Set(c, newCalleeStatus)

	if err != nil {
		GetLogEntry(c, vh.lgr).Errorf("Error %v", err)
		return http.StatusInternalServerError
	}

	// for better user experience
	vh.sendEvents(c, chatId, calleeUserId, db.CallStatusBeingInvited, userPrincipalDto.UserId, userPrincipalDto.Avatar, tetATet)

	return http.StatusOK
}

func (vh *InviteHandler) addAsEntered(ctx context.Context, tx *db.Tx, tokenId uuid.UUID, userId, chatId int64, tetATet bool) error {
	return tx.AddAsEntered(ctx, tokenId, userId, chatId, tetATet)
}

func (vh *InviteHandler) ownerRemoveFromCalling(ctx context.Context, tx *db.Tx, callee int64, chatId int64, userPrincipalDto *auth.AuthResult, ownerTokenId uuid.UUID, tetATet bool) int {

	statuses, err := tx.GetBeingInvitedByOwnerAndCalleeId(ctx, dto.UserCallStateId{TokenId: ownerTokenId, UserId: userPrincipalDto.UserId}, callee, chatId)
	if err != nil {
		GetLogEntry(ctx, vh.lgr).Errorf("Error during getting stetuses %v", err)
		return http.StatusInternalServerError
	}

	// softRemoveExtended has an internal deduplication in order not to send multiple events to one user in case multiple tokens of he
	code := vh.softRemoveExtended(ctx, tx, statuses, db.CallStatusRemoving, &overrideArg{
		overrideOwnerId:     userPrincipalDto.UserId,
		overrideOwnerAvatar: userPrincipalDto.Avatar,
		overrideTetATet:     tetATet,
		overrideChatId:      chatId,
	})
	if code != http.StatusOK {
		return code
	}

	// if we remove user from call - send them EventMissedCall notification
	// sendMissedCallNotification has an internal deduplication in order not to send multiple events to one user in case multiple tokens of he
	vh.sendMissedCallNotification(ctx, chatId, userPrincipalDto, statuses)

	return http.StatusOK
}

// user enters to call somehow, either by clicking green tube or opening .../video link
func (vh *InviteHandler) ProcessEnterToDial(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context(), vh.lgr).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	// check my access to chat
	if ok, err := vh.chatClient.CheckAccess(c.Request().Context(), userPrincipalDto.UserId, chatId); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	// https://docs.livekit.io/guides/getting-started/#generating-access-tokens-(jwt)
	// https://github.com/nkonev/videochat/blob/8fd81bccbe5f552de1ca123e2ba855dfe814cf66/development.md#generate-livekit-token
	aKey := vh.config.LivekitConfig.Api.Key
	aSecret := vh.config.LivekitConfig.Api.Secret
	aRoomId := utils.GetRoomNameFromId(chatId)

	var tokenId uuid.UUID
	var token string

	// during entering into dial. Returns status: true which means that frontend should (initially) draw the calling.
	// Now it used only in tet-a-tet.
	// If we are in the tet-a-tet
	basicChatInfo, err := vh.chatClient.GetBasicChatInfo(c.Request().Context(), chatId, userPrincipalDto.UserId) // tet-a-tet
	if err != nil {
		return err
	}

	err = db.Transact(c.Request().Context(), vh.database, func(tx *db.Tx) error {
		allPrevMyStates, err := tx.GetByCalleeUserIdFromAllChats(c.Request().Context(), userPrincipalDto.UserId)
		if err != nil {
			return err
		}

		prevStatesToRemove := make([]dto.UserCallState, 0)
		for _, st := range allPrevMyStates {
			if utils.ContainsString(db.GetStatusesToRemoveOnEnter(), st.Status) {
				prevStatesToRemove = append(prevStatesToRemove, st)
			}
		}
		// first of all we remove our previous invitations (smb. invites me)
		vh.softRemoveExtended(c.Request().Context(), tx, prevStatesToRemove, db.CallStatusRemoving, nil)

		var enterToChatInitializedByMe bool = true
		var myInvitation *dto.UserCallState = nil
		for _, st := range allPrevMyStates {
			if st.OwnerUserId != nil && st.Status == db.CallStatusBeingInvited && chatId == st.ChatId {
				enterToChatInitializedByMe = false
				myInvitation = &st
				break
			}
		}

		// if videochat does not exist OR user call status can be overridden
		// then it means that I'm the owner and I need to create it
		if enterToChatInitializedByMe {
			tokenIdStr := c.QueryParam("tokenId")
			// if token was set from /invite call before
			if len(tokenIdStr) > 0 {
				tokenId, err = uuid.Parse(tokenIdStr)
				if err != nil {
					GetLogEntry(c.Request().Context(), vh.lgr).Errorf("Error during parsing provided token, error=%v", err)
					return err
				}
			} else {
				tokenId = uuid.New()
			}

			token, err = vh.getJoinToken(aKey, aSecret, aRoomId, userPrincipalDto, tokenId)
			if err != nil {
				GetLogEntry(c.Request().Context(), vh.lgr).Errorf("Error during getting token, userId=%v, chatId=%v, error=%v", userPrincipalDto.UserId, chatId, err)
				return err
			}

			// and put myself with a status "inCall"
			// add ourself status
			err = vh.addAsEntered(
				c.Request().Context(),
				tx,
				tokenId,
				userPrincipalDto.UserId,
				chatId,
				basicChatInfo.TetATet,
			)
			if err != nil {
				GetLogEntry(c.Request().Context(), vh.lgr).Errorf("Error during adding as owner %v", err)
				return c.NoContent(http.StatusInternalServerError)
			}

			oppositeTetATetUserId, err := vh.getOppositeUserOfTetATetIfPossible(c.Request().Context(), basicChatInfo, chatId, userPrincipalDto)
			if err != nil {
				GetLogEntry(c.Request().Context(), vh.lgr).Errorf("Error %v", err)
				return c.NoContent(http.StatusInternalServerError)
			}
			if oppositeTetATetUserId != nil {
				GetLogEntry(c.Request().Context(), vh.lgr).Infof("Adding user %v to call because it is tet-a-tet chat and he isn't in video", *oppositeTetATetUserId)
				vh.addAsCallee(c.Request().Context(), tx, *oppositeTetATetUserId, chatId, userPrincipalDto, tokenId, basicChatInfo.TetATet)
			}
		} else { // we enter to somebody's chat
			// token_taken = true and owner_[user,token] = null are not actual because of hardRemove()
			tokenId = myInvitation.TokenId
			token, err = vh.getJoinToken(aKey, aSecret, aRoomId, userPrincipalDto, tokenId)
			if err != nil {
				GetLogEntry(c.Request().Context(), vh.lgr).Errorf("Error during getting token, userId=%v, chatId=%v, error=%v", userPrincipalDto.UserId, chatId, err)
				return err
			}

			err = vh.addAsEntered(
				c.Request().Context(),
				tx,
				tokenId,
				userPrincipalDto.UserId,
				chatId,
				basicChatInfo.TetATet,
			)
			if err != nil {
				GetLogEntry(c.Request().Context(), vh.lgr).Errorf("Error during adding as non-owner %v", err)
				return err
			}
			vh.sendEvents(c.Request().Context(), chatId, userPrincipalDto.UserId, db.CallStatusInCall, *myInvitation.OwnerUserId, *myInvitation.OwnerAvatar, basicChatInfo.TetATet)
		}

		vh.stateChangedEventService.NotifyAllChatsAboutUsersInVideoStatus(c.Request().Context(), tx, []int64{userPrincipalDto.UserId})

		return nil
	})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, TokenResponse{
		TokenId: tokenId,
		Token:   token,
	})
}

func (vh *InviteHandler) getOppositeUserOfTetATetIfPossible(c context.Context, basicChatInfo *dto.BasicChatDto, chatId int64, me *auth.AuthResult) (*int64, error) {
	usersOfChat := basicChatInfo.ParticipantIds // here are only first 20 users, but it's enough for sake tet-a-tet purposes
	// in this block we start calling in case valid tet-a-tet
	if basicChatInfo.TetATet && len(usersOfChat) == 2 {
		var oppositeUser *int64 = utils.GetOppositeUser(usersOfChat, me.UserId)

		// uniq users by userId
		usersOfVideo, err := vh.userService.GetVideoParticipants(c, chatId)
		if err != nil {
			GetLogEntry(c, vh.lgr).Errorf("Error %v", err)
			return nil, err
		}

		var oppositeUserOfVideo *int64 = utils.GetOppositeUser2(usersOfVideo, me.UserId)

		// create the call for the opposite user
		// oppositeUserOfVideo is need for case when your counterpart enters into call (not entered until this moment) and this (oppositeUserOfVideo == nil) prevents us to start calling him back
		// and we(behalf user) doesn't have incoming call
		if oppositeUserOfVideo == nil && oppositeUser != nil {
			// we should call the counterpart (opposite user)
			return oppositeUser, nil
		}
	}
	return nil, nil
}

type TokenResponse struct {
	TokenId uuid.UUID `json:"tokenId"`
	Token   string    `json:"token"`
}

func (vh *InviteHandler) getJoinToken(apiKey, apiSecret, room string, authResult *auth.AuthResult, tokenId uuid.UUID) (string, error) {
	canPublish := true
	canSubscribe := true

	aId := utils.MakeIdentityFromUserId(authResult.UserId)

	at := lkauth.NewAccessToken(apiKey, apiSecret)
	grant := &lkauth.VideoGrant{
		RoomJoin:     true,
		Room:         room,
		CanPublish:   &canPublish,
		CanSubscribe: &canSubscribe,
	}

	mds, err := utils.MakeMetadata(authResult.UserId, authResult.UserLogin, authResult.Avatar, tokenId)
	if err != nil {
		return "", err
	}

	validFor := vh.config.VideoTokenValidTime
	at.AddGrant(grant).
		SetIdentity(aId).SetValidFor(validFor).SetMetadata(mds)

	return at.ToJWT()
}

func (vh *InviteHandler) hardRemove(c context.Context, tx *db.Tx, userCallStates []dto.UserCallState) {
	userCallStateIds := make([]dto.UserCallStateId, 0)
	for _, st := range userCallStates {
		// prepare userCallStateIds
		userCallStateIds = append(userCallStateIds, dto.UserCallStateId{
			TokenId: st.TokenId,
			UserId:  st.UserId,
		})
	}
	err := tx.RemoveByUserCallStates(c, userCallStateIds)
	if err != nil {
		GetLogEntry(c, vh.lgr).Errorf("Error during removing from db: %v", err)
	}
}

func (vh *InviteHandler) ProcessCancelInvitation(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context(), vh.lgr).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	// check my access to chat
	if ok, err := vh.chatClient.CheckAccess(c.Request().Context(), userPrincipalDto.UserId, chatId); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	code, err := db.TransactWithResult(c.Request().Context(), vh.database, func(tx *db.Tx) (int, error) {
		myPrevInvitingStates, err := tx.GetBeingInvitedByCalleeIdAndChatId(c.Request().Context(), userPrincipalDto.UserId, chatId)
		if err != nil {
			return 0, err
		}
		byOwnerId := map[int64][]dto.UserCallState{}
		for _, st := range myPrevInvitingStates {
			oid := utils.OwnerIdToNoUser(st.OwnerUserId)
			byOwnerId[oid] = append(byOwnerId[oid], st)
		}

		for ownerId, states := range byOwnerId {
			if len(states) > 0 {
				vh.softRemoveExtended(c.Request().Context(), tx, states, db.CallStatusCancelling, &overrideArg{
					overrideOwnerId:     ownerId,
					overrideOwnerAvatar: utils.NullToEmpty(states[0].OwnerAvatar),
					overrideTetATet:     states[0].ChatTetATet,
					overrideChatId:      chatId,
				})
			}
		}

		return http.StatusOK, nil
	})

	return c.NoContent(code)
}

// question: how not to overwhelm the system by iterating over all the users and all the chats ?
// answer: using opened rooms and rooms are going to be closed - see livekit's room.empty_timeout

type overrideArg struct {
	overrideOwnerId     int64
	overrideOwnerAvatar string
	overrideTetATet     bool
	overrideChatId      int64
}

func (vh *InviteHandler) softRemoveExtended(
	c context.Context,
	tx *db.Tx,
	usersToRemove []dto.UserCallState,
	callStatus string,
	overrideArg *overrideArg,
) int {
	var err error
	// we remove callee by setting status
	var sentUserIds = map[int64]bool{}
	for _, userId := range usersToRemove {

		err = tx.SetRemoving(c, dto.UserCallStateId{TokenId: userId.TokenId, UserId: userId.UserId}, callStatus)
		if err != nil {
			GetLogEntry(c, vh.lgr).Errorf("Error %v", err)
			continue
		}

		if _, ok := sentUserIds[userId.UserId]; !ok {

			var chatId int64
			var ownerId int64 = db.NoUser
			var ownerAvatar string
			var tetATet bool
			if overrideArg != nil {
				chatId = overrideArg.overrideChatId
				ownerId = overrideArg.overrideOwnerId
				ownerAvatar = overrideArg.overrideOwnerAvatar
				tetATet = overrideArg.overrideTetATet
			} else {
				chatId = userId.ChatId
				if userId.OwnerUserId != nil {
					ownerId = *userId.OwnerUserId
				}
				if userId.OwnerAvatar != nil {
					ownerAvatar = *userId.OwnerAvatar
				}
				tetATet = userId.ChatTetATet
			}

			vh.sendEvents(c, chatId, userId.UserId, callStatus, ownerId, ownerAvatar, tetATet)
			sentUserIds[userId.UserId] = true
		}
	}
	if err != nil {
		GetLogEntry(c, vh.lgr).Errorf("Error %v", err)
		return http.StatusInternalServerError
	}

	return http.StatusOK
}

func (vh *InviteHandler) sendEvents(c context.Context, chatId int64, userId int64, callStatus string, ownerId int64, ownerAvatar string, tetATet bool) {
	var usersOfDial []int64 = []int64{userId}

	inviteNames, err := vh.chatClient.GetChatNameForInvite(c, chatId, ownerId, usersOfDial)
	if err != nil {
		GetLogEntry(c, vh.lgr).Error(err, "Failed during getting chat invite names")
		return
	}

	// send the new status (= invitation) immediately to users (callees)
	// send updates for ChatParticipants (blinking green tube)
	m := map[int64]string{userId: callStatus}
	vh.stateChangedEventService.SendDialEvents(c, chatId, m, ownerId, ownerAvatar, tetATet, inviteNames)
}

// owner stops call by exiting
func (vh *InviteHandler) ProcessExit(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context(), vh.lgr).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	tokenIdStr := c.QueryParam("tokenId")
	tokenId, err := uuid.Parse(tokenIdStr)
	if err != nil {
		return err
	}

	// check my access to chat
	if ok, err := vh.chatClient.CheckAccess(c.Request().Context(), userPrincipalDto.UserId, chatId); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	err = db.Transact(c.Request().Context(), vh.database, func(tx *db.Tx) error {
		calleesIOwe, err := tx.GetUserOwnedBeingInvitedCallees(c.Request().Context(), dto.UserCallStateId{
			TokenId: tokenId,
			UserId:  userPrincipalDto.UserId,
		})
		if err != nil {
			return err
		}

		var iAmLeavingOwner bool = false

		if len(calleesIOwe) > 0 {
			iAmLeavingOwner = true
		}

		myState, err := tx.Get(c.Request().Context(), dto.UserCallStateId{
			TokenId: tokenId,
			UserId:  userPrincipalDto.UserId,
		})
		if err != nil {
			return err
		}

		if iAmLeavingOwner { // owner leaving
			ownerId := *calleesIOwe[0].OwnerUserId // == userPrincipalDto.UserId
			ownerAvatar := utils.NullToEmpty(calleesIOwe[0].OwnerAvatar)
			tetATet := calleesIOwe[0].ChatTetATet

			// set myself to status Removing
			vh.softRemoveExtended(c.Request().Context(), tx, []dto.UserCallState{*myState}, db.CallStatusRemoving, &overrideArg{
				overrideOwnerId:     ownerId,
				overrideOwnerAvatar: ownerAvatar,
				overrideTetATet:     tetATet,
				overrideChatId:      chatId,
			})

			// set callees to status Removing
			vh.softRemoveExtended(c.Request().Context(), tx, calleesIOwe, db.CallStatusRemoving, &overrideArg{
				overrideOwnerId:     ownerId,
				overrideOwnerAvatar: ownerAvatar,
				overrideTetATet:     tetATet,
				overrideChatId:      chatId,
			})

			// for being invited participants to dial - send EventMissedCall notification
			vh.sendMissedCallNotification(c.Request().Context(), chatId, userPrincipalDto, calleesIOwe)
		} else {
			// set myself to temporarily status
			var ownerId = utils.OwnerIdToNoUser(myState.OwnerUserId)
			var ownerAvatar = utils.NullToEmpty(myState.OwnerAvatar)
			var tetATet = myState.ChatTetATet
			vh.softRemoveExtended(c.Request().Context(), tx, []dto.UserCallState{*myState}, db.CallStatusRemoving, &overrideArg{
				overrideOwnerId:     ownerId,
				overrideOwnerAvatar: ownerAvatar,
				overrideTetATet:     tetATet,
				overrideChatId:      chatId,
			})
		}
		vh.stateChangedEventService.NotifyAllChatsAboutUsersInVideoStatus(c.Request().Context(), tx, []int64{userPrincipalDto.UserId})
		return nil
	})

	if err != nil {
		GetLogEntry(c.Request().Context(), vh.lgr).Errorf("Error during leaving: %v", err)
		return err
	}

	return c.NoContent(http.StatusOK)
}

func (vh *InviteHandler) sendMissedCallNotification(ctx context.Context, chatId int64, userPrincipalDto *auth.AuthResult, statuses []dto.UserCallState) {

	missedUserMap := map[int64][]dto.UserCallState{}
	for _, status := range statuses {
		missedUserMap[status.UserId] = append(missedUserMap[status.UserId], status)
	}
	missedUsersList := make([]int64, 0)
	for k, v := range missedUserMap {
		if len(v) > 0 {
			missedUsersList = append(missedUsersList, k)
		}
	}

	if len(missedUsersList) > 0 {
		if chatNames, err := vh.chatClient.GetChatNameForInvite(ctx, chatId, userPrincipalDto.UserId, missedUsersList); err != nil {
			GetLogEntry(ctx, vh.lgr).Errorf("Error %v", err)
		} else {
			for _, chatName := range chatNames {
				states := missedUserMap[chatName.UserId]
				sentToUser := false
				for _, state := range states {
					if state.Status == db.CallStatusBeingInvited && !sentToUser {
						// here send missed call notification
						var missedCall = dto.NotificationEvent{
							EventType:              EventMissedCall,
							ChatId:                 chatId,
							UserId:                 chatName.UserId,
							MissedCallNotification: &dto.MissedCallNotification{chatName.Name},
							ByUserId:               userPrincipalDto.UserId,
							ByLogin:                userPrincipalDto.UserLogin,
						}
						if len(userPrincipalDto.Avatar) > 0 {
							missedCall.ByAvatar = &userPrincipalDto.Avatar
						}

						err = vh.notificationPublisher.Publish(ctx, missedCall)
						if err != nil {
							GetLogEntry(ctx, vh.lgr).Errorf("Error %v", err)
						}

						sentToUser = true
					}
				}

			}
		}
	}
}

// send current dial statuses to WebSocket
func (vh *InviteHandler) SendCurrentInVideoStatuses(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context(), vh.lgr).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	userIdParam := c.QueryParam("userId")
	userIds := make([]int64, 0)
	split := strings.Split(userIdParam, ",")
	for _, us := range split {
		if us == "" {
			continue
		}
		parseInt64, err := utils.ParseInt64(us)
		if err != nil {
			GetLogEntry(c.Request().Context(), vh.lgr).Errorf("unable to parse %v", err)
		} else {
			userIds = append(userIds, parseInt64)
		}
	}

	if len(userIds) > 0 {
		err := db.Transact(c.Request().Context(), vh.database, func(tx *db.Tx) error {
			vh.stateChangedEventService.NotifyAllChatsAboutUsersInVideoStatus(c.Request().Context(), tx, userIds)
			return nil
		})
		if err != nil {
			GetLogEntry(c.Request().Context(), vh.lgr).Errorf("Error %v", err)
		}
	}

	return c.NoContent(http.StatusOK)
}

func (vh *InviteHandler) GetMyBeingInvitedStatus(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context(), vh.lgr).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	invitation, err := db.TransactWithResult(c.Request().Context(), vh.database, func(tx *db.Tx) (dto.VideoCallInvitation, error) {

		myStates, err := tx.GetMyBeingInvitedStatus(c.Request().Context(), userPrincipalDto.UserId)
		if err != nil {
			return dto.VideoCallInvitation{}, err
		}

		var inviteName string
		var chatId int64 = db.NoChat
		var status string = db.CallStatusNotFound
		var avatar *string

		if len(myStates) > 0 {
			myState := myStates[0]
			chatId = myState.ChatId
			var ownerId = utils.OwnerIdToNoUser(myState.OwnerUserId)
			inviteNames, err := vh.chatClient.GetChatNameForInvite(c.Request().Context(), myState.ChatId, ownerId, []int64{userPrincipalDto.UserId})
			if err != nil {
				return dto.VideoCallInvitation{}, err
			}

			if len(inviteNames) == 0 {
				return dto.VideoCallInvitation{}, errors.New("not found invitation names")
			}
			chatInviteName := inviteNames[0]
			inviteName = chatInviteName.Name
			status = myState.Status
			avatar = services.GetAvatar(utils.NullToEmpty(myState.OwnerAvatar), myState.ChatTetATet)
		}

		invitation := dto.VideoCallInvitation{
			ChatId:   chatId,
			ChatName: inviteName,
			Status:   status,
			Avatar:   avatar,
		}
		return invitation, nil
	})
	if err != nil {
		GetLogEntry(c.Request().Context(), vh.lgr).Errorf("Error: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, invitation)
}
