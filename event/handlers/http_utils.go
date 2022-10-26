package handlers

import (
	"github.com/labstack/echo/v4"
	"nkonev.name/event/utils"
	"strings"
)

func GetPathParamAsInt64(c echo.Context, name string) (int64, error) {
	paramString := c.Param(name)
	param, err := utils.ParseInt64(paramString)
	if err != nil {
		return 0, err
	}
	return param, nil
}

func GetQueryParamAsInt64(c echo.Context, name string) (int64, error) {
	paramString := c.QueryParam(name)
	param, err := utils.ParseInt64(paramString)
	if err != nil {
		return 0, err
	}
	return param, nil
}

func GetQueryParamsAsInt64Slice(c echo.Context, name string) ([]int64, error) {
	q := c.Request().URL.Query() // Parse only once
	values := q[name]
	if len(values) == 0 {
		return []int64{}, nil
	}
	ret := []int64{}
	if len(values) == 1 {
		split := strings.Split(values[0], ",")
		for _, paramString := range split {
			param, err := utils.ParseInt64(paramString)
			if err != nil {
				return nil, err
			}
			ret = append(ret, param)
		}
	}
	return ret, nil
}

func GetQueryParamAsBoolean(c echo.Context, name string) (bool, error) {
	paramString := c.QueryParam(name)
	param, err := utils.GetBooleanWithError(paramString)
	if err != nil {
		return false, err
	}
	return param, nil
}
