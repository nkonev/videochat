package handlers

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/araddon/dateparse"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"net/http"
	"nkonev.name/chat/auth"
	"nkonev.name/chat/client"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/services"
	"nkonev.name/chat/utils"
	"strings"
	"time"
)

func getUsersRemotely(userIdSet map[int64]bool, restClient *client.RestClient, c echo.Context) (map[int64]*dto.User, error) {
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

func getUsersRemotelyOrEmptyFromSlice(userIds []int64, restClient *client.RestClient, c echo.Context) map[int64]*dto.User {
	return getUsersRemotelyOrEmpty(utils.ArrayToSet(userIds), restClient, c)
}

func getUsersRemotelyOrEmpty(userIdSet map[int64]bool, restClient *client.RestClient, c echo.Context) map[int64]*dto.User {
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

func SanitizeMessage(policy *services.SanitizerPolicy, input string) string {
	return policy.Sanitize(input)
}

func Trim(str string) string {
	return strings.TrimSpace(str)
}

func TrimAmdSanitize(policy *services.SanitizerPolicy, input string) string {
	return Trim(SanitizeMessage(policy, input))
}

type MediaUrlErr struct {
	url   string
	where string
}

func (s *MediaUrlErr) Error() string {
	return fmt.Sprintf("Media url is not allowed in %v: %v", s.where, s.url)
}

func TrimAmdSanitizeMessage(policy *services.SanitizerPolicy, input string) (string, error) {
	sanitizedHtml := Trim(SanitizeMessage(policy, input))

	whitelist := viper.GetString("message.allowedMediaUrls")
	wlArr := strings.Split(whitelist, ",")

	iframeWhitelist := viper.GetString("message.allowedIframeUrls")
	iframeWlArr := strings.Split(iframeWhitelist, ",")

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(sanitizedHtml))
	if err != nil {
		Logger.Warnf("Unable to read html: %v", err)
		return "", errors.New("Unable to read html")
	}

	var retErr error
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		maybeImage := s.First()
		if maybeImage != nil {
			src, exists := maybeImage.Attr("src")
			if exists && !utils.ContainsUrl(wlArr, src) {
				Logger.Infof("Filtered not allowed url in image src %v", src)
				retErr = &MediaUrlErr{src, "image src"}
			}
		}
	})

	doc.Find("video").Each(func(i int, s *goquery.Selection) {
		maybeVideo := s.First()
		if maybeVideo != nil {
			src, srcExists := maybeVideo.Attr("src")
			if srcExists && !utils.ContainsUrl(wlArr, src) {
				Logger.Infof("Filtered not allowed url in video src %v", src)
				retErr = &MediaUrlErr{src, "video src"}
			}

			poster, posterExists := maybeVideo.Attr("poster")
			if posterExists && !utils.ContainsUrl(wlArr, poster) {
				Logger.Infof("Filtered not allowed url in video poster %v", poster)
				retErr = &MediaUrlErr{src, "video poster"}
			}
		}
	})
	doc.Find("iframe").Each(func(i int, s *goquery.Selection) {
		maybeIframe := s.First()
		if maybeIframe != nil {
			src, exists := maybeIframe.Attr("src")
			if exists && !utils.ContainsUrl(iframeWlArr, src) {
				Logger.Infof("Filtered not allowed url in iframe src %v", src)
				retErr = &MediaUrlErr{src, "iframe src"}
			}
		}
	})

	return sanitizedHtml, retErr
}

func ValidateAndRespondError(c echo.Context, v validation.Validatable) (bool, error) {
	if err := v.Validate(); err != nil {
		logger.GetLogEntry(c.Request().Context()).Debugf("Error during validation: %v", err)
		return false, c.JSON(http.StatusBadRequest, err)
	}
	return true, nil
}

func createMessagePreview(cleanTagsPolicy *services.StripTagsPolicy, text, login string) string {
	tmp := cleanTagsPolicy.Sanitize(loginPrefix(login) + text)
	runes := []rune(tmp)
	size := utils.Min(len(runes), viper.GetInt("previewMaxTextSize"))
	ret := string(runes[:size])
	return ret
}

func loginPrefix(login string) string {
	return login + ": "
}

func createMessagePreviewWithoutLogin(cleanTagsPolicy *services.StripTagsPolicy, text string) string {
	tmp := cleanTagsPolicy.Sanitize(text)
	runes := []rune(tmp)
	size := utils.Min(len(runes), viper.GetInt("previewMaxTextSize"))
	ret := string(runes[:size])
	return ret
}
