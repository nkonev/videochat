package handlers

import (
	"encoding/base64"
	"net/http"
	"time"

	"github.com/araddon/dateparse"
	"github.com/labstack/echo/v4"
	"nkonev.name/video/auth"
	"nkonev.name/video/config"
	"nkonev.name/video/logger"
	"nkonev.name/video/utils"
)

type AuthMiddleware echo.MiddlewareFunc

func ExtractAuth(request *http.Request, lgr *logger.Logger) (*auth.AuthResult, error) {
	expiresInString := request.Header.Get("X-Auth-ExpiresIn") // in GMT. in milliseconds from java
	t, err := dateparse.ParseIn(expiresInString, time.UTC)
	lgr.WithTracing(request.Context()).Infof("Extracted session expiration time: %v", t)

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
	
	permissions := request.Header.Values("X-Auth-Permission")

	avatar := request.Header.Get("X-Auth-Avatar")

	return &auth.AuthResult{
		UserId:      i,
		UserLogin:   string(decodedString),
		Avatar:      avatar,
		ExpiresAt:   t.Unix(),
		Roles:       roles,
		Permissions: permissions,
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
func authorize(config *config.ExtendedConfig, request *http.Request, lgr *logger.Logger) (*auth.AuthResult, bool, error) {
	whitelistStr := config.AuthConfig.ExcludePaths
	whitelist := utils.StringsToRegexpArray(whitelistStr)
	if utils.CheckUrlInWhitelist(request.Context(), lgr, whitelist, request.RequestURI) {
		return nil, true, nil
	}
	auth, err := ExtractAuth(request, lgr)
	if err != nil {
		lgr.WithTracing(request.Context()).Infof("Error during extract AuthResult: %v", err)
		return nil, false, nil
	}
	lgr.WithTracing(request.Context()).Infof("Success AuthResult: %v", *auth)
	return auth, false, nil
}

func ConfigureAuthMiddleware(config *config.ExtendedConfig, lgr *logger.Logger) AuthMiddleware {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authResult, whitelist, err := authorize(config, c.Request(), lgr)
			if err != nil {
				lgr.WithTracing(c.Request().Context()).Errorf("Error during authorize: %v", err)
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
