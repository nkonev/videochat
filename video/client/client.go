package client

import (
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
	client         *http.Client
	baseUrl        string
	accessPath     string
	isAdminPath    string
	aaaBaseUrl     string
	aaaGetUsersUrl string
	tracer         trace.Tracer
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
		client:         client,
		baseUrl:        config.ChatConfig.ChatUrlConfig.Base,
		accessPath:     config.ChatConfig.ChatUrlConfig.Access,
		isAdminPath:    config.ChatConfig.ChatUrlConfig.IsChatAdmin,
		aaaBaseUrl:     config.AaaConfig.AaaUrlConfig.Base,
		aaaGetUsersUrl: config.AaaConfig.AaaUrlConfig.GetUsers,
		tracer:         trcr,
	}
}

func (h *RestClient) CheckAccess(userId int64, chatId int64, c context.Context) (bool, error) {
	url := fmt.Sprintf("%v%v?userId=%v&chatId=%v", h.baseUrl, h.accessPath, userId, chatId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		Logger.Error(err, "Error during create GET")
		return false, err
	}

	ctx, span := h.tracer.Start(c, "access.Check")
	defer span.End()
	req = req.WithContext(ctx)

	response, err := h.client.Do(req)
	if err != nil {
		Logger.Error(err, "Transport error during checking access")
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		return true, nil
	} else if response.StatusCode == http.StatusUnauthorized {
		return false, nil
	} else {
		err := errors.New("Unexpected status on checkAccess")
		Logger.Error(err, "Unexpected status on checkAccess", "httpCode", response.StatusCode)
		return false, err
	}
}

func (h *RestClient) IsAdmin(userId int64, chatId int64, c context.Context) (bool, error) {
	url := fmt.Sprintf("%v%v?userId=%v&chatId=%v", h.baseUrl, h.isAdminPath, userId, chatId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		Logger.Error(err, "Error during create GET")
		return false, err
	}

	ctx, span := h.tracer.Start(c, "chat.IsAdmin")
	defer span.End()
	req = req.WithContext(ctx)

	response, err := h.client.Do(req)
	if err != nil {
		Logger.Error(err, "Transport error during checking access")
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		return true, nil
	} else if response.StatusCode == http.StatusUnauthorized {
		return false, nil
	} else {
		err := errors.New("Unexpected status on checkAccess")
		Logger.Error(err, "Unexpected status on checkAccess", "httpCode", response.StatusCode)
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
		Logger.Errorln("Failed during parse aaa url:", err)
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
		Logger.Warningln("Failed to request get users response:", err)
		return nil, err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code != 200 {
		Logger.Warningln("Users response responded non-200 code: ", code)
		return nil, err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Logger.Errorln("Failed to decode get users response:", err)
		return nil, err
	}

	users := &[]*dto.User{}
	if err := json.Unmarshal(bodyBytes, users); err != nil {
		Logger.Errorln("Failed to parse users:", err)
		return nil, err
	}
	return *users, nil
}
