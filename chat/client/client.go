package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"nkonev.name/chat/config"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/utils"

	"go.opentelemetry.io/otel/trace"
)

type restClient struct {
	*http.Client
	protocolHostPort string
	tracer           trace.Tracer
	cfg              *config.AppConfig
	lgr              *logger.LoggerWrapper
	clientName       string
}

// You should call defer httpResp.Body.Close()
func queryRawResponse[ReqDto any](ctx context.Context, rc *restClient, behalfUserId int64, method, url, opName string, req *ReqDto, queryParams *url.Values) (*http.Response, error) {
	contentType := "application/json;charset=UTF-8"
	fullUrl := utils.StringToUrl(rc.protocolHostPort + url)
	if queryParams != nil {
		fullUrl.RawQuery = queryParams.Encode()
	}

	requestHeaders := map[string][]string{
		"Accept-Encoding": {"gzip, deflate"},
		"Accept":          {contentType},
		"Content-Type":    {contentType},
	}

	if behalfUserId != dto.NonExistentUser {
		requestHeaders[utils.HeaderUserId] = []string{utils.ToString(behalfUserId)}
	}

	httpReq := &http.Request{
		Method: method,
		Header: requestHeaders,
		URL:    fullUrl,
	}

	if req != nil {
		bytesData, err := json.Marshal(req)
		if err != nil {
			rc.lgr.ErrorContext(ctx, fmt.Sprintf("Failed during marshalling request body for %v:", opName), logger.AttributeError, err)
			return nil, err
		}
		reader := bytes.NewReader(bytesData)

		httpReq.Body = io.NopCloser(reader)
	}

	ctx, span := rc.tracer.Start(ctx, opName)
	defer span.End()
	httpReq = httpReq.WithContext(ctx)

	if rc.cfg.Http.Dump {
		dumpReq, err := httputil.DumpRequestOut(httpReq, true)
		if err != nil {
			return nil, err
		}
		if rc.cfg.Http.PrettyLog && !rc.cfg.Logger.Json {
			fmt.Printf("%s >>>\n", rc.clientName)
			fmt.Printf("%s\n", string(dumpReq))
		} else {
			rc.lgr.InfoContext(ctx, fmt.Sprintf("%s >>>", rc.clientName))
			rc.lgr.InfoContext(ctx, string(dumpReq))
		}
	}

	httpResp, err := rc.Do(httpReq)
	if err != nil {
		rc.lgr.WarnContext(ctx, fmt.Sprintf("Failed to request %v response:", opName), logger.AttributeError, err)
		return nil, err
	}
	code := httpResp.StatusCode
	if !(code >= 200 && code < 300) {
		rc.lgr.WarnContext(ctx, fmt.Sprintf("%v response responded non-2xx code: ", opName), "code", code)
		return nil, fmt.Errorf("%v response responded non-2xx code: %v", opName, code)
	}

	if rc.cfg.Http.Dump {
		dumpResp, err := httputil.DumpResponse(httpResp, true)
		if err != nil {
			return nil, err
		}
		if rc.cfg.Http.PrettyLog && !rc.cfg.Logger.Json {
			fmt.Printf("%s <<<\n", rc.clientName)
			fmt.Printf("%s\n", string(dumpResp))
		} else {
			rc.lgr.InfoContext(ctx, fmt.Sprintf("%s <<<", rc.clientName))
			rc.lgr.InfoContext(ctx, string(dumpResp))
		}
	}
	return httpResp, err
}

func query[ReqDto any, ResDto any](ctx context.Context, rc *restClient, behalfUserId int64, method, url, opName string, req *ReqDto, queryParams *url.Values) (ResDto, error) {
	var resp ResDto
	var err error
	httpResp, err := queryRawResponse(ctx, rc, behalfUserId, method, url, opName, req, queryParams)
	if err != nil {
		return resp, err
	}
	defer httpResp.Body.Close()

	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		rc.lgr.WarnContext(ctx, fmt.Sprintf("Failed to decode %v response:", opName), logger.AttributeError, err)
		return resp, err
	}

	if len(bodyBytes) > 0 { // to handle 204 no content
		if err = json.Unmarshal(bodyBytes, &resp); err != nil {
			rc.lgr.ErrorContext(ctx, fmt.Sprintf("Failed to parse %v response:", opName), logger.AttributeError, err)
			return resp, err
		}
	}
	return resp, nil
}

func queryNoResponse[ReqDto any](ctx context.Context, rc *restClient, behalfUserId int64, method, url, opName string, req *ReqDto, queryParams *url.Values) error {
	var err error
	httpResp, err := queryRawResponse(ctx, rc, behalfUserId, method, url, opName, req, queryParams)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	return nil
}
