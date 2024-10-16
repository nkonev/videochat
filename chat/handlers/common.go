package handlers

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/araddon/dateparse"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/guregu/null"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"net/http"
	"net/url"
	"nkonev.name/chat/auth"
	"nkonev.name/chat/client"
	"nkonev.name/chat/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/services"
	"nkonev.name/chat/utils"
	"strings"
	"time"
)

func getUsersRemotely(ctx context.Context, userIdSet map[int64]bool, restClient *client.RestClient) (map[int64]*dto.User, error) {
	var userIds = utils.SetToArray(userIdSet)
	length := len(userIds)
	GetLogEntry(ctx).Infof("Requested user length is %v", length)
	if length == 0 {
		return map[int64]*dto.User{}, nil
	}
	users, err := restClient.GetUsers(ctx, userIds)
	if err != nil {
		return nil, err
	}
	var ownersObjects = map[int64]*dto.User{}
	for _, u := range users {
		ownersObjects[u.Id] = u
	}
	return ownersObjects, nil
}

func getUserOnlinesRemotely(ctx context.Context, userIdSet map[int64]bool, restClient *client.RestClient) (map[int64]*dto.UserOnline, error) {
	var userIds = utils.SetToArray(userIdSet)
	length := len(userIds)
	GetLogEntry(ctx).Infof("Requested user length is %v", length)
	if length == 0 {
		return map[int64]*dto.UserOnline{}, nil
	}
	users, err := restClient.GetOnlines(ctx, userIds)
	if err != nil {
		return nil, err
	}
	var ownersObjects = map[int64]*dto.UserOnline{}
	for _, u := range users {
		ownersObjects[u.Id] = u
	}
	return ownersObjects, nil
}

func getUsersRemotelyOrEmptyFromSlice(ctx context.Context, userIds []int64, restClient *client.RestClient) map[int64]*dto.User {
	return getUsersRemotelyOrEmpty(ctx, utils.ArrayToSet(userIds), restClient)
}

func getUserOnlinesRemotelyOrEmptyFromSlice(ctx context.Context, userIds []int64, restClient *client.RestClient) map[int64]*dto.UserOnline {
	return getUserOnlinesRemotelyOrEmpty(ctx, utils.ArrayToSet(userIds), restClient)
}

func getUsersRemotelyOrEmpty(ctx context.Context, userIdSet map[int64]bool, restClient *client.RestClient) map[int64]*dto.User {
	if remoteUsers, err := getUsersRemotely(ctx, userIdSet, restClient); err != nil {
		GetLogEntry(ctx).Warn("Error during getting users from aaa")
		return map[int64]*dto.User{}
	} else {
		return remoteUsers
	}
}

