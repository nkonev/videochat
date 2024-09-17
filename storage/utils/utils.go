package utils

import (
	"context"
	"fmt"
	"github.com/oklog/ulid/v2"
	"github.com/spf13/viper"
	"net/url"
	. "nkonev.name/storage/logger"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const USER_PRINCIPAL_DTO = "userPrincipalDto"

type H map[string]interface{}

const MessageIdNonExistent = -1

const converted = "converted"
const underscoreConverted = "_" + converted
const ConvertedContentType = "video/webm"

const maxFilenameLength = 255 // this allows filesystem
const MaxFilenameLength = maxFilenameLength - 32 // =223. reserve 32 symbols for things like "_converted"

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

func RemoveExtension(fileName string) string {
	idx := strings.LastIndex(fileName, ".")
	if idx > 0 {
		firstPart := fileName[0:idx]
		return firstPart
	} else {
		return fileName
	}
}

func GetFilename(aKey string) string {
	split := strings.Split(aKey, "/")
	if len(split) > 1 {
		return split[len(split)-1]
	} else {
		return aKey
	}
}

func GetKeyForConverted(minioKey string) string {
	if IsVideo(minioKey) {
		idx := strings.LastIndex(minioKey, ".")
		if idx > 0 {
			firstPart := minioKey[0:idx]
			extPart := minioKey[idx+1:]
			extPart = strings.ToLower(extPart)
			return firstPart + underscoreConverted + "." + extPart
		} else {
			return minioKey
		}
	} else {
		return minioKey
	}
}

func IsConverted(minioKey string) bool {
	return strings.Contains(GetFilename(minioKey), underscoreConverted)
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

const UrlStoragePublicGetFile = "/api/storage/public/download"
const UrlStoragePublicPreviewFile = "/api/storage/public/download/embed/preview"
const UrlStorageGetFile = "/api/storage/download"
const UrlStorageGetFilePublicExternal = "/api/storage/public/download"
const UrlBasePreview = "/api/storage/embed/preview"
const UrlBasePublicPreview = "/api/storage/public/download/embed/preview"

// returns monotonically decreasing lexically sequence to use S3's lexical sorting
func GetFileItemId() string {
	location := time.UTC
	dt0 := time.Date(viper.GetInt("ulid.topYear"), time.January, 1, 0, 0, 0, 0, location)
	dt1 := time.Now().UTC()
	delta := dt0.UnixMilli() - dt1.UnixMilli()
	initializingReverseTime := time.UnixMilli(delta)
	ms := ulid.Timestamp(initializingReverseTime)
	id, _ := ulid.New(ms, ulid.DefaultEntropy())
	return id.String()
}

func ContainsUrl(elems []string, elem string) bool {
	parsedUrlToTest, err := url.Parse(elem)
	if err != nil {
		Logger.Infof("Unable to parse urlToTest %v", elem)
		return false
	}
	for i := 0; i < len(elems); i++ {
		parsedAllowedUrl, err := url.Parse(elems[i])
		if err != nil {
			Logger.Infof("Unable to parse allowedUrl %v", elems[i])
			return false
		}

		if parsedUrlToTest.Host == parsedAllowedUrl.Host && parsedUrlToTest.Scheme == parsedAllowedUrl.Scheme {
			return true
		}
	}
	return false
}

func nonLetterSplit(c rune) bool {
	return !unicode.IsLetter(c) && !unicode.IsNumber(c) && c != '.' && c != '-' && c != '+' && c != '_' && c != ' '
}

// output of this fun eventually goes to sanitizer in chat
func CleanFilename(ctx context.Context, input string, shouldAddDateToTheFilename bool) string {
	words := strings.FieldsFunc(input, nonLetterSplit)
	tmp := strings.Join(words, "")
	trimmedFilename := strings.TrimSpace(tmp)

	filenameParts := strings.Split(trimmedFilename, ".")
	hasExt := len(filenameParts) == 2
	newFileName := ""
	if hasExt && shouldAddDateToTheFilename {
		newFileName = filenameParts[0] + "_" + time.Now().UTC().Format("20060102150405") + "." + filenameParts[1]
	} else {
		newFileName = trimmedFilename
	}

	lenInBytes := len(newFileName)
	if lenInBytes > MaxFilenameLength {
		// https://github.com/minio/minio/discussions/18571
		GetLogEntry(ctx).Infof("Filename %v has more than %v bytes (%v), so we're going to strip it", newFileName, MaxFilenameLength, lenInBytes)
		nameAndExt := strings.Split(newFileName, ".")

		name := nameAndExt[0]
		ext := ""
		if hasExt {
			ext = nameAndExt[1]
		}
		newStrippedFileName := ""
		for i:=1; i <= lenInBytes; i++{
			newStrippedFileName = firstN(name, MaxFilenameLength-i)
			if hasExt {
				newStrippedFileName += ("." + ext)
			}
			if len(newStrippedFileName) <= MaxFilenameLength {
				break
			}
		}
		newFileName = newStrippedFileName
	}

	return newFileName
}

func firstN(str string, n int) string {
	v := []rune(str)
	if n >= len(v) {
		return str
	}
	return string(v[:n])
}

func NullableToBoolean(v *bool) bool {
	if v == nil {
		return false
	} else {
		return *v
	}
}
