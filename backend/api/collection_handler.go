package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/FedeBP/pumoide/backend/apperrors"
	"github.com/FedeBP/pumoide/backend/models"
	"github.com/FedeBP/pumoide/backend/utils"
	"github.com/google/uuid"
)

const (
	ActionExport           = "export"
	ActionImport           = "import"
	ActionAddRequest       = "addRequest"
	ActionUpdateCollection = "updateCollection"
	ActionDeleteCollection = "deleteCollection"
	ActionDeleteRequest    = "deleteRequest"
)

type CollectionHandler struct {
	DefaultPath string
	Logger      *log.Logger
}

func NewCollectionHandler(defaultPath string, logger *log.Logger) *CollectionHandler {
	return &CollectionHandler{DefaultPath: defaultPath, Logger: logger}
}

func (h *CollectionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("action")
	switch r.Method {
	case http.MethodGet:
		if action == ActionExport {
			h.exportCollection(w, r)
		} else {
			h.getCollections(w, r)
		}
	case http.MethodPost:
		if action == ActionImport {
			h.importCollection(w, r)
		} else {
			h.createCollection(w, r)
		}
	case http.MethodPut:
		switch action {
		case ActionAddRequest:
			h.addRequestToCollection(w, r)
		case ActionUpdateCollection:
			h.updateCollection(w, r)
		default:
			apperrors.RespondWithError(w, http.StatusBadRequest, "Invalid action", nil, h.Logger)
		}
	case http.MethodDelete:
		switch action {
		case ActionDeleteCollection:
			h.deleteCollection(w, r)
		case ActionDeleteRequest:
			h.deleteRequestFromCollection(w, r)
			apperrors.RespondWithError(w, http.StatusBadRequest, "Invalid action", nil, h.Logger)
		}
	default:
		apperrors.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed", nil, h.Logger)
	}
}

func (h *CollectionHandler) getCollectionPath(r *http.Request) string {
	path := r.URL.Query().Get("path")
	if path == "" {
		return h.DefaultPath
	}
	return path
}

// GET methods

func (h *CollectionHandler) getCollections(w http.ResponseWriter, r *http.Request) {
	collectionPath := h.getCollectionPath(r)

	files, err := filepath.Glob(filepath.Join(collectionPath, "*.json"))
	if err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to read collections", err, h.Logger)
		return
	}

	var collections []models.Collection
	for _, file := range files {
		collection, err := models.LoadCollection(collectionPath, filepath.Base(file[:len(file)-5]))
		if err != nil {
			h.Logger.Printf("Failed to load collection %s: %v", file, err)
			continue
		}
		collections = append(collections, *collection)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(collections); err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to encode collections", err, h.Logger)
		return
	}
}

func (h *CollectionHandler) exportCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get("id")
	if collectionID == "" {
		apperrors.RespondWithError(w, http.StatusBadRequest, "Collection ID is required", nil, h.Logger)
		return
	}

	collection, err := models.LoadCollection(h.getCollectionPath(r), collectionID)
	if err != nil {
		apperrors.RespondWithError(w, http.StatusNotFound, "Failed to load collection", err, h.Logger)
		return
	}

	exportedCollection := collection.ToExportedCollection()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.json", collection.Name))
	if err := json.NewEncoder(w).Encode(exportedCollection); err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to encode exported collection", err, h.Logger)
	}
}

// POST methods

func (h *CollectionHandler) createCollection(w http.ResponseWriter, r *http.Request) {
	var collection models.Collection
	if err := json.NewDecoder(r.Body).Decode(&collection); err != nil {
		apperrors.RespondWithError(w, http.StatusBadRequest, "Invalid request body", err, h.Logger)
		return
	}

	for _, req := range collection.Requests {
		if !req.Method.IsValid() {
			var message = fmt.Sprintf("Invalid HTTP method in request '%s': %s", req.Name, req.Method)
			apperrors.RespondWithError(w, http.StatusBadRequest, message, nil, h.Logger)
			return
		}
	}

	collection.ID = uuid.New().String()

	savePath := h.getCollectionPath(r)
	if err := utils.EnsureDir(savePath); err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to create directory", err, h.Logger)
		return
	}

	if err := collection.Save(savePath); err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to save collection", err, h.Logger)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(collection); err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to encode created collection", err, h.Logger)
	}
}

