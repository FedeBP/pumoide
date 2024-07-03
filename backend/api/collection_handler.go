package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/FedeBP/pumoide/backend/errors"
	"github.com/FedeBP/pumoide/backend/models"
	"github.com/FedeBP/pumoide/backend/utils"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type CollectionHandler struct {
	DefaultPath string
	Logger      *logrus.Logger
}

func (h *CollectionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get(utils.Action)
	switch r.Method {
	case http.MethodGet:
		if action == utils.ActionExport {
			h.exportCollection(w, r)
		} else {
			h.getCollections(w, r)
		}
	case http.MethodPost:
		if action == utils.ActionImport {
			h.importCollection(w, r)
		} else {
			h.createCollection(w, r)
		}
	case http.MethodPut:
		switch action {
		case utils.ActionAddRequest:
			h.addRequestToCollection(w, r)
		case utils.ActionUpdateCollection:
			h.updateCollection(w, r)
		default:
			errors.RespondWithError(w, http.StatusBadRequest, utils.InvalidActionErr, nil, h.Logger)
		}
	case http.MethodDelete:
		switch action {
		case utils.ActionDeleteCollection:
			h.deleteCollection(w, r)
		case utils.ActionDeleteRequest:
			h.deleteRequestFromCollection(w, r)
		default:
			errors.RespondWithError(w, http.StatusBadRequest, utils.InvalidActionErr, nil, h.Logger)
		}
	default:
		errors.RespondWithError(w, http.StatusMethodNotAllowed, utils.MethodNotAllowedErr, nil, h.Logger)
	}
}

func (h *CollectionHandler) getCollectionPath(r *http.Request) string {
	path := r.URL.Query().Get(utils.Path)
	if path == utils.EmptyString {
		return h.DefaultPath
	}
	return path
}

// GET methods

func (h *CollectionHandler) getCollections(w http.ResponseWriter, r *http.Request) {
	collectionPath := h.getCollectionPath(r)

	files, err := filepath.Glob(filepath.Join(collectionPath, "*.json"))
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToReadCollectionErr, err, h.Logger)
		return
	}

	var collections []models.Collection
	for _, file := range files {
		collection, err := models.LoadCollection(collectionPath, filepath.Base(file[:len(file)-5]))
		if err != nil {
			h.Logger.Printf(utils.FailedToLoadCollectionErr+" %s: %v", file, err)
			continue
		}
		collections = append(collections, *collection)
	}

	if len(collections) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set(utils.ContentType, utils.AppJson)
	if err := json.NewEncoder(w).Encode(collections); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToWriteResponseErr, err, h.Logger)
		return
	}
}

func (h *CollectionHandler) exportCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get(utils.ID)
	if collectionID == utils.EmptyString {
		errors.RespondWithError(w, http.StatusBadRequest, utils.CollectionIdRequiredErr, nil, h.Logger)
		return
	}

	collection, err := models.LoadCollection(h.getCollectionPath(r), collectionID)
	if err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToLoadCollectionErr, err, h.Logger)
		return
	}

	exportedCollection := collection.ToExportedCollection()

	w.Header().Set(utils.ContentType, utils.AppJson)
	w.Header().Set(utils.ContentDisposition, fmt.Sprintf("attachment; filename=%s.json", collection.Name))
	if err := json.NewEncoder(w).Encode(exportedCollection); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToWriteResponseErr, err, h.Logger)
	}
}

// POST methods

func (h *CollectionHandler) createCollection(w http.ResponseWriter, r *http.Request) {
	var collection models.Collection
	if err := json.NewDecoder(r.Body).Decode(&collection); err != nil {
		errors.RespondWithError(w, http.StatusBadRequest, utils.FailedToReadCollectionErr, err, h.Logger)
		return
	}

	for i := range collection.Requests {
		if collection.Requests[i].ID == utils.EmptyString {
			collection.Requests[i].ID = uuid.New().String()
		}
		if !collection.Requests[i].Method.IsValid() {
			message := fmt.Sprintf(utils.InvalidHTTPMethodErr+" '%s': %s", collection.Requests[i].Name, collection.Requests[i].Method)
			errors.RespondWithError(w, http.StatusBadRequest, message, nil, h.Logger)
			return
		}
	}

	collection.ID = uuid.New().String()

	savePath := h.getCollectionPath(r)
	if err := utils.EnsureDir(savePath); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToCreateDirErr, err, h.Logger)
		return
	}

	if err := collection.Save(savePath); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToSaveDirErr, err, h.Logger)
		return
	}

	w.Header().Set(utils.ContentType, utils.AppJson)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(collection); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToWriteResponseErr, err, h.Logger)
	}
}

func (h *CollectionHandler) importCollection(w http.ResponseWriter, r *http.Request) {
	var importedCollection models.ImportedCollection
	if err := json.NewDecoder(r.Body).Decode(&importedCollection); err != nil {
		errors.RespondWithError(w, http.StatusBadRequest, utils.FailedToReadCollectionErr, err, h.Logger)
		return
	}

	newCollection, _ := models.NewCollectionFromImported(importedCollection)

	for _, req := range newCollection.Requests {
		if !req.Method.IsValid() {
			var message = fmt.Sprintf(utils.InvalidHTTPMethodErr+" '%s': %s", req.Name, req.Method)
			errors.RespondWithError(w, http.StatusBadRequest, message, nil, h.Logger)
			return
		}
	}

	collectionPath := h.getCollectionPath(r)
	if err := newCollection.Save(collectionPath); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToSaveCollectionErr, err, h.Logger)
		return
	}

	w.Header().Set(utils.ContentType, utils.AppJson)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(newCollection); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToWriteResponseErr, err, h.Logger)
	}
}

