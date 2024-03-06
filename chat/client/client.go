package client

import (
	"bytes"
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
	"nkonev.name/chat/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
	"strings"
)

type RestClient struct {
	*http.Client
	tracer trace.Tracer
}

func NewRestClient() *RestClient {
	tr := &http.Transport{
		MaxIdleConns:       viper.GetInt("http.maxIdleConns"),
		IdleConnTimeout:    viper.GetDuration("http.idleConnTimeout"),
		DisableCompression: viper.GetBool("http.disableCompression"),
	}
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	trR := otelhttp.NewTransport(tr)
	client := &http.Client{Transport: trR}
	trcr := otel.Tracer("rest/client")

	return &RestClient{client, trcr}
}

func (rc RestClient) GetUsers(userIds []int64, c context.Context) ([]*dto.User, error) {
	contentType := "application/json;charset=UTF-8"
	url0 := viper.GetString("aaa.url.base")
	url1 := viper.GetString("aaa.url.getUsers")
	fullUrl := url0 + url1

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

	ctx, span := rc.tracer.Start(c, "users.Get")
	defer span.End()
	request = request.WithContext(ctx)
	resp, err := rc.Do(request)
	if err != nil {
		GetLogEntry(c).Warningln("Failed to request get users response:", err)
		return nil, err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code != 200 {
		GetLogEntry(c).Warningln("Users response responded non-200 code: ", code)
		return nil, errors.New("Users response responded non-200 code")
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

func (rc RestClient) GetOnlines(userIds []int64, c context.Context) ([]*dto.UserOnline, error) {
	contentType := "application/json;charset=UTF-8"
	url0 := viper.GetString("aaa.url.base")
	url1 := viper.GetString("aaa.url.getOnlines")
	fullUrl := url0 + url1

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

	ctx, span := rc.tracer.Start(c, "users.Onlines")
	defer span.End()
	request = request.WithContext(ctx)
	resp, err := rc.Do(request)
	if err != nil {
		GetLogEntry(c).Warningln("Failed to request get users response:", err)
		return nil, err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code != 200 {
		GetLogEntry(c).Warningln("Users response responded non-200 code: ", code)
		return nil, errors.New("Users response responded non-200 code")
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		GetLogEntry(c).Errorln("Failed to decode get users response:", err)
		return nil, err
	}

	users := &[]*dto.UserOnline{}
	if err := json.Unmarshal(bodyBytes, users); err != nil {
		GetLogEntry(c).Errorln("Failed to parse users:", err)
		return nil, err
	}
	return *users, nil
}

type searchUsersResuqstDto struct {
	Page         int     `json:"page"`
	Size         int     `json:"size"`
	UserIds      []int64 `json:"userIds"`
	SearchString string  `json:"searchString"`
	Including    bool    `json:"including"`
}

type searchUsersResponseDto struct {
	Users []*dto.User `json:"users"`
	Count int         `json:"count"`
}

func (rc RestClient) SearchGetUsers(searchString string, including bool, ids []int64, page, size int, c context.Context) ([]*dto.User, int, error) {
	contentType := "application/json;charset=UTF-8"
	url0 := viper.GetString("aaa.url.base")
	url1 := viper.GetString("aaa.url.searchUsers")
	fullUrl := url0 + url1

	requestHeaders := map[string][]string{
		"Accept-Encoding": {"gzip, deflate"},
		"Accept":          {contentType},
		"Content-Type":    {contentType},
	}

	parsedUrl, err := url.Parse(fullUrl)
	if err != nil {
		GetLogEntry(c).Errorln("Failed during parse aaa url:", err)
		return nil, 0, err
	}

	req := searchUsersResuqstDto{
		UserIds:      ids,
		SearchString: searchString,
		Including:    including,
		Page:         page,
		Size:         size,
	}

	bytesData, err := json.Marshal(req)
	if err != nil {
		GetLogEntry(c).Errorln("Failed during marshalling:", err)
		return nil, 0, err
	}
	reader := bytes.NewReader(bytesData)

	nopCloser := ioutil.NopCloser(reader)

	request := &http.Request{
		Method: "POST",
		Header: requestHeaders,
		URL:    parsedUrl,
		Body:   nopCloser,
	}

	ctx, span := rc.tracer.Start(c, "users.Search")
	defer span.End()
	request = request.WithContext(ctx)
	resp, err := rc.Do(request)
	if err != nil {
		GetLogEntry(c).Warningln("Failed to request get users response:", err)
		return nil, 0, err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code != 200 {
		GetLogEntry(c).Warningln("Users response responded non-200 code: ", code)
		return nil, 0, errors.New("Users response responded non-200 code")
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		GetLogEntry(c).Errorln("Failed to decode get users response:", err)
		return nil, 0, err
	}

	respDto := &searchUsersResponseDto{}
	if err := json.Unmarshal(bodyBytes, respDto); err != nil {
		GetLogEntry(c).Errorln("Failed to parse users:", err)
		return nil, 0, err
	}
	return respDto.Users, respDto.Count, nil
}

type UserExists struct {
	Exists bool  `json:"exists"`
	UserId int64 `json:"userId"`
}


func (rc RestClient) CheckAreUsersExists(userIds []int64, c context.Context) (*[]UserExists, error) {

	var chatIdsString []string
	for _, chatIdInt := range userIds {
		chatIdsString = append(chatIdsString, utils.Int64ToString(chatIdInt))
	}

	join := strings.Join(chatIdsString, ",")

	url0 := viper.GetString("aaa.url.base")
	url1 := viper.GetString("aaa.url.checkUsersExistsPath")

	fullUrl := fmt.Sprintf("%v?userId=%v", url0 + url1, join)

	parsedUrl, err := url.Parse(fullUrl)
	if err != nil {
		GetLogEntry(c).Errorln("Failed during parse aaa url:", err)
		return nil, err
	}

	request := &http.Request{
		Method: "GET",
		URL:    parsedUrl,
		Header: map[string][]string{
			echo.HeaderContentType: {"application/json"},
		},
	}

	ctx, span := rc.tracer.Start(c, "users.CheckExists")
	defer span.End()
	request = request.WithContext(ctx)

	response, err := rc.Do(request)
	if err != nil {
		GetLogEntry(c).Error(err, "Transport error during checking user presence")
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		err = errors.New(fmt.Sprintf("Unexpected status checking user presence %v", response.StatusCode))
		return nil, err
	}

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		GetLogEntry(c).Errorln("Failed to decode get user presence response:", err)
		return nil, err
	}

	resultMap := new([]UserExists)
	if err := json.Unmarshal(bodyBytes, resultMap); err != nil {
		GetLogEntry(c).Errorln("Failed to parse result:", err)
		return nil, err
	}
	return resultMap, nil
}
