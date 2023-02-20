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
	. "nkonev.name/video/logger"
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
}

func NewRestClient(config *config.ExtendedConfig) *RestClient {
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
	}
}

func (h *RestClient) CheckAccess(userId int64, chatId int64, c context.Context) (bool, error) {
	url := fmt.Sprintf("%v%v?userId=%v&chatId=%v", h.chatBaseUrl, h.accessPath, userId, chatId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		GetLogEntry(c).Error(err, "Error during create GET")
		return false, err
	}

	ctx, span := h.tracer.Start(c, "access.Check")
	defer span.End()
	req = req.WithContext(ctx)

	response, err := h.client.Do(req)
	if err != nil {
		GetLogEntry(c).Error(err, "Transport error during checking access")
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		return true, nil
	} else if response.StatusCode == http.StatusUnauthorized {
		return false, nil
	} else {
		err := errors.New("Unexpected status on checkAccess")
		GetLogEntry(c).Error(err, "Unexpected status on checkAccess", "httpCode", response.StatusCode)
		return false, err
	}
}

func (h *RestClient) IsAdmin(userId int64, chatId int64, c context.Context) (bool, error) {
	url := fmt.Sprintf("%v%v?userId=%v&chatId=%v", h.chatBaseUrl, h.isAdminPath, userId, chatId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		GetLogEntry(c).Error(err, "Error during create GET")
		return false, err
	}

	ctx, span := h.tracer.Start(c, "chat.IsAdmin")
	defer span.End()
	req = req.WithContext(ctx)

	response, err := h.client.Do(req)
	if err != nil {
		GetLogEntry(c).Error(err, "Transport error during checking access")
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		return true, nil
	} else if response.StatusCode == http.StatusUnauthorized {
		return false, nil
	} else {
		err := errors.New("Unexpected status on checkAccess")
		GetLogEntry(c).Error(err, "Unexpected status on checkAccess", "httpCode", response.StatusCode)
		return false, err
	}
}

func (h *RestClient) GetUsers(userIds []int64, c context.Context) ([]*dto.User, error) {
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
		GetLogEntry(c).Errorln("Failed during parse aaa url:", err)
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
		GetLogEntry(c).Warningln("Failed to request get users response:", err)
		return nil, err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code != 200 {
		GetLogEntry(c).Warningln("Users response responded non-200 code: ", code)
		return nil, err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		GetLogEntry(c).Errorln("Failed to decode get users response:", err)
		return nil, err
	}

	users := &[]*dto.User{}
	if err := json.Unmarshal(bodyBytes, users); err != nil {
		GetLogEntry(c).Errorln("Failed to parse users:", err)
		return nil, err
	}
	return *users, nil
}

func (h *RestClient) DoesParticipantBelongToChat(chatId int64, userIds []int64, c context.Context) ([]*dto.ParticipantBelongsToChat, error) {
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
		GetLogEntry(c).Errorln("Failed during parse chat url:", err)
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
		GetLogEntry(c).Warningln("Failed to request DoesParticipantBelongToChat response:", err)
		return nil, err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code != 200 {
		GetLogEntry(c).Warningln("DoesParticipantBelongToChat response responded non-200 code: ", code)
		return nil, err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		GetLogEntry(c).Errorln("Failed to decode DoesParticipantBelongToChat response:", err)
		return nil, err
	}

	users := &dto.ParticipantsBelongToChat{}
	if err := json.Unmarshal(bodyBytes, users); err != nil {
		GetLogEntry(c).Errorln("Failed to parse DoesParticipantBelongToChat:", err)
		return nil, err
	}
	return users.Users, nil
}

