package api

import (
	"bytes"
	"encoding/json"
	"github.com/FedeBP/pumoide/backend/models"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateCollection(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "api_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to delete temp dir: %v", err)
		}
	}(tempDir)

	// Create test data
	collection := models.Collection{
		Name: "Test Collection",
		Requests: []models.Request{
			{Method: "GET", URL: "http://example.com"},
		},
	}
	body, _ := json.Marshal(collection)

	// Create request
	req, err := http.NewRequest("POST", "/pumoide-api/collections?path="+tempDir, bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler := http.HandlerFunc(handleCollections)
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	// Check the response body
	var responseCollection models.Collection
	err = json.Unmarshal(rr.Body.Bytes(), &responseCollection)
	if err != nil {
		t.Fatal(err)
	}
	if responseCollection.Name != collection.Name {
		t.Errorf("handler returned unexpected body: got %v want %v", responseCollection.Name, collection.Name)
	}
}

func TestGetCollections(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "api_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to delete temp dir: %v", err)
		}
	}(tempDir)

	// Create some test collections
	collections := []models.Collection{
		{ID: "test1", Name: "Test Collection 1"},
		{ID: "test2", Name: "Test Collection 2"},
	}
	for _, c := range collections {
		err := c.Save(tempDir)
		if err != nil {
			t.Fatalf("Failed to save temp dir: %v", err)
		}
	}

	// Create request
	req, err := http.NewRequest("GET", "/pumoide-api/collections?path="+tempDir, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler := http.HandlerFunc(handleCollections)
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body
	var responseCollections []models.Collection
	err = json.Unmarshal(rr.Body.Bytes(), &responseCollections)
	if err != nil {
		t.Fatal(err)
	}
	if len(responseCollections) != len(collections) {
		t.Errorf("handler returned unexpected number of collections: got %v want %v", len(responseCollections), len(collections))
	}
}

func TestAddRequestToCollection(t *testing.T) {
	// Setup: Create a temporary directory and a test collection
	tempDir, _ := os.MkdirTemp("", "api_test")
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to create temp dir %v", err)
		}
	}(tempDir)
	collection := models.Collection{ID: "test1", Name: "Test Collection"}
	err := collection.Save(tempDir)
	if err != nil {
		t.Fatalf("Failed to save collection: %v", err)
	}

	// Create test request data
	request := models.Request{Method: "GET", URL: "http://example.com/api"}
	body, _ := json.Marshal(request)

	// Create request
	req, _ := http.NewRequest("PUT", "/pumoide-api/collections?action=addRequest&id=test1&path="+tempDir, bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	// Call the handler
	handler := http.HandlerFunc(handleCollections)
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	// Check if the request was added to the collection
	updatedCollection, err := models.LoadCollection(tempDir, "test1")
	if err != nil {
		t.Fatalf("Failed to load collection: %v", err)
	}
	if len(updatedCollection.Requests) != 1 {
		t.Errorf("request was not added to collection")
	}
}

func TestDeleteRequestFromCollection(t *testing.T) {
	// Setup: Create a temporary directory and a test collection with a request
	tempDir, _ := os.MkdirTemp("", "api_test")
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
	}(tempDir)
	collection := models.Collection{
		ID:       "test1",
		Name:     "Test Collection",
		Requests: []models.Request{{ID: "req1", Method: "GET", URL: "http://example.com/api"}},
	}
	err := collection.Save(tempDir)
	if err != nil {
		t.Fatalf("Failed to save collection: %v", err)
	}

	// Create request
	req, _ := http.NewRequest("DELETE", "/pumoide-api/collections?action=deleteRequest&collectionId=test1&requestId=req1&path="+tempDir, nil)
	rr := httptest.NewRecorder()

	// Call the handler
	handler := http.HandlerFunc(handleCollections)
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check if the request was removed from the collection
	updatedCollection, err := models.LoadCollection(tempDir, "test1")
	if err != nil {
		t.Fatalf("Failed to load collection: %v", err)
	}
	if len(updatedCollection.Requests) != 0 {
		t.Errorf("request was not removed from collection")
	}
}

