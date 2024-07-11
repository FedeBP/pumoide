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

	updatedCollection.ID = id

	if err := updatedCollection.Save(s.DefaultPath); err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToSaveCollection, err)
	}

	return &updatedCollection, nil
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
		Name       string          `json:"name"`
		Request    json.RawMessage `json:"request"`
	}
	if err := json.Unmarshal(requestData, &requestInfo); err != nil {
		return nil, errors.NewAppError(http.StatusBadRequest, constants.ErrInvalidRequestBody, err)
	}

	newRequest, err := models.CreateRequestFromJSON(requestInfo.Name, requestInfo.Request)
	if err != nil {
		return nil, errors.NewAppError(http.StatusBadRequest, constants.ErrInvalidRequest, err)
	}

	if err := s.addRequestToFolder(&collection.Item, requestInfo.FolderPath, newRequest); err != nil {
		return nil, err
	}

	if err := collection.Save(s.DefaultPath); err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToSaveCollection, err)
	}

	return newRequest, nil
}

func (s *CollectionService) addRequestToFolder(items *[]models.Item, folderPath []string, request domain.Request) error {
	if len(folderPath) == 0 {
		requestJSON, err := json.Marshal(request)
		if err != nil {
			return errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToEncodeRequest, err)
		}
		*items = append(*items, models.Item{
			ID:      uuid.New().String(),
			Name:    request.GetName(),
			Request: requestJSON,
		})
		return nil
	}

	for i := range *items {
		if (*items)[i].Name == folderPath[0] {
			if len(folderPath) == 1 {
				if (*items)[i].Item == nil {
					(*items)[i].Item = []models.Item{}
				}
				requestJSON, err := json.Marshal(request)
				if err != nil {
					return errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToEncodeRequest, err)
				}
				(*items)[i].Item = append((*items)[i].Item, models.Item{
					ID:      uuid.New().String(),
					Name:    request.GetName(),
					Request: requestJSON,
				})
				return nil
			}
			return s.addRequestToFolder(&(*items)[i].Item, folderPath[1:], request)
		}
	}

	return errors.NewAppError(http.StatusNotFound, constants.ErrFolderNotFound, nil)
}

func (s *CollectionService) DeleteRequestFromCollection(collectionID string, requestID string) error {
	collection, err := models.LoadCollection(s.DefaultPath, collectionID)
	if err != nil {
		return errors.NewAppError(http.StatusNotFound, constants.ErrFailedToLoadCollection, err)
	}

	deleted := s.deleteRequestFromItems(&collection.Item, requestID)
	if !deleted {
		return errors.NewAppError(http.StatusNotFound, constants.ErrRequestNotFound, nil)
	}

	if err := collection.Save(s.DefaultPath); err != nil {
		return errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToSaveCollection, err)
	}

	return nil
}

func (s *CollectionService) deleteRequestFromItems(items *[]models.Item, requestID string) bool {
	for i := range *items {
		if (*items)[i].Request != nil {
			var req struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal((*items)[i].Request, &req); err == nil && req.ID == requestID {
				*items = append((*items)[:i], (*items)[i+1:]...)
				return true
			}
		}
		if len((*items)[i].Item) > 0 {
			if s.deleteRequestFromItems(&(*items)[i].Item, requestID) {
				return true
			}
		}
	}
	return false
}

func (s *CollectionService) ExportCollection(collectionID string) (*models.Collection, error) {
	collection, err := models.LoadCollection(s.DefaultPath, collectionID)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NewAppError(http.StatusNoContent, constants.ErrCollectionNotFound, err)
		}
		return nil, errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToLoadCollection, err)
	}

	exportableCollection := collection.ToExportable()

	if exportableCollection == nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToExportCollection, nil)
	}

	return exportableCollection, nil
}

func (s *CollectionService) AddFolderToCollection(collectionID string, folderName string, parentPath []string) error {
	collection, err := models.LoadCollection(s.DefaultPath, collectionID)
	if err != nil {
		return errors.NewAppError(http.StatusNotFound, constants.ErrFailedToLoadCollection, err)
	}

	if err := collection.AddFolder(folderName, parentPath...); err != nil {
		return err
	}

	if err := collection.Save(s.DefaultPath); err != nil {
		return errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToSaveCollection, err)
	}

	return nil
}

func (s *CollectionService) DeleteFolderFromCollection(collectionID string, folderPath []string) error {
	collection, err := models.LoadCollection(s.DefaultPath, collectionID)
	if err != nil {
		return errors.NewAppError(http.StatusNotFound, constants.ErrFailedToLoadCollection, err)
	}

	if err := collection.DeleteFolder(folderPath); err != nil {
		return err
	}

	if err := collection.Save(s.DefaultPath); err != nil {
		return errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToSaveCollection, err)
	}

	return nil
}
