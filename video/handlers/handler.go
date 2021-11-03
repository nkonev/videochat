package handlers

import (
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/pion/ion-sfu/cmd/signal/json-rpc/server"
	log "github.com/pion/ion-sfu/pkg/logger"
	"github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/turn/v2"
	"github.com/sourcegraph/jsonrpc2"
	websocketjsonrpc2 "github.com/sourcegraph/jsonrpc2/websocket"
	"io/fs"
	"net/http"
	"nkonev.name/video/config"
	"nkonev.name/video/dto"
	"nkonev.name/video/service"
	"nkonev.name/video/utils"
	"strings"
)

//go:embed static
var embeddedFiles embed.FS

type Handler struct {
	upgrader        *websocket.Upgrader
	sfu 			*sfu.SFU
	conf            *config.ExtendedConfig
	httpFs          *http.FileSystem
	service *service.ExtendedService
}


type JsonRpcExtendedHandler struct {
	*server.JSONSignal
	service *service.ExtendedService
}

type ContextData struct {
	userId int64
	chatId int64
	login string
}

// key is an unexported type for keys defined in this package.
// This prevents collisions with keys defined in other packages.
type key int

// contextDataKey is the key for user.User values in Contexts. It is
// unexported; clients use user.NewContext and user.FromContext
// instead of using this key directly.
var contextDataKey key

// NewContext returns a new Context that carries value u.
func NewContext(ctx context.Context, u *ContextData) context.Context {
	return context.WithValue(ctx, contextDataKey, u)
}

// FromContext returns the User value stored in ctx, if any.
func FromContext(ctx context.Context) (*ContextData, bool) {
	u, ok := ctx.Value(contextDataKey).(*ContextData)
	return u, ok
}

type UserByStreamId struct {
	StreamId string `json:"streamId"`
	IncludeOtherStreamIds bool `json:"includeOtherStreamIds"`
}

func NewHandler(
	upgrader *websocket.Upgrader,
	sfu *sfu.SFU,
	conf *config.ExtendedConfig,
	service *service.ExtendedService,
) Handler {
	fsys, err := fs.Sub(embeddedFiles, "static")
	if err != nil {
		panic("Cannot open static embedded dir")
	}
	staticDir := http.FS(fsys)

	handler := Handler{
		upgrader: upgrader,
		sfu: sfu,
		conf:     conf,
		httpFs:   &staticDir,
		service: service,
	}
	return handler
}

var logger = log.New()