func getUserOnlinesRemotelyOrEmpty(ctx context.Context, userIdSet map[int64]bool, restClient *client.RestClient) map[int64]*dto.UserOnline {
	if remoteUsers, err := getUserOnlinesRemotely(ctx, userIdSet, restClient); err != nil {
		GetLogEntry(ctx).Warn("Error during getting users from aaa")
		return map[int64]*dto.UserOnline{}
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

	anAvatar := request.Header.Get("X-Auth-Avatar")
	var theAvatar *string = nil
	if len(anAvatar) > 0 {
		theAvatar = &anAvatar
	}

	return &auth.AuthResult{
		UserId:    i,
		UserLogin: string(decodedString),
		ExpiresAt: t.Unix(),
		Roles:     roles,
		Avatar:    theAvatar,
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
				GetLogEntry(c.Request().Context()).Errorf("Error during authorize: %v", err)
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

func TrimAmdSanitizeChatTitle(policy *services.StripTagsPolicy, title string) string {
	t := Trim(policy.Sanitize(title))
	return t
}

type MediaUrlErr struct {
	url   string
	where string
}

func (s *MediaUrlErr) Error() string {
	return fmt.Sprintf("Media url is not allowed in %v: %v", s.where, s.url)
}

func TrimAmdSanitizeMessage(ctx context.Context, policy *services.SanitizerPolicy, input string) (string, error) {
	sanitizedHtml := Trim(SanitizeMessage(policy, input))

	whitelist := viper.GetString("message.allowedMediaUrls")
	wlArr := strings.Split(whitelist, ",")
	frontendUrl := viper.GetString("frontendUrl")
	wlArr = append(wlArr, frontendUrl)
	wlArr = append(wlArr, "") // storage urls without protocol://host:port

	iframeWhitelist := viper.GetString("message.allowedIframeUrls")
	iframeWlArr := strings.Split(iframeWhitelist, ",")

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(sanitizedHtml))
	if err != nil {
		GetLogEntry(ctx).Warnf("Unable to read html: %v", err)
		return "", errors.New("Unable to read html")
	}

	var retErr error
	maxMediasCount := viper.GetInt("message.maxMedias")
	mediaCount := 0

	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		maybeImage := s.First()
		if maybeImage != nil {
			src, exists := maybeImage.Attr("src")
			if exists && !utils.ContainsUrl(wlArr, src) {
				GetLogEntry(ctx).Infof("Filtered not allowed url in image src %v", src)
				retErr = &MediaUrlErr{src, "image src"}
			}
			if exists {
				fixedSrc, err := removeProtocolHostPortIfNeed(src, frontendUrl)
				if err != nil {
					retErr = err
				}
				maybeImage.SetAttr("src", fixedSrc)
			}

			original, originalExists := maybeImage.Attr("data-original")
			if originalExists && !utils.ContainsUrl(wlArr, original) {
				GetLogEntry(ctx).Infof("Filtered not allowed url in image src %v", original)
				retErr = &MediaUrlErr{original, "image src"}
			}
			if originalExists {
				fixedSrc, err := removeProtocolHostPortIfNeed(original, frontendUrl)
				if err != nil {
					retErr = err
				}
				maybeImage.SetAttr("data-original", fixedSrc)
			}

			mediaCount++
		}
	})
	if retErr != nil {
		return "", retErr
	}

	doc.Find("video").Each(func(i int, s *goquery.Selection) {
		maybeVideo := s.First()
		if maybeVideo != nil {
			src, srcExists := maybeVideo.Attr("src")
			if srcExists && !utils.ContainsUrl(wlArr, src) {
				GetLogEntry(ctx).Infof("Filtered not allowed url in video src %v", src)
				retErr = &MediaUrlErr{src, "video src"}
			}
			if srcExists {
				fixedSrc, err := removeProtocolHostPortIfNeed(src, frontendUrl)
				if err != nil {
					retErr = err
				}
				maybeVideo.SetAttr("src", fixedSrc)
			}

			poster, posterExists := maybeVideo.Attr("poster")
			if posterExists && !utils.ContainsUrl(wlArr, poster) {
				GetLogEntry(ctx).Infof("Filtered not allowed url in video poster %v", poster)
				retErr = &MediaUrlErr{src, "video poster"}
			}
			if posterExists {
				fixedPoster, err := removeProtocolHostPortIfNeed(poster, frontendUrl)
				if err != nil {
					retErr = err
				}
				maybeVideo.SetAttr("poster", fixedPoster)
			}

			mediaCount++
		}
	})
	if retErr != nil {
		return "", retErr
	}

	doc.Find("iframe").Each(func(i int, s *goquery.Selection) {
		maybeIframe := s.First()
		if maybeIframe != nil {
			src, exists := maybeIframe.Attr("src")
			if exists && !utils.ContainsUrl(iframeWlArr, src) {
				GetLogEntry(ctx).Infof("Filtered not allowed url in iframe src %v", src)
				retErr = &MediaUrlErr{src, "iframe src"}
			}
			mediaCount++
		}
	})
	if retErr != nil {
		return "", retErr
	}

	doc.Find("audio").Each(func(i int, s *goquery.Selection) {
		maybeAudio := s.First()
		if maybeAudio != nil {
			src, exists := maybeAudio.Attr("src")
			if exists && !utils.ContainsUrl(wlArr, src) {
				GetLogEntry(ctx).Infof("Filtered not allowed url in audio src %v", src)
				retErr = &MediaUrlErr{src, "audio src"}
			}
			if exists {
				fixedSrc, err := removeProtocolHostPortIfNeed(src, frontendUrl)
				if err != nil {
					retErr = err
				}
				maybeAudio.SetAttr("src", fixedSrc)
			}

			mediaCount++
		}
	})
	if retErr != nil {
		return "", retErr
	}
	if mediaCount > maxMediasCount {
		return "", errors.New("Too many medias")
	}

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		maybeA := s.First()
		if maybeA != nil {
			src, exists := maybeA.Attr("href")
			if exists {
				fixedSrc, err := removeProtocolHostPortIfNeed(src, frontendUrl)
				if err != nil {
					retErr = err
				}
				maybeA.SetAttr("href", fixedSrc)
			}
		}
	})
	if retErr != nil {
		return "", retErr
	}

	ret, err := doc.Find("html").Find("body").Html()
	if err != nil {
		GetLogEntry(ctx).Warnf("Unagle to write html: %v", err)
		return "", err
	}

	return ret, nil
}

