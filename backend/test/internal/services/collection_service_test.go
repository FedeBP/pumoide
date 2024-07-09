package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/FedeBP/pumoide/backend/internal/models"
	"github.com/FedeBP/pumoide/backend/internal/services"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestCollectionService_CreateCollection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "collection_test")
	assert.NoError(t, err)
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			return
		}
	}(tempDir)

	logger := logrus.New()
	service := services.NewCollectionService(tempDir, logger)

	collectionData := []byte(`{
		"name": "Test Collection",
		"description": "A test collection",
		"requests": [
			{
				"name": "Test Request",
				"type": "rest",
				"method": "GET",
				"url": "https://models.example.com/test"
			}
		]
	}`)

	collection, err := service.CreateCollection(collectionData)
	assert.NoError(t, err)
	assert.NotNil(t, collection)
	assert.Equal(t, "Test Collection", collection.Name)
	assert.Equal(t, "A test collection", collection.Description)
	assert.Len(t, collection.Requests, 1)

	files, err := filepath.Glob(filepath.Join(tempDir, "*.json"))
	assert.NoError(t, err)
	assert.Len(t, files, 1)
}

func TestCollectionService_GetCollections(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "collection_test")
	assert.NoError(t, err)
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			return
		}
	}(tempDir)

	logger := logrus.New()
	service := services.NewCollectionService(tempDir, logger)

	collections := []models.Collection{
		{ID: "1", Name: "Collection 1"},
		{ID: "2", Name: "Collection 2"},
	}

	for _, c := range collections {
		data, _ := json.Marshal(c)
		err = os.WriteFile(filepath.Join(tempDir, c.ID+".json"), data, 0644)
		assert.NoError(t, err)
	}

	result, err := service.GetCollections()
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "Collection 1", result[0].Name)
	assert.Equal(t, "Collection 2", result[1].Name)
}

func TestCollectionService_DeleteCollection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "collection_test")
	assert.NoError(t, err)
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			return
		}
	}(tempDir)

	logger := logrus.New()
	service := services.NewCollectionService(tempDir, logger)

	collection := models.Collection{ID: "test", Name: "Test Collection"}
	data, _ := json.Marshal(collection)
	err = os.WriteFile(filepath.Join(tempDir, collection.ID+".json"), data, 0644)
	assert.NoError(t, err)

	err = service.DeleteCollection(collection.ID)
	assert.NoError(t, err)

	_, err = os.Stat(filepath.Join(tempDir, collection.ID+".json"))
	assert.True(t, os.IsNotExist(err))
}
