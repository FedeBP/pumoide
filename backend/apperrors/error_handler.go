package apperrors

import (
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
)

type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func NewAppError(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func RespondWithError(w http.ResponseWriter, statusCode int, message string, err error, logger *logrus.Logger) {
	appErr := NewAppError(statusCode, message, err)
	if err != nil {
		logger.Printf("[%d] %s: %v", statusCode, message, err)
	} else {
		logger.Printf("[%d] %s", statusCode, message)
	}
	http.Error(w, appErr.Error(), statusCode)
}
