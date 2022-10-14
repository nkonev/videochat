package utils

import (
	"fmt"
	dto2 "nkonev.name/chat/dto"
	. "nkonev.name/chat/logger"
	"regexp"
	"strconv"
)

const USER_PRINCIPAL_DTO = "userPrincipalDto"
const PersonalChannelPrefix = "#"

type H map[string]interface{}

func StringsToRegexpArray(strings []string) []regexp.Regexp {
	regexps := make([]regexp.Regexp, len(strings))
	for i, str := range strings {
		r, err := regexp.Compile(str)
		if err != nil {
			panic(err)
		} else {
			regexps[i] = *r
		}
	}
	return regexps
}

func CheckUrlInWhitelist(whitelist []regexp.Regexp, uri string) bool {
	for _, regexp0 := range whitelist {
		if regexp0.MatchString(uri) {
			Logger.Infof("Skipping authentication for %v because it matches %v", uri, regexp0.String())
			return true
		}
	}
	return false
}

const maxSize = 100
const defaultSize = 20
const defaultPage = 0

func FixPage(page int) int {
	if page < 0 {
		return defaultPage
	} else {
		return page
	}
}

func FixPageString(page string) int {
	atoi, err := strconv.Atoi(page)
	if err != nil {
		return defaultPage
	} else {
		return FixPage(atoi)
	}
}

func FixSize(size int) int {
	if size > maxSize || size < 1 {
		return defaultSize
	} else {
		return size
	}
}

func FixSizeString(size string) int {
	atoi, err := strconv.Atoi(size)
	if err != nil {
		return defaultSize
	} else {
		return FixSize(atoi)
	}

}

func GetOffset(page int, size int) int {
	return page * size
}

func GetBoolean(s string) bool {
	if parseBool, err := strconv.ParseBool(s); err != nil {
		return false
	} else {
		return parseBool
	}
}

func GetBooleanWithError(s string) (bool, error) {
	if parseBool, err := strconv.ParseBool(s); err != nil {
		return false, err
	} else {
		return parseBool, nil
	}
}

func ParseInt64(s string) (int64, error) {
	if i, err := strconv.ParseInt(s, 10, 64); err != nil {
		return 0, err
	} else {
		return i, nil
	}
}

func Int64ToString(i int64) string {
	return fmt.Sprintf("%v", i)
}

func InterfaceToString(i interface{}) string {
	return fmt.Sprintf("%v", i)
}

const CHANNEL_PREFIX_CHAT_MESSAGES = "chatMessages"

func SetToArray(set map[int64]bool) []int64 {
	var ownerIds []int64
	for k, _ := range set {
		ownerIds = append(ownerIds, k)
	}
	return ownerIds
}

func GetIndexOf(ids []int64, elem int64) int {
	for i := 0; i < len(ids); i++ {
		if ids[i] == elem {
			return i
		}
	}
	return -1
}

func Contains(ids []int64, elem int64) bool {
	return GetIndexOf(ids, elem) != -1
}

func Remove(ids []int64, elem int64) []int64 {
	if !Contains(ids, elem) {
		return ids
	} else {
		var newArr = []int64{}
		for _, id := range ids {
			if id != elem {
				newArr = append(newArr, id)
			}
		}
		return newArr
	}
}

func SecondsToStringMilliseconds(seconds int64) string {
	return fmt.Sprintf("%v000", seconds)
}

func ReplaceChatNameToLoginForTetATet(chatDto dto2.ChatDtoWithTetATet, participant *dto2.User, behalfParticipantId int64) {
	if chatDto.GetIsTetATet() && participant.Id != behalfParticipantId {
		chatDto.SetName(participant.Login)
	}
}
