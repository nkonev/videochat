package utils

import (
	"context"
	"fmt"
	"github.com/guregu/null"
	"github.com/rotisserie/eris"
	"net/url"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"
	"regexp"
	"strconv"
	"strings"
)

const USER_PRINCIPAL_DTO = "userPrincipalDto"

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

func CheckUrlInWhitelist(ctx context.Context, lgr *logger.Logger, whitelist []regexp.Regexp, uri string) bool {
	for _, regexp0 := range whitelist {
		if regexp0.MatchString(uri) {
			lgr.WithTracing(ctx).Infof("Skipping authentication for %v because it matches %v", uri, regexp0.String())
			return true
		}
	}
	return false
}

const maxSize = 100
const DefaultSize = 20
const DefaultPage = 0
const DefaultOffset = 0

const PaginationMaxSize = 200

func FixPage(page int) int {
	if page < 0 {
		return DefaultPage
	} else {
		return page
	}
}

func FixPageString(page string) int {
	atoi, err := strconv.Atoi(page)
	if err != nil {
		return DefaultPage
	} else {
		return FixPage(atoi)
	}
}

func FixId(id string) *int64 {
	atoi, err := ParseInt64(id)
	if err != nil {
		return nil
	} else {
		return &atoi
	}
}

func FixSize(size int) int {
	if size > maxSize || size < 1 {
		return DefaultSize
	} else {
		return size
	}
}

func FixSizeString(size string) int {
	atoi, err := strconv.Atoi(size)
	if err != nil {
		return DefaultSize
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
		return false, eris.Wrap(err, "unable to parse bool")
	} else {
		return parseBool, nil
	}
}

func ParseInt64(s string) (int64, error) {
	if i, err := strconv.ParseInt(s, 10, 64); err != nil {
		return 0, eris.Wrap(err, "unable to parse int")
	} else {
		return i, nil
	}
}

func Int64ToString(i int64) string {
	return fmt.Sprintf("%v", i)
}

func Float64ToString(i float64) string {
	return fmt.Sprintf("%v", i)
}

func InterfaceToString(i interface{}) string {
	return fmt.Sprintf("%v", i)
}

func SetToArray(set map[int64]bool) []int64 {
	var ownerIds []int64
	for k, v := range set {
		if v {
			ownerIds = append(ownerIds, k)
		}
	}
	return ownerIds
}

func ArrayToSet(arr []int64) map[int64]bool {
	var ownerIds map[int64]bool = map[int64]bool{}
	for _, el := range arr {
		ownerIds[el] = true
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

func ContainsUrl(lgr *logger.Logger, elems []string, elem string) bool {
	parsedUrlToTest, err := url.Parse(elem)
	if err != nil {
		lgr.Infof("Unable to parse urlToTest %v", elem)
		return false
	}
	for i := 0; i < len(elems); i++ {
		parsedAllowedUrl, err := url.Parse(elems[i])
		if err != nil {
			lgr.Infof("Unable to parse allowedUrl %v", elems[i])
			return false
		}

		if ContainUrl(parsedUrlToTest, parsedAllowedUrl) {
			return true
		}
	}
	return false
}

func ContainUrl(parsedUrlToTest, parsedAllowedUrl *url.URL) bool {
	if parsedUrlToTest.Host == parsedAllowedUrl.Host && parsedUrlToTest.Scheme == parsedAllowedUrl.Scheme {
		return true
	} else {
		return false
	}
}

func Remove(ids []int64, elem int64) []int64 {
	var newArr = []int64{}
	for _, id := range ids {
		if id != elem {
			newArr = append(newArr, id)
		}
	}
	return newArr
}

func SecondsToStringMilliseconds(seconds int64) string {
	return fmt.Sprintf("%v000", seconds)
}

func setLastSeenForteATet(
	participantsOnline map[int64]bool,
	participant *dto.User, behalfParticipantId int64, isSingleTetATetParticipant bool, chatDto dto.ChatDtoWithTetATet,
) {
	// leave LastSeenDateTime not null only if the opposite user isn't online
	if participant.Id != behalfParticipantId {
		if !isSingleTetATetParticipant {
			chatDto.SetLastSeenDateTime(participant.LastSeenDateTime)

			onl, ok := participantsOnline[participant.Id]
			if ok {
				if onl { // if the opposite user is online we don't need to show last login
					chatDto.SetLastSeenDateTime(null.TimeFromPtr(nil))
				}
			}
		}
	}
}

func ReplaceForTetATet(chatDto dto.ChatDtoWithTetATet, participantsOnline map[int64]bool, participant *dto.User, behalfParticipantId int64, isSingleTetATetParticipant bool) {
	if chatDto.GetIsTetATet() {
		if participant.Id != behalfParticipantId || isSingleTetATetParticipant {
			chatDto.SetName(participant.Login)
			chatDto.SetAvatar(participant.Avatar)
			chatDto.SetShortInfo(participant.ShortInfo)
			chatDto.SetLoginColor(participant.LoginColor)
		}

		// we cannot use the same branch participant.Id != behalfParticipantId because the second condition is the different
		setLastSeenForteATet(participantsOnline, participant, behalfParticipantId, isSingleTetATetParticipant, chatDto)
	}
}

func GetType(aDto interface{}) string {
	strName := fmt.Sprintf("%T", aDto)
	return strName
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func GetInt64BoolMap(ids []int64) map[int64]bool {
	r := map[int64]bool{}
	for _, id := range ids {
		r[id] = true
	}
	return r
}

const FileParam = "file"
const UrlStoragePublicGetFile = "/api/storage/public/download"
const UrlStorageEmbedPreview = "/embed/preview"

func SetImagePreviewExtension(key string) string {
	return SetExtension(key, "jpg")
}

func SetExtension(fileName string, newExtension string) string {
	idx := strings.LastIndex(fileName, ".")
	if idx > 0 {
		firstPart := fileName[0:idx]
		return firstPart + "." + newExtension
	} else {
		return fileName
	}
}

func RemoveExtension(fileName string) string {
	idx := strings.LastIndex(fileName, ".")
	if idx > 0 {
		firstPart := fileName[0:idx]
		return firstPart
	} else {
		return fileName
	}
}

func NullableToBoolean(pt *bool) bool {
	if pt != nil && *pt {
		return true
	} else {
		return false
	}
}

func UrlEncode(input string) string {
	params := url.Values{}
	params.Add("prefix", input)
	tmp := params.Encode()
	after, _ := strings.CutPrefix(tmp, "prefix=")
	return after
}
