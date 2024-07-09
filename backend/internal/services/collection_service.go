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

	collection.ID = uuid.New().String()

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
	existingCollection.Requests = updatedCollection.Requests

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

	var requestType struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(requestData, &requestType); err != nil {
		return nil, errors.NewAppError(http.StatusBadRequest, constants.ErrInvalidRequestBody, err)
	}

	var newRequest domain.Request
	switch requestType.Type {
	case "rest":
		newRequest = &models.RESTRequest{}
	case "websocket":
		newRequest = &models.WebSocketRequest{}
	default:
		return nil, errors.NewAppError(http.StatusBadRequest, "Unknown request type", nil)
	}

	if err := json.Unmarshal(requestData, newRequest); err != nil {
		return nil, errors.NewAppError(http.StatusBadRequest, constants.ErrInvalidRequestBody, err)
	}

	if err := newRequest.Validate(); err != nil {
		return nil, errors.NewAppError(http.StatusBadRequest, constants.ErrInvalidRequest, err)
	}

	if newRequest.GetID() == "" {
		newRequest.SetID(uuid.New().String())
	}

	collection.Requests = append(collection.Requests, newRequest)

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

	if !collection.RemoveRequest(requestID) {
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
			return nil, errors.NewAppError(http.StatusNotFound, constants.ErrFailedToLoadCollection, err)
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
