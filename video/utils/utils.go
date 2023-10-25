package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/livekit/protocol/livekit"
	"nkonev.name/video/dto"
	. "nkonev.name/video/logger"
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

func ParseBoolean(str string) (bool, error) {
	return strconv.ParseBool(str)
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

func GetRoomNameFromId(chatId int64) string {
	return fmt.Sprintf("chat%v", chatId)
}

func MakeIdentityFromUserId(userId int64) string {
	return fmt.Sprintf("%v_%v", userId, uuid.New().String())
}

func GetUserIdFromIdentity(identity string) (int64, error) {
	split := strings.Split(identity, "_")
	if len(split) != 2 {
		return 0, errors.New("Should be 2 parts")
	}
	return ParseInt64(split[0])
}

func IsNotHumanUser(identity string) bool {
	split := strings.Split(identity, "_")
	if len(split) != 2 {
		return true
	}
	return split[0] == "EG"
}

func MakeMetadata(userId int64, userLogin string, avatar string) (string, error) {
	md := &dto.MetadataDto{
		UserId: userId,
		Login:  userLogin,
		Avatar: avatar,
	}

	bytes, err := json.Marshal(md)
	if err != nil {
		return "", err
	}

	mds := string(bytes)
	return mds, nil
}

func ParseMetadata(metadata string) (*dto.MetadataDto, error) {
	md := &dto.MetadataDto{}
	err := json.Unmarshal([]byte(metadata), md)
	return md, err
}

func ParseParticipantMetadataOrNull(participant *livekit.ParticipantInfo) (*dto.MetadataDto, error) {
	if IsNotHumanUser(participant.Identity) {
		return nil, nil
	}

	return ParseMetadata(participant.Metadata)
}

func GetRoomIdFromName(chatName string) (int64, error) {
	var chatId int64
	if _, err := fmt.Sscanf(chatName, "chat%d", &chatId); err != nil {
		return 0, err
	} else {
		return chatId, nil
	}
}

func GetType(aDto interface{}) string {
	strName := fmt.Sprintf("%T", aDto)
	return strName
}
