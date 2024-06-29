package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/FedeBP/pumoide/backend/apperrors"
	"github.com/FedeBP/pumoide/backend/models"
)

type MethodHandler struct {
	Logger *log.Logger
}

func NewMethodHandler(logger *log.Logger) *MethodHandler {
	return &MethodHandler{Logger: logger}
}

func (h *MethodHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apperrors.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed", nil, h.Logger)
		return
	}

	validMethods := models.GetValidMethods()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(validMethods); err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to encode response", err, h.Logger)
		return
	}
}
