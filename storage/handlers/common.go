package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/araddon/dateparse"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/common/expfmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"math/big"
	"net/http"
	"nkonev.name/storage/auth"
	"nkonev.name/storage/client"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/s3"
	"nkonev.name/storage/utils"
	"strings"
	"time"
)

type AuthMiddleware echo.MiddlewareFunc

func ExtractAuth(request *http.Request, lgr *log.Logger) (*auth.AuthResult, error) {
	expiresInString := request.Header.Get("X-Auth-ExpiresIn") // in GMT. in milliseconds from java
	t, err := dateparse.ParseIn(expiresInString, time.UTC)
	GetLogEntry(request.Context(), lgr).Infof("Extracted session expiration time: %v", t)

	if err != nil {
		return nil, err
	}

	userIdString := request.Header.Get("X-Auth-UserId")
	i, err := utils.ParseInt64(userIdString)
	if err != nil {
		return nil, err
	}

	decodedString, err := base64.StdEncoding.DecodeString(request.Header.Get("X-Auth-Username"))
	if err != nil {
		return nil, err
	}

	roles := request.Header.Values("X-Auth-Role")

	return &auth.AuthResult{
		UserId:    i,
		UserLogin: string(decodedString),
		ExpiresAt: t.Unix(),
		Roles:     roles,
	}, nil
}

// https://www.keycloak.org/docs/latest/securing_apps/index.html#upstream-headers
// authorize checks authentication of each requests (websocket establishment or regular ones)
//
// Parameters:
//
//   - `request` : http request to check
//   - `httpClient` : client to check authorization
//
// Returns:
//
//   - *AuthResult pointer or nil
//   - is whitelisted
//   - error
func authorize(request *http.Request, lgr *log.Logger) (*auth.AuthResult, bool, error) {
	whitelistStr := viper.GetStringSlice("auth.exclude")
	whitelist := utils.StringsToRegexpArray(whitelistStr)
	if utils.CheckUrlInWhitelist(request.Context(), lgr, whitelist, request.RequestURI) {
		return nil, true, nil
	}
	auth, err := ExtractAuth(request, lgr)
	if err != nil {
		GetLogEntry(request.Context(), lgr).Infof("Error during extract AuthResult: %v", err)
		return nil, false, nil
	}
	GetLogEntry(request.Context(), lgr).Infof("Success AuthResult: %v", *auth)
	return auth, false, nil
}

func ConfigureAuthMiddleware(lgr *log.Logger) AuthMiddleware {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authResult, whitelist, err := authorize(c.Request(), lgr)
			if err != nil {
				GetLogEntry(c.Request().Context(), lgr).Errorf("Error during authorize: %v", err)
				return err
			} else if whitelist {
				return next(c)
			} else if authResult == nil {
				return c.JSON(http.StatusUnauthorized, &utils.H{"status": "unauthorized"})
			} else {
				c.Set(utils.USER_PRINCIPAL_DTO, authResult)
				return next(c)
			}
		}
	}
}

func Convert(h http.Handler) echo.HandlerFunc {
	return func(c echo.Context) error {
		h.ServeHTTP(c.Response().Writer, c.Request())
		return nil
	}
}

func getMaxAllowedConsumption(ctx context.Context, lgr *log.Logger, restClient *client.RestClient, isUnlimited bool) (int64, error) {
	if isUnlimited {
		return calcTotalSize(ctx, lgr, restClient)
	} else {
		return viper.GetInt64("limits.default.all.users.limit"), nil
	}
}

func calcTotalSize(ctx context.Context, lgr *log.Logger, restClient *client.RestClient) (int64, error) {
	var totalClusterSize = big.NewFloat(0)

	clusterMetrics, err := restClient.GetMinioMetricsCluster(ctx)
	if err != nil {
		GetLogEntry(ctx, lgr).Errorf("Error during getting bucket consumption %v", err)
		return 0, err
	}

	var parser expfmt.TextParser

	mf, err := parser.TextToMetricFamilies(strings.NewReader(clusterMetrics))
	if err != nil {
		GetLogEntry(ctx, lgr).Errorf("Error during parsing bucket consumption %v", err)
		return 0, err
	}

	mfi := mf["minio_cluster_capacity_usable_total_bytes"]

	for _, me := range mfi.Metric {
		if me != nil && me.Gauge != nil && me.Gauge.Value != nil {
			addable := big.NewFloat(*me.Gauge.Value)
			newV := big.NewFloat(0)
			newV.Add(totalClusterSize, addable)
			totalClusterSize = newV
		}
	}

	resV, _ := totalClusterSize.Int64()

	return resV, nil
}

func calcBucketsConsumption(ctx context.Context, lgr *log.Logger, restClient *client.RestClient) (int64, error) {
	var totalBucketConsumption = big.NewFloat(0)

	bucketMetrics, err := restClient.GetMinioMetricsBucket(ctx)
	if err != nil {
		GetLogEntry(ctx, lgr).Errorf("Error during getting bucket consumption %v", err)
		return 0, err
	}

	var parser expfmt.TextParser

	mf, err := parser.TextToMetricFamilies(strings.NewReader(bucketMetrics))
	if err != nil {
		GetLogEntry(ctx, lgr).Errorf("Error during parsing bucket consumption %v", err)
		return 0, err
	}

	mfi := mf["minio_bucket_usage_total_bytes"]

	for _, me := range mfi.Metric {
		if me != nil && me.Gauge != nil && me.Gauge.Value != nil {
			addable := big.NewFloat(*me.Gauge.Value)
			newV := big.NewFloat(0)
			newV.Add(totalBucketConsumption, addable)
			totalBucketConsumption = newV
		}
	}

	resV, _ := totalBucketConsumption.Int64()

	return resV, nil
}

func checkUserLimit(ctx context.Context, lgr *log.Logger, minioClient *s3.InternalMinioClient, bucketName string, userPrincipalDto *auth.AuthResult, desiredSize int64, restClient *client.RestClient) (bool, int64, int64, error) {
	limitsEnabled := viper.GetBool("limits.enabled")
	// TODO take into account userId
	consumption, err := calcBucketsConsumption(ctx, lgr, restClient)
	if err != nil {
		GetLogEntry(ctx, lgr).Errorf("Error during getting consumption %v", err)
		return false, 0, 0, err
	}

	isUnlimited := (userPrincipalDto != nil && userPrincipalDto.HasRole("ROLE_ADMIN")) || !limitsEnabled

	maxAllowed, err := getMaxAllowedConsumption(ctx, lgr, restClient, isUnlimited)
	if err != nil {
		GetLogEntry(ctx, lgr).Errorf("Error during calculating max allowed %v", err)
		return false, 0, 0, err
	}
	available := maxAllowed - consumption
	GetLogEntry(ctx, lgr).Debugf("Max allowed %v, isUnlimited %v, consumption %v, available %v", maxAllowed, isUnlimited, consumption, available)

	if desiredSize > available {
		GetLogEntry(ctx, lgr).Infof("Upload too large %v+%v>%v bytes", consumption, desiredSize, maxAllowed)
		return false, consumption, available, nil
	}
	return true, consumption, available, nil
}

func cacheableResponse(c echo.Context, ttl time.Duration) {
	if c.Request().URL.Query().Get("cache") != "false" {
		delta := viper.GetDuration("response.cache.delta")
		cacheControlValue := fmt.Sprintf("public, max-age=%v", (ttl - delta).Seconds())
		c.Response().Header().Set("Cache-Control", cacheControlValue)
	}
}

func avatarCacheableResponse(c echo.Context) {
	cacheableResponse(c, viper.GetDuration("response.cache.avatar"))
}
