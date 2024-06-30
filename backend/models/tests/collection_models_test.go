package tests

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/FedeBP/pumoide/backend/apperrors"
	"github.com/FedeBP/pumoide/backend/models"
)

func TestCollectionSaveAndLoad(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "collection_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to delete temp dir: %v", err)
		}
	}(tempDir)

	collection := &models.Collection{
		ID:          "test-id",
		Name:        "Test Collection",
		Description: "Test Description",
		Requests: []models.Request{
			{
				ID:     "req1",
				Name:   "Test Request",
				Method: models.MethodGet,
				URL:    "http://example.com",
				Headers: []models.Header{
					{Key: "Content-Type", Value: "application/json"},
				},
				Body: `{"key": "value"}`,
			},
		},
	}

	err = collection.Save(tempDir)
	if err != nil {
		t.Fatalf("Failed to save collection: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tempDir, "test-id.json")); os.IsNotExist(err) {
		t.Fatalf("Collection file was not created")
	}

	loadedCollection, err := models.LoadCollection(tempDir, "test-id")
	if err != nil {
		t.Fatalf("Failed to load collection: %v", err)
	}

	if collection.ID != loadedCollection.ID {
		t.Errorf("Loaded collection ID does not match: got %v, want %v", loadedCollection.ID, collection.ID)
	}

	if collection.Name != loadedCollection.Name {
		t.Errorf("Loaded collection Name does not match: got %v, want %v", loadedCollection.Name, collection.Name)
	}

	if collection.Description != loadedCollection.Description {
		t.Errorf("Loaded collection Description does not match: got %v, want %v", loadedCollection.Description, collection.Description)
	}

	if len(collection.Requests) != len(loadedCollection.Requests) {
		t.Errorf("Loaded collection Requests length does not match: got %v, want %v", len(loadedCollection.Requests), len(collection.Requests))
	}

	if len(loadedCollection.Requests) > 0 {
		originalReq := collection.Requests[0]
		loadedReq := loadedCollection.Requests[0]

		if originalReq.ID != loadedReq.ID {
			t.Errorf("Loaded request ID does not match: got %v, want %v", loadedReq.ID, originalReq.ID)
		}

		if originalReq.Name != loadedReq.Name {
			t.Errorf("Loaded request Name does not match: got %v, want %v", loadedReq.Name, originalReq.Name)
		}

		if originalReq.Method != loadedReq.Method {
			t.Errorf("Loaded request Method does not match: got %v, want %v", loadedReq.Method, originalReq.Method)
		}

		if originalReq.URL != loadedReq.URL {
			t.Errorf("Loaded request URL does not match: got %v, want %v", loadedReq.URL, originalReq.URL)
		}

		if originalReq.Body != loadedReq.Body {
			t.Errorf("Loaded request Body does not match: got %v, want %v", loadedReq.Body, originalReq.Body)
		}

		if len(originalReq.Headers) != len(loadedReq.Headers) {
			t.Errorf("Loaded request Headers length does not match: got %v, want %v", len(loadedReq.Headers), len(originalReq.Headers))
		}

		if len(loadedReq.Headers) > 0 {
			if originalReq.Headers[0].Key != loadedReq.Headers[0].Key || originalReq.Headers[0].Value != loadedReq.Headers[0].Value {
				t.Errorf("Loaded request Header does not match: got %v, want %v", loadedReq.Headers[0], originalReq.Headers[0])
			}
		}
	}
}

func TestAddRequest(t *testing.T) {
	collection := &models.Collection{
		ID:   "test-id",
		Name: "Test Collection",
	}

	request := models.Request{
		ID:     "req1",
		Name:   "Test Request",
		Method: models.MethodGet,
		URL:    "http://example.com",
	}

	err := collection.AddRequest(request)
	if err != nil {
		_ = apperrors.NewAppError(http.StatusInternalServerError, "Failed to add request", err)
		return
	}

	if len(collection.Requests) != 1 {
		t.Errorf("Request was not added to collection: got %v requests, want 1", len(collection.Requests))
	}

	if collection.Requests[0].ID != request.ID {
		t.Errorf("Added request ID does not match: got %v, want %v", collection.Requests[0].ID, request.ID)
	}
}

func TestRemoveRequest(t *testing.T) {
	collection := &models.Collection{
		ID:   "test-id",
		Name: "Test Collection",
		Requests: []models.Request{
			{ID: "req1", Name: "Request 1"},
			{ID: "req2", Name: "Request 2"},
		},
	}

	removed := collection.RemoveRequest("req1")
	if !removed {
		t.Errorf("RemoveRequest returned false, expected true")
	}

	if len(collection.Requests) != 1 {
		t.Errorf("Request was not removed from collection: got %v requests, want 1", len(collection.Requests))
	}

	if collection.Requests[0].ID != "req2" {
		t.Errorf("Incorrect request removed: got ID %v, want req2", collection.Requests[0].ID)
	}

	removed = collection.RemoveRequest("non-existent")
	if removed {
		t.Errorf("RemoveRequest returned true for non-existent request, expected false")
	}
}

func TestMethodIsValid(t *testing.T) {
	validMethods := []models.Method{models.MethodGet, models.MethodPost, models.MethodPut, models.MethodDelete, models.MethodPatch, models.MethodHead, models.MethodOptions, models.MethodTrace, models.MethodConnect}
	for _, method := range validMethods {
		if !method.IsValid() {
			t.Errorf("Method %s should be valid", method)
		}
	}

	invalidMethod := models.Method("INVALID")
	if invalidMethod.IsValid() {
		t.Errorf("Method INVALID should not be valid")
	}
}
