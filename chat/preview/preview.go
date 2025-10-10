package preview

import (
	"nkonev.name/chat/sanitizer"
	"nkonev.name/chat/utils"
)

func LoginPrefix(login string) string {
	return login + ": "
}

func CreateMessagePreview(cleanTagsPolicy *sanitizer.StripTagsPolicy, previewMaxTextSize int, text, login string) string {
	input := LoginPrefix(login) + text
	return CreateMessagePreviewWithoutLogin(cleanTagsPolicy, previewMaxTextSize, input)
}

func CreateMessagePreviewWithoutLogin(cleanTagsPolicy *sanitizer.StripTagsPolicy, previewMaxTextSize int, text string) string {
	return stripTagsAndCut(cleanTagsPolicy, previewMaxTextSize, text)
}

func stripTagsAndCut(cleanTagsPolicy *sanitizer.StripTagsPolicy, sizeToCut int, text string) string {
	tmp := cleanTagsPolicy.Sanitize(text)

	if tmp == "" {
		return tmp
	}

	runes := []rune(tmp)
	textLen := len(runes)
	size := utils.Min(sizeToCut, textLen)
	ret := string(runes[:size])
	if textLen > sizeToCut {
		ret += "..."
	}
	return ret
}
