package handlers

import (
	"bytes"
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
	"net/url"
	"nkonev.name/video/config"
	"strconv"
	"strings"
	"sync"
)

//go:embed static
var embeddedFiles embed.FS

type Handler struct {
	client          *http.Client
	upgrader        *websocket.Upgrader
	sfu             *sfu.SFU
	conf            *config.ExtendedConfig
	httpFs          *http.FileSystem
	peerUserIdIndex connectionsLockableMap
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
		client:   client,
		upgrader: upgrader,
		sfu:      sfu,
		conf:     conf,
		httpFs:   &staticDir,
		peerUserIdIndex: connectionsLockableMap{
			RWMutex:            sync.RWMutex{},
			connectionWithData: connectionWithData{},
		},
	}
	return handler
}

var logger = log.New()

func (h *Handler) SfuHandler(w http.ResponseWriter, r *http.Request) {
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
	p := server.NewJSONSignal(peer0, logger)
	defer p.Close()

	jc := jsonrpc2.NewConn(r.Context(), websocketjsonrpc2.NewObjectStream(c), p)
	<-jc.DisconnectNotify()
}

func (h *Handler) storeToIndex(peer0 *sfu.Peer, userId, peerId, streamId, login string, videoMute, audioMute bool) {
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

func (h *Handler) removeFromIndex(peer0 *sfu.Peer, userId string, conn *websocket.Conn) {
	logger.Info("Removing peer from map", "peer", peer0.ID(), "userId", userId)
	h.peerUserIdIndex.Lock()
	defer h.peerUserIdIndex.Unlock()
	conn.Close()
	delete(h.peerUserIdIndex.connectionWithData, peer0)
}

func (h *Handler) getExtendedConnectionInfo(peer0 *sfu.Peer) *ExtendedPeerInfo {
	h.peerUserIdIndex.RLock()
	defer h.peerUserIdIndex.RUnlock()
	s, ok := h.peerUserIdIndex.connectionWithData[peer0]
	if ok {
		return &s
	} else {
		return nil
	}
}

func (h *Handler) Users(w http.ResponseWriter, r *http.Request) {
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

func (h *Handler) UserByStreamId(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatId := vars["chatId"]
	streamId := r.URL.Query().Get("streamId")
	userId := r.Header.Get("X-Auth-UserId") // behalf
	if ok, err := h.checkAccess(userId, chatId); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	session, _ := h.sfu.GetSession(fmt.Sprintf("chat%v", chatId))
	if session != nil {
		for _, peer := range session.Peers() {
			if h.peerIsAlive(peer) {
				if pwm := h.getPeerMetadataByStreamId(chatId, streamId); pwm != nil && pwm.ExtendedPeerInfo != nil && pwm.ExtendedPeerInfo.streamId != "" {
					w.Header().Set("Content-Type", "application/json")
					d := NotifyDto{
						PeerId:    pwm.ExtendedPeerInfo.peerId,
						StreamId:  pwm.ExtendedPeerInfo.streamId,
						Login:     pwm.ExtendedPeerInfo.login,
						VideoMute: pwm.ExtendedPeerInfo.videoMute,
						AudioMute: pwm.ExtendedPeerInfo.audioMute,
					}
					marshal, err := json.Marshal(d)
					if err != nil {
						logger.Error(err, "Error during marshalling peerWithMetadata to json")
						w.WriteHeader(http.StatusInternalServerError)
					} else {
						_, err := w.Write(marshal)
						if err != nil {
							logger.Error(err, "Error during sending json")
						}
					}
					return
				}
			}
		}
	}
	w.WriteHeader(http.StatusNoContent)
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

func (h *Handler) countPeers(chatId string) int64 {
	var usersCount int64 = 0
	session, _ := h.sfu.GetSession(fmt.Sprintf("chat%v", chatId))
	if session != nil {
		for _, peer := range session.Peers() {
			if h.peerIsAlive(peer) {
				usersCount++
			}
		}
	}
	return usersCount
}

func (h *Handler) notify(chatId string, data *NotifyDto) error {
	var usersCount = h.countPeers(chatId)
	var chatNotifyDto = chatNotifyDto{}
	if data != nil {
		logger.Info("Notifying with data", "streamId", data.StreamId, "login", data.Login)
		chatNotifyDto.Data = data
	} else {
		logger.Info("Notifying without data")
	}
	chatNotifyDto.UsersCount = usersCount
	parseInt64, err2 := ParseInt64(chatId)
	if err2 != nil {
		logger.Error(err2, "Failed during parse chat id")
		return err2
	}
	chatNotifyDto.ChatId = parseInt64

	url0 := h.conf.ChatConfig.ChatUrlConfig.Base
	url1 := h.conf.ChatConfig.ChatUrlConfig.Notify

	fullUrl := fmt.Sprintf("%v%v", url0, url1)
	parsedUrl, err := url.Parse(fullUrl)
	if err != nil {
		logger.Error(err, "Failed during parse chat url")
		return err
	}

	marshal, err2 := json.Marshal(chatNotifyDto)
	if err2 != nil {
		logger.Error(err2, "Failed during marshal chatNotifyDto")
		return err2
	}

	r := ioutil.NopCloser(bytes.NewReader(marshal))
	contentType := "application/json;charset=UTF-8"

	requestHeaders := map[string][]string{
		"Accept-Encoding": {"gzip, deflate"},
		"Accept":          {contentType},
		"Content-Type":    {contentType},
	}

	req := &http.Request{
		Method: http.MethodPut,
		URL:    parsedUrl,
		Body:   r,
		Header: requestHeaders,
	}

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

type NotifyDto struct {
	PeerId    string `json:"peerId"`
	StreamId  string `json:"streamId"`
	Login     string `json:"login"`
	VideoMute bool   `json:"videoMute"`
	AudioMute bool   `json:"audioMute"`
}

func (h *Handler) NotifyChatParticipants(w http.ResponseWriter, r *http.Request) {
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
		if peerF := h.getPeerByPeerId(chatId, bodyStruct.PeerId); peerF != nil {
			h.storeToIndex(peerF, userId, bodyStruct.PeerId, bodyStruct.StreamId, bodyStruct.Login, bodyStruct.VideoMute, bodyStruct.AudioMute)
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

func (h *Handler) peerIsAlive(peer *sfu.Peer) bool {
	if peer == nil {
		return false
	}
	return peer.Publisher().SignalingState() != webrtc.SignalingStateClosed
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

func (h *Handler) checkAccess(userIdString string, chatIdString string) (bool, error) {
	url0 := h.conf.ChatConfig.ChatUrlConfig.Base
	url1 := h.conf.ChatConfig.ChatUrlConfig.Access

	response, err := h.client.Get(url0 + url1 + "?userId=" + userIdString + "&chatId=" + chatIdString)
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

func (h *Handler) Kick(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatId := vars["chatId"]
	userToKickId := r.URL.Query().Get("userId")
	var notifyBool bool = r.URL.Query().Get("notify") == "true"

	if h.kick(chatId, userToKickId, notifyBool) != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

type peerWithMetadata struct {
	*sfu.Peer
	*ExtendedPeerInfo
	*sfu.Session
}

func (h *Handler) getPeerMetadatas(chatId, userId string) []peerWithMetadata {
	session, _ := h.sfu.GetSession(fmt.Sprintf("chat%v", chatId))
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

func (h *Handler) getPeerMetadataByStreamId(chatId, streamId string) *peerWithMetadata {
	session, _ := h.sfu.GetSession(fmt.Sprintf("chat%v", chatId)) // ChatVideo.vue
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

func (h *Handler) getPeerByPeerId(chatId, peerId string) *sfu.Peer {
	session, _ := h.sfu.GetSession(fmt.Sprintf("chat%v", chatId)) // ChatVideo.vue
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

func (h *Handler) kick(chatId, userId string, notifyBool bool) error {
	logger.Info("Invoked kick", "chatId", chatId, "userId", userId, "notify", notifyBool)

	metadatas := h.getPeerMetadatas(chatId, userId)
	for _, metadata := range metadatas {
		metadata.Peer.Close()
		metadata.Session.RemovePeer(metadata.Peer.ID())
		h.notify(chatId, nil)
	}

	// send kick notification through chat's personal centrifuge channel
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