func (h *RestClient) GetChatParticipantIds(chatId int64, c context.Context) ([]int64, error) {
	contentType := "application/json;charset=UTF-8"
	fullUrl := h.chatBaseUrl + h.chatParticipantIdsPath

	requestHeaders := map[string][]string{
		"Accept-Encoding": {"gzip, deflate"},
		"Accept":          {contentType},
		"Content-Type":    {contentType},
	}

	parsedUrl, err := url.Parse(fullUrl + "?chatId=" + utils.Int64ToString(chatId))
	if err != nil {
		GetLogEntry(c).Errorln("Failed during parse chat participant ids url:", err)
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
		GetLogEntry(c).Warningln("Failed to request chat participant ids response:", err)
		return nil, err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code != 200 {
		GetLogEntry(c).Warningln("Chat response responded non-200 code: ", code)
		return nil, err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		GetLogEntry(c).Errorln("Failed to decode chat participant ids response:", err)
		return nil, err
	}

	userIds := new([]int64)
	if err := json.Unmarshal(bodyBytes, userIds); err != nil {
		GetLogEntry(c).Errorln("Failed to parse chat participant ids:", err)
		return nil, err
	}
	return *userIds, nil
}

func (h *RestClient) GetChatNameForInvite(chatId int64, behalfUserId int64, participantIds []int64, c context.Context) ([]*dto.ChatName, error) {
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
		GetLogEntry(c).Errorln("Failed during parse name for invite url:", err)
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
		GetLogEntry(c).Warningln("Failed to request name for invite response:", err)
		return nil, err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code != 200 {
		GetLogEntry(c).Warningln("Chat name for invite response responded non-200 code: ", code)
		return nil, err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		GetLogEntry(c).Errorln("Failed to decode name for invite response:", err)
		return nil, err
	}

	ret := new([]*dto.ChatName)
	if err := json.Unmarshal(bodyBytes, ret); err != nil {
		GetLogEntry(c).Errorln("Failed to parse name for invite:", err)
		return nil, err
	}
	return *ret, nil
}

type S3Request struct {
	FileName string `json:"fileName"`
	ChatId   int64  `json:"chatId"`
	OwnerId  int64  `json:"ownerId"`
}

func (h *RestClient) GetS3(filename string, chatId int64, userId int64, c context.Context) (*dto.S3Response, error) {
	contentType := "application/json;charset=UTF-8"
	fullUrl := h.storageBaseUrl + h.storageS3Path

	requestHeaders := map[string][]string{
		"Accept-Encoding": {"gzip, deflate"},
		"Accept":          {contentType},
		"Content-Type":    {contentType},
	}

	parsedUrl, err := url.Parse(fullUrl)
	if err != nil {
		GetLogEntry(c).Errorln("Failed during parse storage s3 url:", err)
		return nil, err
	}

	req := S3Request{
		FileName: filename,
		ChatId:   chatId,
		OwnerId:  userId,
	}

	bytesData, err := json.Marshal(req)
	if err != nil {
		GetLogEntry(c).Errorln("Failed during marshalling:", err)
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
		GetLogEntry(c).Warningln("Failed to request s3 response:", err)
		return nil, err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code != 200 {
		GetLogEntry(c).Warningln("Chat name for s3 response responded non-200 code: ", code)
		return nil, err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		GetLogEntry(c).Errorln("Failed to decode s3 response:", err)
		return nil, err
	}

	ret := new(dto.S3Response)
	if err := json.Unmarshal(bodyBytes, ret); err != nil {
		GetLogEntry(c).Errorln("Failed to parse s3:", err)
		return nil, err
	}
	return ret, nil
}

func (h *RestClient) GetBasicChatInfo(chatId int64, userId int64, c context.Context) (*dto.BasicChatDto, error) {
	contentType := "application/json;charset=UTF-8"
	fullUrl := h.chatBaseUrl + h.chatBasicInfoPath + "/" + utils.Int64ToString(chatId) + "?userId=" + utils.Int64ToString(userId)

	requestHeaders := map[string][]string{
		"Accept-Encoding": {"gzip, deflate"},
		"Accept":          {contentType},
		"Content-Type":    {contentType},
	}

	parsedUrl, err := url.Parse(fullUrl)
	if err != nil {
		GetLogEntry(c).Errorln("Failed during parse BasicChatInfo for invite url:", err)
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
		GetLogEntry(c).Warningln("Failed to request BasicChatInfo for invite response:", err)
		return nil, err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code != 200 {
		GetLogEntry(c).Warningln("Chat BasicChatInfo for invite response responded non-200 code: ", code)
		return nil, err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		GetLogEntry(c).Errorln("Failed to decode BasicChatInfo for invite response:", err)
		return nil, err
	}

	ret := new(dto.BasicChatDto)
	if err := json.Unmarshal(bodyBytes, ret); err != nil {
		GetLogEntry(c).Errorln("Failed to parse BasicChatInfo for invite:", err)
		return nil, err
	}
	return ret, nil
}