// PUT methods

func (h *CollectionHandler) updateCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get(utils.ID)
	if collectionID == utils.EmptyString {
		errors.RespondWithError(w, http.StatusBadRequest, utils.CollectionIdRequiredErr, nil, h.Logger)
		return
	}

	var updatedCollection models.Collection
	if err := json.NewDecoder(r.Body).Decode(&updatedCollection); err != nil {
		errors.RespondWithError(w, http.StatusBadRequest, utils.FailedToReadCollectionErr, err, h.Logger)
		return
	}

	for _, req := range updatedCollection.Requests {
		if !req.Method.IsValid() {
			var message = fmt.Sprintf(utils.InvalidHTTPMethodErr+" '%s': %s", req.Name, req.Method)
			errors.RespondWithError(w, http.StatusBadRequest, message, nil, h.Logger)
			return
		}
	}

	collectionPath := h.getCollectionPath(r)

	existingCollection, err := models.LoadCollection(collectionPath, collectionID)
	if err != nil {
		errors.RespondWithError(w, http.StatusNotFound, utils.FailedToLoadCollectionErr, err, h.Logger)
		return
	}

	existingCollection.Name = updatedCollection.Name
	existingCollection.Description = updatedCollection.Description
	existingCollection.Requests = updatedCollection.Requests

	if err := existingCollection.Save(collectionPath); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToSaveCollectionErr, err, h.Logger)
		return
	}

	w.Header().Set(utils.ContentType, utils.AppJson)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(existingCollection); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToWriteResponseErr, err, h.Logger)
	}
}

func (h *CollectionHandler) addRequestToCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get(utils.ID)
	if collectionID == utils.EmptyString {
		errors.RespondWithError(w, http.StatusBadRequest, utils.CollectionIdRequiredErr, nil, h.Logger)
		return
	}

	var request models.Request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		errors.RespondWithError(w, http.StatusBadRequest, utils.InvalidRequestBodyErr, err, h.Logger)
		return
	}

	if !request.Method.IsValid() {
		var message = fmt.Sprintf(utils.InvalidHTTPMethodErr+" '%s': %s", request.Name, request.Method)
		errors.RespondWithError(w, http.StatusBadRequest, message, nil, h.Logger)
		return
	}

	request.ID = uuid.New().String()

	collectionPath := h.getCollectionPath(r)

	collection, err := models.LoadCollection(collectionPath, collectionID)
	if err != nil {
		errors.RespondWithError(w, http.StatusNotFound, utils.FailedToLoadCollectionErr, err, h.Logger)
		return
	}

	err = collection.AddRequest(request)
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToSaveRequestErr, err, h.Logger)
		return
	}
	if err := collection.Save(collectionPath); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToSaveCollectionErr, err, h.Logger)
		return
	}

	w.Header().Set(utils.ContentType, utils.AppJson)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(request); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToEncodeRequestErr, err, h.Logger)
	}
}

// DELETE methods

func (h *CollectionHandler) deleteCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get(utils.ID)
	if collectionID == utils.EmptyString {
		errors.RespondWithError(w, http.StatusBadRequest, utils.CollectionIdRequiredErr, nil, h.Logger)
		return
	}

	collectionPath := h.getCollectionPath(r)
	filePath := filepath.Join(collectionPath, collectionID+".json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		errors.RespondWithError(w, http.StatusNotFound, utils.CollectionNotFoundErr, err, h.Logger)
		return
	}

	if err := os.Remove(filePath); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToDeleteCollectionErr, err, h.Logger)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	if _, err := w.Write([]byte(utils.CollectionDeletedSuccess)); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToWriteResponseErr, err, h.Logger)
	}
}

func (h *CollectionHandler) deleteRequestFromCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get(utils.CollectionID)
	requestID := r.URL.Query().Get(utils.RequestID)
	if collectionID == utils.EmptyString || requestID == utils.EmptyString {
		errors.RespondWithError(w, http.StatusBadRequest, utils.CollectionIdRequiredErr, nil, h.Logger)
		return
	}

	collectionPath := h.getCollectionPath(r)

	collection, err := models.LoadCollection(collectionPath, collectionID)
	if err != nil {
		errors.RespondWithError(w, http.StatusNotFound, utils.FailedToLoadCollectionErr, err, h.Logger)
		return
	}

	if !collection.RemoveRequest(requestID) {
		errors.RespondWithError(w, http.StatusNotFound, utils.RequestNotFoundErr, err, h.Logger)
		return
	}

	if err := collection.Save(collectionPath); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToSaveCollectionErr, err, h.Logger)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	if _, err := w.Write([]byte(utils.RequestDeletedSuccess)); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToWriteResponseErr, err, h.Logger)
	}
}
