package handlers

import (
	"context"
	"encoding/base64"
	"github.com/araddon/dateparse"
	"github.com/labstack/echo/v4"
	"github.com/microcosm-cc/bluemonday"
	"github.com/spf13/viper"
	"net/http"
	"nkonev.name/chat/auth"
	"nkonev.name/chat/client"
	"nkonev.name/chat/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
	"strings"
	"time"
)

func getUsersRemotely(userIdSet map[int64]bool, restClient client.RestClient, c echo.Context) (map[int64]*dto.User, error) {
	var userIds = utils.SetToArray(userIdSet)
	length := len(userIds)
	Logger.Infof("Requested user length is %v", length)
	if length == 0 {
		return map[int64]*dto.User{}, nil
	}
	users, err := restClient.GetUsers(userIds, c.Request().Context())
	if err != nil {
		return nil, err
	}
	var ownersObjects = map[int64]*dto.User{}
	for _, u := range users {
		ownersObjects[u.Id] = u
	}
	return ownersObjects, nil
}

func getUsersRemotelyOrEmpty(userIdSet map[int64]bool, restClient client.RestClient, c echo.Context) map[int64]*dto.User {
	if remoteUsers, err := getUsersRemotely(userIdSet, restClient, c); err != nil {
		GetLogEntry(c.Request().Context()).Warn("Error during getting users from aaa")
		return map[int64]*dto.User{}
	} else {
		return remoteUsers
	}
}

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
				httpContext := context.WithValue(c.Request().Context(), utils.USER_PRINCIPAL_DTO, authResult)
				httpRequestWithContext := c.Request().WithContext(httpContext)
				c.SetRequest(httpRequestWithContext)
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

func SanitizeMessage(policy *bluemonday.Policy, input string) string {
	return policy.Sanitize(input)
}

func Trim(str string) string {
	return strings.TrimSpace(str)
}

func TrimAmdSanitize(policy *bluemonday.Policy, input string) string {
	return Trim(SanitizeMessage(policy, input))
}