func (h *Handler) SfuHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatId, userId, err := parseChatIdAndUserId(vars["chatId"], r.Header.Get("X-Auth-UserId"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	decodedString, err := base64.StdEncoding.DecodeString(r.Header.Get("X-Auth-Username"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	userLogin := string(decodedString)

	if ok, err := h.service.CheckAccess(userId, chatId); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	r = r.WithContext(NewContext(r.Context(), &ContextData{userId: userId, chatId: chatId, login: userLogin}))

	c, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error(err, "Unable to upgrade request to websocket", "user_id", userId, "chat_id", chatId)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer c.Close()

	peer0 := sfu.NewPeer(h.sfu)
	// we can't store it here because peer's stream is not initialized yet - TODO recheck
	// h.service.StoreToIndex(...)
	defer h.service.RemoveFromIndex(peer0, userId, c)
	defer h.service.NotifyAboutLeaving(chatId)
	p := server.NewJSONSignal(peer0, logger)
	je := &JsonRpcExtendedHandler{p, h.service}
	defer p.Close()

	jc := jsonrpc2.NewConn(r.Context(), websocketjsonrpc2.NewObjectStream(c), je)
	<-jc.DisconnectNotify()
}


func (h *Handler) CountUsers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatId, userId, err := parseChatIdAndUserId(vars["chatId"], r.Header.Get("X-Auth-UserId"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if ok, err := h.service.CheckAccess(userId, chatId); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	response := CountUsersResponse{}
	response.UsersCount = h.service.CountPeers(chatId)

	w.Header().Set("Content-Type", "application/json")
	marshal, err := json.Marshal(response)
	if err != nil {
		logger.Error(err, "Error during marshalling UsersResponse to json")
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		_, err := w.Write(marshal)
		if err != nil {
			logger.Error(err, "Error during sending json")
		}
	}
}

func (h *Handler) Users(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	chatId, err := utils.ParseInt64(vars["chatId"])
	if err != nil {
		logger.Error(err, "Failed during parse chat id")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response, err := h.service.GetPeersByChatId(chatId)
	if err != nil {
		logger.Error(err, "Failed during getting peers by chat id")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	marshal, err := json.Marshal(response)
	if err != nil {
		logger.Error(err, "Error during marshalling UsersResponse to json")
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		_, err := w.Write(marshal)
		if err != nil {
			logger.Error(err, "Error during sending json")
		}
	}
}

func (h *Handler) UserByStreamId(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	streamId := r.URL.Query().Get("streamId")
	chatId, userId, err := parseChatIdAndUserId(vars["chatId"], r.Header.Get("X-Auth-UserId")) // behalf this userId
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	userDto, otherStreamIds, err := h.service.UserByStreamId(chatId, streamId, true, userId)
	if err != nil {
		if errors.Is(err, &service.ErrorNoAccess{}) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	resp := UserDtoWrapper{}
	if userDto != nil {
		resp.Found = true
		resp.UserDto = userDto
	}
	resp.OtherStreamIds = otherStreamIds
	marshal, err := json.Marshal(resp)
	if err != nil {
		logger.Error(err, "Error during marshalling peerWithMetadata to json")
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(marshal)
		if err != nil {
			logger.Error(err, "Error during sending json")
		}
	}
	return
}

type ICEServerConfigDto struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username"`
	Credential string   `json:"credential"`
}

type FrontendConfigDto struct {
	ICEServers []ICEServerConfigDto `json:"iceServers"`
}

func (h *Handler) Config(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	frontendConfig := h.conf.FrontendConfig
	var responseSliceFrontendConfig = FrontendConfigDto{}

	for _, s := range frontendConfig.ICEServers {
		var newElement = ICEServerConfigDto{
			URLs: s.ICEServerConfig.URLs,
			Username: s.ICEServerConfig.Username,
			Credential: s.ICEServerConfig.Credential,
		}
		if s.LongTermCredentialDuration != 0 {
			username, password, err := turn.GenerateLongTermCredentials(h.conf.Turn.Auth.Secret, s.LongTermCredentialDuration)
			if err != nil {
				logger.Error(err, "Error during GenerateLongTermCredentials")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			newElement.Credential = password
			newElement.Username = username
		}
		responseSliceFrontendConfig.ICEServers = append(responseSliceFrontendConfig.ICEServers, newElement)
	}

	marshal, err := json.Marshal(responseSliceFrontendConfig)
	if err != nil {
		logger.Error(err, "Error during marshalling ConfigResponse to json")
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		_, err := w.Write(marshal)
		if err != nil {
			logger.Error(err, "Error during sending json")
		}
	}
}

type CountUsersResponse struct {
	UsersCount int64 `json:"usersCount"`
}

func (h *Handler) Static() http.HandlerFunc {
	fileServer := http.FileServer(*h.httpFs)
	return func(w http.ResponseWriter, r *http.Request) {
		reqUrl := r.RequestURI
		if reqUrl == "/" || reqUrl == "/index.html" || reqUrl == "/favicon.ico" || strings.HasPrefix(reqUrl, "/build") || strings.HasPrefix(reqUrl, "/assets") || reqUrl == "/git.json" {
			fileServer.ServeHTTP(w, r)
		}
	}
}


func (h *Handler) Kick(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatId, userToKickId, err := parseChatIdAndUserId(vars["chatId"], r.URL.Query().Get("userId"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	silent, err := utils.ParseBoolean(r.URL.Query().Get("silent"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if h.service.KickUser(chatId, userToKickId, silent) != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func parseChatIdAndUserId(chatId, userId string) (int64, int64, error) {
	chatIdInt64, err := utils.ParseInt64(chatId)
	if err != nil {
		logger.Error(err, "Failed during parse chat id")
		return -1, -1, err
	}
	userId64, err := utils.ParseInt64(userId)
	if err != nil {
		logger.Error(err, "Failed during parse user id")
		return -1, -1, err
	}
	return chatIdInt64, userId64, nil
}

type UserDtoWrapper struct {
	UserDto *dto.StoreNotifyDto `json:"userDto"`
	Found bool              `json:"found"`
	OtherStreamIds []string `json:"otherStreamIds"`
}

func (p *JsonRpcExtendedHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	replyError := func(err error) {
		if errors.Is(err, &service.ErrorNoAccess{}) {
			_ = conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
				Code:    401,
				Message: err.Error(),
			})
		} else {
			_ = conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
				Code:    500,
				Message: fmt.Sprintf("%s", err),
			})
		}
	}

	fromContext, b := FromContext(ctx)
	if !b {
		err := errors.New("unable to extract data from context")
		p.Logger.Error(err, "problem with getting data from context")
		replyError(err)
		return
	}

	switch req.Method {
	case "offer":
		streamId, err := getStreamId(req.Params)
		if err != nil {
			p.Logger.Error(err, "connect: error parsing stream id")
			replyError(err)
			break
		}

		logger.Info("Extracted streamId from sdp", "stream_id", streamId)
		// we need to sync mutes with front default or find the correct approach to send if after client has connected.
		// appending to getAndPublishLocalMediaStream() Promise is too early and leads to "Not found peer metadata by"
		p.service.StoreToIndex(streamId, fromContext.userId, fromContext.login, false, false)

		p.JSONSignal.Handle(ctx, conn, req)

	case "userByStreamId":
		var userByStreamId UserByStreamId
		err := json.Unmarshal(*req.Params, &userByStreamId)
		if err != nil {
			p.Logger.Error(err, "error parsing UserByStreamId request")
			replyError(err)
			break
		}
		userDto, otherStreamIds, err := p.service.UserByStreamId(fromContext.chatId, userByStreamId.StreamId, userByStreamId.IncludeOtherStreamIds, fromContext.userId)
		if err != nil {
			replyError(err)
			break
		}
		resp := UserDtoWrapper{}
		if userDto != nil {
			resp.Found = true
			resp.UserDto = userDto
		}
		resp.OtherStreamIds = otherStreamIds
		_ = conn.Reply(ctx, req.ID, resp)

	case "putUserData": {
		var bodyStruct dto.UserInputDto
		err := json.Unmarshal(*req.Params, &bodyStruct)
		if err != nil {
			p.Logger.Error(err, "error parsing StoreNotifyDto request")
			break
		}
		if p.service.ExistsPeerByStreamId(fromContext.chatId, bodyStruct.StreamId) {
			p.service.StoreToIndex(bodyStruct.StreamId, fromContext.userId, fromContext.login, bodyStruct.VideoMute, bodyStruct.AudioMute)
			notificationDto := &dto.StoreNotifyDto{
				UserId:    fromContext.userId,
				StreamId:  bodyStruct.StreamId,
				Login:     fromContext.login,
				VideoMute: bodyStruct.VideoMute,
				AudioMute: bodyStruct.AudioMute,
			}
			if err := p.service.Notify(fromContext.chatId, notificationDto); err != nil {
				p.Logger.Error(err, "error during sending notification")
			}
		} else {
			logger.Info("Not found peer metadata by", "chat_id", fromContext.chatId, "stream_id", bodyStruct.StreamId)
		}
	}
	default:
		p.JSONSignal.Handle(ctx, conn, req)
	}
}

func getStreamId(params *json.RawMessage) (string, error) {
	var negotiation server.Negotiation
	err := json.Unmarshal(*params, &negotiation)
	if err != nil {
		return "", errors.New("error parsing offer from jsonrpc params")
	}
	// get stream id from negotiation.Desc, userId from context and put metadata to store
	unmarshalledSdp, err := negotiation.Desc.Unmarshal()
	if err != nil {
		return "", errors.New("error parsing sdp from negotiation")
	}

	if unmarshalledSdp == nil {
		return "", errors.New("Missed SDP")
	}
	for _, mediaDescription := range unmarshalledSdp.MediaDescriptions {
		for _, attribute := range mediaDescription.Attributes {
			if attribute.Key == "msid" {
				split := strings.Split(attribute.Value, " ")
				if len(split) < 1 {
					return "", errors.New("Invalid msid " + attribute.Value)
				}
				msid := split[0]
				return msid, nil
			}
		}
	}
	return "", errors.New("Unable to find at least one media attribute")
}
