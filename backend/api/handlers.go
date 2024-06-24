package api

import (
	"encoding/json"
	"github.com/FedeBP/pumoide/backend/models"
	"github.com/FedeBP/pumoide/backend/utils"
	"github.com/google/uuid"
	"net/http"
	"path/filepath"
)

func handleCollections(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getCollections(w, r)
	case http.MethodPost:
		createCollection(w, r)
	case http.MethodPut:
		action := r.URL.Query().Get("action")
		if action == "addRequest" {
			addRequestToCollection(w, r)
		} else {
			http.Error(w, "Invalid action", http.StatusBadRequest)
		}
	case http.MethodDelete:
		action := r.URL.Query().Get("action")
		if action == "deleteRequest" {
			deleteRequestFromCollection(w, r)
		} else {
			http.Error(w, "Invalid action", http.StatusBadRequest)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func createCollection(w http.ResponseWriter, r *http.Request) {
	var collection models.Collection
	err := json.NewDecoder(r.Body).Decode(&collection)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection.ID = uuid.New().String()

	savePath := r.URL.Query().Get("path")
	if savePath == "" {
		savePath = utils.GetDefaultCollectionsPath()
	}

	err = utils.EnsureDir(savePath)
	if err != nil {
		http.Error(w, "Failed to create directory", http.StatusInternalServerError)
		return
	}

	err = collection.Save(savePath)
	if err != nil {
		http.Error(w, "Failed to save collection", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(collection)
	if err != nil {
		return
	}
}

func getCollections(w http.ResponseWriter, r *http.Request) {
	collectionPath := r.URL.Query().Get("path")
	if collectionPath == "" {
		collectionPath = utils.GetDefaultCollectionsPath()
	}

	files, err := filepath.Glob(filepath.Join(collectionPath, "*.json"))
	if err != nil {
		http.Error(w, "Failed to read collections", http.StatusInternalServerError)
		return
	}

	var collections []models.Collection
	for _, file := range files {
		collection, err := models.LoadCollection(collectionPath, filepath.Base(file[:len(file)-5]))
		if err != nil {
			http.Error(w, "Failed to load collections", http.StatusInternalServerError)
			continue
		}
		collections = append(collections, *collection)
	}

	err = json.NewEncoder(w).Encode(collections)
	if err != nil {
		return
	}
}

func addRequestToCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get("id")
	if collectionID == "" {
		http.Error(w, "Collection ID is required", http.StatusBadRequest)
		return
	}

	var request models.Request
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	request.ID = uuid.New().String()

	collectionPath := r.URL.Query().Get("path")
	if collectionPath == "" {
		collectionPath = utils.GetDefaultCollectionsPath()
	}

	collection, err := models.LoadCollection(collectionPath, collectionID)
	if err != nil {
		http.Error(w, "Failed to load collection", http.StatusNotFound)
		return
	}

	collection.AddRequest(request)
	err = collection.Save(collectionPath)
	if err != nil {
		http.Error(w, "Failed to save collection", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(request)
	if err != nil {
		http.Error(w, "Failed to write header", http.StatusInternalServerError)
		return
	}
}

func deleteRequestFromCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get("collectionId")
	requestID := r.URL.Query().Get("requestId")
	if collectionID == "" || requestID == "" {
		http.Error(w, "Collection ID and Request ID are required", http.StatusBadRequest)
		return
	}

	collectionPath := r.URL.Query().Get("path")
	if collectionPath == "" {
		collectionPath = utils.GetDefaultCollectionsPath()
	}

	collection, err := models.LoadCollection(collectionPath, collectionID)
	if err != nil {
		http.Error(w, "Failed to load collection", http.StatusNotFound)
		return
	}

	if !collection.RemoveRequest(requestID) {
		http.Error(w, "Request not found in collection", http.StatusNotFound)
		return
	}

	err = collection.Save(collectionPath)
	if err != nil {
		http.Error(w, "Failed to save collection", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Request deleted successfully"))
	if err != nil {
		http.Error(w, "Failed to write header", http.StatusInternalServerError)
		return
	}
}
