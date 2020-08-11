package handlers

import (
	"github.com/labstack/echo/v4"
	"nkonev.name/storage/utils"
)

func GetPathParamAsInt64(c echo.Context, name string) (int64, error) {
	paramString := c.Param(name)
	param, err := utils.ParseInt64(paramString)
	if err != nil {
		return 0, err
	}
	return param, nil
}
