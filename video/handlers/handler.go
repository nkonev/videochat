package handlers

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	log "github.com/nkonev/ion-sfu/pkg/logger"
	"github.com/nkonev/ion-sfu/cmd/signal/json-rpc/server"
	"github.com/nkonev/ion-sfu/pkg/sfu"
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
	client    *http.Client
	upgrader  *websocket.Upgrader
	sfu       *sfu.SFU
	conf      *config.ExtendedConfig
	httpFs    *http.FileSystem
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
		client:    client,
		upgrader:  upgrader,
		sfu:       sfu,
		conf:      conf,
		httpFs:    &staticDir,
	}
	return handler
}

var 	logger         = log.New()


func (h *Handler) SfuHandler(w http.ResponseWriter, r *http.Request) {
	userId := r.Header.Get("X-Auth-UserId")
	chatId := r.URL.Query().Get("chatId")
	if !h.checkAccess(h.client, userId, chatId) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	c, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	p := server.NewJSONSignal(sfu.NewPeerWithMetadata(h.sfu, userId), logger)
	//h.addPeerToMap(userId, p.Peer)
	defer p.Close()

	jc := jsonrpc2.NewConn(r.Context(), websocketjsonrpc2.NewObjectStream(c), p)
	<-jc.DisconnectNotify()
}

// GET /api/video/users?chatId=${this.chatId} - responds users count
func (h *Handler) Users(w http.ResponseWriter, r *http.Request) {
	userId := r.Header.Get("X-Auth-UserId")
	chatId := r.URL.Query().Get("chatId")
	if !h.checkAccess(h.client, userId, chatId) {
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
		logger.Info("Error during marshalling UsersResponse to json")
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		_, err := w.Write(marshal)
		if err != nil {
			logger.Info("Error during sending json")
		}
	}
}

// PUT /api/video/notify?chatId=${this.chatId}` -> "/internal/video/notify"
func (h *Handler) NotifyChatParticipants(w http.ResponseWriter, r *http.Request) {
	userId := r.Header.Get("X-Auth-UserId")
	chatId := r.URL.Query().Get("chatId")
	if !h.checkAccess(h.client, userId, chatId) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
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
		logger.Info("Failed during parse chat url: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	req := &http.Request{Method: http.MethodPut, URL: parsedUrl}

	response, err := h.client.Do(req)
	if err != nil {
		logger.Info("Transport error during notifying %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		if response.StatusCode != http.StatusOK {
			logger.Info("Http Error %v during notifying %v", response.StatusCode, err)
			w.WriteHeader(http.StatusInternalServerError)
		}
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

// GET `/api/video/config`
func (h *Handler) Config(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	marshal, err := json.Marshal(h.conf.FrontendConfig)
	if err != nil {
		logger.Info("Error during marshalling ConfigResponse to json")
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		_, err := w.Write(marshal)
		if err != nil {
			logger.Info("Error during sending json")
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

func (h *Handler) checkAccess(client *http.Client, userIdString string, chatIdString string) bool {
	url0 := h.conf.ChatConfig.ChatUrlConfig.Base
	url1 := h.conf.ChatConfig.ChatUrlConfig.Access

	response, err := client.Get(url0 + url1 + "?userId=" + userIdString + "&chatId=" + chatIdString)
	if err != nil {
		logger.Info("Transport error during checking access %v", err)
		return false
	}
	if response.StatusCode == http.StatusOK {
		return true
	} else if response.StatusCode == http.StatusUnauthorized {
		return false
	} else {
		logger.Info("Unexpected status on checkAccess %v", response.StatusCode)
		return false
	}
}

// GET `/internal/kick`
func (h *Handler) Kick(w http.ResponseWriter, r *http.Request) {
	chatId := r.URL.Query().Get("chatId")
	userToKickId := r.URL.Query().Get("userId")
	h.kick(chatId, userToKickId)
}

type UserPeers struct {
	sync.RWMutex
	Peers []*sfu.Peer
}

func (h *Handler) kick(chatId, userId string) {
	// for peer := session.peers
	session, _ := h.sfu.GetSession(fmt.Sprintf("chat%v", chatId)) // ChatVideo.vue
	if session == nil {
		return
	}
	for _, peerF := range session.Peers() {
		// if getUserId(peer) == userId
		/*if h.hasPeer(userId, peerF) {
			// session.disconnect(peer)
			session.RemovePeer(peerF.ID())
		}*/
		gotUserId := peerF.Metadata()
		i := gotUserId.(string)
		if userId == i {
			peerF.Close()
			session.RemovePeer(peerF.ID())
		}
	}
}
/*
func (h *Handler) addPeerToMap(userId string, peer *sfu.Peer) {
	userPeerInterface, _ := h.userPeers.LoadOrStore(userId, &UserPeers{})
	userPeer := userPeerInterface.(*UserPeers)
	log.Infof("Storing peer for userId=%v", userId)
	userPeer.Lock()
	defer userPeer.Unlock()
	userPeer.Peers = append(userPeer.Peers, peer)
}

func (h *Handler) hasPeer(userId string, peer *sfu.Peer) bool {
	if peer == nil {
		return false
	}
	if load, ok := h.userPeers.Load(userId); ok {
		userPeer := load.(*UserPeers)
		userPeer.RLock()
		defer userPeer.RUnlock()
		for _, enumerablePeer := range userPeer.Peers {
			if enumerablePeer.ID() == peer.ID() {
				return true
			}
		}
	}
	return false
}

func (h *Handler) removePeer(peer *sfu.Peer) {
	if peer == nil {
		return
	}
	h.userPeers.Range(func(_, enumerableUserI interface{}) bool {
		enumerableUser := enumerableUserI.(*UserPeers)
		enumerableUser.Lock()
		defer enumerableUser.Unlock()
		for _, userPeer := range enumerableUser.Peers {
			userPeer
		}
		return true
	})

}

func countMapLen(m *sync.Map) int64 {
	var length int64 = 0
	m.Range(func(_, _ interface{}) bool {
		length++
		return true
	})
	return length
}

*/