func (h *CollectionHandler) importCollection(w http.ResponseWriter, r *http.Request) {
	var importedCollection models.ImportedCollection
	if err := json.NewDecoder(r.Body).Decode(&importedCollection); err != nil {
		apperrors.RespondWithError(w, http.StatusBadRequest, "Failed to parse imported collection", err, h.Logger)
		return
	}

	newCollection := models.NewCollectionFromImported(importedCollection)

	for _, req := range newCollection.Requests {
		if !req.Method.IsValid() {
			var message = fmt.Sprintf("Invalid HTTP method in request '%s': %s", req.Name, req.Method)
			apperrors.RespondWithError(w, http.StatusBadRequest, message, nil, h.Logger)
			return
		}
	}

	collectionPath := h.getCollectionPath(r)
	if err := newCollection.Save(collectionPath); err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to save imported collection", err, h.Logger)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(newCollection); err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to encode imported collection", err, h.Logger)
	}
}

// PUT methods

func (h *CollectionHandler) updateCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get("id")
	if collectionID == "" {
		apperrors.RespondWithError(w, http.StatusBadRequest, "Collection ID is required", nil, h.Logger)
		return
	}

	var updatedCollection models.Collection
	if err := json.NewDecoder(r.Body).Decode(&updatedCollection); err != nil {
		apperrors.RespondWithError(w, http.StatusBadRequest, "Failed to parse updated collection", err, h.Logger)
		return
	}

	for _, req := range updatedCollection.Requests {
		if !req.Method.IsValid() {
			var message = fmt.Sprintf("Invalid HTTP method in request '%s': %s", req.Name, req.Method)
			apperrors.RespondWithError(w, http.StatusBadRequest, message, nil, h.Logger)
			return
		}
	}

	collectionPath := h.getCollectionPath(r)

	existingCollection, err := models.LoadCollection(collectionPath, collectionID)
	if err != nil {
		apperrors.RespondWithError(w, http.StatusNotFound, "Failed to load collection", err, h.Logger)
		return
	}

	existingCollection.Name = updatedCollection.Name
	existingCollection.Description = updatedCollection.Description
	existingCollection.Requests = updatedCollection.Requests

	if err := existingCollection.Save(collectionPath); err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to save updated collection", err, h.Logger)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(existingCollection); err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to encode updated collection", err, h.Logger)
	}
}

func (h *CollectionHandler) addRequestToCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get("id")
	if collectionID == "" {
		apperrors.RespondWithError(w, http.StatusBadRequest, "Collection ID is required", nil, h.Logger)
		return
	}

	var request models.Request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		apperrors.RespondWithError(w, http.StatusBadRequest, "Failed to parse request", err, h.Logger)
		return
	}

	if !request.Method.IsValid() {
		var message = fmt.Sprintf("Invalid HTTP method in request '%s': %s", request.Name, request.Method)
		apperrors.RespondWithError(w, http.StatusBadRequest, message, nil, h.Logger)
		return
	}

	request.ID = uuid.New().String()

	collectionPath := h.getCollectionPath(r)

	collection, err := models.LoadCollection(collectionPath, collectionID)
	if err != nil {
		apperrors.RespondWithError(w, http.StatusNotFound, "Failed to load collection", err, h.Logger)
		return
	}

	collection.AddRequest(request)
	if err := collection.Save(collectionPath); err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to save collection", err, h.Logger)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(request); err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to encode added request", err, h.Logger)
	}
}

// DELETE methods

func (h *CollectionHandler) deleteCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get("id")
	if collectionID == "" {
		apperrors.RespondWithError(w, http.StatusBadRequest, "Collection ID is required", nil, h.Logger)
		return
	}

	collectionPath := h.getCollectionPath(r)
	filePath := filepath.Join(collectionPath, collectionID+".json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		apperrors.RespondWithError(w, http.StatusNotFound, "Collection not found", err, h.Logger)
		return
	}

	if err := os.Remove(filePath); err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to delete collection", err, h.Logger)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("Collection deleted successfully")); err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to write response", err, h.Logger)
	}
}

func (h *CollectionHandler) deleteRequestFromCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get("collectionId")
	requestID := r.URL.Query().Get("requestId")
	if collectionID == "" || requestID == "" {
		apperrors.RespondWithError(w, http.StatusBadRequest, "Collection ID is required", nil, h.Logger)
		return
	}

	collectionPath := h.getCollectionPath(r)

	collection, err := models.LoadCollection(collectionPath, collectionID)
	if err != nil {
		apperrors.RespondWithError(w, http.StatusNotFound, "Failed to load collection", err, h.Logger)
		return
	}

	if !collection.RemoveRequest(requestID) {
		apperrors.RespondWithError(w, http.StatusNotFound, "Request not found in collection", err, h.Logger)
		return
	}

	if err := collection.Save(collectionPath); err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to save collection", err, h.Logger)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("Request deleted successfully")); err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to write response", err, h.Logger)
	}
}
