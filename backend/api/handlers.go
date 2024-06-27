package api

import (
	"encoding/json"
	"fmt"
	"github.com/FedeBP/pumoide/backend/models"
	"github.com/FedeBP/pumoide/backend/utils"
	"github.com/google/uuid"
	"net/http"
	"os"
	"path/filepath"
)

func handleCollections(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("action")
	switch r.Method {
	case http.MethodGet:
		if r.URL.Query().Get("action") == "export" {
			exportCollection(w, r)
		} else {
			getCollections(w, r)
		}
	case http.MethodPost:
		createCollection(w, r)
	case http.MethodPut:
		switch action {
		case "addRequest":
			addRequestToCollection(w, r)
		case "updateCollection":
			updateCollection(w, r)
		case "importCollection":
			importCollection(w, r)
		default:
			http.Error(w, "Invalid action", http.StatusBadRequest)
		}
	case http.MethodDelete:
		switch action {
		case "deleteCollection":
			deleteCollection(w, r)
		case "deleteRequest":
			deleteRequestFromCollection(w, r)
		default:
			http.Error(w, "Invalid action", http.StatusBadRequest)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// GET handlers
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

func exportCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get("id")
	if collectionID == "" {
		http.Error(w, "Collection ID is required", http.StatusBadRequest)
		return
	}

	collectionPath := utils.GetDefaultCollectionsPath()
	collection, err := models.LoadCollection(collectionPath, collectionID)
	if err != nil {
		http.Error(w, "Failed to load collection", http.StatusNotFound)
		return
	}

	exportedCollection := struct {
		Info struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Schema      string `json:"schema"`
		} `json:"info"`
		Item []struct {
			Name    string `json:"name"`
			Request struct {
				Method string            `json:"method"`
				URL    string            `json:"url"`
				Header []models.Header   `json:"header"`
				Body   map[string]string `json:"body"`
			} `json:"request"`
		} `json:"item"`
	}{
		Info: struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Schema      string `json:"schema"`
		}{
			Name:        collection.Name,
			Description: collection.Description,
			Schema:      "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
	}

	for _, req := range collection.Requests {
		item := struct {
			Name    string `json:"name"`
			Request struct {
				Method string            `json:"method"`
				URL    string            `json:"url"`
				Header []models.Header   `json:"header"`
				Body   map[string]string `json:"body"`
			} `json:"request"`
		}{
			Name: req.Name,
			Request: struct {
				Method string            `json:"method"`
				URL    string            `json:"url"`
				Header []models.Header   `json:"header"`
				Body   map[string]string `json:"body"`
			}{
				Method: req.Method,
				URL:    req.URL,
				Header: req.Headers,
				Body: map[string]string{
					"mode": "raw",
					"raw":  req.Body,
				},
			},
		}
		exportedCollection.Item = append(exportedCollection.Item, item)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.json", collection.Name))
	err = json.NewEncoder(w).Encode(exportedCollection)
	if err != nil {
		http.Error(w, "Failed to export collection", http.StatusInternalServerError)
		return
	}
}

// POST handlers
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

func importCollection(w http.ResponseWriter, r *http.Request) {
	var importedCollection struct {
		Info struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"info"`
		Item []struct {
			Name    string `json:"name"`
			Request struct {
				Method string `json:"method"`
				URL    string `json:"url"`
				Header []struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				} `json:"header"`
				Body struct {
					Mode string `json:"mode"`
					Raw  string `json:"raw"`
				} `json:"body"`
			} `json:"request"`
		} `json:"item"`
	}

	err := json.NewDecoder(r.Body).Decode(&importedCollection)
	if err != nil {
		http.Error(w, "Failed to parse imported collection", http.StatusBadRequest)
		return
	}

	newCollection := models.Collection{
		ID:          uuid.New().String(),
		Name:        importedCollection.Info.Name,
		Description: importedCollection.Info.Description,
	}

	for _, item := range importedCollection.Item {
		newRequest := models.Request{
			ID:     uuid.New().String(),
			Name:   item.Name,
			Method: item.Request.Method,
			URL:    item.Request.URL,
		}

		for _, header := range item.Request.Header {
			newRequest.Headers = append(newRequest.Headers, models.Header{
				Key:   header.Key,
				Value: header.Value,
			})
		}

		if item.Request.Body.Mode == "raw" {
			newRequest.Body = item.Request.Body.Raw
		}
		newCollection.Requests = append(newCollection.Requests, newRequest)
	}

	collectionPath := utils.GetDefaultCollectionsPath()
	err = newCollection.Save(collectionPath)
	if err != nil {
		http.Error(w, "Failed to save imported collection", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(newCollection)
	if err != nil {
		http.Error(w, "Failed to write header", http.StatusInternalServerError)
		return
	}
}

// PUT handlers
func updateCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get("id")
	if collectionID == "" {
		http.Error(w, "Collection ID is required", http.StatusBadRequest)
		return
	}

	var updatedCollection models.Collection
	err := json.NewDecoder(r.Body).Decode(&updatedCollection)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collectionPath := r.URL.Query().Get("path")
	if collectionPath == "" {
		collectionPath = utils.GetDefaultCollectionsPath()
	}

	existingCollection, err := models.LoadCollection(collectionPath, collectionID)
	if err != nil {
		http.Error(w, "Failed to load collection", http.StatusNotFound)
		return
	}

	existingCollection.Name = updatedCollection.Name
	existingCollection.Requests = updatedCollection.Requests

	err = existingCollection.Save(collectionPath)
	if err != nil {
		http.Error(w, "Failed to save updated collection", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(existingCollection)
	if err != nil {
		http.Error(w, "Failed to write header", http.StatusInternalServerError)
		return
	}
}

// DELETE handlers
func deleteCollection(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get("id")
	if collectionID == "" {
		http.Error(w, "Collection ID is required", http.StatusBadRequest)
		return
	}

	collectionPath := r.URL.Query().Get("path")
	if collectionPath == "" {
		collectionPath = utils.GetDefaultCollectionsPath()
	}

	filePath := filepath.Join(collectionPath, collectionID+".json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}

	err := os.Remove(filePath)
	if err != nil {
		http.Error(w, "Failed to delete collection", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Collection deleted successfully"))
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
