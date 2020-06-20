package handlers

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v4"
	"net/http"
	"nkonev.name/chat/logger"
)

func ValidateAndRespondError(c echo.Context, v validation.Validatable) (bool, error) {
	if err := v.Validate(); err != nil {
		logger.GetLogEntry(c.Request()).Debugf("Error during validation: %v", err)
		return false, c.JSON(http.StatusBadRequest, err)
	}
	return true, nil
}
