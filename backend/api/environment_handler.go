package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/FedeBP/pumoide/backend/errors"
	"github.com/FedeBP/pumoide/backend/models"
	"github.com/FedeBP/pumoide/backend/utils"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type EnvironmentHandler struct {
	DefaultPath string
	Logger      *logrus.Logger
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
		errors.RespondWithError(w, http.StatusMethodNotAllowed, utils.MethodNotAllowedErr, nil, h.Logger)
	}
}

func (h *EnvironmentHandler) getEnvironments(w http.ResponseWriter) {
	files, err := filepath.Glob(filepath.Join(h.DefaultPath, "*.json"))
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToReadEnvironmentErr, err, h.Logger)
		return
	}

	var environments []models.Environment
	for _, file := range files {
		environment, err := models.LoadEnvironment(h.DefaultPath, filepath.Base(file[:len(file)-5]))
		if err != nil {
			h.Logger.Printf(utils.FailedToLoadEnvironmentErr+" %s: %v", file, err)
			continue
		}
		environments = append(environments, *environment)
	}

	if len(environments) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set(utils.ContentType, utils.AppJson)
	if err := json.NewEncoder(w).Encode(environments); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToWriteResponseErr, err, h.Logger)
		return
	}
}

func (h *EnvironmentHandler) createEnvironment(w http.ResponseWriter, r *http.Request) {
	var environment models.Environment
	err := json.NewDecoder(r.Body).Decode(&environment)
	if err != nil {
		errors.RespondWithError(w, http.StatusBadRequest, utils.FailedToReadEnvironmentErr, err, h.Logger)
		return
	}

	environment.ID = uuid.New().String()
	err = environment.Save(h.DefaultPath)
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToSaveEnvironmentErr, err, h.Logger)
		return
	}

	w.Header().Set(utils.ContentType, utils.AppJson)
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(environment)
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToWriteResponseErr, err, h.Logger)
		return
	}
}

func (h *EnvironmentHandler) updateEnvironment(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get(utils.ID)
	if id == utils.EmptyString {
		errors.RespondWithError(w, http.StatusBadRequest, utils.EnvironmentIdRequiredErr, nil, h.Logger)
		return
	}

	var updatedEnvironment models.Environment
	err := json.NewDecoder(r.Body).Decode(&updatedEnvironment)
	if err != nil {
		errors.RespondWithError(w, http.StatusBadRequest, utils.FailedToReadEnvironmentErr, err, h.Logger)
		return
	}

	existingEnvironment, err := models.LoadEnvironment(h.DefaultPath, id)
	if err != nil {
		errors.RespondWithError(w, http.StatusNotFound, utils.EnvironmentNotFoundErr, err, h.Logger)
		return
	}

	existingEnvironment.Name = updatedEnvironment.Name
	existingEnvironment.Variables = updatedEnvironment.Variables

	err = existingEnvironment.Save(h.DefaultPath)
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToSaveEnvironmentErr, err, h.Logger)
		return
	}

	w.Header().Set(utils.ContentType, utils.AppJson)
	err = json.NewEncoder(w).Encode(existingEnvironment)
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToWriteResponseErr, err, h.Logger)
		return
	}
}

func (h *EnvironmentHandler) deleteEnvironment(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get(utils.ID)
	if id == utils.EmptyString {
		errors.RespondWithError(w, http.StatusBadRequest, utils.EnvironmentIdRequiredErr, nil, h.Logger)
		return
	}

	filePath := filepath.Join(h.DefaultPath, id+".json")
	err := os.Remove(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			errors.RespondWithError(w, http.StatusNotFound, utils.EnvironmentNotFoundErr, err, h.Logger)
		} else {
			errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToDeleteEnvironmentErr, err, h.Logger)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
	_, err = w.Write([]byte(utils.EnvironmentDeletedSuccess))
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToWriteResponseErr, err, h.Logger)
	}
}
