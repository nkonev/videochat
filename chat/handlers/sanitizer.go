package handlers

import "github.com/microcosm-cc/bluemonday"

func CreateSanitizer() *bluemonday.Policy {
	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("style").OnElements("span", "p")
	return policy
}
