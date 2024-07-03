package api

import (
	"encoding/json"
	"net/http"

	"github.com/FedeBP/pumoide/backend/errors"
	"github.com/FedeBP/pumoide/backend/models"
	"github.com/FedeBP/pumoide/backend/utils"
	"github.com/sirupsen/logrus"
)

type MethodHandler struct {
	Logger *logrus.Logger
}

func (h *MethodHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errors.RespondWithError(w, http.StatusMethodNotAllowed, utils.MethodNotAllowedErr, nil, h.Logger)
		return
	}

	validMethods := models.GetValidMethods()

	if len(validMethods) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set(utils.ContentType, utils.AppJson)
	if err := json.NewEncoder(w).Encode(validMethods); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToWriteResponseErr, err, h.Logger)
		return
	}
}
