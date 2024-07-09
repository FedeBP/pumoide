package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/FedeBP/pumoide/backend/internal/domain"
	"github.com/FedeBP/pumoide/backend/internal/models"
	"github.com/FedeBP/pumoide/backend/internal/utils"
	"github.com/FedeBP/pumoide/backend/pkg/constants"
	"github.com/FedeBP/pumoide/backend/pkg/errors"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type CollectionHandler struct {
	DefaultPath string
	Logger      *logrus.Logger
}

func (h *CollectionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get(constants.Action)
	switch r.Method {
	case http.MethodGet:
		if action == constants.ActionExport {
			h.exportCollection(w, r)
		} else {
			h.getCollections(w, r)
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
			errors.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidAction, nil, h.Logger)
		}
	case http.MethodDelete:
		switch action {
		case constants.ActionDeleteCollection:
			h.deleteCollection(w, r)
		case constants.ActionDeleteRequest:
			h.deleteRequestFromCollection(w, r)
		default:
			errors.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidAction, nil, h.Logger)
		}
	default:
		errors.RespondWithError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed, nil, h.Logger)
	}
}

func (h *CollectionHandler) getCollectionPath(r *http.Request) string {
	path := r.URL.Query().Get(constants.Path)
	if path == constants.EmptyString {
		return h.DefaultPath
	}
	return path
}

// GET methods

func (h *CollectionHandler) getCollections(w http.ResponseWriter, r *http.Request) {
	collectionPath := h.getCollectionPath(r)

	files, err := filepath.Glob(filepath.Join(collectionPath, "*.json"))
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToReadCollection, err, h.Logger)
		return
	}

	var collections []models.Collection
	for _, file := range files {
		collection, err := models.LoadCollection(collectionPath, filepath.Base(file[:len(file)-5]))
		if err != nil {
			h.Logger.Printf(constants.ErrFailedToLoadCollection+" %s: %v", file, err)
			continue
		}
		collections = append(collections, *collection)
	}

	if len(collections) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set(constants.ContentType, constants.AppJson)
	if err := json.NewEncoder(w).Encode(collections); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err, h.Logger)
		return
	}
}

func (h *CollectionHandler) exportCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get(constants.ID)
	if collectionID == constants.EmptyString {
		errors.RespondWithError(w, http.StatusBadRequest, constants.ErrCollectionIDRequired, nil, h.Logger)
		return
	}

	collection, err := models.LoadCollection(h.getCollectionPath(r), collectionID)
	if err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToLoadCollection, err, h.Logger)
		return
	}

	exportedCollection := collection.ToExportedCollection()

	w.Header().Set(constants.ContentType, constants.AppJson)
	w.Header().Set(constants.ContentDisposition, fmt.Sprintf("attachment; filename=%s.json", collection.Name))
	if err := json.NewEncoder(w).Encode(exportedCollection); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err, h.Logger)
	}
}

// POST methods

func (h *CollectionHandler) createCollection(w http.ResponseWriter, r *http.Request) {
	var collection models.Collection
	if err := json.NewDecoder(r.Body).Decode(&collection); err != nil {
		errors.RespondWithError(w, http.StatusBadRequest, constants.ErrFailedToReadCollection, err, h.Logger)
		return
	}

	if err := collection.Validate(); err != nil {
		errors.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidCollection, err, h.Logger)
		return
	}

	collection.ID = uuid.New().String()

	savePath := h.getCollectionPath(r)
	if err := utils.EnsureDir(savePath); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToCreateDir, err, h.Logger)
		return
	}

	if err := collection.Save(savePath); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToSaveDir, err, h.Logger)
		return
	}

	w.Header().Set(constants.ContentType, constants.AppJson)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(collection); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err, h.Logger)
	}
}

func (h *CollectionHandler) importCollection(w http.ResponseWriter, r *http.Request) {
	var importedCollection models.ImportedCollection
	if err := json.NewDecoder(r.Body).Decode(&importedCollection); err != nil {
		errors.RespondWithError(w, http.StatusBadRequest, constants.ErrFailedToReadCollection, err, h.Logger)
		return
	}

	newCollection, _ := models.NewCollectionFromImported(importedCollection)

	collectionPath := h.getCollectionPath(r)
	if err := newCollection.Save(collectionPath); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToSaveCollection, err, h.Logger)
		return
	}

	w.Header().Set(constants.ContentType, constants.AppJson)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(newCollection); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err, h.Logger)
	}
}

// PUT methods

