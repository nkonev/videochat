package handlers

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/nkonev/ion-sfu/cmd/signal/json-rpc/server"
	"github.com/nkonev/ion-sfu/pkg/sfu"
	log "github.com/pion/ion-log"
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
	client *http.Client
	upgrader *websocket.Upgrader
	sessionUserPeer *sync.Map
	sfu *sfu.SFU
	conf *config.ExtendedConfig
	httpFs *http.FileSystem
}

func NewHandler(
	client *http.Client,
	upgrader *websocket.Upgrader,
	sfu *sfu.SFU,
	conf *config.ExtendedConfig,
) Handler {
	fsys, err := fs.Sub(embeddedFiles, "static")
	if err != nil {
		log.Panicf("Cannot open static embedded dir")
	}
	staticDir := http.FS(fsys)

	return Handler{
		client:          client,
		upgrader:        upgrader,
		sessionUserPeer: &sync.Map{},
		sfu:             sfu,
		conf:            conf,
		httpFs: 		&staticDir,
	}
}


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

	p := server.NewJSONSignal(sfu.NewPeer(h.sfu))
	addPeerToMap(h.sessionUserPeer, chatId, userId, p)
	defer p.Close()
	defer removePeerFromMap(h.sessionUserPeer, chatId, userId)

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

	chatInterface, ok := h.sessionUserPeer.Load(chatId)
	response := UsersResponse{}
	if ok {
		chat := chatInterface.(*sync.Map)
		response.UsersCount = countMapLen(chat)
	}
	w.Header().Set("Content-Type", "application/json")
	marshal, err := json.Marshal(response)
	if err != nil {
		log.Errorf("Error during marshalling UsersResponse to json")
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		_, err := w.Write(marshal)
		if err != nil {
			log.Errorf("Error during sending json")
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
	chatInterface, ok := h.sessionUserPeer.Load(chatId)
	if ok {
		chat := chatInterface.(*sync.Map)
		usersCount = countMapLen(chat)
	}

	url0 := h.conf.ChatConfig.ChatUrlConfig.Base
	url1 := h.conf.ChatConfig.ChatUrlConfig.Notify

	fullUrl := fmt.Sprintf("%v%v?usersCount=%v&chatId=%v", url0, url1, usersCount, chatId)
	parsedUrl, err := url.Parse(fullUrl)
	if err != nil {
		log.Errorf("Failed during parse chat url: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	req := &http.Request{Method: http.MethodPut, URL: parsedUrl}

	response, err := h.client.Do(req)
	if err != nil {
		log.Errorf("Transport error during notifying %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		if response.StatusCode != http.StatusOK {
			log.Errorf("Http Error %v during notifying %v", response.StatusCode, err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

// GET `/api/video/config`
func (h *Handler) Config(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	marshal, err := json.Marshal(h.conf.FrontendConfig)
	if err != nil {
		log.Errorf("Error during marshalling ConfigResponse to json")
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		_, err := w.Write(marshal)
		if err != nil {
			log.Errorf("Error during sending json")
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
		log.Errorf("Transport error during checking access %v", err)
		return false
	}
	if response.StatusCode == http.StatusOK {
		return true
	} else if response.StatusCode == http.StatusUnauthorized {
		return false
	} else {
		log.Errorf("Unexpected status on checkAccess %v", response.StatusCode)
		return false
	}
}

func addPeerToMap(sessionUserPeer *sync.Map, chatId string, userId string, peer *server.JSONSignal) {
	userPeerInterface, _ := sessionUserPeer.LoadOrStore(chatId, &sync.Map{})
	userPeer := userPeerInterface.(*sync.Map)
	log.Infof("Storing peer for userId=%v to chatId=%v", userId, chatId)
	userPeer.Store(userId, peer)
}

func removePeerFromMap(sessionUserPeer *sync.Map, chatId string, userId string) {
	userPeerInterface, ok := sessionUserPeer.Load(chatId)
	if !ok {
		log.Errorf("Cannot remove chatId=%v from sessionUserPeer", chatId)
		return
	}
	userPeer := userPeerInterface.(*sync.Map)
	log.Infof("Removing peer for userId=%v from chatId=%v", userId, chatId)
	userPeer.Delete(userId)

	userPeerLength := countMapLen(userPeer)

	if userPeerLength == 0 {
		log.Infof("For chatId=%v there is no peers, removing user %v", chatId, userId)
		sessionUserPeer.Delete(chatId)
	}
}

func countMapLen(m *sync.Map) int64 {
	var length int64 = 0
	m.Range(func(_, _ interface{}) bool {
		length++
		return true
	})
	return length
}

