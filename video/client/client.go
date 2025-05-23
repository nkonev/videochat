package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"io/ioutil"
	"net/http"
	"net/url"
	"nkonev.name/video/config"
	"nkonev.name/video/dto"
	"nkonev.name/video/logger"
	"nkonev.name/video/utils"
	"strings"
)

type RestClient struct {
	client                          *http.Client
	chatBaseUrl                     string
	accessPath                      string
	isAdminPath                     string
	doesParticipantBelongToChatPath string
	chatParticipantIdsPath          string
	chatInviteNamePath              string
	chatBasicInfoPath               string
	aaaBaseUrl                      string
	aaaGetUsersUrl                  string
	storageBaseUrl                  string
	storageS3Path                   string
	tracer                          trace.Tracer
	lgr                             *logger.Logger
}

func NewRestClient(config *config.ExtendedConfig, lgr *logger.Logger) *RestClient {
	tr := &http.Transport{
		MaxIdleConns:       config.RestClientConfig.MaxIdleConns,
		IdleConnTimeout:    config.RestClientConfig.IdleConnTimeout,
		DisableCompression: config.RestClientConfig.DisableCompression,
	}
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	trR := otelhttp.NewTransport(tr)
	client := &http.Client{Transport: trR}
	trcr := otel.Tracer("rest/client")

	return &RestClient{
		client:                          client,
		chatBaseUrl:                     config.ChatConfig.ChatUrlConfig.Base,
		accessPath:                      config.ChatConfig.ChatUrlConfig.Access,
		isAdminPath:                     config.ChatConfig.ChatUrlConfig.IsChatAdmin,
		doesParticipantBelongToChatPath: config.ChatConfig.ChatUrlConfig.DoesParticipantBelongToChat,
		chatParticipantIdsPath:          config.ChatConfig.ChatUrlConfig.ChatParticipantIds,
		chatInviteNamePath:              config.ChatConfig.ChatUrlConfig.ChatInviteName,
		chatBasicInfoPath:               config.ChatConfig.ChatUrlConfig.ChatBasicInfoPath,
		aaaBaseUrl:                      config.AaaConfig.AaaUrlConfig.Base,
		aaaGetUsersUrl:                  config.AaaConfig.AaaUrlConfig.GetUsers,
		storageBaseUrl:                  config.StorageConfig.StorageUrlConfig.Base,
		storageS3Path:                   config.StorageConfig.StorageUrlConfig.S3,
		tracer:                          trcr,
		lgr:                             lgr,
	}
}

func (h *RestClient) CheckAccess(c context.Context, userId int64, chatId int64) (bool, error) {
	url := fmt.Sprintf("%v%v?userId=%v&chatId=%v", h.chatBaseUrl, h.accessPath, userId, chatId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		h.lgr.WithTracing(c).Errorw("Error during create GET", err)
		return false, err
	}

	ctx, span := h.tracer.Start(c, "access.Check")
	defer span.End()
	req = req.WithContext(ctx)

	response, err := h.client.Do(req)
	if err != nil {
		h.lgr.WithTracing(c).Errorw("Transport error during checking access", err)
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		return true, nil
	} else if response.StatusCode == http.StatusUnauthorized {
		return false, nil
	} else {
		err := errors.New("Unexpected status on checkAccess")
		h.lgr.WithTracing(c).Errorw("Unexpected status on checkAccess", err, "httpCode", response.StatusCode)
		return false, err
	}
}

func (h *RestClient) IsAdmin(c context.Context, userId int64, chatId int64) (bool, error) {
	url := fmt.Sprintf("%v%v?userId=%v&chatId=%v", h.chatBaseUrl, h.isAdminPath, userId, chatId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		h.lgr.WithTracing(c).Errorw("Error during create GET", err)
		return false, err
	}

	ctx, span := h.tracer.Start(c, "chat.IsAdmin")
	defer span.End()
	req = req.WithContext(ctx)

	response, err := h.client.Do(req)
	if err != nil {
		h.lgr.WithTracing(c).Errorw("Transport error during checking access", err)
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		return true, nil
	} else if response.StatusCode == http.StatusUnauthorized {
		return false, nil
	} else {
		err := errors.New("Unexpected status on checkAccess")
		h.lgr.WithTracing(c).Errorw("Unexpected status on checkAccess", err, "httpCode", response.StatusCode)
		return false, err
	}
}

