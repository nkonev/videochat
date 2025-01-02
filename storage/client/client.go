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
	"nkonev.name/storage/logger"
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
	lgr                    *logger.Logger
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

func NewChatAccessClient(lgr *logger.Logger) *RestClient {
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
		lgr:                    lgr,
	}
}

func (h *RestClient) CheckAccess(c context.Context, userId *int64, chatId int64) (bool, error) {
	return h.CheckAccessExtended(c, userId, chatId, utils.ChatIdNonExistent, utils.MessageIdNonExistent, "")
}

// overrideChatId and overrideMessageId come together
func (h *RestClient) CheckAccessExtended(c context.Context, userId *int64, chatId int64, overrideChatId, overrideMessageId int64, fileItemUuid string) (bool, error) {
	var url0 string

	parsed, err := url.Parse(fmt.Sprintf("%v%v", h.baseUrl, h.accessPath))
	if err != nil {
		return false, err
	}
	query := parsed.Query()

	if overrideMessageId != utils.MessageIdNonExistent {
		query.Set("chatId", utils.Int64ToString(chatId))
		query.Set(utils.OverrideChatId, utils.Int64ToString(overrideChatId))
		query.Set(utils.OverrideMessageId, utils.Int64ToString(overrideMessageId))
		query.Set("fileItemUuid", fileItemUuid)
	} else {
		query.Set("chatId", utils.Int64ToString(chatId))
		if userId != nil {
			query.Set("userId", utils.Int64ToString(*userId))
		}
		query.Set("considerCanResend", utils.BooleanToString(true))
	}
	parsed.RawQuery = query.Encode()
	url0 = parsed.String()

	req, err := http.NewRequest("GET", url0, nil)
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

func (h *RestClient) RemoveFileItem(c context.Context, chatId int64, fileItemUuid string, userId int64) {
	fullUrl := fmt.Sprintf("%v%v?chatId=%v&fileItemUuid=%v&userId=%v", h.baseUrl, h.removeFileItemPath, chatId, fileItemUuid, userId)

	parsedUrl, err := url.Parse(fullUrl)
	if err != nil {
		h.lgr.WithTracing(c).Errorln("Failed during parse chat url:", err)
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
		h.lgr.WithTracing(c).Errorf("Transport error during removing file item %v", err)
		return
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		return
	} else {
		h.lgr.WithTracing(c).Errorf("Unexpected status on removing file item %v: %v", err, response.StatusCode)
		return
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

	ctx, span := h.tracer.Start(c, "Users.Get")
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

type ChatExists struct {
	Exists bool  `json:"exists"`
	ChatId int64 `json:"chatId"`
}

func (h *RestClient) CheckIsChatExists(c context.Context, chatIds []int64) (*[]ChatExists, error) {

	var chatIdsString []string
	for _, chatIdInt := range chatIds {
		chatIdsString = append(chatIdsString, utils.Int64ToString(chatIdInt))
	}

	join := strings.Join(chatIdsString, ",")

	fullUrl := fmt.Sprintf("%v%v?chatId=%v", h.baseUrl, h.checkChatExistsPath, join)

	parsedUrl, err := url.Parse(fullUrl)
	if err != nil {
		h.lgr.WithTracing(c).Errorln("Failed during parse chat url:", err)
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
		h.lgr.WithTracing(c).Errorw("Transport error during checking chat presence", err)
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		err = errors.New(fmt.Sprintf("Unexpected status checking chat presence %v", response.StatusCode))
		return nil, err
	}

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		h.lgr.WithTracing(c).Errorln("Failed to decode get chat presence response:", err)
		return nil, err
	}

	resultMap := new([]ChatExists)
	if err := json.Unmarshal(bodyBytes, resultMap); err != nil {
		h.lgr.WithTracing(c).Errorln("Failed to parse result:", err)
		return nil, err
	}
	return resultMap, nil
}

func (h *RestClient) GetChatParticipantIdsByPage(c context.Context, chatId int64, page, size int) ([]int64, error) {
	contentType := "application/json;charset=UTF-8"
	fullUrl := h.baseUrl + h.chatParticipantIdsPath

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

func buildMinioPrefix() string {
	var fullUrl = ""
	if viper.GetBool("minio.secured") {
		fullUrl += "https://"
	} else {
		fullUrl += "http://"
	}
	fullUrl += viper.GetString("minio.internalEndpoint")
	return fullUrl
}

func (h *RestClient) GetMinioMetricsCluster(c context.Context) (string, error) {
	contentType := "text/plain"

	var fullUrl = buildMinioPrefix() + "/minio/v2/metrics/cluster"

	requestHeaders := map[string][]string{
		"Accept": {contentType},
	}

	parsedUrl, err := url.Parse(fullUrl)
	if err != nil {
		h.lgr.WithTracing(c).Errorln("Failed during parse aaa url:", err)
		return "", err
	}
	request := &http.Request{
		Method: "GET",
		Header: requestHeaders,
		URL:    parsedUrl,
	}

	resp, err := h.client.Do(request)
	if err != nil {
		h.lgr.WithTracing(c).Warnln("Failed to request get minio response:", err)
		return "", err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code != 200 {
		h.lgr.WithTracing(c).Warnln("minio response responded non-200 code: ", code)
		return "", err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		h.lgr.WithTracing(c).Errorln("Failed to decode minio response:", err)
		return "", err
	}

	return string(bodyBytes), nil
}

func (h *RestClient) GetMinioMetricsBucket(c context.Context) (string, error) {
	contentType := "text/plain"

	var fullUrl = buildMinioPrefix() + "/minio/v2/metrics/bucket"

	requestHeaders := map[string][]string{
		"Accept": {contentType},
	}

	parsedUrl, err := url.Parse(fullUrl)
	if err != nil {
		h.lgr.WithTracing(c).Errorln("Failed during parse aaa url:", err)
		return "", err
	}
	request := &http.Request{
		Method: "GET",
		Header: requestHeaders,
		URL:    parsedUrl,
	}

	resp, err := h.client.Do(request)
	if err != nil {
		h.lgr.WithTracing(c).Warnln("Failed to request get minio response:", err)
		return "", err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code != 200 {
		h.lgr.WithTracing(c).Warnln("minio response responded non-200 code: ", code)
		return "", err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		h.lgr.WithTracing(c).Errorln("Failed to decode minio response:", err)
		return "", err
	}

	return string(bodyBytes), nil
}
