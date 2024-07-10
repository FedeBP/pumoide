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
			return
		}
	}(tempDir)

	logger := logrus.New()
	service := services.NewCollectionService(tempDir, logger)

	var collectionID string

	t.Run("CreateCollection", func(t *testing.T) {
		collectionData := []byte(`{
            "name": "Test Collection",
            "description": "A test collection",
            "items": [
                {
                    "name": "Folder 1",
                    "folder": {
                        "items": [
                            {
                                "name": "Request 1",
                                "request": {
                                    "id": "request-1",
                                    "name": "Request 1",
                                    "type": "rest",
                                    "method": "GET",
                                    "url": "https://api.example.com/test"
                                }
                            }
                        ]
                    }
                },
                {
                    "name": "Request 2",
                    "request": {
                        "id": "request-2",
                        "name": "Request 2",
                        "type": "rest",
                        "method": "POST",
                        "url": "https://api.example.com/test"
                    }
                }
            ]
        }`)

		collection, err := service.CreateCollection(collectionData)
		require.NoError(t, err)
		assert.NotEmpty(t, collection.ID)
		collectionID = collection.ID
		assert.Equal(t, "Test Collection", collection.Name)
		assert.Equal(t, "A test collection", collection.Description)
		assert.Len(t, collection.Items, 2)
		assert.NotNil(t, collection.Items[0].Folder)
		assert.Len(t, collection.Items[0].Folder.Items, 1)
		assert.NotNil(t, collection.Items[1].Request)
	})

	t.Run("GetCollections", func(t *testing.T) {
		collections, err := service.GetCollections()
		require.NoError(t, err)
		assert.Len(t, collections, 1)
	})

	t.Run("UpdateCollection", func(t *testing.T) {
		updatedData := []byte(`{
            "name": "Updated Collection",
            "description": "An updated test collection",
            "items": [
                {
                    "name": "Folder 1",
                    "folder": {
                        "items": [
                            {
                                "name": "Request 1",
                                "request": {
                                    "id": "request-1",
                                    "name": "Request 1",
                                    "type": "rest",
                                    "method": "GET",
                                    "url": "https://api.example.com/test"
                                }
                            }
                        ]
                    }
                },
                {
                    "name": "New Folder",
                    "folder": {
                        "items": []
                    }
                }
            ]
        }`)

		updatedCollection, err := service.UpdateCollection(collectionID, updatedData)
		require.NoError(t, err)
		assert.Equal(t, "Updated Collection", updatedCollection.Name)
		assert.Equal(t, "An updated test collection", updatedCollection.Description)
		assert.Len(t, updatedCollection.Items, 2)
		assert.NotNil(t, updatedCollection.Items[0].Folder)
		assert.NotNil(t, updatedCollection.Items[1].Folder)
	})

	t.Run("AddRequestToCollection", func(t *testing.T) {
		requestData := []byte(`{
            "folderPath": ["New Folder"],
            "request": {
                "id": "test-request-id",
                "name": "New Request",
                "type": "rest",
                "method": "GET",
                "url": "https://api.example.com/new"
            }
        }`)

		newRequest, err := service.AddRequestToCollection(collectionID, requestData)
		require.NoError(t, err)
		assert.Equal(t, "New Request", newRequest.GetName())

		updatedCollection, err := models.LoadCollection(tempDir, collectionID)
		require.NoError(t, err)
		assert.Len(t, updatedCollection.Items, 2)
		assert.Len(t, updatedCollection.Items[1].Folder.Items, 1)

		addedRequest := updatedCollection.Items[1].Folder.Items[0]
		var requestDetails struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		err = json.Unmarshal(addedRequest.Request, &requestDetails)
		require.NoError(t, err)
		assert.Equal(t, "test-request-id", requestDetails.ID)
		assert.Equal(t, "New Request", requestDetails.Name)
	})

	t.Run("DeleteRequestFromCollection", func(t *testing.T) {
		err = service.DeleteRequestFromCollection(collectionID, "test-request-id")
		require.NoError(t, err)

		updatedCollection, err := models.LoadCollection(tempDir, collectionID)
		require.NoError(t, err)

		requestFound := false
		var searchForRequest func(items []models.Item)
		searchForRequest = func(items []models.Item) {
			for _, item := range items {
				if item.Request != nil {
					var req struct {
						ID string `json:"id"`
					}
					err := json.Unmarshal(item.Request, &req)
					require.NoError(t, err)
					if req.ID == "test-request-id" {
						requestFound = true
						return
					}
				}
				if item.Folder != nil {
					searchForRequest(item.Folder.Items)
				}
			}
		}
		searchForRequest(updatedCollection.Items)
		assert.False(t, requestFound, "Request should have been deleted")
	})

	t.Run("ExportCollection", func(t *testing.T) {
		exportedCollection, err := service.ExportCollection(collectionID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Collection", exportedCollection.Info.Name)
		assert.Equal(t, "An updated test collection", exportedCollection.Info.Description)
		assert.Len(t, exportedCollection.Item, 2)
		assert.NotEmpty(t, exportedCollection.Item[0].Item)
		assert.Empty(t, exportedCollection.Item[1].Item)
	})

	t.Run("ImportCollection", func(t *testing.T) {
		importData := []byte(`{
            "info": {
                "name": "Imported Collection",
                "description": "An imported test collection"
            },
            "item": [
                {
                    "name": "Imported Folder",
                    "item": [
                        {
                            "name": "Imported Request",
                            "request": {
                                "name": "Imported Request",
                                "type": "rest",
                                "method": "GET",
                                "url": "https://api.example.com/imported"
                            }
                        }
                    ]
                }
            ]
        }`)

		importedCollection, err := service.ImportCollection(importData)
		require.NoError(t, err)
		assert.Equal(t, "Imported Collection", importedCollection.Name)
		assert.Equal(t, "An imported test collection", importedCollection.Description)
		assert.Len(t, importedCollection.Items, 1)
		assert.NotNil(t, importedCollection.Items[0].Folder)
		assert.Len(t, importedCollection.Items[0].Folder.Items, 1)
	})

	t.Run("DeleteCollection", func(t *testing.T) {
		collections, err := service.GetCollections()
		require.NoError(t, err)
		initialCount := len(collections)

		for _, collection := range collections {
			err = service.DeleteCollection(collection.ID)
			require.NoError(t, err)
		}

		remainingCollections, err := service.GetCollections()
		require.NoError(t, err)
		assert.Len(t, remainingCollections, 0)
		assert.Equal(t, initialCount-len(collections), len(remainingCollections))
	})
}
