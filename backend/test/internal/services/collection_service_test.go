package services_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/FedeBP/pumoide/backend/internal/models"
	"github.com/FedeBP/pumoide/backend/internal/services"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectionService(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "collection_test")
	require.NoError(t, err)
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to remove temp directory: %v", err)
		}
	}(tempDir)

	logger := logrus.New()
	service := services.NewCollectionService(tempDir, logger)

	var collectionID string

	t.Run("CreateCollection", func(t *testing.T) {
		collectionData := []byte(`{
            "info": {
                "name": "Test Collection",
                "description": "A test collection"
            },
            "item": [
                {
                    "name": "Folder 1",
                    "item": [
                        {
                            "name": "Request 1",
                            "request": {
                                "method": "GET",
                                "url": "https://api.example.com/test"
                            }
                        }
                    ]
                },
                {
                    "name": "New Folder",
                    "item": []
                }
            ]
        }`)

		collection, err := service.CreateCollection(collectionData)
		require.NoError(t, err)
		assert.NotEmpty(t, collection.ID)
		collectionID = collection.ID
		assert.Equal(t, "Test Collection", collection.Info.Name)
		assert.Equal(t, "A test collection", collection.Info.Description)
		assert.Len(t, collection.Item, 2)
		assert.NotEmpty(t, collection.Item[0].Item)
		assert.Empty(t, collection.Item[1].Item)
	})

	t.Run("GetCollections", func(t *testing.T) {
		collections, err := service.GetCollections()
		require.NoError(t, err)
		assert.Len(t, collections, 1)
	})

	t.Run("UpdateCollection", func(t *testing.T) {
		updatedData := []byte(`{
            "info": {
                "name": "Updated Collection",
                "description": "An updated test collection"
            },
            "item": [
                {
                    "name": "Folder 1",
                    "item": [
                        {
                            "name": "Request 1",
                            "request": {
                                "method": "GET",
                                "url": "https://api.example.com/test"
                            }
                        }
                    ]
                },
                {
                    "name": "New Folder",
                    "item": []
                }
            ]
        }`)

		updatedCollection, err := service.UpdateCollection(collectionID, updatedData)
		require.NoError(t, err)
		assert.Equal(t, "Updated Collection", updatedCollection.Info.Name)
		assert.Equal(t, "An updated test collection", updatedCollection.Info.Description)
		assert.Len(t, updatedCollection.Item, 2)
		assert.NotEmpty(t, updatedCollection.Item[0].Item)
		assert.Empty(t, updatedCollection.Item[1].Item)
	})

	t.Run("DeleteRequestFromCollection", func(t *testing.T) {
		require.NotEmpty(t, collectionID, "Collection ID should not be empty")

		requestData := []byte(`{
            "folderPath": ["New Folder"],
            "name": "Request to Delete",
            "request": {
                "id": "request-to-delete-id",
                "method": "POST",
                "url": "https://api.example.com/delete-me"
            }
        }`)

		_, err := service.AddRequestToCollection(collectionID, requestData)
		require.NoError(t, err)

		updatedCollection, err := models.LoadCollection(tempDir, collectionID)
		require.NoError(t, err)
		var newFolder *models.Item
		for _, item := range updatedCollection.Item {
			if item.Name == "New Folder" {
				newFolder = &item
				break
			}
		}
		require.NotNil(t, newFolder, "New Folder not found")
		require.NotEmpty(t, newFolder.Item, "New Folder should not be empty")

		var addedRequest *models.Item
		for _, item := range newFolder.Item {
			if item.Name == "Request to Delete" {
				addedRequest = &item
				break
			}
		}
		require.NotNil(t, addedRequest, "Added request not found")

		var reqDetails struct {
			ID string `json:"id"`
		}
		err = json.Unmarshal(addedRequest.Request, &reqDetails)
		require.NoError(t, err)
		require.NotEmpty(t, reqDetails.ID, "Request ID should not be empty")

		err = service.DeleteRequestFromCollection(collectionID, reqDetails.ID)
		require.NoError(t, err)

		updatedCollection, err = models.LoadCollection(tempDir, collectionID)
		require.NoError(t, err)
		newFolder = nil
		for _, item := range updatedCollection.Item {
			if item.Name == "New Folder" {
				newFolder = &item
				break
			}
		}
		require.NotNil(t, newFolder, "New Folder not found")
		for _, item := range newFolder.Item {
			assert.NotEqual(t, "Request to Delete", item.Name, "Request should have been deleted")
		}
	})

	t.Run("DeleteRequestFromCollection", func(t *testing.T) {
		requestData := []byte(`{
        "folderPath": ["New Folder"],
        "name": "Request to Delete",
        "request": {
            "id": "request-to-delete-id",
            "method": "POST",
            "url": "https://api.example.com/delete-me"
        }
    	}`)

		_, err := service.AddRequestToCollection(collectionID, requestData)
		require.NoError(t, err)

		updatedCollection, err := models.LoadCollection(tempDir, collectionID)
		require.NoError(t, err)
		var newFolder *models.Item
		for _, item := range updatedCollection.Item {
			if item.Name == "New Folder" {
				newFolder = &item
				break
			}
		}
		require.NotNil(t, newFolder, "New Folder not found")
		require.NotEmpty(t, newFolder.Item, "New Folder should not be empty")

		var addedRequest *models.Item
		for _, item := range newFolder.Item {
			if item.Name == "Request to Delete" {
				addedRequest = &item
				break
			}
		}
		require.NotNil(t, addedRequest, "Added request not found")

		var reqDetails struct {
			ID string `json:"id"`
		}
		err = json.Unmarshal(addedRequest.Request, &reqDetails)
		require.NoError(t, err)
		require.NotEmpty(t, reqDetails.ID, "Request ID should not be empty")

		err = service.DeleteRequestFromCollection(collectionID, reqDetails.ID)
		require.NoError(t, err)

		updatedCollection, err = models.LoadCollection(tempDir, collectionID)
		require.NoError(t, err)
		newFolder = nil
		for _, item := range updatedCollection.Item {
			if item.Name == "New Folder" {
				newFolder = &item
				break
			}
		}
		require.NotNil(t, newFolder, "New Folder not found")
		for _, item := range newFolder.Item {
			assert.NotEqual(t, "Request to Delete", item.Name, "Request should have been deleted")
		}
	})

	t.Run("ExportCollection", func(t *testing.T) {
		requestData := []byte(`{
        "folderPath": ["New Folder"],
        "name": "New Request",
        "request": {
            "method": "GET",
            "url": "https://api.example.com/new"
        }
    	}`)

		_, err := service.AddRequestToCollection(collectionID, requestData)
		require.NoError(t, err)

		exportedCollection, err := service.ExportCollection(collectionID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Collection", exportedCollection.Info.Name)
		assert.Equal(t, "An updated test collection", exportedCollection.Info.Description)
		assert.Len(t, exportedCollection.Item, 2)
		assert.NotEmpty(t, exportedCollection.Item[0].Item)
		assert.NotEmpty(t, exportedCollection.Item[1].Item)

		var newFolder *models.Item
		for _, item := range exportedCollection.Item {
			if item.Name == "New Folder" {
				newFolder = &item
				break
			}
		}
		require.NotNil(t, newFolder, "New Folder not found")
		require.Len(t, newFolder.Item, 1, "New Folder should contain 1 item")
		assert.Equal(t, "New Request", newFolder.Item[0].Name)
	})

	t.Run("DeleteCollection", func(t *testing.T) {
		collections, err := service.GetCollections()
		require.NoError(t, err)
		initialCount := len(collections)

		err = service.DeleteCollection(collectionID)
		require.NoError(t, err)

		remainingCollections, err := service.GetCollections()
		require.NoError(t, err)
		assert.Len(t, remainingCollections, initialCount-1)
	})
}
