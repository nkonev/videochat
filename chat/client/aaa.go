package client

import (
	"context"
	"crypto/tls"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"net/http"
	"net/url"
	"nkonev.name/chat/config"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
)

type aaaRestClient struct {
	restClient
}

type AaaRestClient interface {
	GetUsers(ctx context.Context, userIds []int64) ([]*dto.User, error)
	SearchGetUsers(ctx context.Context, searchString string, including bool, ids []int64, page int64, size int32) ([]*dto.User, int64, error)
	GetOnlines(ctx context.Context, userIds []int64) ([]*dto.UserOnline, error)
	CheckAreUsersExists(ctx context.Context, userIds []int64) ([]dto.UserExists, error)
}

func NewAAARestClient(cfg *config.AppConfig, lgr *logger.LoggerWrapper) AaaRestClient {
	tr := &http.Transport{
		MaxIdleConns:       cfg.Http.MaxIdleConns,
		IdleConnTimeout:    cfg.Http.IdleConnTimeout,
		DisableCompression: cfg.Http.DisableCompression,
	}
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	trR := otelhttp.NewTransport(tr)
	client := &http.Client{Transport: trR}
	trcr := otel.Tracer("rest/client")

	return &aaaRestClient{restClient{client, cfg.Aaa.Url.Base, trcr, cfg, lgr, "[aaa client]"}}
}

func (rc *aaaRestClient) GetUsers(ctx context.Context, userIds []int64) ([]*dto.User, error) {
	if len(userIds) == 0 {
		return []*dto.User{}, nil
	}

	queryParams := url.Values{}
	for _, u := range userIds {
		queryParams.Add("userId", utils.ToString(u))
	}
	resp, err := query[any, []*dto.User](ctx, &rc.restClient, dto.NonExistentUser, http.MethodGet, rc.cfg.Aaa.Url.GetUsers, "user.Get", nil, &queryParams)
	if err != nil {
		return []*dto.User{}, err
	}
	return resp, nil
}

func (rc *aaaRestClient) SearchGetUsers(ctx context.Context, searchString string, including bool, ids []int64, page int64, size int32) ([]*dto.User, int64, error) {
	req := dto.SearchUsersRequestDto{
		UserIds:      ids,
		SearchString: searchString,
		Including:    including,
		Page:         page,
		Size:         size,
	}

	respDto, err := query[dto.SearchUsersRequestDto, dto.SearchUsersResponseDto](ctx, &rc.restClient, dto.NonExistentUser, http.MethodPost, rc.cfg.Aaa.Url.SearchUsers, "user.Search", &req, nil)
	if err != nil {
		return nil, 0, err
	}
	return respDto.Users, respDto.Count, nil
}

func (rc *aaaRestClient) GetOnlines(ctx context.Context, userIds []int64) ([]*dto.UserOnline, error) {
	if len(userIds) == 0 {
		return []*dto.UserOnline{}, nil
	}

	queryParams := url.Values{}
	for _, u := range userIds {
		queryParams.Add("userId", utils.ToString(u))
	}
	resp, err := query[any, []*dto.UserOnline](ctx, &rc.restClient, dto.NonExistentUser, http.MethodGet, rc.cfg.Aaa.Url.GetUserOnlines, "user.GetOnlines", nil, &queryParams)
	if err != nil {
		return []*dto.UserOnline{}, err
	}
	return resp, nil
}

func (rc *aaaRestClient) CheckAreUsersExists(ctx context.Context, userIds []int64) ([]dto.UserExists, error) {
	if len(userIds) == 0 {
		return []dto.UserExists{}, nil
	}

	queryParams := url.Values{}
	for _, u := range userIds {
		queryParams.Add("userId", utils.ToString(u))
	}
	resp, err := query[any, []dto.UserExists](ctx, &rc.restClient, dto.NonExistentUser, http.MethodGet, rc.cfg.Aaa.Url.GetUserExists, "user.GetExists", nil, &queryParams)
	if err != nil {
		return []dto.UserExists{}, err
	}
	return resp, nil

}
