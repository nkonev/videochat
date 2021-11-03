package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	log "github.com/pion/ion-sfu/pkg/logger"
	"github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/webrtc/v3"
	"net/http"
	"nkonev.name/video/config"
	"nkonev.name/video/dto"
	"nkonev.name/video/producer"
	"sync"
	"time"
)

var logger = log.New()

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
	streamId  string
	login     string
	videoMute bool
	audioMute bool
}
// streamId:ExtendedPeerInfo
type connectionWithData map[string]ExtendedPeerInfo
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

func (h *ExtendedService) StoreToIndex(streamId string, userId int64, login string, videoMute, audioMute bool) {
	logger.Info("Storing peer to map",  "stream_id", streamId, "user_id", userId, "login", login, "video_mute", videoMute, "audio_mute", audioMute)
	h.peerUserIdIndex.Lock()
	defer h.peerUserIdIndex.Unlock()
	h.peerUserIdIndex.connectionWithData[streamId] = ExtendedPeerInfo{
		userId:    userId,
		streamId:  streamId,
		login:     login,
		videoMute: videoMute,
		audioMute: audioMute,
	}
}

func (h *ExtendedService) RemoveFromIndex(peer0 sfu.Peer, userId int64, conn *websocket.Conn) {
	logger.Info("Removing peer from map", "peer_id", peer0.ID(), "user_id", userId)
	h.peerUserIdIndex.Lock()
	defer h.peerUserIdIndex.Unlock()
	if streamId, err := getStreamId(peer0); err != nil {
		logger.Error(err, "Unable to get streamId", "peer_id", peer0.ID(), "user_id", userId)
	} else {
		delete(h.peerUserIdIndex.connectionWithData, streamId)
	}
}

func (h *ExtendedService) getExtendedConnectionInfo(peer0 sfu.Peer) *ExtendedPeerInfo {
	h.peerUserIdIndex.RLock()
	defer h.peerUserIdIndex.RUnlock()

	if streamId, err := getStreamId(peer0); err != nil {
		logger.Error(err, "Unable to get streamId", "peer_id", peer0.ID())
		return nil
	} else {
		s, ok := h.peerUserIdIndex.connectionWithData[streamId]
		if ok {
			return &s
		} else {
			return nil
		}
	}
}

type ErrorNoAccess struct {}
func (e *ErrorNoAccess) Error() string { return "No access" }

type errorInternal struct {}
func (e *errorInternal) Error() string { return "Internal error" }

func getStreamId(peer0 sfu.Peer) (string, error) {
	if (peer0.Publisher() != nil && len(peer0.Publisher().Tracks())!=0) {
		return peer0.Publisher().Tracks()[0].StreamID(), nil
	} else {
		return "", errors.New("Peer " + peer0.ID() + " has no stream id")
	}
}

func (h *ExtendedService) UserByStreamId(chatId int64, interestingStreamId string, includeOtherStreamIds bool, behalfUserId int64) (*dto.StoreNotifyDto, []string, error) {
	if ok, err := h.CheckAccess(behalfUserId, chatId); err != nil {
		return nil, nil, &errorInternal{}
	} else if !ok {
		return nil, nil, &ErrorNoAccess{}
	}

	var sessionInfoDto *dto.StoreNotifyDto
	var otherStreamIds = []string{}

	session := h.getSessionWithoutCreatingAnew(chatId)
	if session != nil {
		for _, peer := range session.Peers() {
			if h.peerIsAlive(peer) {

				eci := h.getExtendedConnectionInfo(peer)

				if eci != nil {
					if interestingStreamId == eci.streamId {
						sessionInfoDto = &dto.StoreNotifyDto{
							StreamId:  eci.streamId,
							Login:     eci.login,
							VideoMute: eci.videoMute,
							AudioMute: eci.audioMute,
							UserId:	   eci.userId,
						}
					} else if includeOtherStreamIds {
						otherStreamIds = append(otherStreamIds, eci.streamId)
					}
				}
			}
		}
	}
	return sessionInfoDto, otherStreamIds, nil
}

// sent to chat through RabbitMQ
type chatNotifyDto struct {
	Data       *dto.StoreNotifyDto `json:"data"`
	UsersCount int64           `json:"usersCount"`
	ChatId     int64           `json:"chatId"`
}

