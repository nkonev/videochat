package services

import "github.com/microcosm-cc/bluemonday"

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
	policy.AllowAttrs("data-original").OnElements("img", "video")
	policy.AllowAttrs("data-type", "data-id").OnElements("span")
	policy.AllowAttrs("target", "class", "data-id", "data-label").OnElements("a")
	policy.AllowElements("video")
	policy.AllowAttrs("src", "class", "poster", "controls").OnElements("video")
	policy.AllowElements("iframe")
	policy.AllowAttrs("src", "class", "allowfullscreen", "frameborder").OnElements("iframe")
	policy.AllowAttrs("class").OnElements("div")
	policy.AllowElements("audio")
	policy.AllowAttrs("src", "class", "controls").OnElements("audio")
	return &SanitizerPolicy{policy}
}

func CreateStripTags() *StripTagsPolicy {
	return &StripTagsPolicy{bluemonday.StrictPolicy()}
}

func StripStripSourcePolicy() *StripSourcePolicy {
	policy := bluemonday.StrictPolicy()
	policy.SkipElementsContent("code", "pre")
	return &StripSourcePolicy{policy}
}
