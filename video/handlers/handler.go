package handlers

import (
	"context"
	"embed"
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

	if ok, err := h.service.CheckAccess(userId, chatId); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	r = r.WithContext(NewContext(r.Context(), &ContextData{userId: userId, chatId: chatId}))

	c, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error(err, "Unable to upgrade request to websocket", "user_id", userId, "chat_id", chatId)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer c.Close()

	peer0 := sfu.NewPeer(h.sfu)
	h.service.StoreToIndex(peer0, userId, "", "", "", false, false)
	defer h.service.RemoveFromIndex(peer0, userId, c)
	defer h.service.NotifyAboutLeaving(chatId)
	p := server.NewJSONSignal(peer0, logger)
	je := &JsonRpcExtendedHandler{p, h.service}
	defer p.Close()

	jc := jsonrpc2.NewConn(r.Context(), websocketjsonrpc2.NewObjectStream(c), je)
	<-jc.DisconnectNotify()
}


func (h *Handler) Users(w http.ResponseWriter, r *http.Request) {
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

	response := UsersResponse{}
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

func (h *Handler) UserByStreamId(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	streamId := r.URL.Query().Get("streamId")
	chatId, userId, err := parseChatIdAndUserId(vars["chatId"], r.Header.Get("X-Auth-UserId")) // behalf this userId
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	userDto, err := h.service.UserByStreamId(chatId, streamId, userId)
	if err != nil {
		if errors.Is(err, &service.ErrorNoAccess{}) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	if userDto == nil{
		w.WriteHeader(http.StatusNoContent)
		return
	}
	marshal, err := json.Marshal(userDto)
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

type UsersResponse struct {
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


	if h.service.KickUser(chatId, userToKickId) != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func parseChatIdAndUserId(chatId, userId string) (int64, int64, error) {
	chatIdInt64, err := service.ParseInt64(chatId)
	if err != nil {
		logger.Error(err, "Failed during parse chat id")
		return -1, -1, err
	}
	userId64, err := service.ParseInt64(userId)
	if err != nil {
		logger.Error(err, "Failed during parse user id")
		return -1, -1, err
	}
	return chatIdInt64, userId64, nil
}

type UserDtoWrapper struct {
	UserDto *dto.StoreNotifyDto `json:"userDto"`
	Found bool              `json:"found"`
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
		p.Logger.Error(err, "problem with getting tata from context")
		replyError(err)

	}

	switch req.Method {
	case "userByStreamId":
		var userByStreamId UserByStreamId
		err := json.Unmarshal(*req.Params, &userByStreamId)
		if err != nil {
			p.Logger.Error(err, "error parsing UserByStreamId request")
			replyError(err)
			break
		}
		userDto, err := p.service.UserByStreamId(fromContext.chatId, userByStreamId.StreamId, fromContext.userId)
		if err != nil {
			replyError(err)
			break
		}
		resp := UserDtoWrapper{}
		if userDto != nil {
			resp.Found = true
			resp.UserDto = userDto
		}
		_ = conn.Reply(ctx, req.ID, resp)

	case "putUserData": {
		var bodyStruct dto.StoreNotifyDto
		err := json.Unmarshal(*req.Params, &bodyStruct)
		if err != nil {
			p.Logger.Error(err, "error parsing StoreNotifyDto request")
			break
		}
		if sfuPeer := p.service.GetPeerByPeerId(fromContext.chatId, bodyStruct.PeerId); sfuPeer != nil {
			p.service.StoreToIndex(sfuPeer, fromContext.chatId, bodyStruct.PeerId, bodyStruct.StreamId, bodyStruct.Login, bodyStruct.VideoMute, bodyStruct.AudioMute)
			if err := p.service.Notify(fromContext.chatId, &bodyStruct); err != nil {
				p.Logger.Error(err, "error during sending notification")
			}
		} else {
			logger.Info("Not found peer metadata by", "chat_id", fromContext.chatId, "peer_id", bodyStruct.PeerId)
		}
	}
	default:
		p.JSONSignal.Handle(ctx, conn, req)
	}
}
