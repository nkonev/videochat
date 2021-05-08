package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/webrtc/v3"
	"net/http"
	"nkonev.name/video/config"
	"nkonev.name/video/producer"
	"strconv"
	"sync"
	"github.com/gorilla/websocket"
	"time"
)

type ExtendedService struct {
	sfu             *sfu.SFU
	peerUserIdIndex connectionsLockableMap
	rabbitMqPublisher *producer.RabbitPublisher
	conf            *config.ExtendedConfig
	client          *http.Client
}

type ExtendedPeerInfo struct {
	userId int64
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

func NewExtendedService(
	sfu *sfu.SFU,
	conf *config.ExtendedConfig,
	rabbitMqPublisher *producer.RabbitPublisher,
	client *http.Client,
) ExtendedService {
	handler := ExtendedService{
		sfu:      sfu,
		conf:     conf,
		peerUserIdIndex: connectionsLockableMap{
			RWMutex:            sync.RWMutex{},
			connectionWithData: connectionWithData{},
		},
		rabbitMqPublisher: rabbitMqPublisher,
		client:   client,
	}
	return handler
}

func (h *ExtendedService) storeToIndex(peer0 *sfu.Peer, userId int64, peerId, streamId, login string, videoMute, audioMute bool) {
	logger.Info("Storing peer to map", "peer_id", peer0.ID(), "userId", userId, "streamId", streamId, "login", login)
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

func (h *ExtendedService) removeFromIndex(peer0 *sfu.Peer, userId int64, conn *websocket.Conn) {
	logger.Info("Removing peer from map", "peer_id", peer0.ID(), "userId", userId)
	h.peerUserIdIndex.Lock()
	defer h.peerUserIdIndex.Unlock()
	conn.Close()
	delete(h.peerUserIdIndex.connectionWithData, peer0)
}

func (h *ExtendedService) getExtendedConnectionInfo(peer0 *sfu.Peer) *ExtendedPeerInfo {
	h.peerUserIdIndex.RLock()
	defer h.peerUserIdIndex.RUnlock()
	s, ok := h.peerUserIdIndex.connectionWithData[peer0]
	if ok {
		return &s
	} else {
		return nil
	}
}

type errorNoAccess struct {}
func (e *errorNoAccess) Error() string { return "No access" }

type errorInternal struct {}
func (e *errorInternal) Error() string { return "Internal error" }

func (h *ExtendedService) userByStreamId(chatId int64, interestingStreamId string, behalfUserId int64) (*StoreNotifyDto, error) {
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
					d := StoreNotifyDto{
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

// sent to chat through RabbitMQ
type chatNotifyDto struct {
	Data       *StoreNotifyDto `json:"data"`
	UsersCount int64           `json:"usersCount"`
	ChatId     int64           `json:"chatId"`
}

func ParseInt64(s string) (int64, error) {
	if i, err := strconv.ParseInt(s, 10, 64); err != nil {
		return 0, err
	} else {
		return i, nil
	}
}

func (h *ExtendedService) getSessionWithoutCreatingAnew(chatId int64) *sfu.Session {
	sessionName := fmt.Sprintf("chat%v", chatId)
	if session, ok := h.sfu.GetSessions()[sessionName]; ok {
		return session
	} else {
		return nil
	}
}


func (h *ExtendedService) countPeers(chatId int64) int64 {
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

func (h *ExtendedService) notify(chatId int64, data *StoreNotifyDto) error {
	var usersCount = h.countPeers(chatId)
	var chatNotifyDto = chatNotifyDto{}
	if data != nil {
		logger.Info("Notifying with data", "chatId", chatId, "streamId", data.StreamId, "login", data.Login)
		chatNotifyDto.Data = data
	} else {
		logger.Info("Notifying without data", "chatId", chatId)
	}
	chatNotifyDto.UsersCount = usersCount
	chatNotifyDto.ChatId = chatId

	marshal, err2 := json.Marshal(chatNotifyDto)
	if err2 != nil {
		logger.Error(err2, "Failed during marshal chatNotifyDto")
		return err2
	}

	return h.rabbitMqPublisher.Publish(marshal)
}

func (h *ExtendedService) peerIsAlive(peer *sfu.Peer) bool {
	if peer == nil {
		return false
	}
	return peer.Publisher().SignalingState() != webrtc.SignalingStateClosed
}

func (h *ExtendedService) checkAccess(userId int64, chatId int64) (bool, error) {
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

type peerWithMetadata struct {
	*sfu.Peer
	*ExtendedPeerInfo
	*sfu.Session
}

func (h *ExtendedService) getPeerMetadatas(chatId, userId int64) []peerWithMetadata {
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


func (h *ExtendedService) getPeerMetadataByStreamId(chatId int64, streamId string) *peerWithMetadata {
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

func (h *ExtendedService) getPeerByPeerId(chatId int64, peerId string) *sfu.Peer {
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

func (h *ExtendedService) KickUser(chatId, userId int64) error {
	logger.Info("Invoked kick", "chatId", chatId, "userId", userId)

	metadatas := h.getPeerMetadatas(chatId, userId)
	for _, metadata := range metadatas {
		metadata.Peer.Close()
		metadata.Session.RemovePeer(metadata.Peer.ID())
		h.notify(chatId, nil)
	}

	return nil
}

func (h *ExtendedService) notifyAboutLeaving(chatId int64) {
	if err := h.notify(chatId, nil); err != nil {
		logger.Error(err, "error during sending leave notification")
	} else {
		logger.Info("Successfully sent notification about leaving")
	}
}

func (h *ExtendedService) notifyAllChats() {
	for sessionName, _ := range h.sfu.GetSessions() {
		var chatId int64
		if _, err := fmt.Sscanf(sessionName, "chat%d", &chatId); err != nil {
			logger.Error(err, "error during reading chat id from session", "sessionName", sessionName)
		} else {
			if err = h.notify(chatId, nil); err != nil {
				logger.Error(err, "error during sending periodic notification")
			}
		}
	}
}

func (h *ExtendedService) Schedule() *chan struct{} {
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

