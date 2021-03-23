package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	uberCompat "github.com/nkonev/jaeger-uber-propagation-compat/propagation"
	"github.com/spf13/viper"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
	"io/ioutil"
	"net/http"
	"net/url"
	"nkonev.name/chat/handlers/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
	"strings"
)

type RestClient struct {
	*http.Client
}

func NewRestClient() RestClient {
	tr := &http.Transport{
		MaxIdleConns:       viper.GetInt("http.maxIdleConns"),
		IdleConnTimeout:    viper.GetDuration("http.idleConnTimeout"),
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
		Logger.Errorln("Failed during parse aaa url:", err)
		return nil, err
	}
	request := &http.Request{
		Method: "GET",
		Header: requestHeaders,
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

func (rc RestClient) Kick(chatId int64, userId int64) error {
	contentType := "application/json;charset=UTF-8"
	url0 := viper.GetString("video.url.base")
	urlKickTemplate := viper.GetString("video.url.kick")
	urlKick := fmt.Sprintf(urlKickTemplate, chatId)
	fullUrl := fmt.Sprintf("%v%v?userId=%v", url0, urlKick, userId)

	requestHeaders := map[string][]string{
		"Accept-Encoding": {"gzip, deflate"},
		"Accept":          {contentType},
		"Content-Type":    {contentType},
	}

	parsedUrl, err := url.Parse(fullUrl)
	if err != nil {
		Logger.Errorln("Failed during parse video url:", err)
		return err
	}
	request := &http.Request{
		Method: "PUT",
		Header: requestHeaders,
		URL:    parsedUrl,
	}

	resp, err := rc.Do(request)
	if err != nil {
		Logger.Warningln("Failed to request kick response:", err)
		return err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code != 200 {
		Logger.Warningln("kick response responded non-200 code: ", code)
		return err
	}
	return nil
}
