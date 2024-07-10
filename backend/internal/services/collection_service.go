package services

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/FedeBP/pumoide/backend/internal/domain"
	"github.com/FedeBP/pumoide/backend/internal/models"
	"github.com/FedeBP/pumoide/backend/pkg/constants"
	"github.com/FedeBP/pumoide/backend/pkg/errors"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type CollectionService struct {
	DefaultPath string
	Logger      *logrus.Logger
}

func NewCollectionService(defaultPath string, logger *logrus.Logger) *CollectionService {
	return &CollectionService{
		DefaultPath: defaultPath,
		Logger:      logger,
	}
}

func (s *CollectionService) GetCollections() ([]models.Collection, error) {
	files, err := filepath.Glob(filepath.Join(s.DefaultPath, "*.json"))
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToReadCollection, err)
	}

	var collections []models.Collection
	for _, file := range files {
		collection, err := models.LoadCollection(s.DefaultPath, filepath.Base(file[:len(file)-5]))
		if err != nil {
			s.Logger.Printf(constants.ErrFailedToLoadCollection+" %s: %v", file, err)
			continue
		}
		collections = append(collections, *collection)
	}

	return collections, nil
}

func (s *CollectionService) CreateCollection(collectionData []byte) (*models.Collection, error) {
	var collection models.Collection
	if err := json.Unmarshal(collectionData, &collection); err != nil {
		return nil, errors.NewAppError(http.StatusBadRequest, constants.ErrFailedToReadCollection, err)
	}

	if collection.ID == constants.EmptyString {
		collection.ID = uuid.New().String()
	}

	if err := collection.Save(s.DefaultPath); err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToSaveCollection, err)
	}

	return &collection, nil
}

func (s *CollectionService) UpdateCollection(id string, collectionData []byte) (*models.Collection, error) {
	var updatedCollection models.Collection
	if err := json.Unmarshal(collectionData, &updatedCollection); err != nil {
		return nil, errors.NewAppError(http.StatusBadRequest, constants.ErrFailedToReadCollection, err)
	}

	existingCollection, err := models.LoadCollection(s.DefaultPath, id)
	if err != nil {
		return nil, errors.NewAppError(http.StatusNotFound, constants.ErrFailedToLoadCollection, err)
	}

	existingCollection.Name = updatedCollection.Name
	existingCollection.Description = updatedCollection.Description
	existingCollection.Items = updatedCollection.Items

	if err := existingCollection.Save(s.DefaultPath); err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToSaveCollection, err)
	}

	return existingCollection, nil
}

func (s *CollectionService) DeleteCollection(id string) error {
	filePath := filepath.Join(s.DefaultPath, id+".json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return errors.NewAppError(http.StatusNotFound, constants.ErrCollectionNotFound, err)
	}

	if err := os.Remove(filePath); err != nil {
		return errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToDeleteCollection, err)
	}

	return nil
}

func (s *CollectionService) AddRequestToCollection(collectionID string, requestData []byte) (domain.Request, error) {
	collection, err := models.LoadCollection(s.DefaultPath, collectionID)
	if err != nil {
		return nil, errors.NewAppError(http.StatusNotFound, constants.ErrFailedToLoadCollection, err)
	}

	var requestInfo struct {
		FolderPath []string        `json:"folderPath"`
		Request    json.RawMessage `json:"request"`
	}
	if err := json.Unmarshal(requestData, &requestInfo); err != nil {
		return nil, errors.NewAppError(http.StatusBadRequest, constants.ErrInvalidRequestBody, err)
	}

	newRequest, err := models.CreateRequestFromJSON(requestInfo.Request)
	if err != nil {
		return nil, errors.NewAppError(http.StatusBadRequest, constants.ErrInvalidRequest, err)
	}

	targetItems := &collection.Items
	for _, folderName := range requestInfo.FolderPath {
		found := false
		for i, item := range *targetItems {
			if item.Folder != nil && item.Name == folderName {
				targetItems = &(*targetItems)[i].Folder.Items
				found = true
				break
			}
		}
		if !found {
			return nil, errors.NewAppError(http.StatusNotFound, constants.ErrFolderNotFound, nil)
		}
	}

	*targetItems = append(*targetItems, models.Item{
		Name:    newRequest.GetName(),
		Request: requestInfo.Request,
	})

	if err := collection.Save(s.DefaultPath); err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToSaveCollection, err)
	}

	return newRequest, nil
}

func (s *CollectionService) DeleteRequestFromCollection(collectionID string, requestID string) error {
	collection, err := models.LoadCollection(s.DefaultPath, collectionID)
	if err != nil {
		return errors.NewAppError(http.StatusNotFound, constants.ErrFailedToLoadCollection, err)
	}

	deleted := false
	var removeRequestFromItems func(*[]models.Item) bool
	removeRequestFromItems = func(items *[]models.Item) bool {
		for i, item := range *items {
			if item.Request != nil {
				var req struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal(item.Request, &req); err == nil && req.ID == requestID {
					*items = append((*items)[:i], (*items)[i+1:]...)
					return true
				}
			}
			if item.Folder != nil {
				if removeRequestFromItems(&item.Folder.Items) {
					return true
				}
			}
		}
		return false
	}

	deleted = removeRequestFromItems(&collection.Items)

	if !deleted {
		return errors.NewAppError(http.StatusNotFound, constants.ErrRequestNotFound, nil)
	}

	if err := collection.Save(s.DefaultPath); err != nil {
		return errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToSaveCollection, err)
	}

	return nil
}

func (s *CollectionService) ExportCollection(collectionID string) (*models.ExportedCollection, error) {
	collection, err := models.LoadCollection(s.DefaultPath, collectionID)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NewAppError(http.StatusNoContent, constants.ErrFailedToLoadCollection, err)
		}
		return nil, errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToLoadCollection, err)
	}

	exportedCollection := collection.ToExportedCollection()

	return &exportedCollection, nil
}

func (s *CollectionService) ImportCollection(importedCollectionData []byte) (*models.Collection, error) {
	var importedCollection models.ImportedCollection
	if err := json.Unmarshal(importedCollectionData, &importedCollection); err != nil {
		return nil, errors.NewAppError(http.StatusBadRequest, constants.ErrFailedToReadCollection, err)
	}

	newCollection, err := models.NewCollectionFromImported(importedCollection)
	if err != nil {
		return nil, errors.NewAppError(http.StatusBadRequest, constants.ErrInvalidCollection, err)
	}

	newCollection.ID = uuid.New().String()

	if err := newCollection.Save(s.DefaultPath); err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToSaveCollection, err)
	}

	return newCollection, nil
}
