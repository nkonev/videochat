package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/guregu/null"
	uberCompat "github.com/nkonev/jaeger-uber-propagation-compat/propagation"
	"github.com/spf13/viper"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
	"io/ioutil"
	"net/http"
	"net/url"
	"nkonev.name/chat/handlers/dto"
	. "nkonev.name/chat/logger"
	name_nkonev_aaa "nkonev.name/chat/proto"
	"nkonev.name/chat/utils"
)

type RestClient struct {
	*http.Client
}

func NewRestClient() RestClient {
	tr := &http.Transport{
		MaxIdleConns:       viper.GetInt("http.idle.conns.max"),
		IdleConnTimeout:    viper.GetDuration("http.idle.connTimeout"),
		DisableCompression: viper.GetBool("http.disableCompression"),
	}
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	trR := &ochttp.Transport{
		Base:        tr,
		Propagation: &uberCompat.HTTPFormat{},
	}
	client := &http.Client{Transport: trR}
	return RestClient{client}
}

// https://developers.google.com/protocol-buffers/docs/gotutorial
func (rc RestClient) GetUsers(userIds []int64, c context.Context) ([]*dto.User, error) {
	contentType := "application/x-protobuf;charset=UTF-8"
	url0 := viper.GetString("aaa.url.base")
	url1 := viper.GetString("aaa.url.getUsers")
	fullUrl := url0 + url1
	userReq := &name_nkonev_aaa.UsersRequest{UserIds: userIds}
	useRequestBytes, err := proto.Marshal(userReq)
	if err != nil {
		Logger.Warningln("Failed to encode get users request: ", err)
		return nil, err
	}

	userRequestReader := bytes.NewReader(useRequestBytes)

	requestHeaders := map[string][]string{
		"Accept-Encoding": {"gzip, deflate"},
		"Accept":          {contentType},
		"Content-Type":    {contentType},
	}

	if err != nil {
		Logger.Warningln("Error during inserting tracing")
	}

	parsedUrl, err := url.Parse(fullUrl)
	if err != nil {
		Logger.Errorln("Failed during parse aaa url:", err)
		return nil, err
	}
	userRequestReadCloser := ioutil.NopCloser(userRequestReader)
	request := &http.Request{
		Method: "GET",
		Header: requestHeaders,
		Body:   userRequestReadCloser,
		URL:    parsedUrl,
	}

	ctx, span := trace.StartSpan(c, "users.Get")
	defer span.End()
	request = request.WithContext(ctx)
	resp, err := rc.Do(request)
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
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Logger.Errorln("Failed to decode get users response:", err)
		return nil, err
	}
	users := &name_nkonev_aaa.UsersResponse{}
	if err := proto.Unmarshal(body, users); err != nil {
		Logger.Errorln("Failed to parse users:", err)
		return nil, err
	}

	var arr []*dto.User
	for _, nu := range users.Users {
		participant := convertToParticipant(nu)
		arr = append(arr, &participant)
	}
	return arr, nil
}

func convertToParticipant(user *name_nkonev_aaa.UserDto) dto.User {
	var nullableAvatar = null.NewString(user.Avatar, user.Avatar != "")
	return dto.User{
		Id:     user.Id,
		Login:  user.Login,
		Avatar: nullableAvatar,
	}
}

type sessionRequest struct {
	ChatId string `json:"customSessionId"`
}

type sessionResponse struct {
	Id string `json:"id"`
}

// session is chat
func (rc RestClient) CreateOpenviduSession(chatId int64) (string, error) {
	contentType := "application/json;charset=UTF-8"
	url0 := viper.GetString("openvidu.url.base")
	url1 := "/openvidu/api/sessions"
	fullUrl := url0 + url1

	requestHeaders := map[string][]string{
		"Accept-Encoding": {"gzip, deflate"},
		"Accept":          {contentType},
		"Content-Type":    {contentType},
	}

	parsedUrl, err := url.Parse(fullUrl)
	if err != nil {
		Logger.Errorln("Failed during parse openvidu url:", err)
		return "", err
	}

	sessionName := fmt.Sprintf("chat%v", chatId)
	sreq := sessionRequest{sessionName}
	marshalledBytes, err := json.Marshal(sreq)
	if err != nil {
		Logger.Errorln("Error during marshalling create session request:", err)
		return "", err
	}
	reader := bytes.NewReader(marshalledBytes)
	readCloser := ioutil.NopCloser(reader)

	request := &http.Request{
		Method: "POST",
		Header: requestHeaders,
		Body:   readCloser,
		URL:    parsedUrl,
	}
	request.SetBasicAuth(viper.GetString("openvidu.user"), viper.GetString("openvidu.password"))

	resp, err := rc.Do(request)
	if err != nil {
		Logger.Warningln("Failed to request create session response:", err)
		return "", err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code == 409 {
		return sessionName, nil
	}
	if !(code >= 200 && code < 300) {
		Logger.Errorln("Openvidu create session response responded non-200 code: ", code)
		return "", errors.New("Bad response code of create session response")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Logger.Errorln("Failed to decode create session response:", err)
		return "", err
	}
	sres := &sessionResponse{}
	if err := json.Unmarshal(body, sres); err != nil {
		Logger.Errorln("Failed to parse create session response:", err)
		return "", err
	}

	return sres.Id, nil
}

type tokenResponse struct {
	Token string `json:"token"`
}

// connection is chat user
func (rc RestClient) CreateOpenviduConnection(sessionId string) (string, error) {
	contentType := "application/json;charset=UTF-8"
	url0 := viper.GetString("openvidu.url.base")
	url1 := "/openvidu/api/sessions/"
	fullUrl := url0 + url1 + sessionId + "/connection"

	requestHeaders := map[string][]string{
		"Accept-Encoding": {"gzip, deflate"},
		"Accept":          {contentType},
		"Content-Type":    {contentType},
	}

	parsedUrl, err := url.Parse(fullUrl)
	if err != nil {
		Logger.Errorln("Failed during parse openvidu url:", err)
		return "", err
	}

	sreq := utils.H{}
	marshalledBytes, err := json.Marshal(sreq)
	if err != nil {
		Logger.Errorln("Error during marshalling create token request:", err)
		return "", err
	}
	reader := bytes.NewReader(marshalledBytes)
	readCloser := ioutil.NopCloser(reader)

	request := &http.Request{
		Method: "POST",
		Header: requestHeaders,
		Body:   readCloser,
		URL:    parsedUrl,
	}
	request.SetBasicAuth(viper.GetString("openvidu.user"), viper.GetString("openvidu.password"))

	resp, err := rc.Do(request)
	if err != nil {
		Logger.Warningln("Failed to request create token response:", err)
		return "", err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if !(code >= 200 && code < 300) {
		Logger.Errorln("Openvidu create token response responded non-200 code: ", code)
		return "", errors.New("Bad response code of create token response")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Logger.Errorln("Failed to decode create token response:", err)
		return "", err
	}
	sres := &tokenResponse{}
	if err := json.Unmarshal(body, sres); err != nil {
		Logger.Errorln("Failed to parse create token response:", err)
		return "", err
	}

	return sres.Token, nil
}
