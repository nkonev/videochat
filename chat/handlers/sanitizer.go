package handlers

import "github.com/microcosm-cc/bluemonday"

func CreateSanitizer() *bluemonday.Policy {
	return bluemonday.UGCPolicy()
}
