package utils

import "strconv"

func ParseBoolean(str string) (bool, error) {
	return strconv.ParseBool(str)
}