func (h *RestClient) GetUsers(c context.Context, userIds []int64) ([]*dto.User, error) {
	contentType := "application/json;charset=UTF-8"
	fullUrl := h.aaaBaseUrl + h.aaaGetUsersUrl

	var userIdsString []string
	for _, userIdInt := range userIds {
		userIdsString = append(userIdsString, utils.Int64ToString(userIdInt))
	}

	join := strings.Join(userIdsString, ",")

	requestHeaders := map[string][]string{
		"Accept-Encoding": {"gzip, deflate"},
		"Accept":          {contentType},
		"Content-Type":    {contentType},
	}

	parsedUrl, err := url.Parse(fullUrl + "?userId=" + join)
	if err != nil {
		h.lgr.WithTracing(c).Errorln("Failed during parse aaa url:", err)
		return nil, err
	}
	request := &http.Request{
		Method: "GET",
		Header: requestHeaders,
		URL:    parsedUrl,
	}

	ctx, span := h.tracer.Start(c, "chat.GetUsers")
	defer span.End()
	request = request.WithContext(ctx)

	resp, err := h.client.Do(request)
	if err != nil {
		h.lgr.WithTracing(c).Warnln("Failed to request get users response:", err)
		return nil, err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code != 200 {
		h.lgr.WithTracing(c).Warnln("Users response responded non-200 code: ", code)
		return nil, err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		h.lgr.WithTracing(c).Errorln("Failed to decode get users response:", err)
		return nil, err
	}

	users := &[]*dto.User{}
	if err := json.Unmarshal(bodyBytes, users); err != nil {
		h.lgr.WithTracing(c).Errorln("Failed to parse users:", err)
		return nil, err
	}
	return *users, nil
}

func (h *RestClient) DoesParticipantBelongToChat(c context.Context, chatId int64, userIds []int64) ([]*dto.ParticipantBelongsToChat, error) {

	if len(userIds) == 0 {
		return make([]*dto.ParticipantBelongsToChat, 0), nil
	}

	contentType := "application/json;charset=UTF-8"
	fullUrl := h.chatBaseUrl + h.doesParticipantBelongToChatPath

	var userIdsString []string
	for _, userIdInt := range userIds {
		userIdsString = append(userIdsString, utils.Int64ToString(userIdInt))
	}

	join := strings.Join(userIdsString, ",")

	requestHeaders := map[string][]string{
		"Accept-Encoding": {"gzip, deflate"},
		"Accept":          {contentType},
		"Content-Type":    {contentType},
	}

	parsedUrl, err := url.Parse(fullUrl + "?userId=" + join + "&chatId=" + fmt.Sprintf("%v", chatId))
	if err != nil {
		h.lgr.WithTracing(c).Errorln("Failed during parse chat url:", err)
		return nil, err
	}
	request := &http.Request{
		Method: "GET",
		Header: requestHeaders,
		URL:    parsedUrl,
	}

	ctx, span := h.tracer.Start(c, "chat.DoesParticipantBelongToChat")
	defer span.End()
	request = request.WithContext(ctx)

	resp, err := h.client.Do(request)
	if err != nil {
		h.lgr.WithTracing(c).Warnln("Failed to request DoesParticipantBelongToChat response:", err)
		return nil, err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code != 200 {
		h.lgr.WithTracing(c).Warnln("DoesParticipantBelongToChat response responded non-200 code: ", code)
		return nil, err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		h.lgr.WithTracing(c).Errorln("Failed to decode DoesParticipantBelongToChat response:", err)
		return nil, err
	}

	users := &dto.ParticipantsBelongToChat{}
	if err := json.Unmarshal(bodyBytes, users); err != nil {
		h.lgr.WithTracing(c).Errorln("Failed to parse DoesParticipantBelongToChat:", err)
		return nil, err
	}
	return users.Users, nil
}

func (h *RestClient) GetChatParticipantIdsByPage(c context.Context, chatId int64, page, size int) ([]int64, error) {
	contentType := "application/json;charset=UTF-8"
	fullUrl := h.chatBaseUrl + h.chatParticipantIdsPath

	requestHeaders := map[string][]string{
		"Accept-Encoding": {"gzip, deflate"},
		"Accept":          {contentType},
		"Content-Type":    {contentType},
	}

	parsedUrl, err := url.Parse(fullUrl + "?chatId=" + utils.Int64ToString(chatId) + "&page=" + utils.IntToString(page) + "&size=" + utils.IntToString(size))
	if err != nil {
		h.lgr.WithTracing(c).Errorln("Failed during parse chat participant ids url:", err)
		return nil, err
	}
	request := &http.Request{
		Method: "GET",
		Header: requestHeaders,
		URL:    parsedUrl,
	}

	ctx, span := h.tracer.Start(c, "chat.GetParticipantIds")
	defer span.End()
	request = request.WithContext(ctx)

	resp, err := h.client.Do(request)
	if err != nil {
		h.lgr.WithTracing(c).Warnln("Failed to request chat participant ids response:", err)
		return nil, err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code != 200 {
		h.lgr.WithTracing(c).Warnln("Chat response responded non-200 code: ", code)
		return nil, err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		h.lgr.WithTracing(c).Errorln("Failed to decode chat participant ids response:", err)
		return nil, err
	}

	userIds := new([]int64)
	if err := json.Unmarshal(bodyBytes, userIds); err != nil {
		h.lgr.WithTracing(c).Errorln("Failed to parse chat participant ids:", err)
		return nil, err
	}
	return *userIds, nil
}

func (h *RestClient) GetChatParticipantIds(c context.Context, chatId int64, consumer func(participantIds []int64) error) error {
	var lastError error
	shouldContinue := true
	for page := 0; shouldContinue; page++ {
		portion, err := h.GetChatParticipantIdsByPage(c, chatId, page, utils.DefaultSize)
		if len(portion) < utils.DefaultSize {
			shouldContinue = false
		}
		if err != nil {
			h.lgr.WithTracing(c).Warnf("got error %v", err)
			lastError = err
			continue
		}
		err = consumer(portion)
		if err != nil {
			h.lgr.WithTracing(c).Errorf("Got error during invoking consumer portion %v", err)
			lastError = err
			continue
		}
	}
	return lastError
}

func (h *RestClient) GetChatNameForInvite(c context.Context, chatId int64, behalfUserId int64, participantIds []int64) ([]*dto.ChatName, error) {
	contentType := "application/json;charset=UTF-8"
	fullUrl := h.chatBaseUrl + h.chatInviteNamePath

	requestHeaders := map[string][]string{
		"Accept-Encoding": {"gzip, deflate"},
		"Accept":          {contentType},
		"Content-Type":    {contentType},
	}

	var userIdsString []string
	for _, userIdInt := range participantIds {
		userIdsString = append(userIdsString, utils.Int64ToString(userIdInt))
	}

	joinedParticipantIds := strings.Join(userIdsString, ",")

	parsedUrl, err := url.Parse(fullUrl + "?chatId=" + utils.Int64ToString(chatId) + "&behalfUserId=" + utils.Int64ToString(behalfUserId) + "&userIds=" + joinedParticipantIds)
	if err != nil {
		h.lgr.WithTracing(c).Errorln("Failed during parse name for invite url:", err)
		return nil, err
	}
	request := &http.Request{
		Method: "GET",
		Header: requestHeaders,
		URL:    parsedUrl,
	}

	ctx, span := h.tracer.Start(c, "chat.GetNameForInvite")
	defer span.End()
	request = request.WithContext(ctx)

	resp, err := h.client.Do(request)
	if err != nil {
		h.lgr.WithTracing(c).Warnln("Failed to request name for invite response:", err)
		return nil, err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code != 200 {
		h.lgr.WithTracing(c).Warnln("Chat name for invite response responded non-200 code: ", code)
		return nil, err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		h.lgr.WithTracing(c).Errorln("Failed to decode name for invite response:", err)
		return nil, err
	}

	ret := new([]*dto.ChatName)
	if err := json.Unmarshal(bodyBytes, ret); err != nil {
		h.lgr.WithTracing(c).Errorln("Failed to parse name for invite:", err)
		return nil, err
	}
	return *ret, nil
}

type S3Request struct {
	FileName string `json:"fileName"`
	ChatId   int64  `json:"chatId"`
	OwnerId  int64  `json:"ownerId"`
}

func (h *RestClient) GetS3(c context.Context, filename string, chatId int64, userId int64) (*dto.S3Response, error) {
	contentType := "application/json;charset=UTF-8"
	fullUrl := h.storageBaseUrl + h.storageS3Path

	requestHeaders := map[string][]string{
		"Accept-Encoding": {"gzip, deflate"},
		"Accept":          {contentType},
		"Content-Type":    {contentType},
	}

	parsedUrl, err := url.Parse(fullUrl)
	if err != nil {
		h.lgr.WithTracing(c).Errorln("Failed during parse storage s3 url:", err)
		return nil, err
	}

	req := S3Request{
		FileName: filename,
		ChatId:   chatId,
		OwnerId:  userId,
	}

	bytesData, err := json.Marshal(req)
	if err != nil {
		h.lgr.WithTracing(c).Errorln("Failed during marshalling:", err)
		return nil, err
	}
	reader := bytes.NewReader(bytesData)

	nopCloser := ioutil.NopCloser(reader)

	request := &http.Request{
		Method: "POST",
		Header: requestHeaders,
		URL:    parsedUrl,
		Body:   nopCloser,
	}

	ctx, span := h.tracer.Start(c, "storage.GetS3")
	defer span.End()
	request = request.WithContext(ctx)

	resp, err := h.client.Do(request)
	if err != nil {
		h.lgr.WithTracing(c).Warnln("Failed to request s3 response:", err)
		return nil, err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code != 200 {
		h.lgr.WithTracing(c).Warnln("Chat name for s3 response responded non-200 code: ", code)
		return nil, err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		h.lgr.WithTracing(c).Errorln("Failed to decode s3 response:", err)
		return nil, err
	}

	ret := new(dto.S3Response)
	if err := json.Unmarshal(bodyBytes, ret); err != nil {
		h.lgr.WithTracing(c).Errorln("Failed to parse s3:", err)
		return nil, err
	}
	return ret, nil
}

func (h *RestClient) GetBasicChatInfo(c context.Context, chatId int64, userId int64) (*dto.BasicChatDto, error) {
	contentType := "application/json;charset=UTF-8"
	fullUrl := h.chatBaseUrl + h.chatBasicInfoPath + "/" + utils.Int64ToString(chatId)

	requestHeaders := map[string][]string{
		"Accept-Encoding": {"gzip, deflate"},
		"Accept":          {contentType},
		"Content-Type":    {contentType},
	}

	parsedUrl, err := url.Parse(fullUrl)
	if err != nil {
		h.lgr.WithTracing(c).Errorln("Failed during parse BasicChatInfo for invite url:", err)
		return nil, err
	}
	request := &http.Request{
		Method: "GET",
		Header: requestHeaders,
		URL:    parsedUrl,
	}

	ctx, span := h.tracer.Start(c, "chat.GetBasicChatInfo")
	defer span.End()
	request = request.WithContext(ctx)

	resp, err := h.client.Do(request)
	if err != nil {
		h.lgr.WithTracing(c).Warnln("Failed to request BasicChatInfo for invite response:", err)
		return nil, err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code != 200 {
		h.lgr.WithTracing(c).Warnln("Chat BasicChatInfo for invite response responded non-200 code: ", code)
		return nil, err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		h.lgr.WithTracing(c).Errorln("Failed to decode BasicChatInfo for invite response:", err)
		return nil, err
	}

	ret := new(dto.BasicChatDto)
	if err := json.Unmarshal(bodyBytes, ret); err != nil {
		h.lgr.WithTracing(c).Errorln("Failed to parse BasicChatInfo for invite:", err)
		return nil, err
	}
	return ret, nil
}

func (h *RestClient) CloseIdleConnections() {
	h.client.CloseIdleConnections()
}

func (h *RestClient) GetClient() *http.Client {
	return h.client
}
