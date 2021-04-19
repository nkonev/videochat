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
	"github.com/pion/webrtc/v3"
	"github.com/sourcegraph/jsonrpc2"
	websocketjsonrpc2 "github.com/sourcegraph/jsonrpc2/websocket"
	"io/fs"
	"io/ioutil"
	"net/http"
	"nkonev.name/video/config"
	"nkonev.name/video/producer"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed static
var embeddedFiles embed.FS

type HttpHandler struct {
	client          *http.Client
	upgrader        *websocket.Upgrader
	sfu             *sfu.SFU
	conf            *config.ExtendedConfig
	httpFs          *http.FileSystem
	peerUserIdIndex connectionsLockableMap
	rabbitMqPublisher *producer.RabbitPublisher
}
type ExtendedPeerInfo struct {
	userId string
	// will be added after PUT /notify
	peerId    string
	streamId  string
	login     string
	videoMute bool
	audioMute bool
}
type connectionWithData map[*sfu.Peer]ExtendedPeerInfo
type connectionsLockableMap struct {
	sync.RWMutex
	connectionWithData
}

type JsonRpcExtendedHandler struct {
	*server.JSONSignal
	parentHandler *HttpHandler
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
	client *http.Client,
	upgrader *websocket.Upgrader,
	sfu *sfu.SFU,
	conf *config.ExtendedConfig,
	rabbitMqPublisher *producer.RabbitPublisher,
) HttpHandler {
	fsys, err := fs.Sub(embeddedFiles, "static")
	if err != nil {
		panic("Cannot open static embedded dir")
	}
	staticDir := http.FS(fsys)

	handler := HttpHandler{
		client:   client,
		upgrader: upgrader,
		sfu:      sfu,
		conf:     conf,
		httpFs:   &staticDir,
		peerUserIdIndex: connectionsLockableMap{
			RWMutex:            sync.RWMutex{},
			connectionWithData: connectionWithData{},
		},
		rabbitMqPublisher: rabbitMqPublisher,
	}
	return handler
}

var logger = log.New()

