package handlers

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
	policy.AllowAttrs("class").OnElements("img")
	policy.AllowAttrs("target").OnElements("a")
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
