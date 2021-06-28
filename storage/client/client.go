package client

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"net/http"
	"net/url"
	. "nkonev.name/storage/logger"
)

type ChatAccessClient struct {
	client          *http.Client
	baseUrl string
	accessPath string
	removeFileItemPath string
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

func NewChatAccessClient() *ChatAccessClient {
	client := newRestClient()
	return &ChatAccessClient {
		client: client,
		baseUrl: viper.GetString("chat.url.base"),
		accessPath: viper.GetString("chat.url.access"),
		removeFileItemPath: viper.GetString("chat.url.removeFileItem"),
	}
}

func (h *ChatAccessClient) CheckAccess(userId int64, chatId int64) (bool, error) {
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

func (h *ChatAccessClient) RemoveFileItem(chatId int64, fileItemUuid string, userId int64) {
	fullUrl := fmt.Sprintf("%v%v?chatId=%v&fileItemUuid=%v&userId=%v", h.baseUrl, h.removeFileItemPath, chatId, fileItemUuid, userId)

	parsedUrl, err := url.Parse(fullUrl)
	if err != nil {
		Logger.Errorln("Failed during parse chat url:", err)
		return
	}

	request := &http.Request{
		Method: "DELETE",
		URL:    parsedUrl,
	}

	response, err := h.client.Do(request)
	if err != nil {
		Logger.Error(err, "Transport error during removing file item")
		return
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		return
	} else {
		Logger.Error(err, "Unexpected status on removing file item", "httpCode", response.StatusCode)
		return
	}

}

