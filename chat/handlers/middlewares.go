package handlers

import (
	"fmt"
	"github.com/araddon/dateparse"
	"github.com/centrifugal/centrifuge"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"net/http"
	"nkonev.name/chat/auth"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
	"strconv"
)

type AuthMiddleware echo.MiddlewareFunc

func ExtractAuth(request *http.Request) (*auth.AuthResult, error) {
	expiresInString := request.Header.Get("X-Auth-ExpiresIn") // in GMT. in milliseconds from java
	t, err := dateparse.ParseLocal(expiresInString)
	GetLogEntry(request).Infof("Extracted session expiration time: %v", t)

	if err != nil {
		return nil, err
	}

	userIdString := request.Header.Get("X-Auth-UserId")
	i, err := strconv.ParseInt(userIdString, 10, 64)
	if err != nil {
		return nil, err
	}

	return &auth.AuthResult{
		UserId:    i,
		UserLogin: request.Header.Get("X-Auth-Username"),
		ExpiresAt: t.Unix(),
	}, nil
}

// https://www.keycloak.org/docs/latest/securing_apps/index.html#upstream-headers
// authorize checks authentication of each requests (websocket establishment or regular ones)
//
// Parameters:
//
//  - `request` : http request to check
//  - `httpClient` : client to check authorization
//
// Returns:
//
//  - *AuthResult pointer or nil
//  - is whitelisted
//  - error
func authorize(request *http.Request) (*auth.AuthResult, bool, error) {
	whitelistStr := viper.GetStringSlice("auth.exclude")
	whitelist := utils.StringsToRegexpArray(whitelistStr)
	if utils.CheckUrlInWhitelist(whitelist, request.RequestURI) {
		return nil, true, nil
	}
	auth, err := ExtractAuth(request)
	if err != nil {
		GetLogEntry(request).Infof("Error during extract AuthResult: %v", err)
		return nil, false, nil
	}
	GetLogEntry(request).Infof("Success AuthResult: %v", *auth)
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

func CentrifugeAuthMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authResult, _, err := authorize(r)
		if err != nil {
			Logger.Errorf("Error during try to authenticate centrifuge request: %v", err)
			return
		} else if authResult == nil {
			Logger.Errorf("Not authenticated centrifuge request")
			return
		} else {
			ctx := r.Context()
			newCtx := centrifuge.SetCredentials(ctx, &centrifuge.Credentials{
				UserID:   fmt.Sprintf("%v", authResult.UserId),
				ExpireAt: authResult.ExpiresAt,
				Info:     []byte(fmt.Sprintf("{\"login\": \"%v\"}", authResult.UserLogin)),
			})
			r = r.WithContext(newCtx)
			h.ServeHTTP(w, r)
		}
	})
}

func Convert(h http.Handler) echo.HandlerFunc {
	return func(c echo.Context) error {
		h.ServeHTTP(c.Response().Writer, c.Request())
		return nil
	}
}
