package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/FedeBP/pumoide/backend/models"
	"github.com/google/uuid"
)

type EnvironmentHandler struct {
	DefaultPath string
}

func NewEnvironmentHandler(defaultPath string) *EnvironmentHandler {
	return &EnvironmentHandler{DefaultPath: defaultPath}
}

func (h *EnvironmentHandler) Handle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getEnvironments(w, r)
	case http.MethodPost:
		h.createEnvironment(w, r)
	case http.MethodPut:
		h.updateEnvironment(w, r)
	case http.MethodDelete:
		h.deleteEnvironment(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *EnvironmentHandler) getEnvironments(w http.ResponseWriter, r *http.Request) {
	files, err := filepath.Glob(filepath.Join(h.DefaultPath, "*.json"))
	if err != nil {
		http.Error(w, "Failed to read environments", http.StatusInternalServerError)
		return
	}

	var environments []models.Environment
	for _, file := range files {
		environment, err := models.LoadEnvironment(h.DefaultPath, filepath.Base(file[:len(file)-5]))
		if err != nil {
			continue
		}
		environments = append(environments, *environment)
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(environments)
	if err != nil {
		http.Error(w, "Failed to encode environments: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *EnvironmentHandler) createEnvironment(w http.ResponseWriter, r *http.Request) {
	var environment models.Environment
	err := json.NewDecoder(r.Body).Decode(&environment)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	environment.ID = uuid.New().String()
	err = environment.Save(h.DefaultPath)
	if err != nil {
		http.Error(w, "Failed to save environment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(environment)
	if err != nil {
		http.Error(w, "Failed to encode environment: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *EnvironmentHandler) updateEnvironment(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Environment ID is required", http.StatusBadRequest)
		return
	}

	var updatedEnvironment models.Environment
	err := json.NewDecoder(r.Body).Decode(&updatedEnvironment)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	existingEnvironment, err := models.LoadEnvironment(h.DefaultPath, id)
	if err != nil {
		http.Error(w, "Environment not found", http.StatusNotFound)
		return
	}

	existingEnvironment.Name = updatedEnvironment.Name
	existingEnvironment.Variables = updatedEnvironment.Variables

	err = existingEnvironment.Save(h.DefaultPath)
	if err != nil {
		http.Error(w, "Failed to save updated environment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(existingEnvironment)
	if err != nil {
		http.Error(w, "Failed to encode environment: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *EnvironmentHandler) deleteEnvironment(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Environment ID is required", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(h.DefaultPath, id+".json")
	err := os.Remove(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Environment not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to delete environment", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Environment deleted successfully"))
	if err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}