func TestUpdateCollection(t *testing.T) {
	// Setup: Create a temporary directory and a test collection
	tempDir, _ := os.MkdirTemp("", "api_test")
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
	}(tempDir)
	collection := models.Collection{
		ID:   "test1",
		Name: "Test Collection",
		Requests: []models.Request{
			{ID: "req1", Method: "GET", URL: "http://example.com/api"},
		},
	}
	err := collection.Save(tempDir)
	if err != nil {
		t.Fatalf("Failed to save collection: %v", err)
	}

	// Create updated collection data
	updatedCollection := models.Collection{
		Name: "Updated Test Collection",
		Requests: []models.Request{
			{ID: "req1", Method: "POST", URL: "http://example.com/api/updated"},
			{ID: "req2", Method: "GET", URL: "http://example.com/api/new"},
		},
	}
	body, _ := json.Marshal(updatedCollection)

	// Create request
	req, _ := http.NewRequest("PUT", "/pumoide-api/collections?action=updateCollection&id=test1&path="+tempDir, bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	// Call the handler
	handler := http.HandlerFunc(handleCollections)
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check if the collection was updated
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
	// Setup: Create a temporary directory and a test collection
	tempDir, _ := os.MkdirTemp("", "api_test")
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
	}(tempDir)
	collection := models.Collection{
		ID:   "test1",
		Name: "Test Collection",
		Requests: []models.Request{
			{ID: "req1", Method: "GET", URL: "http://example.com/api"},
		},
	}
	err := collection.Save(tempDir)
	if err != nil {
		t.Fatalf("Failed to save collection: %v", err)
	}

	// Create request
	req, _ := http.NewRequest("DELETE", "/pumoide-api/collections?action=deleteCollection&id=test1&path="+tempDir, nil)
	rr := httptest.NewRecorder()

	// Call the handler
	handler := http.HandlerFunc(handleCollections)
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check if the collection file was deleted
	_, err = os.Stat(filepath.Join(tempDir, "test1.json"))
	if !os.IsNotExist(err) {
		t.Errorf("collection file was not deleted")
	}
}

func TestImportCollection(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "api_test")
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
	}(tempDir)

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

	req, _ := http.NewRequest("POST", "/pumoide-api/collections?action=import&path="+tempDir, strings.NewReader(importJSON))
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(handleCollections)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	var responseCollection models.Collection
	err := json.Unmarshal(rr.Body.Bytes(), &responseCollection)
	if err != nil {
		t.Errorf("failet to unmarshall %v", err)
		return
	}

	if responseCollection.Name != "Imported Collection" {
		t.Errorf("handler returned unexpected name: got %v want %v", responseCollection.Name, "Imported Collection")
	}

	if len(responseCollection.Requests) != 1 {
		t.Errorf("handler returned unexpected number of requests: got %v want %v", len(responseCollection.Requests), 1)
	}
}

func TestExportCollection(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "api_test")
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
	}(tempDir)

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
	err := collection.Save(tempDir)
	if err != nil {
		t.Errorf("Failed to save collection %v", err)
		return
	}

	req, _ := http.NewRequest("GET", "/pumoide-api/collections?action=export&id=test1&path="+tempDir, nil)
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(handleCollections)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var exportedCollection struct {
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
	}
	err = json.Unmarshal(rr.Body.Bytes(), &exportedCollection)
	if err != nil {
		t.Errorf("failed to unmarshall %v", err)
		return
	}

	if exportedCollection.Info.Name != "Test Collection" {
		t.Errorf("handler returned unexpected name: got %v want %v", exportedCollection.Info.Name, "Test Collection")
	}

	if len(exportedCollection.Item) != 1 {
		t.Errorf("handler returned unexpected number of requests: got %v want %v", len(exportedCollection.Item), 1)
	}
}
