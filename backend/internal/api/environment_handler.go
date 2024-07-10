package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/FedeBP/pumoide/backend/internal/domain"
	"github.com/FedeBP/pumoide/backend/internal/services"
	"github.com/FedeBP/pumoide/backend/pkg/constants"
	customErrors "github.com/FedeBP/pumoide/backend/pkg/errors"
	"github.com/sirupsen/logrus"
)

type EnvironmentHandler struct {
	Service *services.EnvironmentService
	Logger  *logrus.Logger
}

func NewEnvironmentHandler(service *services.EnvironmentService, logger *logrus.Logger) *EnvironmentHandler {
	return &EnvironmentHandler{
		Service: service,
		Logger:  logger,
	}
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
		customErrors.RespondWithError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed, nil, h.Logger)
	}
}

func (h *EnvironmentHandler) getEnvironments(w http.ResponseWriter) {
	environments, err := h.Service.GetEnvironments()
	if err != nil {
		customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToReadEnvironment, err, h.Logger)
		return
	}

	if len(environments) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set(constants.ContentType, constants.AppJson)
	if err := json.NewEncoder(w).Encode(environments); err != nil {
		customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err, h.Logger)
		return
	}
}

func (h *EnvironmentHandler) createEnvironment(w http.ResponseWriter, r *http.Request) {
	var environment domain.Environment
	if err := json.NewDecoder(r.Body).Decode(&environment); err != nil {
		customErrors.RespondWithError(w, http.StatusBadRequest, constants.ErrFailedToReadEnvironment, err, h.Logger)
		return
	}

	if err := h.Service.CreateEnvironment(&environment); err != nil {
		customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToSaveEnvironment, err, h.Logger)
		return
	}

	w.Header().Set(constants.ContentType, constants.AppJson)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(environment); err != nil {
		customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err, h.Logger)
		return
	}
}

func (h *EnvironmentHandler) updateEnvironment(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get(constants.ID)
	if id == constants.EmptyString {
		customErrors.RespondWithError(w, http.StatusBadRequest, constants.ErrEnvironmentIDRequired, nil, h.Logger)
		return
	}

	var updatedEnvironment domain.Environment
	if err := json.NewDecoder(r.Body).Decode(&updatedEnvironment); err != nil {
		customErrors.RespondWithError(w, http.StatusBadRequest, constants.ErrFailedToReadEnvironment, err, h.Logger)
		return
	}

	err := h.Service.UpdateEnvironment(id, &updatedEnvironment)
	if err != nil {
		var appErr *customErrors.AppError
		if errors.As(err, &appErr) {
			customErrors.RespondWithError(w, appErr.Code, appErr.Message, appErr.Err, h.Logger)
		} else {
			customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToSaveEnvironment, err, h.Logger)
		}
		return
	}

	w.Header().Set(constants.ContentType, constants.AppJson)
	if err := json.NewEncoder(w).Encode(updatedEnvironment); err != nil {
		customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err, h.Logger)
		return
	}
}

func (h *EnvironmentHandler) deleteEnvironment(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get(constants.ID)
	if id == constants.EmptyString {
		customErrors.RespondWithError(w, http.StatusBadRequest, constants.ErrEnvironmentIDRequired, nil, h.Logger)
		return
	}

	err := h.Service.DeleteEnvironment(id)
	if err != nil {
		var appErr *customErrors.AppError
		if errors.As(err, &appErr) {
			customErrors.RespondWithError(w, appErr.Code, appErr.Message, appErr.Err, h.Logger)
		} else {
			customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToDeleteEnvironment, err, h.Logger)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
