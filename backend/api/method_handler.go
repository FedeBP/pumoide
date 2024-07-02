package api

import (
	"encoding/json"
	"net/http"

	"github.com/FedeBP/pumoide/backend/apperrors"
	"github.com/FedeBP/pumoide/backend/models"
	"github.com/FedeBP/pumoide/backend/utils"
	"github.com/sirupsen/logrus"
)

type MethodHandler struct {
	Logger *logrus.Logger
}

func (h *MethodHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apperrors.RespondWithError(w, http.StatusMethodNotAllowed, utils.MethodNotAllowedErr, nil, h.Logger)
		return
	}

	validMethods := models.GetValidMethods()

	w.Header().Set(utils.ContentType, utils.AppJson)
	if err := json.NewEncoder(w).Encode(validMethods); err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToWriteResponseErr, err, h.Logger)
		return
	}
}
