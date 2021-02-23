package handlers

import (
	"github.com/labstack/echo/v4"
	"nkonev.name/chat/utils"
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