func (h *CollectionHandler) updateCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get(constants.ID)
	if collectionID == constants.EmptyString {
		errors.RespondWithError(w, http.StatusBadRequest, constants.ErrCollectionIDRequired, nil, h.Logger)
		return
	}

	var updatedCollection models.Collection
	if err := json.NewDecoder(r.Body).Decode(&updatedCollection); err != nil {
		errors.RespondWithError(w, http.StatusBadRequest, constants.ErrFailedToReadCollection, err, h.Logger)
		return
	}

	for _, req := range updatedCollection.Requests {
		if err := req.Validate(); err != nil {
			errors.RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request '%s': %v", req.GetName(), err), nil, h.Logger)
			return
		}
	}

	collectionPath := h.getCollectionPath(r)

	existingCollection, err := models.LoadCollection(collectionPath, collectionID)
	if err != nil {
		errors.RespondWithError(w, http.StatusNotFound, constants.ErrFailedToLoadCollection, err, h.Logger)
		return
	}

	existingCollection.Name = updatedCollection.Name
	existingCollection.Description = updatedCollection.Description
	existingCollection.Requests = updatedCollection.Requests

	if err := existingCollection.Save(collectionPath); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToSaveCollection, err, h.Logger)
		return
	}

	w.Header().Set(constants.ContentType, constants.AppJson)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(existingCollection); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err, h.Logger)
	}
}

func (h *CollectionHandler) addRequestToCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get(constants.ID)
	if collectionID == constants.EmptyString {
		errors.RespondWithError(w, http.StatusBadRequest, constants.ErrCollectionIDRequired, nil, h.Logger)
		return
	}

	var rawRequest json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&rawRequest); err != nil {
		errors.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidRequestBody, err, h.Logger)
		return
	}

	var baseRequest struct {
		Type domain.RequestType `json:"type"`
	}
	if err := json.Unmarshal(rawRequest, &baseRequest); err != nil {
		errors.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidRequestBody, err, h.Logger)
		return
	}

	var newRequest domain.Request
	switch baseRequest.Type {
	case "rest":
		var restRequest models.RESTRequest
		if err := json.Unmarshal(rawRequest, &restRequest); err != nil {
			errors.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidRequestBody, err, h.Logger)
			return
		}
		newRequest = &restRequest
	case "websocket":
		var wsRequest models.WebSocketRequest
		if err := json.Unmarshal(rawRequest, &wsRequest); err != nil {
			errors.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidRequestBody, err, h.Logger)
			return
		}
		newRequest = &wsRequest
	default:
		errors.RespondWithError(w, http.StatusBadRequest, "Unsupported request type", nil, h.Logger)
		return
	}

	newRequest.SetID(uuid.New().String())

	collectionPath := h.getCollectionPath(r)

	collection, err := models.LoadCollection(collectionPath, collectionID)
	if err != nil {
		errors.RespondWithError(w, http.StatusNotFound, constants.ErrFailedToLoadCollection, err, h.Logger)
		return
	}

	if err := collection.AddRequest(newRequest); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToSaveRequest, err, h.Logger)
		return
	}

	if err := collection.Save(collectionPath); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToSaveCollection, err, h.Logger)
		return
	}

	w.Header().Set(constants.ContentType, constants.AppJson)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(newRequest); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToEncodeRequest, err, h.Logger)
	}
}

// DELETE methods

func (h *CollectionHandler) deleteCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get(constants.ID)
	if collectionID == constants.EmptyString {
		errors.RespondWithError(w, http.StatusBadRequest, constants.ErrCollectionIDRequired, nil, h.Logger)
		return
	}

	collectionPath := h.getCollectionPath(r)
	filePath := filepath.Join(collectionPath, collectionID+".json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		errors.RespondWithError(w, http.StatusNotFound, constants.ErrCollectionNotFound, err, h.Logger)
		return
	}

	if err := os.Remove(filePath); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToDeleteCollection, err, h.Logger)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	if _, err := w.Write([]byte(constants.CollectionDeletedSuccess)); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err, h.Logger)
	}
}

func (h *CollectionHandler) deleteRequestFromCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get(constants.CollectionID)
	requestID := r.URL.Query().Get(constants.RequestID)
	if collectionID == constants.EmptyString || requestID == constants.EmptyString {
		errors.RespondWithError(w, http.StatusBadRequest, constants.ErrCollectionIDRequired, nil, h.Logger)
		return
	}

	collectionPath := h.getCollectionPath(r)

	collection, err := models.LoadCollection(collectionPath, collectionID)
	if err != nil {
		errors.RespondWithError(w, http.StatusNotFound, constants.ErrFailedToLoadCollection, err, h.Logger)
		return
	}

	if !collection.RemoveRequest(requestID) {
		errors.RespondWithError(w, http.StatusNotFound, constants.ErrRequestNotFound, err, h.Logger)
		return
	}

	if err := collection.Save(collectionPath); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToSaveCollection, err, h.Logger)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	if _, err := w.Write([]byte(constants.RequestDeletedSuccess)); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err, h.Logger)
	}
}
