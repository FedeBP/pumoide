package errors

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/FedeBP/pumoide/backend/pkg/constants"
	"github.com/sirupsen/logrus"
)

type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e == nil {
		return "<nil>"
	}
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

	if logger != nil {
		logEntry := logger.WithFields(logrus.Fields{
			"statusCode": statusCode,
			"message":    message,
		})
		if err != nil {
			logEntry = logEntry.WithError(err)
		}
		logEntry.Error("Pumoide error")
	}

	w.Header().Set(constants.ContentType, constants.AppJson)
	w.WriteHeader(statusCode)
	encodeErr := json.NewEncoder(w).Encode(struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Error   string `json:"error,omitempty"`
	}{
		Code:    appErr.Code,
		Message: appErr.Message,
		Error:   appErr.Error(),
	})
	if encodeErr != nil && logger != nil {
		logger.WithError(encodeErr).Error(constants.ErrFailedToWriteResponse)
	}
}
