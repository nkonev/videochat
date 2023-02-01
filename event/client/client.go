package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	. "nkonev.name/event/logger"
	"nkonev.name/event/utils"
	"strings"
)

type RestClient struct {
	client               *http.Client
	chatBaseUrl          string
	accessPath           string
	aaaBaseUrl           string
	requestForOnlinePath string
	tracer               trace.Tracer
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

	return &RestClient{
		client:               client,
		chatBaseUrl:          viper.GetString("chat.url.base"),
		accessPath:           viper.GetString("chat.url.access"),
		aaaBaseUrl:           viper.GetString("aaa.url.base"),
		requestForOnlinePath: viper.GetString("aaa.url.requestForOnline"),
		tracer:               trcr,
	}
}

func (h *RestClient) CheckAccess(userId int64, chatId int64, c context.Context) (bool, error) {
	url := fmt.Sprintf("%v%v?userId=%v&chatId=%v", h.chatBaseUrl, h.accessPath, userId, chatId)

	req, err := http.NewRequest("GET", url, nil)
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

func (h *RestClient) AskForUserOnline(userIds []int64, c context.Context) {
	var userIdsString []string
	for _, userIdInt := range userIds {
		userIdsString = append(userIdsString, utils.Int64ToString(userIdInt))
	}

	join := strings.Join(userIdsString, ",")

	url := h.aaaBaseUrl + h.requestForOnlinePath + "?userId=" + join

	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		GetLogEntry(c).Error(err, "Error during create GET")
		return
	}

	ctx, span := h.tracer.Start(c, "online.Request")
	defer span.End()
	req = req.WithContext(ctx)

	response, err := h.client.Do(req)
	if err != nil {
		GetLogEntry(c).Error(err, "Transport error during online.Request")
		return
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		return
	} else {
		err := errors.New("Unexpected status on online.Request")
		GetLogEntry(c).Error(err, "Unexpected status on online.Request", response.StatusCode)
		return
	}
}
