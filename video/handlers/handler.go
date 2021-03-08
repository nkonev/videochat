package handlers

import (
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
	"net/http"
	"net/url"
	"nkonev.name/video/config"
	"strings"
	"sync"
)

//go:embed static
var embeddedFiles embed.FS

type Handler struct {
	client      *http.Client
	upgrader    *websocket.Upgrader
	sfu         *sfu.SFU
	conf        *config.ExtendedConfig
	httpFs      *http.FileSystem
	connections connectionsLockableMap
}
type extendedConnectionInfo struct {
	userId string
	websocketConnection *websocket.Conn
}
type connectionWithData map[*sfu.Peer]extendedConnectionInfo
type connectionsLockableMap struct {
	sync.RWMutex
	connectionWithData
}

func NewHandler(
	client *http.Client,
	upgrader *websocket.Upgrader,
	sfu *sfu.SFU,
	conf *config.ExtendedConfig,
) Handler {
	fsys, err := fs.Sub(embeddedFiles, "static")
	if err != nil {
		panic("Cannot open static embedded dir")
	}
	staticDir := http.FS(fsys)

	handler := Handler{
		client:      client,
		upgrader:    upgrader,
		sfu:         sfu,
		conf:        conf,
		httpFs:      &staticDir,
		connections: connectionsLockableMap{
			RWMutex:            sync.RWMutex{},
			connectionWithData: connectionWithData{},
		},
	}
	return handler
}

var 	logger         = log.New()

func (h *Handler) SfuHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatId := vars["chatId"]
	userId := r.Header.Get("X-Auth-UserId")
	if !h.checkAccess(userId, chatId) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	c, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error(err, "Unable to upgrade request to websocket", "userId", userId, "chatId", chatId)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer c.Close()

	peer0 := sfu.NewPeer(h.sfu)
	h.addToConnMap(peer0, userId, c)
	defer h.removeFromConnMap(peer0, userId, c)
	p := server.NewJSONSignal(peer0, logger)
	defer p.Close()

	jc := jsonrpc2.NewConn(r.Context(), websocketjsonrpc2.NewObjectStream(c), p)
	<-jc.DisconnectNotify()
}

func (h *Handler) addToConnMap(peer0 *sfu.Peer, userId string, conn *websocket.Conn) {
	logger.Info("Adding peer to map", "peer", peer0.ID(), "userId", userId)
	h.connections.Lock()
	defer h.connections.Unlock()
	h.connections.connectionWithData[peer0] = extendedConnectionInfo{userId, conn}
}

func (h *Handler) removeFromConnMap(peer0 *sfu.Peer, userId string, conn *websocket.Conn) {
	logger.Info("Removing peer from map", "peer", peer0.ID(), "userId", userId)
	h.connections.Lock()
	defer h.connections.Unlock()
	conn.Close()
	delete(h.connections.connectionWithData, peer0)
}

func (h *Handler) getFromConnMap(peer0 *sfu.Peer) string {
	h.connections.RLock()
	defer h.connections.RUnlock()
	s, ok := h.connections.connectionWithData[peer0]
	if ok {
		return s.userId
	} else {
		return ""
	}
}

