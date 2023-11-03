package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"io/ioutil"
	"net/http"
	"net/url"
	"nkonev.name/storage/dto"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/utils"
	"strings"
)

type RestClient struct {
	client                 *http.Client
	baseUrl                string
	accessPath             string
	removeFileItemPath     string
	aaaBaseUrl             string
	aaaGetUsersUrl         string
	checkChatExistsPath    string
	chatParticipantIdsPath string
	tracer                 trace.Tracer
}

func newRestClient() *http.Client {
	tr := &http.Transport{
		MaxIdleConns:       viper.GetInt("http.maxIdleConns"),
		IdleConnTimeout:    viper.GetDuration("http.idleConnTimeout"),
		DisableCompression: viper.GetBool("http.disableCompression"),
	}
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	trR := otelhttp.NewTransport(tr)
	client := &http.Client{Transport: trR}
	return client
}

func NewChatAccessClient() *RestClient {
	client := newRestClient()
	trcr := otel.Tracer("rest/client")

	return &RestClient{
		client:                 client,
		baseUrl:                viper.GetString("chat.url.base"),
		accessPath:             viper.GetString("chat.url.access"),
		removeFileItemPath:     viper.GetString("chat.url.removeFileItem"),
		aaaBaseUrl:             viper.GetString("aaa.url.base"),
		aaaGetUsersUrl:         viper.GetString("aaa.url.getUsers"),
		checkChatExistsPath:    viper.GetString("chat.url.checkChatExistsPath"),
		chatParticipantIdsPath: viper.GetString("chat.url.chatParticipants"),
		tracer:                 trcr,
	}
}

func (h *RestClient) CheckAccess(userId *int64, chatId int64, c context.Context) (bool, error) {
	builder := fmt.Sprintf("%v%v?chatId=%v&considerCanResend=true", h.baseUrl, h.accessPath, chatId)
	if userId != nil {
		builder += "&userId=" + utils.Int64ToString(*userId)
	}
	url0 := builder

	req, err := http.NewRequest("GET", url0, nil)
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

func (h *RestClient) RemoveFileItem(chatId int64, fileItemUuid string, userId int64, c context.Context) {
	fullUrl := fmt.Sprintf("%v%v?chatId=%v&fileItemUuid=%v&userId=%v", h.baseUrl, h.removeFileItemPath, chatId, fileItemUuid, userId)

	parsedUrl, err := url.Parse(fullUrl)
	if err != nil {
		GetLogEntry(c).Errorln("Failed during parse chat url:", err)
		return
	}

	request := &http.Request{
		Method: "DELETE",
		URL:    parsedUrl,
	}

	ctx, span := h.tracer.Start(c, "fileItem.Remove")
	defer span.End()
	request = request.WithContext(ctx)

	response, err := h.client.Do(request)
	if err != nil {
		GetLogEntry(c).Errorf("Transport error during removing file item %v", err)
		return
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		return
	} else {
		GetLogEntry(c).Errorf("Unexpected status on removing file item %v: %v", err, response.StatusCode)
		return
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

	ctx, span := h.tracer.Start(c, "Users.Get")
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

type ChatExists struct {
	Exists bool `json:"exists"`
	ChatId int64 `json:"chatId"`
}


func (h *RestClient) CheckIsChatExists(chatIds []int64, c context.Context) (*[]ChatExists, error) {

	var chatIdsString []string
	for _, chatIdInt := range chatIds {
		chatIdsString = append(chatIdsString, utils.Int64ToString(chatIdInt))
	}

	join := strings.Join(chatIdsString, ",")

	fullUrl := fmt.Sprintf("%v%v?chatId=%v", h.baseUrl, h.checkChatExistsPath, join)

	parsedUrl, err := url.Parse(fullUrl)
	if err != nil {
		GetLogEntry(c).Errorln("Failed during parse chat url:", err)
		return nil, err
	}

	request := &http.Request{
		Method: "GET",
		URL:    parsedUrl,
		Header: map[string][]string{
			echo.HeaderContentType: {"application/json"},
		},
	}

	ctx, span := h.tracer.Start(c, "Chat.CheckExists")
	defer span.End()
	request = request.WithContext(ctx)

	response, err := h.client.Do(request)
	if err != nil {
		GetLogEntry(c).Error(err, "Transport error during checking chat presence")
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		err = errors.New(fmt.Sprintf("Unexpected status checking chat presence %v", response.StatusCode))
		return nil, err
	}

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		GetLogEntry(c).Errorln("Failed to decode get chat presence response:", err)
		return nil, err
	}

	resultMap := new([]ChatExists)
	if err := json.Unmarshal(bodyBytes, resultMap); err != nil {
		GetLogEntry(c).Errorln("Failed to parse result:", err)
		return nil, err
	}
	return resultMap, nil
}

func (h *RestClient) GetChatParticipantIds(chatId int64, c context.Context) ([]int64, error) {
	contentType := "application/json;charset=UTF-8"
	fullUrl := h.baseUrl + h.chatParticipantIdsPath

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
