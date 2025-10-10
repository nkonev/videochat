package sanitizer

import (
	"context"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/microcosm-cc/bluemonday"
	"net/url"
	"nkonev.name/chat/config"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
	"strings"
)

type SanitizerPolicy struct {
	*bluemonday.Policy
}

type StripTagsPolicy struct {
	*bluemonday.Policy
}

type StripSourcePolicy struct {
	*bluemonday.Policy
}

func CreateSanitizer() *SanitizerPolicy {
	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("style").OnElements("span", "p", "strong", "em", "s", "u", "img", "mark")
	policy.AllowAttrs("class").OnElements("img", "span")
	policy.AllowAttrs("data-original", "data-width", "data-height", "data-allowfullscreen").OnElements("img")
	policy.AllowAttrs("data-type", "data-id").OnElements("span")
	policy.AllowAttrs("target", "class", "data-id", "data-label", "data-type", "data-mention-suggestion-char").OnElements("a")
	policy.AllowAttrs("class", "data-id", "data-label", "data-type", "data-mention-suggestion-char").OnElements("span")
	policy.AllowAttrs("class").OnElements("div")
	return &SanitizerPolicy{policy}
}

func CreateStripTags() *StripTagsPolicy {
	return &StripTagsPolicy{bluemonday.StrictPolicy()}
}

func CreateStripSource() *StripSourcePolicy {
	policy := CreateSanitizer()
	policy.SkipElementsContent("code", "pre", "blockquote")
	return &StripSourcePolicy{policy.Policy}
}

func TrimAmdSanitizeChatTitle(policy *StripTagsPolicy, title string) string {
	t := Trim(policy.Sanitize(title))
	return t
}

func Trim(str string) string {
	return strings.TrimSpace(str)
}

func SanitizeMessage(policy *SanitizerPolicy, input string) string {
	return policy.Sanitize(input)
}

func TrimAmdSanitize(policy *SanitizerPolicy, input string) string {
	return Trim(SanitizeMessage(policy, input))
}

func TrimAmdSanitizeMessage(ctx context.Context, cfg *config.AppConfig, lgr *logger.LoggerWrapper, policy *SanitizerPolicy, input string) (string, error) {
	sanitizedHtml := Trim(SanitizeMessage(policy, input))

	whitelist := cfg.Message.AllowedMediaUrls
	wlArr := strings.Split(whitelist, ",")
	frontendUrl := cfg.FrontendUrl
	wlArr = append(wlArr, frontendUrl)
	wlArr = append(wlArr, "") // storage urls without protocol://host:port

	iframeWhitelist := cfg.Message.AllowedIframeUrls
	iframeWlArr := strings.Split(iframeWhitelist, ",")

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(sanitizedHtml))
	if err != nil {
		lgr.WarnContext(ctx, "Unable to read html", logger.AttributeError, err)
		return "", errors.New("Unable to read html")
	}

	var retErr error
	maxMediasCount := cfg.Message.MaxMedias
	mediaCount := 0

	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		maybeImage := s.First()
		if maybeImage != nil {
			src, exists := maybeImage.Attr("src")
			if exists && !utils.ContainsUrl(ctx, lgr, wlArr, src) {
				lgr.InfoContext(ctx, "Filtered not allowed url in image src", "src", src)
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
			if originalExists && (!utils.ContainsUrl(ctx, lgr, wlArr, original) && !utils.ContainsUrl(ctx, lgr, iframeWlArr, original)) {
				lgr.InfoContext(ctx, "Filtered not allowed url in image src", "src", original)
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

	if mediaCount > maxMediasCount {
		retErr = &MediaOverflowErr{maxMediasCount, mediaCount}
		return "", retErr
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
		lgr.WarnContext(ctx, "Unable to write html", logger.AttributeError, err)
		return "", err
	}

	return ret, nil
}

type MediaUrlErr struct {
	url   string
	where string
}

func (s *MediaUrlErr) Error() string {
	return fmt.Sprintf("Media url is not allowed in %v: %v", s.where, s.url)
}

type MediaOverflowErr struct {
	allowed int
	given   int
}

func (s *MediaOverflowErr) Error() string {
	return fmt.Sprintf("Too many medias: allowed %v, given %v", s.allowed, s.given)
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
