package api

import (
	"bytes"
	"encoding/json"
	"github.com/FedeBP/pumoide/backend/models"
	"net/http"
	"net/http/httptest"
	"os"
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
