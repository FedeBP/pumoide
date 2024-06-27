package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

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
}

func NewCollectionHandler(defaultPath string) *CollectionHandler {
	return &CollectionHandler{DefaultPath: defaultPath}
}

func (h *CollectionHandler) Handle(w http.ResponseWriter, r *http.Request) {
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
			http.Error(w, "Invalid action", http.StatusBadRequest)
		}
	case http.MethodDelete:
		switch action {
		case ActionDeleteCollection:
			h.deleteCollection(w, r)
		case ActionDeleteRequest:
			h.deleteRequestFromCollection(w, r)
		default:
			http.Error(w, "Invalid action", http.StatusBadRequest)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
		http.Error(w, "Failed to read collections", http.StatusInternalServerError)
		return
	}

	var collections []models.Collection
	for _, file := range files {
		collection, err := models.LoadCollection(collectionPath, filepath.Base(file[:len(file)-5]))
		if err != nil {
			continue
		}
		collections = append(collections, *collection)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(collections); err != nil {
		http.Error(w, "Failed to encode collections", http.StatusInternalServerError)
	}
}

func (h *CollectionHandler) exportCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get("id")
	if collectionID == "" {
		http.Error(w, "Collection ID is required", http.StatusBadRequest)
		return
	}

	collection, err := models.LoadCollection(h.getCollectionPath(r), collectionID)
	if err != nil {
		http.Error(w, "Failed to load collection", http.StatusNotFound)
		return
	}

	exportedCollection := collection.ToExportedCollection()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.json", collection.Name))
	if err := json.NewEncoder(w).Encode(exportedCollection); err != nil {
		http.Error(w, "Failed to encode exported collection", http.StatusInternalServerError)
	}
}

// POST methods

func (h *CollectionHandler) createCollection(w http.ResponseWriter, r *http.Request) {
	var collection models.Collection
	if err := json.NewDecoder(r.Body).Decode(&collection); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for _, req := range collection.Requests {
		if !req.Method.IsValid() {
			http.Error(w, fmt.Sprintf("Invalid HTTP method in request '%s': %s", req.Name, req.Method), http.StatusBadRequest)
			return
		}
	}

	collection.ID = uuid.New().String()

	savePath := h.getCollectionPath(r)
	if err := utils.EnsureDir(savePath); err != nil {
		http.Error(w, "Failed to create directory", http.StatusInternalServerError)
		return
	}

	if err := collection.Save(savePath); err != nil {
		http.Error(w, "Failed to save collection", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(collection); err != nil {
		http.Error(w, "Failed to encode created collection", http.StatusInternalServerError)
	}
}

func (h *CollectionHandler) importCollection(w http.ResponseWriter, r *http.Request) {
	var importedCollection models.ImportedCollection
	if err := json.NewDecoder(r.Body).Decode(&importedCollection); err != nil {
		http.Error(w, "Failed to parse imported collection: "+err.Error(), http.StatusBadRequest)
		return
	}

	newCollection := models.NewCollectionFromImported(importedCollection)

	for _, req := range newCollection.Requests {
		if !req.Method.IsValid() {
			http.Error(w, fmt.Sprintf("Invalid HTTP method in imported request '%s': %s", req.Name, req.Method), http.StatusBadRequest)
			return
		}
	}

	collectionPath := h.getCollectionPath(r)
	if err := newCollection.Save(collectionPath); err != nil {
		http.Error(w, "Failed to save imported collection: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(newCollection); err != nil {
		http.Error(w, "Failed to encode imported collection: "+err.Error(), http.StatusInternalServerError)
	}
}

// PUT methods

func (h *CollectionHandler) updateCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get("id")
	if collectionID == "" {
		http.Error(w, "Collection ID is required", http.StatusBadRequest)
		return
	}

	var updatedCollection models.Collection
	if err := json.NewDecoder(r.Body).Decode(&updatedCollection); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for _, req := range updatedCollection.Requests {
		if !req.Method.IsValid() {
			http.Error(w, fmt.Sprintf("Invalid HTTP method in request '%s': %s", req.Name, req.Method), http.StatusBadRequest)
			return
		}
	}

	collectionPath := h.getCollectionPath(r)

	existingCollection, err := models.LoadCollection(collectionPath, collectionID)
	if err != nil {
		http.Error(w, "Failed to load collection", http.StatusNotFound)
		return
	}

	existingCollection.Name = updatedCollection.Name
	existingCollection.Description = updatedCollection.Description
	existingCollection.Requests = updatedCollection.Requests

	if err := existingCollection.Save(collectionPath); err != nil {
		http.Error(w, "Failed to save updated collection", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(existingCollection); err != nil {
		http.Error(w, "Failed to encode updated collection", http.StatusInternalServerError)
	}
}

func (h *CollectionHandler) addRequestToCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get("id")
	if collectionID == "" {
		http.Error(w, "Collection ID is required", http.StatusBadRequest)
		return
	}

	var request models.Request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !request.Method.IsValid() {
		http.Error(w, fmt.Sprintf("Invalid HTTP method: %s", request.Method), http.StatusBadRequest)
		return
	}

	request.ID = uuid.New().String()

	collectionPath := h.getCollectionPath(r)

	collection, err := models.LoadCollection(collectionPath, collectionID)
	if err != nil {
		http.Error(w, "Failed to load collection", http.StatusNotFound)
		return
	}

	collection.AddRequest(request)
	if err := collection.Save(collectionPath); err != nil {
		http.Error(w, "Failed to save collection", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(request); err != nil {
		http.Error(w, "Failed to encode added request", http.StatusInternalServerError)
	}
}

// DELETE methods

func (h *CollectionHandler) deleteCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get("id")
	if collectionID == "" {
		http.Error(w, "Collection ID is required", http.StatusBadRequest)
		return
	}

	collectionPath := h.getCollectionPath(r)
	filePath := filepath.Join(collectionPath, collectionID+".json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}

	if err := os.Remove(filePath); err != nil {
		http.Error(w, "Failed to delete collection", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("Collection deleted successfully")); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

func (h *CollectionHandler) deleteRequestFromCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get("collectionId")
	requestID := r.URL.Query().Get("requestId")
	if collectionID == "" || requestID == "" {
		http.Error(w, "Collection ID and Request ID are required", http.StatusBadRequest)
		return
	}

	collectionPath := h.getCollectionPath(r)

	collection, err := models.LoadCollection(collectionPath, collectionID)
	if err != nil {
		http.Error(w, "Failed to load collection", http.StatusNotFound)
		return
	}

	if !collection.RemoveRequest(requestID) {
		http.Error(w, "Request not found in collection", http.StatusNotFound)
		return
	}

	if err := collection.Save(collectionPath); err != nil {
		http.Error(w, "Failed to save collection", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("Request deleted successfully")); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}