func (h *Handler) Users(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatId := vars["chatId"]
	userId := r.Header.Get("X-Auth-UserId")
	if !h.checkAccess(userId, chatId) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	response := UsersResponse{}

	if session, _:= h.sfu.GetSession(fmt.Sprintf("chat%v", chatId)); session != nil {
		var usersCount int64
		for _, peer:= range session.Peers() {
			if h.peerIsAlive(peer) {
				usersCount++
			}
		}
		response.UsersCount = usersCount
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

func (h *Handler) notify(chatId string) error {
	var usersCount int64 = 0
	session, _ := h.sfu.GetSession(fmt.Sprintf("chat%v", chatId))
	if session != nil {
		for _, peer:= range session.Peers() {
			if h.peerIsAlive(peer) {
				usersCount++
			}
		}
	}

	url0 := h.conf.ChatConfig.ChatUrlConfig.Base
	url1 := h.conf.ChatConfig.ChatUrlConfig.Notify

	fullUrl := fmt.Sprintf("%v%v?usersCount=%v&chatId=%v", url0, url1, usersCount, chatId)
	parsedUrl, err := url.Parse(fullUrl)
	if err != nil {
		logger.Error(err, "Failed during parse chat url")
		return err
	}

	req := &http.Request{Method: http.MethodPut, URL: parsedUrl}

	response, err := h.client.Do(req)
	if err != nil {
		logger.Error(err, "Transport error during notifying")
		return err
	} else {
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			logger.Error(err, "Http Error during notifying", "httpCode", response.StatusCode, "chatId", chatId)
			return err
		}
	}
	return nil
}

func (h *Handler) NotifyChatParticipants(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatId := vars["chatId"]
	userId := r.Header.Get("X-Auth-UserId")
	if !h.checkAccess(userId, chatId) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if err := h.notify(chatId); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *Handler) peerIsAlive(peer *sfu.Peer) bool {
	if peer == nil {
		return false
	}
	return peer.Publisher().SignalingState() != webrtc.SignalingStateClosed
}

func (h *Handler) peerIsClosed(peer *sfu.Peer) bool {
	if peer == nil {
		return false
	}
	return peer.Publisher().SignalingState() == webrtc.SignalingStateClosed
}

func (h *Handler) Config(w http.ResponseWriter, r *http.Request) {
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

func (h *Handler) Static() http.HandlerFunc {
	fileServer := http.FileServer(*h.httpFs)
	return func(w http.ResponseWriter, r *http.Request) {
		reqUrl := r.RequestURI
		if reqUrl == "/" || reqUrl == "/index.html" || reqUrl == "/favicon.ico" || strings.HasPrefix(reqUrl, "/build") || strings.HasPrefix(reqUrl, "/assets") || reqUrl == "/git.json" {
			fileServer.ServeHTTP(w, r)
		}
	}
}

func (h *Handler) checkAccess(userIdString string, chatIdString string) bool {
	url0 := h.conf.ChatConfig.ChatUrlConfig.Base
	url1 := h.conf.ChatConfig.ChatUrlConfig.Access

	response, err := h.client.Get(url0 + url1 + "?userId=" + userIdString + "&chatId=" + chatIdString)
	if err != nil {
		logger.Error(err, "Transport error during checking access")
		return false
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		return true
	} else if response.StatusCode == http.StatusUnauthorized {
		return false
	} else {
		logger.Error(errors.New("Unexpected status on checkAccess"), "Unexpected status on checkAccess", "httpCode", response.StatusCode)
		return false
	}
}

func (h *Handler) Kick(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatId := vars["chatId"]
	userToKickId := r.URL.Query().Get("userId")
	var notifyBool bool = r.URL.Query().Get("notify") == "true"

	if h.kick(chatId, userToKickId, notifyBool) != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *Handler) kick(chatId, userId string, notifyBool bool) error {
	logger.Info("Invoked kick", "chatId", chatId, "userId", userId, "notify", notifyBool)
	session, _ := h.sfu.GetSession(fmt.Sprintf("chat%v", chatId)) // ChatVideo.vue
	if session == nil {
		return nil
	}
	for _, peerF := range session.Peers() {
		if userId == h.getFromConnMap(peerF) {
			peerF.Close()
			session.RemovePeer(peerF.ID())
			h.notify(chatId)
		}
	}

	// send notification through chat's personal centrifuge channel
	if notifyBool {
		url0 := h.conf.ChatConfig.ChatUrlConfig.Base
		url1 := h.conf.ChatConfig.ChatUrlConfig.Kick

		fullUrl := fmt.Sprintf("%v%v?userId=%v&chatId=%v", url0, url1, userId, chatId)
		parsedUrl, err := url.Parse(fullUrl)
		if err != nil {
			logger.Error(err, "Failed during parse chat url")
			return err
		}

		req := &http.Request{Method: http.MethodPut, URL: parsedUrl}

		response, err := h.client.Do(req)
		if err != nil {
			logger.Error(err, "Transport error during kicking")
			return err
		} else {
			defer response.Body.Close()
			if response.StatusCode != http.StatusOK {
				logger.Error(err, "Http Error during kicking", "httpCode", response.StatusCode, "chatId", chatId)
				return err
			}
		}
	}
	return nil
}
