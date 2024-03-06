package utils

import (
	"fmt"
	"github.com/spf13/viper"
	. "nkonev.name/storage/logger"
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
const DefaultSize = 20
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

func IntToString(i int) string {
	return fmt.Sprintf("%v", i)
}

func InterfaceToString(i interface{}) string {
	return fmt.Sprintf("%v", i)
}

func BooleanToString(i bool) string {
	return fmt.Sprintf("%v", i)
}

func ParseBoolean(str string) (bool, error) {
	return strconv.ParseBool(str)
}

func ParseBooleanOr(str string, defaultVal bool) bool {
	parseBool, err := strconv.ParseBool(str)
	if err != nil {
		return defaultVal
	}
	return parseBool
}

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

func GetStringIndexOf(ids []string, elem string) int {
	for i := 0; i < len(ids); i++ {
		if ids[i] == elem {
			return i
		}
	}
	return -1
}

func StringContains(ids []string, elem string) bool {
	return GetStringIndexOf(ids, elem) != -1
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

type Tuple struct {
	MinioKey string `json:"minioKey"`
	Filename string `json:"filename"`
	Exists   bool   `json:"exists"`
}

func GetDotExtensionStr(fileName string) string {
	split := strings.Split(fileName, ".")
	if len(split) > 1 {
		ext := split[len(split)-1]
		ext = strings.ToLower(ext)
		return "." + ext
	} else {
		return ""
	}
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

func IsImage(minioKey string) bool {
	imageTypes := viper.GetStringSlice("types.image")
	imageTypes2 := toLower(imageTypes)
	return StringContains(imageTypes2, GetDotExtensionStr(minioKey))
}

func IsVideo(minioKey string) bool {
	videoTypes := viper.GetStringSlice("types.video")
	videoTypes2 := toLower(videoTypes)
	return StringContains(videoTypes2, GetDotExtensionStr(minioKey))
}

func IsAudio(minioKey string) bool {
	videoTypes := viper.GetStringSlice("types.audio")
	videoTypes2 := toLower(videoTypes)
	return StringContains(videoTypes2, GetDotExtensionStr(minioKey))
}

func IsPlainText(minioKey string) bool {
	plainTextTypes := viper.GetStringSlice("types.plainText")
	plainTextTypes2 := toLower(plainTextTypes)
	return StringContains(plainTextTypes2, GetDotExtensionStr(minioKey))
}

func toLower(imageTypes []string) []string {
	var imageTypes2 []string = []string{}
	for _, it := range imageTypes {
		imageTypes2 = append(imageTypes2, strings.ToLower(it))
	}
	return imageTypes2
}

func GetType(aDto interface{}) string {
	strName := fmt.Sprintf("%T", aDto)
	return strName
}

const UrlStoragePublicGetFile = "/storage/public/download"
const UrlStoragePublicPreviewFile = "/storage/public/download/embed/preview"
const UrlStorageGetFile = "/storage/download"
const UrlStorageGetFilePublicExternal = "/public/download"
