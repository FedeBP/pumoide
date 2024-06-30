package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/FedeBP/pumoide/backend/apperrors"
	"github.com/FedeBP/pumoide/backend/models"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type EnvironmentHandler struct {
	DefaultPath string
	Logger      *logrus.Logger
}

func NewEnvironmentHandler(defaultPath string, logger *logrus.Logger) *EnvironmentHandler {
	return &EnvironmentHandler{DefaultPath: defaultPath, Logger: logger}
}

func (h *EnvironmentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getEnvironments(w)
	case http.MethodPost:
		h.createEnvironment(w, r)
	case http.MethodPut:
		h.updateEnvironment(w, r)
	case http.MethodDelete:
		h.deleteEnvironment(w, r)
	default:
		apperrors.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed", nil, h.Logger)
	}
}

func (h *EnvironmentHandler) getEnvironments(w http.ResponseWriter) {
	files, err := filepath.Glob(filepath.Join(h.DefaultPath, "*.json"))
	if err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to read environments", err, h.Logger)
		return
	}

	var environments []models.Environment
	for _, file := range files {
		environment, err := models.LoadEnvironment(h.DefaultPath, filepath.Base(file[:len(file)-5]))
		if err != nil {
			h.Logger.Printf("Failed to load environment %s: %v", file, err)
			continue
		}
		environments = append(environments, *environment)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(environments); err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to encode environments", err, h.Logger)
		return
	}
}

func (h *EnvironmentHandler) createEnvironment(w http.ResponseWriter, r *http.Request) {
	var environment models.Environment
	err := json.NewDecoder(r.Body).Decode(&environment)
	if err != nil {
		apperrors.RespondWithError(w, http.StatusBadRequest, "Failed to parse environment", err, h.Logger)
		return
	}

	environment.ID = uuid.New().String()
	err = environment.Save(h.DefaultPath)
	if err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to save environment", err, h.Logger)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(environment)
	if err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to encode environment", err, h.Logger)
		return
	}
}

func (h *EnvironmentHandler) updateEnvironment(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		apperrors.RespondWithError(w, http.StatusBadRequest, "Environment ID is required", nil, h.Logger)
		return
	}

	var updatedEnvironment models.Environment
	err := json.NewDecoder(r.Body).Decode(&updatedEnvironment)
	if err != nil {
		apperrors.RespondWithError(w, http.StatusBadRequest, "Failed to parse updated environment", err, h.Logger)
		return
	}

	existingEnvironment, err := models.LoadEnvironment(h.DefaultPath, id)
	if err != nil {
		apperrors.RespondWithError(w, http.StatusNotFound, "Environment not found", err, h.Logger)
		return
	}

	existingEnvironment.Name = updatedEnvironment.Name
	existingEnvironment.Variables = updatedEnvironment.Variables

	err = existingEnvironment.Save(h.DefaultPath)
	if err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to save updated environment", err, h.Logger)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(existingEnvironment)
	if err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to encode environment", err, h.Logger)
		return
	}
}

func (h *EnvironmentHandler) deleteEnvironment(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		apperrors.RespondWithError(w, http.StatusBadRequest, "Environment ID is required", nil, h.Logger)
		return
	}

	filePath := filepath.Join(h.DefaultPath, id+".json")
	err := os.Remove(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			apperrors.RespondWithError(w, http.StatusNotFound, "Environment not found", err, h.Logger)
		} else {
			apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to delete environment", err, h.Logger)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Environment deleted successfully"))
	if err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to write response", err, h.Logger)
	}
}
