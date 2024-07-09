package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/FedeBP/pumoide/backend/internal/domain"
	"github.com/FedeBP/pumoide/backend/pkg/constants"
	"github.com/FedeBP/pumoide/backend/pkg/errors"
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
		errors.RespondWithError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed, nil, h.Logger)
	}
}

func (h *EnvironmentHandler) getEnvironments(w http.ResponseWriter) {
	files, err := filepath.Glob(filepath.Join(h.DefaultPath, "*.json"))
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToReadEnvironment, err, h.Logger)
		return
	}

	var environments []domain.Environment
	for _, file := range files {
		environment, err := domain.LoadEnvironment(h.DefaultPath, filepath.Base(file[:len(file)-5]))
		if err != nil {
			h.Logger.Printf(constants.ErrFailedToLoadEnvironment+" %s: %v", file, err)
			continue
		}
		environments = append(environments, *environment)
	}

	if len(environments) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set(constants.ContentType, constants.AppJson)
	if err := json.NewEncoder(w).Encode(environments); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err, h.Logger)
		return
	}
}

func (h *EnvironmentHandler) createEnvironment(w http.ResponseWriter, r *http.Request) {
	var environment domain.Environment
	err := json.NewDecoder(r.Body).Decode(&environment)
	if err != nil {
		errors.RespondWithError(w, http.StatusBadRequest, constants.ErrFailedToReadEnvironment, err, h.Logger)
		return
	}

	environment.ID = uuid.New().String()
	err = environment.Save(h.DefaultPath)
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToSaveEnvironment, err, h.Logger)
		return
	}

	w.Header().Set(constants.ContentType, constants.AppJson)
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(environment)
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err, h.Logger)
		return
	}
}

func (h *EnvironmentHandler) updateEnvironment(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get(constants.ID)
	if id == constants.EmptyString {
		errors.RespondWithError(w, http.StatusBadRequest, constants.ErrEnvironmentIDRequired, nil, h.Logger)
		return
	}

	var updatedEnvironment domain.Environment
	err := json.NewDecoder(r.Body).Decode(&updatedEnvironment)
	if err != nil {
		errors.RespondWithError(w, http.StatusBadRequest, constants.ErrFailedToReadEnvironment, err, h.Logger)
		return
	}

	existingEnvironment, err := domain.LoadEnvironment(h.DefaultPath, id)
	if err != nil {
		errors.RespondWithError(w, http.StatusNotFound, constants.ErrEnvironmentNotFound, err, h.Logger)
		return
	}

	existingEnvironment.Name = updatedEnvironment.Name
	existingEnvironment.Variables = updatedEnvironment.Variables

	err = existingEnvironment.Save(h.DefaultPath)
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToSaveEnvironment, err, h.Logger)
		return
	}

	w.Header().Set(constants.ContentType, constants.AppJson)
	err = json.NewEncoder(w).Encode(existingEnvironment)
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err, h.Logger)
		return
	}
}

func (h *EnvironmentHandler) deleteEnvironment(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get(constants.ID)
	if id == constants.EmptyString {
		errors.RespondWithError(w, http.StatusBadRequest, constants.ErrEnvironmentIDRequired, nil, h.Logger)
		return
	}

	filePath := filepath.Join(h.DefaultPath, id+".json")
	err := os.Remove(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			errors.RespondWithError(w, http.StatusNotFound, constants.ErrEnvironmentNotFound, err, h.Logger)
		} else {
			errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToDeleteEnvironment, err, h.Logger)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
	_, err = w.Write([]byte(constants.EnvironmentDeletedSuccess))
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err, h.Logger)
	}
}
