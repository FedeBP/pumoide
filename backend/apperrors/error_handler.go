package apperrors

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
)

type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func NewAppError(code int, message string, err error) AppError {
	return AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func RespondWithError(w http.ResponseWriter, statusCode int, message string, err error, logger *logrus.Logger) {
	appErr := NewAppError(statusCode, message, err)

	logEntry := logger.WithFields(logrus.Fields{
		"statusCode": statusCode,
		"message":    message,
	})
	if err != nil {
		logEntry = logEntry.WithError(err)
	}
	logEntry.Error("API error")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	err = json.NewEncoder(w).Encode(struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Error   string `json:"error,omitempty"`
	}{
		Code:    appErr.Code,
		Message: appErr.Message,
		Error:   appErr.Error(),
	})
	if err != nil {
		logger.WithError(err).Error("Failed to encode error response")
		return
	}
}
