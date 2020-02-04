package client

import (
	"github.com/spf13/viper"
	"net/http"
)

type RestClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type restClientImpl struct {
	delegate *http.Client
}

func NewRestClient() RestClient {
	tr := &http.Transport{
		MaxIdleConns:       viper.GetInt("http.idle.conns.max"),
		IdleConnTimeout:    viper.GetDuration("http.idle.connTimeout"),
		DisableCompression: viper.GetBool("http.disableCompression"),
	}
	client := &http.Client{Transport: tr}
	return &restClientImpl{client}
}

func (rc *restClientImpl) Do(req *http.Request) (*http.Response, error) {
	return rc.delegate.Do(req)
}
