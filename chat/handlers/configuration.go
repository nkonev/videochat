package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
)

func GetConfiguration(c echo.Context) error {
	slice := viper.GetStringSlice("iceServers")
	return c.JSON(200, slice)
}