func (h *ExtendedService) getSessionWithoutCreatingAnew(chatId int64) sfu.Session {
	sessionName := fmt.Sprintf("chat%v", chatId)
	for _, aSession := range h.sfu.GetSessions() {
		if aSession.ID() == sessionName {
			return aSession
		}
	}
	return nil
}


func (h *ExtendedService) CountPeers(chatId int64) int64 {
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

func (h *ExtendedService) Notify(chatId int64, data *dto.StoreNotifyDto) error {
	var usersCount = h.CountPeers(chatId)
	var chatNotifyDto = chatNotifyDto{}
	if data != nil {
		logger.V(3).Info("Notifying with data", "chat_id", chatId, "stream_id", data.StreamId, "login", data.Login)
		chatNotifyDto.Data = data
	} else {
		logger.V(3).Info("Notifying without data", "chat_id", chatId)
	}
	chatNotifyDto.UsersCount = usersCount
	chatNotifyDto.ChatId = chatId

	marshal, err := json.Marshal(chatNotifyDto)
	if err != nil {
		logger.Error(err, "Failed during marshal chatNotifyDto")
		return err
	}

	return h.rabbitMqPublisher.Publish(marshal)
}

func (h *ExtendedService) peerIsAlive(peer sfu.Peer) bool {
	if peer == nil {
		return false
	}
	return peer.Publisher().SignalingState() != webrtc.SignalingStateClosed
}

func (h *ExtendedService) CheckAccess(userId int64, chatId int64) (bool, error) {
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

func (h *ExtendedService) GetPeersByChatId(chatId int64) ([]*dto.StoreNotifyDto, error) {
	var result []*dto.StoreNotifyDto = []*dto.StoreNotifyDto{}

	metadatas := h.getPeerMetadatas(chatId)
	for _, md := range metadatas {
		result = append(result, &dto.StoreNotifyDto{
			StreamId: md.streamId,
			Login: md.login,
			VideoMute: md.videoMute,
			AudioMute: md.audioMute,
			UserId: md.userId,
		})
	}

	return result, nil
}

func (h *ExtendedService) getPeerMetadatas(chatId int64) []*ExtendedPeerInfo {
	session := h.getSessionWithoutCreatingAnew(chatId)
	var result []*ExtendedPeerInfo = []*ExtendedPeerInfo{}
	if session == nil {
		return result
	}
	for _, peerF := range session.Peers() {
		if eci := h.getExtendedConnectionInfo(peerF); eci != nil {
			result = append(result, eci)
		}
	}
	return result
}

type peerWithMetadata struct {
	sfu.Peer
	*ExtendedPeerInfo
	sfu.Session
}

func (h *ExtendedService) getPeerMetadatasForKick(chatId, userId int64) []peerWithMetadata {
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

func (h *ExtendedService) ExistsPeerByStreamId(chatId int64, streamId string) bool {
	session := h.getSessionWithoutCreatingAnew(chatId) // ChatVideo.vue
	if session == nil {
		return false
	}
	h.peerUserIdIndex.RLock()
	defer h.peerUserIdIndex.RUnlock()
	_, ok := h.peerUserIdIndex.connectionWithData[streamId]
	return ok
}

func (h *ExtendedService) KickUser(chatId, userId int64, silent bool) error {
	logger.Info("Invoked kick", "chat_id", chatId, "user_id", userId)

	metadatas := h.getPeerMetadatasForKick(chatId, userId)
	for _, metadata := range metadatas {
		metadata.Peer.Close()
		metadata.Session.RemovePeer(metadata.Peer)
		if !silent {
			h.Notify(chatId, nil)
		}
	}

	return nil
}

func (h *ExtendedService) NotifyAboutLeaving(chatId int64) {
	if err := h.Notify(chatId, nil); err != nil {
		logger.Error(err, "error during sending leave notification")
	} else {
		logger.Info("Successfully sent notification about leaving")
	}
}

func (h *ExtendedService) notifyAllChats() {
	for _, session := range h.sfu.GetSessions() {
		sessionName := session.ID()
		var chatId int64
		if _, err := fmt.Sscanf(sessionName, "chat%d", &chatId); err != nil {
			logger.Error(err, "error during reading chat id from session", "sessionName", sessionName)
		} else {
			if err = h.Notify(chatId, nil); err != nil {
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
				logger.V(3).Info("Invoked chats periodic notificator")
				h.notifyAllChats()
			case <- quit:
				ticker.Stop()
				return
			}
		}
	}()
	return &quit
}

