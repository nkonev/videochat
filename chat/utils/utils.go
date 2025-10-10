package utils

import (
	"context"
	"fmt"
	"net/url"
	"nkonev.name/chat/logger"
	"strconv"
	"strings"
	"time"
)

const maxSize = 100
const DefaultSize = 20
const DefaultPage = 0
const DefaultOffset = 0
const HeaderUserId = "X-Auth-Userid"
const HeaderUserRole = "X-Auth-Role"
const HeaderUserPermission = "X-Auth-Permission"
const HeaderUserLogin = "X-Auth-Username"

func ToString(in any) string {
	return fmt.Sprintf("%v", in)
}

func ParseInt64(s string) (int64, error) {
	if i, err := strconv.ParseInt(s, 10, 64); err != nil {
		return 0, fmt.Errorf("unable to parse int: %v", err)
	} else {
		return i, nil
	}
}

func ParseInt64Nullable(s string) *int64 {
	if i, err := strconv.ParseInt(s, 10, 64); err != nil {
		return nil
	} else {
		return &i
	}
}

func GetBoolean(s string) bool {
	if parseBool, err := strconv.ParseBool(s); err != nil {
		return false
	} else {
		return parseBool
	}
}

func GetBooleanNullable(s string) *bool {
	if parseBool, err := strconv.ParseBool(s); err != nil {
		return nil
	} else {
		return &parseBool
	}
}

func GetBooleanOr(s string, def bool) bool {
	v := GetBooleanNullable(s)
	if v != nil {
		return *v
	}
	return def
}

func GetNullableBooleanOr(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func GetTimeNullable(s string) *time.Time {
	time1, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return nil
	}
	return &time1
}

func GetSliceWithout(exception int64, inputData []int64) []int64 {
	ret := []int64{}
	for _, v := range inputData {
		if v != exception {
			ret = append(ret, v)
		}
	}
	return ret
}

func GetSliceWithoutSlice(exception []int64, inputData []int64) []int64 {
	remaining := make([]int64, len(inputData))
	copy(remaining, inputData)
	for _, toDeleteId := range exception {
		remaining = GetSliceWithout(toDeleteId, remaining)
	}
	return remaining
}

func StringToUrl(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}

func FixPage(page int64) int64 {
	if page < 0 {
		return DefaultPage
	} else {
		return page
	}
}

func FixPageString(page string) int64 {
	atoi, err := ParseInt64(page)
	if err != nil {
		return DefaultPage
	} else {
		return FixPage(atoi)
	}
}

func FixSize(size int32) int32 {
	if size > maxSize || size < 1 {
		return DefaultSize
	} else {
		return size
	}
}

func FixSizeString(size string) int32 {
	atoi, err := strconv.Atoi(size)
	if err != nil {
		return DefaultSize
	} else {
		return FixSize(int32(atoi))
	}

}

func GetOffset(page int64, size int32) int64 {
	return page * int64(size)
}

func ContainsUrl(ctx context.Context, lgr *logger.LoggerWrapper, elems []string, elem string) bool {
	parsedUrlToTest, err := url.Parse(elem)
	if err != nil {
		lgr.InfoContext(ctx, "Unable to parse urlToTest", "url", elem)
		return false
	}
	for i := 0; i < len(elems); i++ {
		parsedAllowedUrl, err := url.Parse(elems[i])
		if err != nil {
			lgr.InfoContext(ctx, "Unable to parse allowedUrl", "url", elems[i])
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

type H map[string]interface{}

type WithId interface {
	GetId() int64
}

func ToMap[T WithId](sliceInput []T) map[int64]T {
	m := make(map[int64]T)
	for _, v := range sliceInput {
		m[v.GetId()] = v
	}
	return m
}

func ToSlice[K comparable, T any](mapInput map[K]T) []T {
	r := make([]T, 0, len(mapInput))
	for _, v := range mapInput {
		r = append(r, v)
	}
	return r
}

func GetType(aDto interface{}) string {
	strName := fmt.Sprintf("%T", aDto)
	return strName
}

func UrlEncode(input string) string {
	params := url.Values{}
	params.Add("prefix", input)
	tmp := params.Encode()
	after, _ := strings.CutPrefix(tmp, "prefix=")
	return after
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

func ComparePointers[E comparable](p1, p2 *E) bool {
	if p1 == p2 {
		return true
	} else if p1 != nil && p2 != nil {
		return *p1 == *p2
	} else {
		return false
	}
}

func SetMapIdBoolToSlice[ID ~int | ~int64 | ~int32 | ~int16](set map[ID]bool) []ID {
	var ownerIds = make([]ID, 0)
	for k, v := range set {
		if v {
			ownerIds = append(ownerIds, k)
		}
	}
	return ownerIds
}

func SliceToSetMapIdBool[ID ~int | ~int64 | ~int32 | ~int16](arr []ID) map[ID]bool {
	var ids map[ID]bool = map[ID]bool{}
	for _, el := range arr {
		ids[el] = true
	}
	return ids
}

func SetMapIdStructToSlice[ID ~int | ~int64 | ~int32 | ~int16](set map[ID]struct{}) []ID {
	var ids = make([]ID, 0)
	for k := range set {
		ids = append(ids, k)
	}
	return ids
}

func SliceToSetMapIdStruct[ID ~int | ~int64 | ~int32 | ~int16](arr []ID) map[ID]struct{} {
	var ids map[ID]struct{} = map[ID]struct{}{}
	for _, el := range arr {
		ids[el] = struct{}{}
	}
	return ids
}

func IntersectSetMapIdStructs[ID ~int | ~int64 | ~int32 | ~int16](set1 map[ID]struct{}, set2 map[ID]struct{}) map[ID]struct{} {
	var ids = make([]ID, 0)
	for k := range set1 {
		if _, ok := set2[k]; ok {
			ids = append(ids, k)
		}
	}
	return nil
}

func IntersectSetSlices[ID ~int | ~int64 | ~int32 | ~int16](set1 []ID, set2 []ID) []ID {
	s1 := SliceToSetMapIdStruct(set1)
	s2 := SliceToSetMapIdStruct(set2)
	res := IntersectSetMapIdStructs(s1, s2)
	return SetMapIdStructToSlice(res)
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
