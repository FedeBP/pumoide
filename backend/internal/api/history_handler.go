package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/FedeBP/pumoide/backend/internal/models"
	"github.com/FedeBP/pumoide/backend/pkg/constants"
	customErrors "github.com/FedeBP/pumoide/backend/pkg/errors"
	"github.com/sirupsen/logrus"
)

type HistoryHandler struct {
	Service *models.History
	Logger  *logrus.Logger
}

func NewHistoryHandler(service *models.History, logger *logrus.Logger) *HistoryHandler {
	return &HistoryHandler{
		Service: service,
		Logger:  logger,
	}
}

func (h *HistoryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get(constants.Action)

	switch r.Method {
	case http.MethodGet:
		switch action {
		case constants.ActionEntry:
			h.getEntry(w, r)
		default:
			h.getEntries(w, r)
		}
	case http.MethodDelete:
		switch action {
		case constants.ActionClearHistory:
			h.clearHistory(w)
		case constants.ActionDeleteEntry:
			h.deleteEntry(w, r)
		default:
			customErrors.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidAction, nil, h.Logger)
		}
	default:
		customErrors.RespondWithError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed, nil, h.Logger)
	}
}

func (h *HistoryHandler) getEntries(w http.ResponseWriter, r *http.Request) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	entries, err := h.Service.GetEntries(page, pageSize)
	if err != nil {
		customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToGetHistory, err, h.Logger)
		return
	}

	w.Header().Set(constants.ContentType, constants.AppJson)
	if err := json.NewEncoder(w).Encode(entries); err != nil {
		customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err, h.Logger)
		return
	}
}

func (h *HistoryHandler) getEntry(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get(constants.ID)
	if id == "" {
		customErrors.RespondWithError(w, http.StatusBadRequest, "Missing history entry ID", nil, h.Logger)
		return
	}

	entry, err := h.Service.GetEntry(id)
	if err != nil {
		var appErr *customErrors.AppError
		if errors.As(err, &appErr) {
			customErrors.RespondWithError(w, appErr.Code, appErr.Message, appErr.Err, h.Logger)
		} else {
			customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToReadResponse, err, h.Logger)
		}
		return
	}

	w.Header().Set(constants.ContentType, constants.AppJson)
	if err := json.NewEncoder(w).Encode(entry); err != nil {
		customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err, h.Logger)
		return
	}
}

func (h *HistoryHandler) deleteEntry(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get(constants.ID)
	if id == "" {
		customErrors.RespondWithError(w, http.StatusBadRequest, "Missing history entry ID", nil, h.Logger)
		return
	}

	err := h.Service.DeleteEntry(id)
	if err != nil {
		var appErr *customErrors.AppError
		if errors.As(err, &appErr) {
			customErrors.RespondWithError(w, appErr.Code, appErr.Message, appErr.Err, h.Logger)
		} else {
			customErrors.RespondWithError(w, http.StatusInternalServerError, "Failed to delete history entry", err, h.Logger)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *HistoryHandler) clearHistory(w http.ResponseWriter) {
	err := h.Service.ClearHistory()
	if err != nil {
		var appErr *customErrors.AppError
		if errors.As(err, &appErr) {
			customErrors.RespondWithError(w, appErr.Code, appErr.Message, appErr.Err, h.Logger)
		} else {
			customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToDeleteHistory, err, h.Logger)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
