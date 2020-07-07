package client

import (
	"bytes"
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/spf13/viper"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
	"io/ioutil"
	"net/http"
	"net/url"
	. "nkonev.name/chat/logger"
	name_nkonev_aaa "nkonev.name/chat/proto"
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
	trR := &ochttp.Transport{
		Base: tr,
	}
	client := &http.Client{Transport: trR}
	return RestClient{client}
}

// https://developers.google.com/protocol-buffers/docs/gotutorial
func (rc RestClient) GetUsers(userIds []int64, c context.Context) ([]*name_nkonev_aaa.UserDto, error) {
	contentType := "application/x-protobuf;charset=UTF-8"
	url0 := viper.GetString("aaa.url.base")
	url1 := viper.GetString("aaa.url.getUsers")
	fullUrl := url0 + url1
	userReq := &name_nkonev_aaa.UsersRequest{UserIds: userIds}
	useRequestBytes, err := proto.Marshal(userReq)
	if err != nil {
		Logger.Errorln("Failed to encode get users request:", err)
		return nil, err
	}

	userRequestReader := bytes.NewReader(useRequestBytes)

	requestHeaders := map[string][]string{
		"Accept-Encoding": {"gzip, deflate"},
		"Accept":          {contentType},
		"Content-Type":    {contentType},
	}

	if err != nil {
		Logger.Infof("Error during inserting tracing")
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
		Logger.Errorln("Failed to request get users response:", err)
		return nil, err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code != 200 {
		Logger.Errorf("Users response responded non-200 code: %v", code)
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
	return users.Users, nil
}
