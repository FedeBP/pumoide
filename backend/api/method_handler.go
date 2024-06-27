package api

import (
	"encoding/json"
	"net/http"

	"github.com/FedeBP/pumoide/backend/models"
)

type MethodHandler struct{}

func NewMethodHandler() *MethodHandler {
	return &MethodHandler{}
}

func (h *MethodHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	validMethods := models.GetValidMethods()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(validMethods); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
