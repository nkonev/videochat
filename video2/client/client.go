package client

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"io/ioutil"
	"net/http"
	"net/url"
	"nkonev.name/video/dto"
	. "nkonev.name/video/logger"
	"nkonev.name/video/utils"
	"strings"
)

type RestClient struct {
	client                 *http.Client
	baseUrl                string
	accessPath             string
	removeFileItemPath     string
	aaaBaseUrl             string
	aaaGetUsersUrl         string
	checkFilesPresencePath string
	checkChatExistsPath    string
}

func newRestClient() *http.Client {
	tr := &http.Transport{
		MaxIdleConns:       viper.GetInt("http.maxIdleConns"),
		IdleConnTimeout:    viper.GetDuration("http.idleConnTimeout"),
		DisableCompression: viper.GetBool("http.disableCompression"),
	}
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{Transport: tr}
	return client
}

func NewChatAccessClient() *RestClient {
	client := newRestClient()

	return &RestClient{
		client:                 client,
		baseUrl:                viper.GetString("chat.url.base"),
		accessPath:             viper.GetString("chat.url.access"),
		removeFileItemPath:     viper.GetString("chat.url.removeFileItem"),
		aaaBaseUrl:             viper.GetString("aaa.url.base"),
		aaaGetUsersUrl:         viper.GetString("aaa.url.getUsers"),
		checkFilesPresencePath: viper.GetString("chat.url.checkEmbeddedFilesPath"),
		checkChatExistsPath:    viper.GetString("chat.url.checkChatExistsPath"),
	}
}

func (h *RestClient) CheckAccess(userId int64, chatId int64) (bool, error) {
	url := fmt.Sprintf("%v%v?userId=%v&chatId=%v", h.baseUrl, h.accessPath, userId, chatId)
	response, err := h.client.Get(url)
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

func (h *RestClient) GetUsers(userIds []int64) ([]*dto.User, error) {
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
