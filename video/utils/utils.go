package utils

import "strconv"

func ParseBoolean(str string) (bool, error) {
	return strconv.ParseBool(str)
}

func ParseInt64(s string) (int64, error) {
	if i, err := strconv.ParseInt(s, 10, 64); err != nil {
		return 0, err
	} else {
		return i, nil
	}
}
