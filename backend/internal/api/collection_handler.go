package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/FedeBP/pumoide/backend/internal/services"
	"github.com/FedeBP/pumoide/backend/pkg/constants"
	customErrors "github.com/FedeBP/pumoide/backend/pkg/errors"
	"github.com/sirupsen/logrus"
)

type CollectionHandler struct {
	Service *services.CollectionService
	Logger  *logrus.Logger
}

func NewCollectionHandler(service *services.CollectionService, logger *logrus.Logger) *CollectionHandler {
	return &CollectionHandler{
		Service: service,
		Logger:  logger,
	}
}

func (h *CollectionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get(constants.Action)
	switch r.Method {
	case http.MethodGet:
		if action == constants.ActionExport {
			h.exportCollection(w, r)
		} else {
			h.getCollections(w)
		}
	case http.MethodPost:
		if action == constants.ActionImport {
			h.importCollection(w, r)
		} else {
			h.createCollection(w, r)
		}
	case http.MethodPut:
		switch action {
		case constants.ActionAddRequest:
			h.addRequestToCollection(w, r)
		case constants.ActionUpdateCollection:
			h.updateCollection(w, r)
		default:
			customErrors.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidAction, nil, h.Logger)
		}
	case http.MethodDelete:
		switch action {
		case constants.ActionDeleteCollection:
			h.deleteCollection(w, r)
		case constants.ActionDeleteRequest:
			h.deleteRequestFromCollection(w, r)
		default:
			customErrors.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidAction, nil, h.Logger)
		}
	default:
		customErrors.RespondWithError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed, nil, h.Logger)
	}
}

// GET methods

func (h *CollectionHandler) getCollections(w http.ResponseWriter) {
	collections, err := h.Service.GetCollections()
	if err != nil {
		customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToReadCollection, err, h.Logger)
		return
	}

	if len(collections) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set(constants.ContentType, constants.AppJson)
	if err := json.NewEncoder(w).Encode(collections); err != nil {
		customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err, h.Logger)
		return
	}
}

func (h *CollectionHandler) exportCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get(constants.ID)
	if collectionID == constants.EmptyString {
		customErrors.RespondWithError(w, http.StatusBadRequest, constants.ErrCollectionIDRequired, nil, h.Logger)
		return
	}

	exportedCollection, err := h.Service.ExportCollection(collectionID)
	if err != nil {
		var appErr *customErrors.AppError
		if errors.As(err, &appErr) {
			if appErr.Code == http.StatusNoContent {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			customErrors.RespondWithError(w, appErr.Code, appErr.Message, appErr.Err, h.Logger)
		} else {
			customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToReadResponse, err, h.Logger)
		}
		return
	}

	w.Header().Set(constants.ContentType, constants.AppJson)
	w.Header().Set(constants.ContentDisposition, fmt.Sprintf("attachment; filename=%s.json", exportedCollection.Info.Name))

	if err := json.NewEncoder(w).Encode(exportedCollection); err != nil {
		customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err, h.Logger)
		return
	}
}

// POST methods

func (h *CollectionHandler) createCollection(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		customErrors.RespondWithError(w, http.StatusBadRequest, constants.ErrFailedToReadCollection, err, h.Logger)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToCloseBody, err, h.Logger)
		}
	}(r.Body)

	collection, err := h.Service.CreateCollection(body)
	if err != nil {
		customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToSaveCollection, err, h.Logger)
		return
	}

	w.Header().Set(constants.ContentType, constants.AppJson)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(collection); err != nil {
		customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err, h.Logger)
	}
}

func (h *CollectionHandler) importCollection(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		customErrors.RespondWithError(w, http.StatusBadRequest, constants.ErrFailedToReadCollection, err, h.Logger)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToCloseBody, err, h.Logger)
		}
	}(r.Body)

	newCollection, err := h.Service.ImportCollection(body)
	if err != nil {
		var appErr *customErrors.AppError
		if errors.As(err, &appErr) {
			customErrors.RespondWithError(w, appErr.Code, appErr.Message, appErr.Err, h.Logger)
		} else {
			customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToSaveCollection, err, h.Logger)
		}
		return
	}

	w.Header().Set(constants.ContentType, constants.AppJson)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(newCollection); err != nil {
		customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err, h.Logger)
		return
	}
}

// PUT methods

func (h *CollectionHandler) updateCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get(constants.ID)
	if collectionID == constants.EmptyString {
		customErrors.RespondWithError(w, http.StatusBadRequest, constants.ErrCollectionIDRequired, nil, h.Logger)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		customErrors.RespondWithError(w, http.StatusBadRequest, constants.ErrFailedToReadCollection, err, h.Logger)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToCloseBody, err, h.Logger)
		}
	}(r.Body)

	updatedCollection, err := h.Service.UpdateCollection(collectionID, body)
	if err != nil {
		customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToSaveCollection, err, h.Logger)
		return
	}

	w.Header().Set(constants.ContentType, constants.AppJson)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(updatedCollection); err != nil {
		customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err, h.Logger)
	}
}

func (h *CollectionHandler) addRequestToCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get(constants.ID)
	if collectionID == constants.EmptyString {
		customErrors.RespondWithError(w, http.StatusBadRequest, constants.ErrCollectionIDRequired, nil, h.Logger)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		customErrors.RespondWithError(w, http.StatusBadRequest, constants.ErrFailedToReadRequestBody, err, h.Logger)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToCloseBody, err, h.Logger)
		}
	}(r.Body)

	newRequest, err := h.Service.AddRequestToCollection(collectionID, body)
	if err != nil {
		var appErr *customErrors.AppError
		if errors.As(err, &appErr) {
			customErrors.RespondWithError(w, appErr.Code, appErr.Message, appErr.Err, h.Logger)
		} else {
			customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToSaveRequest, err, h.Logger)
		}
		return
	}

	w.Header().Set(constants.ContentType, constants.AppJson)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(newRequest); err != nil {
		customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToEncodeRequest, err, h.Logger)
	}
}

// DELETE methods

func (h *CollectionHandler) deleteCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get(constants.ID)
	if collectionID == constants.EmptyString {
		customErrors.RespondWithError(w, http.StatusBadRequest, constants.ErrCollectionIDRequired, nil, h.Logger)
		return
	}

	if err := h.Service.DeleteCollection(collectionID); err != nil {
		customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToDeleteCollection, err, h.Logger)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CollectionHandler) deleteRequestFromCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get(constants.CollectionID)
	requestID := r.URL.Query().Get(constants.RequestID)
	if collectionID == constants.EmptyString || requestID == constants.EmptyString {
		customErrors.RespondWithError(w, http.StatusBadRequest, constants.ErrCollectionIDRequired, nil, h.Logger)
		return
	}

	if err := h.Service.DeleteRequestFromCollection(collectionID, requestID); err != nil {
		customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToDeleteRequest, err, h.Logger)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
