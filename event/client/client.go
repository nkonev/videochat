package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"io/ioutil"
	"net/http"
	"nkonev.name/event/dto"
	"nkonev.name/event/logger"
	"nkonev.name/event/utils"
	"strings"
)

type RestClient struct {
	client               *http.Client
	chatBaseUrl          string
	accessPath           string
	aaaBaseUrl           string
	requestForOnlinePath string
	userExtendedPath     string
	tracer               trace.Tracer
	lgr                  *logger.Logger
}

func NewRestClient(lgr *logger.Logger) *RestClient {
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
		userExtendedPath:     viper.GetString("aaa.url.userExtended"),
		tracer:               trcr,
		lgr:                  lgr,
	}
}

func (h *RestClient) CheckAccess(c context.Context, userId int64, chatId int64) (bool, error) {
	url := fmt.Sprintf("%v%v?userId=%v&chatId=%v", h.chatBaseUrl, h.accessPath, userId, chatId)

	req, err := http.NewRequest("GET", url, nil)
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

func (h *RestClient) AskForUserOnline(c context.Context, userIds []int64) {
	var userIdsString []string
	for _, userIdInt := range userIds {
		userIdsString = append(userIdsString, utils.Int64ToString(userIdInt))
	}

	join := strings.Join(userIdsString, ",")

	url := h.aaaBaseUrl + h.requestForOnlinePath + "?userId=" + join

	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		h.lgr.WithTracing(c).Errorw("Error during create GET", err)
		return
	}

	ctx, span := h.tracer.Start(c, "online.Request")
	defer span.End()
	req = req.WithContext(ctx)

	response, err := h.client.Do(req)
	if err != nil {
		h.lgr.WithTracing(c).Errorw("Transport error during online.Request", err)
		return
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		return
	} else {
		err := errors.New("Unexpected status on online.Request")
		h.lgr.WithTracing(c).Errorw("Unexpected status on online.Request", err, response.StatusCode)
		return
	}
}

func (h *RestClient) GetUserExtended(c context.Context, userId int64, behalfUserId int64) (*dto.UserAccountExtended, error) {

	url := h.aaaBaseUrl + h.userExtendedPath + "/" + utils.Int64ToString(userId) + "?behalfUserId=" + utils.Int64ToString(behalfUserId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		h.lgr.WithTracing(c).Errorw("Error during create GET", err)
		return nil, err
	}

	ctx, span := h.tracer.Start(c, "user.Extended")
	defer span.End()
	req = req.WithContext(ctx)

	response, err := h.client.Do(req)
	if err != nil {
		h.lgr.WithTracing(c).Errorw("Transport error during user.Extended", err)
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		err := errors.New("Unexpected status on user.Extended")
		h.lgr.WithTracing(c).Errorw("Unexpected status on user.Extended", err, response.StatusCode)
		return nil, err
	}
	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		h.lgr.WithTracing(c).Errorln("Failed to decode get users response:", err)
		return nil, err
	}

	user := &dto.UserAccountExtended{}
	if err := json.Unmarshal(bodyBytes, user); err != nil {
		h.lgr.WithTracing(c).Errorln("Failed to parse extended user:", err)
		return nil, err
	}
	return user, nil
}
