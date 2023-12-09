package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/araddon/dateparse"
	"github.com/labstack/echo/v4"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
	"net/http"
	"nkonev.name/storage/auth"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/s3"
	"nkonev.name/storage/utils"
	"syscall"
	"time"
)

type AuthMiddleware echo.MiddlewareFunc

func ExtractAuth(request *http.Request) (*auth.AuthResult, error) {
	expiresInString := request.Header.Get("X-Auth-ExpiresIn") // in GMT. in milliseconds from java
	t, err := dateparse.ParseIn(expiresInString, time.UTC)
	GetLogEntry(request.Context()).Infof("Extracted session expiration time: %v", t)

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
func authorize(request *http.Request) (*auth.AuthResult, bool, error) {
	whitelistStr := viper.GetStringSlice("auth.exclude")
	whitelist := utils.StringsToRegexpArray(whitelistStr)
	if utils.CheckUrlInWhitelist(whitelist, request.RequestURI) {
		return nil, true, nil
	}
	auth, err := ExtractAuth(request)
	if err != nil {
		GetLogEntry(request.Context()).Infof("Error during extract AuthResult: %v", err)
		return nil, false, nil
	}
	GetLogEntry(request.Context()).Infof("Success AuthResult: %v", *auth)
	return auth, false, nil
}

func ConfigureAuthMiddleware() AuthMiddleware {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authResult, whitelist, err := authorize(c.Request())
			if err != nil {
				Logger.Errorf("Error during authorize: %v", err)
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

func getMaxAllowedConsumption(isUnlimited bool) (int64, error) {
	if isUnlimited {
		var stat syscall.Statfs_t
		wd := viper.GetString("limits.stat.dir")
		err := syscall.Statfs(wd, &stat)
		if err != nil {
			return 0, err
		}
		// Available blocks * size per block = available space in bytes
		u := int64(stat.Bavail * uint64(stat.Bsize))
		return u, nil
	} else {
		return viper.GetInt64("limits.default.all.users.limit"), nil
	}
}

func calcUserFilesConsumption(minioClient *s3.InternalMinioClient, bucketName string) int64 {
	var totalBucketConsumption int64
	doneCh := make(chan struct{})
	defer close(doneCh)

	Logger.Debugf("Listing bucket '%v':", bucketName)
	for objInfo := range minioClient.ListObjects(context.Background(), bucketName, minio.ListObjectsOptions{Recursive: true}) {
		totalBucketConsumption += objInfo.Size
	}
	return totalBucketConsumption
}

func checkUserLimit(minioClient *s3.InternalMinioClient, bucketName string, userPrincipalDto *auth.AuthResult, desiredSize int64) (bool, int64, int64, error) {
	limitsEnabled := viper.GetBool("limits.enabled")
	// TODO take on account userId
	consumption := calcUserFilesConsumption(minioClient, bucketName)
	isUnlimited := (userPrincipalDto != nil && userPrincipalDto.HasRole("ROLE_ADMIN")) || !limitsEnabled

	maxAllowed, err := getMaxAllowedConsumption(isUnlimited)
	if err != nil {
		Logger.Errorf("Error during calculating max allowed %v", err)
		return false, 0, 0, err
	}
	available := maxAllowed - consumption

	if desiredSize > available {
		Logger.Infof("Upload too large %v+%v>%v bytes", consumption, desiredSize, maxAllowed)
		return false, consumption, available, nil
	}
	return true, consumption, available, nil
}

func cacheableResponse(c echo.Context, ttl time.Duration) {
	if c.Request().URL.Query().Get("cache") != "false" {
		cacheControlValue := fmt.Sprintf("public, max-age=%v", ttl.Seconds())
		c.Response().Header().Set("Cache-Control", cacheControlValue)
	}
}

func avatarCacheableResponse(c echo.Context) {
	cacheableResponse(c, viper.GetDuration("response.cache.avatar"))
}
