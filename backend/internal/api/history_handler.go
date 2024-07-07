package api

import (
	"encoding/json"
	"github.com/FedeBP/pumoide/backend/internal/utils"
	"net/http"
	"os"
	"strconv"

	"github.com/FedeBP/pumoide/backend/internal/models"
	"github.com/FedeBP/pumoide/backend/pkg/constants"
	"github.com/FedeBP/pumoide/backend/pkg/errors"
	"github.com/sirupsen/logrus"
)

type HistoryHandler struct {
	HistoryManager *models.History
	Logger         *logrus.Logger
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
			errors.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidAction, nil, h.Logger)
		}
	default:
		errors.RespondWithError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed, nil, h.Logger)
	}
}

func (h *HistoryHandler) getEntries(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	entries, err := h.HistoryManager.GetEntries(page, pageSize)
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToGetHistory, err, h.Logger)
		return
	}

	w.Header().Set(constants.ContentType, constants.AppJson)
	err = json.NewEncoder(w).Encode(entries)
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err, h.Logger)
		return
	}
}

func (h *HistoryHandler) getEntry(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get(constants.ID)
	if id == "" {
		errors.RespondWithError(w, http.StatusBadRequest, "Missing history entry ID", nil, h.Logger)
		return
	}

	if err := utils.EnsureDir(h.HistoryManager.GetBasePath()); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, "Failed to create history directory", err, h.Logger)
		return
	}

	entry, err := h.HistoryManager.GetEntry(id)
	if err != nil {
		errors.RespondWithError(w, http.StatusNotFound, "History entry not found", err, h.Logger)
		return
	}

	w.Header().Set(constants.ContentType, constants.AppJson)
	err = json.NewEncoder(w).Encode(entry)
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err, h.Logger)
		return
	}
}

func (h *HistoryHandler) deleteEntry(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get(constants.ID)
	if id == "" {
		errors.RespondWithError(w, http.StatusBadRequest, "Missing history entry ID", nil, h.Logger)
		return
	}

	err := h.HistoryManager.DeleteEntry(id)
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, "Failed to delete history entry", err, h.Logger)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *HistoryHandler) clearHistory(w http.ResponseWriter) {
	h.HistoryManager.GetMutex().Lock()
	defer h.HistoryManager.GetMutex().Unlock()

	if err := os.RemoveAll(h.HistoryManager.GetBasePath()); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToDeleteHistory, err, h.Logger)
		return
	}

	err := utils.EnsureDir(h.HistoryManager.GetBasePath())
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, "Failed to clear history", err, h.Logger)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
