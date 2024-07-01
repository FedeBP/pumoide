package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/FedeBP/pumoide/backend/api"
	"github.com/FedeBP/pumoide/backend/models"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

func init() {
	logger = logrus.New()

	file, err := os.OpenFile("tests.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		logger.Out = file
	} else {
		logger.Debugf("Failed to log to file, using default stderr")
	}
	logger.SetLevel(logrus.DebugLevel)

	logger.SetFormatter(&logrus.JSONFormatter{})
}

func setupTestEnvironment(t *testing.T) (string, *api.CollectionHandler, func()) {
	tempDir, err := os.MkdirTemp("", "api_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	handler := api.NewCollectionHandler(tempDir, logger)

	cleanup := func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("Failed to clean up temp dir: %v", err)
		}
	}

	return tempDir, handler, cleanup
}

func TestCreateCollection(t *testing.T) {
	_, handler, cleanup := setupTestEnvironment(t)
	defer cleanup()

	collection := models.Collection{
		Name: "Test Collection",
		Requests: []models.Request{
			{Method: "GET", Name: "Test Request", URL: "http://example.com"},
		},
	}
	body, err := json.Marshal(collection)
	if err != nil {
		t.Fatalf("Failed to marshal collection: %v", err)
	}

	req, err := http.NewRequest("POST", "/pumoide-api/collections", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	var responseCollection models.Collection
	err = json.Unmarshal(rr.Body.Bytes(), &responseCollection)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if responseCollection.Name != collection.Name {
		t.Errorf("handler returned unexpected body: got %v want %v", responseCollection.Name, collection.Name)
	}
}

func TestGetCollections(t *testing.T) {
	tempDir, handler, cleanup := setupTestEnvironment(t)
	defer cleanup()

	collections := []models.Collection{
		{ID: "test1", Name: "Test Collection 1"},
		{ID: "test2", Name: "Test Collection 2"},
	}
	for _, c := range collections {
		if err := c.Save(tempDir); err != nil {
			t.Fatalf("Failed to save collection: %v", err)
		}
	}

	req, err := http.NewRequest("GET", "/pumoide-api/collections", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var responseCollections []models.Collection
	err = json.Unmarshal(rr.Body.Bytes(), &responseCollections)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if len(responseCollections) != len(collections) {
		t.Errorf("handler returned unexpected number of collections: got %v want %v", len(responseCollections), len(collections))
	}
}

func TestAddRequestToCollection(t *testing.T) {
	tempDir, handler, cleanup := setupTestEnvironment(t)
	defer cleanup()

	collection := models.Collection{ID: "test1", Name: "Test Collection"}
	if err := collection.Save(tempDir); err != nil {
		t.Fatalf("Failed to save collection: %v", err)
	}

	request := models.Request{Method: models.MethodGet, Name: "Test Request", URL: "http://example.com/api"}
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("PUT", "/pumoide-api/collections?action=addRequest&id=test1", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	updatedCollection, err := models.LoadCollection(tempDir, "test1")
	if err != nil {
		t.Fatalf("Failed to load updated collection: %v", err)
	}
	if len(updatedCollection.Requests) != 1 {
		t.Errorf("request was not added to collection")
	}
}

func TestDeleteRequestFromCollection(t *testing.T) {
	tempDir, handler, cleanup := setupTestEnvironment(t)
	defer cleanup()

	collection := models.Collection{
		ID:       "test1",
		Name:     "Test Collection",
		Requests: []models.Request{{ID: "req1", Method: "GET", Name: "Test request", URL: "http://example.com/api"}},
	}
	if err := collection.Save(tempDir); err != nil {
		t.Fatalf("Failed to save collection: %v", err)
	}

	req, err := http.NewRequest("DELETE", "/pumoide-api/collections?action=deleteRequest&collectionId=test1&requestId=req1", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	updatedCollection, err := models.LoadCollection(tempDir, "test1")
	if err != nil {
		t.Fatalf("Failed to load updated collection: %v", err)
	}
	if len(updatedCollection.Requests) != 0 {
		t.Errorf("request was not removed from collection")
	}
}

func TestUpdateCollection(t *testing.T) {
	tempDir, handler, cleanup := setupTestEnvironment(t)
	defer cleanup()

	collection := models.Collection{
		ID:   "test1",
		Name: "Test Collection",
		Requests: []models.Request{
			{ID: "req1", Method: "GET", Name: "Test request", URL: "http://example.com/api"},
		},
	}
	if err := collection.Save(tempDir); err != nil {
		t.Fatalf("Failed to save collection: %v", err)
	}

	updatedCollection := models.Collection{
		Name: "Updated Test Collection",
		Requests: []models.Request{
			{ID: "req1", Method: "POST", Name: "Test request", URL: "http://example.com/api/updated"},
			{ID: "req2", Method: "GET", Name: "Test request 2", URL: "http://example.com/api/new"},
		},
	}
	body, err := json.Marshal(updatedCollection)
	if err != nil {
		t.Fatalf("Failed to marshal updated collection: %v", err)
	}

	req, err := http.NewRequest("PUT", "/pumoide-api/collections?action=updateCollection&id=test1", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	loadedCollection, err := models.LoadCollection(tempDir, "test1")
	if err != nil {
		t.Fatalf("Failed to load updated collection: %v", err)
	}
	if loadedCollection.Name != "Updated Test Collection" {
		t.Errorf("collection name was not updated: got %v want %v", loadedCollection.Name, "Updated Test Collection")
	}
	if len(loadedCollection.Requests) != 2 {
		t.Errorf("incorrect number of requests: got %v want %v", len(loadedCollection.Requests), 2)
	}
}

func TestDeleteCollection(t *testing.T) {
	tempDir, handler, cleanup := setupTestEnvironment(t)
	defer cleanup()

	collection := models.Collection{
		ID:   "test1",
		Name: "Test Collection",
		Requests: []models.Request{
			{ID: "req1", Method: "GET", Name: "Test request", URL: "http://example.com/api"},
		},
	}
	if err := collection.Save(tempDir); err != nil {
		t.Fatalf("Failed to save collection: %v", err)
	}

	req, err := http.NewRequest("DELETE", "/pumoide-api/collections?action=deleteCollection&id=test1", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	_, err = os.Stat(filepath.Join(tempDir, "test1.json"))
	if !os.IsNotExist(err) {
		t.Errorf("collection file was not deleted")
	}
}

func TestImportCollection(t *testing.T) {
	_, handler, cleanup := setupTestEnvironment(t)
	defer cleanup()

	importJSON := `{
        "info": {
            "name": "Imported Collection",
            "description": "This is an imported collection"
        },
        "item": [
            {
                "name": "Get Example",
                "request": {
                    "method": "GET",
                    "url": "https://api.example.com/get",
                    "header": [
                        {
                            "key": "Content-Type",
                            "value": "application/json"
                        }
                    ],
                    "body": {
                        "mode": "raw",
                        "raw": "{\"key\": \"value\"}"
                    }
                }
            }
        ]
    }`

	req, err := http.NewRequest("POST", "/pumoide-api/collections?action=import", bytes.NewBufferString(importJSON))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	var responseCollection models.Collection
	err = json.Unmarshal(rr.Body.Bytes(), &responseCollection)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if responseCollection.Requests[0].Method != models.MethodGet {
		t.Errorf("Imported request has invalid method: got %v, want %v", responseCollection.Requests[0].Method, models.MethodGet)
	}

	if responseCollection.Name != "Imported Collection" {
		t.Errorf("handler returned unexpected name: got %v want %v", responseCollection.Name, "Imported Collection")
	}

	if len(responseCollection.Requests) != 1 {
		t.Errorf("handler returned unexpected number of requests: got %v want %v", len(responseCollection.Requests), 1)
	}
}

func TestExportCollection(t *testing.T) {
	tempDir, handler, cleanup := setupTestEnvironment(t)
	defer cleanup()

	collection := models.Collection{
		ID:          "test1",
		Name:        "Test Collection",
		Description: "This is a test collection",
		Requests: []models.Request{
			{
				ID:     "req1",
				Name:   "Get Example",
				Method: "GET",
				URL:    "https://api.example.com/get",
				Headers: []models.Header{
					{Key: "Content-Type", Value: "application/json"},
				},
				Body: "{\"key\": \"value\"}",
			},
		},
	}
	if err := collection.Save(tempDir); err != nil {
		t.Fatalf("Failed to save collection: %v", err)
	}

	req, err := http.NewRequest("GET", "/pumoide-api/collections?action=export&id=test1", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var exportedCollection models.ExportedCollection
	err = json.Unmarshal(rr.Body.Bytes(), &exportedCollection)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if exportedCollection.Info.Name != "Test Collection" {
		t.Errorf("handler returned unexpected name: got %v want %v", exportedCollection.Info.Name, "Test Collection")
	}

	if len(exportedCollection.Item) != 1 {
		t.Errorf("handler returned unexpected number of requests: got %v want %v", len(exportedCollection.Item), 1)
	}
}
