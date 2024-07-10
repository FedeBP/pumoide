package models

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/FedeBP/pumoide/backend/internal/domain"
	"github.com/FedeBP/pumoide/backend/pkg/constants"
	"github.com/FedeBP/pumoide/backend/pkg/errors"
	"github.com/google/uuid"
)

type Collection struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Items       []Item `json:"items"`
}

type Item struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Request json.RawMessage `json:"request,omitempty"`
	Folder  *Folder         `json:"folder,omitempty"`
}

type Folder struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Items []Item `json:"items"`
}

func (c *Collection) UnmarshalJSON(data []byte) error {
	type Alias Collection
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	for i, item := range c.Items {
		if item.Request != nil {
			req, err := CreateRequestFromJSON(item.Request)
			if err != nil {
				return err
			}
			c.Items[i].Request, err = json.Marshal(req)
			if err != nil {
				return err
			}
		}
		if item.Folder != nil {
			for j, subItem := range item.Folder.Items {
				if subItem.Request != nil {
					req, err := CreateRequestFromJSON(subItem.Request)
					if err != nil {
						return err
					}
					c.Items[i].Folder.Items[j].Request, err = json.Marshal(req)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func (c *Collection) Save(path string) error {
	if err := c.Validate(); err != nil {
		return errors.NewAppError(http.StatusBadRequest, constants.ErrInvalidCollection, err)
	}

	if c.ID == constants.EmptyString {
		c.ID = uuid.New().String()
	}

	c.generateIDs()

	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(path, c.ID+".json"), data, 0644)
}

func (c *Collection) generateIDs() {
	for i := range c.Items {
		c.generateItemID(&c.Items[i])
	}
}

func (c *Collection) generateItemID(item *Item) {
	if item.ID == constants.EmptyString {
		item.ID = uuid.New().String()
	}
	if item.Folder != nil {
		if item.Folder.ID == constants.EmptyString {
			item.Folder.ID = uuid.New().String()
		}
		for i := range item.Folder.Items {
			c.generateItemID(&item.Folder.Items[i])
		}
	}
}

func (c *Collection) Validate() error {
	if c.Name == constants.EmptyString {
		return fmt.Errorf(constants.ErrEmptyCollectionName)
	}

	return c.validateItems(c.Items)
}

func (c *Collection) validateItems(items []Item) error {
	for _, item := range items {
		if item.Request != nil {
			req, err := CreateRequestFromJSON(item.Request)
			if err != nil {
				return fmt.Errorf(constants.ErrInvalidRequestAt, item.ID, err)
			}
			if err := req.Validate(); err != nil {
				return fmt.Errorf(constants.ErrInvalidRequestAt, item.ID, err)
			}
		}
		if item.Folder != nil {
			if err := c.validateItems(item.Folder.Items); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Collection) AddRequest(request domain.Request, folderPath ...string) error {
	if err := request.Validate(); err != nil {
		return errors.NewAppError(http.StatusBadRequest, constants.ErrInvalidRequest, err)
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToEncodeRequest, err)
	}

	newItem := Item{
		ID:      uuid.New().String(),
		Name:    request.GetName(),
		Request: requestJSON,
	}

	if len(folderPath) == 0 {
		c.Items = append(c.Items, newItem)
	} else {
		if err := c.addItemToFolder(newItem, c.Items, folderPath); err != nil {
			return err
		}
	}

	return nil
}

func (c *Collection) addItemToFolder(item Item, items []Item, folderPath []string) error {
	if len(folderPath) == 0 {
		return errors.NewAppError(http.StatusBadRequest, constants.ErrInvalidFolderPath, nil)
	}

	for i, existingItem := range items {
		if existingItem.Folder != nil && existingItem.Folder.Name == folderPath[0] {
			if len(folderPath) == 1 {
				items[i].Folder.Items = append(items[i].Folder.Items, item)
				return nil
			}
			return c.addItemToFolder(item, existingItem.Folder.Items, folderPath[1:])
		}
	}

	return errors.NewAppError(http.StatusNotFound, constants.ErrFolderNotFound, nil)
}

func (c *Collection) RemoveRequest(requestID string) bool {
	return c.removeRequestFromItems(requestID, c.Items)
}

func (c *Collection) removeRequestFromItems(requestID string, items []Item) bool {
	for i, item := range items {
		if item.Request != nil {
			var req struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(item.Request, &req); err == nil && req.ID == requestID {
				items = append(items[:i], items[i+1:]...)
				return true
			}
		}
		if item.Folder != nil {
			if c.removeRequestFromItems(requestID, item.Folder.Items) {
				return true
			}
		}
	}
	return false
}

func (i *Item) GetRequest() (domain.Request, error) {
	if i.Request == nil {
		return nil, nil
	}

	return CreateRequestFromJSON(i.Request)
}

func LoadCollection(path string, id string) (*Collection, error) {
	data, err := os.ReadFile(filepath.Join(path, id+".json"))
	if err != nil {
		return nil, err
	}
	var collection Collection
	err = json.Unmarshal(data, &collection)
	return &collection, err
}