func removeProtocolHostPortIfNeed(src, frontendUrl string) (string, error) {
	parsed, err := url.Parse(src)
	if err != nil {
		return "", err
	}

	parsedAllowedUrl, err := url.Parse(frontendUrl)
	if err != nil {
		return "", err
	}

	if utils.ContainUrl(parsed, parsedAllowedUrl) {
		parsed.Host = ""
		parsed.Scheme = ""
		parsed.User = nil
	}
	return parsed.String(), nil
}

func TrimAmdSanitizeAvatar(ctx context.Context, policy *services.SanitizerPolicy, input null.String) null.String {
	if input.IsZero() {
		return input
	}

	trimmed := Trim(input.String)

	if len(trimmed) == 0 {
		return null.StringFromPtr(nil)
	}

	sanitizedHtml := SanitizeMessage(policy, trimmed)

	whitelist := viper.GetString("chat.allowedAvatarUrls")
	wlArr := strings.Split(whitelist, ",")

	if !utils.ContainsUrl(wlArr, sanitizedHtml) {
		GetLogEntry(ctx).Infof("Filtered chat avatar not allowed url in chat avatar src %v", sanitizedHtml)
		return null.StringFromPtr(nil)
	}

	return null.StringFrom(sanitizedHtml)
}

func ValidateAndRespondError(c echo.Context, v validation.Validatable) (bool, error) {
	if err := v.Validate(); err != nil {
		GetLogEntry(c.Request().Context()).Debugf("Error during validation: %v", err)
		return false, c.JSON(http.StatusBadRequest, err)
	}
	return true, nil
}

func createMessagePreview(cleanTagsPolicy *services.StripTagsPolicy, text, login string) string {
	input := loginPrefix(login) + text
	return createMessagePreviewWithoutLogin(cleanTagsPolicy, input)
}

func loginPrefix(login string) string {
	return login + ": "
}

func createMessagePreviewWithoutLogin(cleanTagsPolicy *services.StripTagsPolicy, text string) string {
	return stripTagsAndCut(cleanTagsPolicy, viper.GetInt("previewMaxTextSize"), text)
}

func stripTagsAndCut(cleanTagsPolicy *services.StripTagsPolicy, sizeToCut int, text string) string {
	tmp := cleanTagsPolicy.Sanitize(text)
	runes := []rune(tmp)
	textLen := len(runes)
	size := utils.Min(sizeToCut, textLen)
	ret := string(runes[:size])
	if textLen > sizeToCut {
		ret += "..."
	}
	return ret
}