func (h *HttpHandler) SfuHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatId := vars["chatId"]
	userId := r.Header.Get("X-Auth-UserId")
	if ok, err := h.checkAccess(userId, chatId); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	chatIdInt64, err2 := ParseInt64(chatId)
	if err2 != nil {
		logger.Error(err2, "Failed during parse chat id")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	userId64, err2 := ParseInt64(userId)
	if err2 != nil {
		logger.Error(err2, "Failed during parse user id")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	r = r.WithContext(NewContext(r.Context(), &ContextData{userId: userId64, chatId: chatIdInt64}))

	c, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error(err, "Unable to upgrade request to websocket", "userId", userId, "chatId", chatId)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer c.Close()

	peer0 := sfu.NewPeer(h.sfu)
	h.storeToIndex(peer0, userId, "", "", "", false, false)
	defer h.removeFromIndex(peer0, userId, c)
	defer h.notifyAboutLeaving(chatId)
	p := server.NewJSONSignal(peer0, logger)
	je := &JsonRpcExtendedHandler{p, h}
	defer p.Close()

	jc := jsonrpc2.NewConn(r.Context(), websocketjsonrpc2.NewObjectStream(c), je)
	<-jc.DisconnectNotify()
}

func (h *HttpHandler) storeToIndex(peer0 *sfu.Peer, userId, peerId, streamId, login string, videoMute, audioMute bool) {
	logger.Info("Storing peer to map", "peer", peer0.ID(), "userId", userId, "streamId", streamId, "login", login)
	h.peerUserIdIndex.Lock()
	defer h.peerUserIdIndex.Unlock()
	h.peerUserIdIndex.connectionWithData[peer0] = ExtendedPeerInfo{
		userId:    userId,
		peerId:    peerId,
		streamId:  streamId,
		login:     login,
		videoMute: videoMute,
		audioMute: audioMute,
	}
}

func (h *HttpHandler) removeFromIndex(peer0 *sfu.Peer, userId string, conn *websocket.Conn) {
	logger.Info("Removing peer from map", "peer", peer0.ID(), "userId", userId)
	h.peerUserIdIndex.Lock()
	defer h.peerUserIdIndex.Unlock()
	conn.Close()
	delete(h.peerUserIdIndex.connectionWithData, peer0)
}

func (h *HttpHandler) getExtendedConnectionInfo(peer0 *sfu.Peer) *ExtendedPeerInfo {
	h.peerUserIdIndex.RLock()
	defer h.peerUserIdIndex.RUnlock()
	s, ok := h.peerUserIdIndex.connectionWithData[peer0]
	if ok {
		return &s
	} else {
		return nil
	}
}

func (h *HttpHandler) Users(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatId := vars["chatId"]
	userId := r.Header.Get("X-Auth-UserId")
	if ok, err := h.checkAccess(userId, chatId); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	response := UsersResponse{}
	response.UsersCount = h.countPeers(chatId)

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

type errorNoAccess struct {}
func (e *errorNoAccess) Error() string { return "No access" }

type errorInternal struct {}
func (e *errorInternal) Error() string { return "Internal error" }

func (h *HttpHandler) userByStreamId(chatId string, interestingStreamId string, behalfUserId string) (*NotifyDto, error) {
	if ok, err := h.checkAccess(behalfUserId, chatId); err != nil {
		return nil, &errorInternal{}
	} else if !ok {
		return nil, &errorNoAccess{}
	}

	session := h.getSessionWithoutCreatingAnew(chatId)
	if session != nil {
		for _, peer := range session.Peers() {
			if h.peerIsAlive(peer) {
				if pwm := h.getPeerMetadataByStreamId(chatId, interestingStreamId); pwm != nil && pwm.ExtendedPeerInfo != nil && pwm.ExtendedPeerInfo.streamId != "" {
					d := NotifyDto{
						PeerId:    pwm.ExtendedPeerInfo.peerId,
						StreamId:  pwm.ExtendedPeerInfo.streamId,
						Login:     pwm.ExtendedPeerInfo.login,
						VideoMute: pwm.ExtendedPeerInfo.videoMute,
						AudioMute: pwm.ExtendedPeerInfo.audioMute,
					}
					return &d, nil
				}
			}
		}
	}
	return nil, nil
}


func (h *HttpHandler) UserByStreamId(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatId := vars["chatId"]
	streamId := r.URL.Query().Get("streamId")
	userId := r.Header.Get("X-Auth-UserId") // behalf

	userDto, err := h.userByStreamId(chatId, streamId, userId)
	if err != nil {
		if errors.Is(err, &errorNoAccess{}) {
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

type chatNotifyDto struct {
	Data       *NotifyDto `json:"data"`
	UsersCount int64      `json:"usersCount"`
	ChatId     int64      `json:"chatId"`
}

func ParseInt64(s string) (int64, error) {
	if i, err := strconv.ParseInt(s, 10, 64); err != nil {
		return 0, err
	} else {
		return i, nil
	}
}

func (h *HttpHandler) getSessionWithoutCreatingAnew(chatId string) *sfu.Session {
	sessionName := fmt.Sprintf("chat%v", chatId)
	if session, ok := h.sfu.GetSessions()[sessionName]; ok {
		return session
	} else {
		return nil
	}
}

func (h *HttpHandler) countPeers(chatId string) int64 {
	var usersCount int64 = 0
	session := h.getSessionWithoutCreatingAnew(chatId)
	if session != nil {
		for _, peer := range session.Peers() {
			if h.peerIsAlive(peer) {
				usersCount++
			}
		}
	}
	return usersCount
}

func (h *HttpHandler) notify(chatId string, data *NotifyDto) error {
	var usersCount = h.countPeers(chatId)
	var chatNotifyDto = chatNotifyDto{}
	if data != nil {
		logger.Info("Notifying with data", "chatId", chatId, "streamId", data.StreamId, "login", data.Login)
		chatNotifyDto.Data = data
	} else {
		logger.Info("Notifying without data", "chatId", chatId)
	}
	chatNotifyDto.UsersCount = usersCount
	parseInt64, err2 := ParseInt64(chatId)
	if err2 != nil {
		logger.Error(err2, "Failed during parse chat id")
		return err2
	}
	chatNotifyDto.ChatId = parseInt64

	marshal, err2 := json.Marshal(chatNotifyDto)
	if err2 != nil {
		logger.Error(err2, "Failed during marshal chatNotifyDto")
		return err2
	}

	return h.rabbitMqPublisher.Publish(marshal)
}

type NotifyDto struct {
	PeerId    string `json:"peerId"`
	StreamId  string `json:"streamId"`
	Login     string `json:"login"`
	VideoMute bool   `json:"videoMute"`
	AudioMute bool   `json:"audioMute"`
}

func (h *HttpHandler) StoreInfoAndNotifyChatParticipants(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatId := vars["chatId"]
	userId := r.Header.Get("X-Auth-UserId")
	if ok, err := h.checkAccess(userId, chatId); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Error(err, "Unable to read body to []byte")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if len(bodyBytes) > 0 {
		logger.Info("Reading optional body")
		var bodyStruct NotifyDto
		err := json.Unmarshal(bodyBytes, &bodyStruct)
		if err != nil {
			logger.Error(err, "Unable to read body's []byte to NotifyDto")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if sfuPeer := h.getPeerByPeerId(chatId, bodyStruct.PeerId); sfuPeer != nil {
			h.storeToIndex(sfuPeer, userId, bodyStruct.PeerId, bodyStruct.StreamId, bodyStruct.Login, bodyStruct.VideoMute, bodyStruct.AudioMute)
			if err := h.notify(chatId, &bodyStruct); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		} else {
			logger.Info("Not found peer metadata by", "chatId", chatId, "peerId", bodyStruct.PeerId)
		}
	} else {
		if err := h.notify(chatId, nil); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}

}

func (h *HttpHandler) peerIsAlive(peer *sfu.Peer) bool {
	if peer == nil {
		return false
	}
	return peer.Publisher().SignalingState() != webrtc.SignalingStateClosed
}

func (h *HttpHandler) Config(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	marshal, err := json.Marshal(h.conf.FrontendConfig)
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

func (h *HttpHandler) Static() http.HandlerFunc {
	fileServer := http.FileServer(*h.httpFs)
	return func(w http.ResponseWriter, r *http.Request) {
		reqUrl := r.RequestURI
		if reqUrl == "/" || reqUrl == "/index.html" || reqUrl == "/favicon.ico" || strings.HasPrefix(reqUrl, "/build") || strings.HasPrefix(reqUrl, "/assets") || reqUrl == "/git.json" {
			fileServer.ServeHTTP(w, r)
		}
	}
}

func (h *HttpHandler) checkAccess(userIdString string, chatIdString string) (bool, error) {
	userId, err := ParseInt64(userIdString)
	if err != nil {
		return false, err
	}
	chatId, err := ParseInt64(chatIdString)
	if err != nil {
		return false, err
	}
	return h.checkAccessInt64(userId, chatId)
}

func (h *HttpHandler) checkAccessInt64(userId int64, chatId int64) (bool, error) {
	url0 := h.conf.ChatConfig.ChatUrlConfig.Base
	url1 := h.conf.ChatConfig.ChatUrlConfig.Access

	url := fmt.Sprintf("%v%v?userId=%v&chatId=%v", url0, url1, userId, chatId)
	response, err := h.client.Get(url)
	if err != nil {
		logger.Error(err, "Transport error during checking access")
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		return true, nil
	} else if response.StatusCode == http.StatusUnauthorized {
		return false, nil
	} else {
		err := errors.New("Unexpected status on checkAccess")
		logger.Error(err, "Unexpected status on checkAccess", "httpCode", response.StatusCode)
		return false, err
	}
}

func (h *HttpHandler) Kick(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatId := vars["chatId"]
	userToKickId := r.URL.Query().Get("userId")

	if h.KickUser(chatId, userToKickId) != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

type peerWithMetadata struct {
	*sfu.Peer
	*ExtendedPeerInfo
	*sfu.Session
}

func (h *HttpHandler) getPeerMetadatas(chatId, userId string) []peerWithMetadata {
	session := h.getSessionWithoutCreatingAnew(chatId)
	var result []peerWithMetadata
	if session == nil {
		return result
	}
	for _, peerF := range session.Peers() {
		if eci := h.getExtendedConnectionInfo(peerF); eci != nil && eci.userId == userId {
			result = append(result, peerWithMetadata{
				peerF,
				eci,
				session,
			})
		}
	}
	return result
}

func (h *HttpHandler) getPeerMetadataByStreamId(chatId, streamId string) *peerWithMetadata {
	session := h.getSessionWithoutCreatingAnew(chatId)// ChatVideo.vue
	if session == nil {
		return nil
	}
	for _, peerF := range session.Peers() {
		if eci := h.getExtendedConnectionInfo(peerF); eci != nil && eci.streamId == streamId {
			return &peerWithMetadata{
				peerF,
				eci,
				session,
			}
		}
	}
	return nil
}

func (h *HttpHandler) getPeerByPeerId(chatId, peerId string) *sfu.Peer {
	session := h.getSessionWithoutCreatingAnew(chatId) // ChatVideo.vue
	if session == nil {
		return nil
	}
	for _, peerF := range session.Peers() {
		if peerF.ID() == peerId {
			return peerF
		}
	}
	return nil
}

func (h *HttpHandler) KickUser(chatId, userId string) error {
	logger.Info("Invoked kick", "chatId", chatId, "userId", userId)

	metadatas := h.getPeerMetadatas(chatId, userId)
	for _, metadata := range metadatas {
		metadata.Peer.Close()
		metadata.Session.RemovePeer(metadata.Peer.ID())
		h.notify(chatId, nil)
	}

	return nil
}

func (h *HttpHandler) notifyAboutLeaving(chatId string) {
	if err := h.notify(chatId, nil); err != nil {
		logger.Error(err, "error during sending leave notification")
	} else {
		logger.Info("Successfully sent notification about leaving")
	}
}

func (h *HttpHandler) notifyAllChats() {
	for sessionName, _ := range h.sfu.GetSessions() {
		var chatId string
		if _, err := fmt.Sscanf(sessionName, "chat%v", &chatId); err != nil {
			logger.Error(err, "error during reading chat id from session", "sessionName", sessionName)
		} else {
			if err = h.notify(chatId, nil); err != nil {
				logger.Error(err, "error during sending periodic notification")
			}
		}
	}
}

func (h *HttpHandler) Schedule() *chan struct{} {
	ticker := time.NewTicker(h.conf.SyncNotificationPeriod)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <- ticker.C:
				logger.Info("Invoked chats periodic notificator")
				h.notifyAllChats()
			case <- quit:
				ticker.Stop()
				return
			}
		}
	}()
	return &quit
}

type UserDtoWrapper struct {
	UserDto *NotifyDto `json:"userDto"`
	Found bool `json:"found"`
}

func (p *JsonRpcExtendedHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	replyError := func(err error) {
		if errors.Is(err, &errorNoAccess{}) {
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
		userDto, err := p.parentHandler.userByStreamId(fmt.Sprintf("%v", fromContext.chatId), userByStreamId.StreamId, fmt.Sprintf("%v", fromContext.userId))
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

	default:
		p.JSONSignal.Handle(ctx, conn, req)
	}
}
