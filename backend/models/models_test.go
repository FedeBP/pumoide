package models

import (
	"os"
	"testing"
)

func TestCollectionSaveAndLoad(t *testing.T) {
	// Create a temporary directory for testing
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

	// Create a test collection
	collection := &Collection{
		ID:   "test-id",
		Name: "Test Collection",
		Requests: []Request{
			{ID: "req1", Method: "GET", URL: "http://example.com"},
		},
	}

	// Test Save
	err = collection.Save(tempDir)
	if err != nil {
		t.Fatalf("Failed to save collection: %v", err)
	}

	// Test Load
	loadedCollection, err := LoadCollection(tempDir, "test-id")
	if err != nil {
		t.Fatalf("Failed to load collection: %v", err)
	}

	// Compare saved and loaded collections
	if collection.ID != loadedCollection.ID || collection.Name != loadedCollection.Name {
		t.Errorf("Loaded collection does not match saved collection")
	}
	if len(collection.Requests) != len(loadedCollection.Requests) {
		t.Errorf("Loaded collection requests do not match saved collection requests")
	}
}
