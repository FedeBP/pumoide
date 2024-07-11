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
	ID   string `json:"id,omitempty"`
	Info struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"info"`
	Item []Item `json:"item"`
}

type Item struct {
	ID      string          `json:"id,omitempty"`
	Name    string          `json:"name"`
	Request json.RawMessage `json:"request,omitempty"`
	Item    []Item          `json:"item,omitempty"`
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

	return c.processItems(&c.Item)
}

func (c *Collection) processItems(items *[]Item) error {
	for i := range *items {
		item := &(*items)[i]
		if item.Request != nil {
			req, err := CreateRequestFromJSON(item.Name, item.Request)
			if err != nil {
				return err
			}
			item.Request, err = json.Marshal(req)
			if err != nil {
				return err
			}
		}
		if len(item.Item) > 0 {
			if err := c.processItems(&item.Item); err != nil {
				return err
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
	c.generateItemIDs(&c.Item)
}

func (c *Collection) generateItemIDs(items *[]Item) {
	for i := range *items {
		item := &(*items)[i]
		if item.ID == constants.EmptyString {
			item.ID = uuid.New().String()
		}
		if len(item.Item) > 0 {
			c.generateItemIDs(&item.Item)
		}
	}
}

func (c *Collection) Validate() error {
	if c.Info.Name == constants.EmptyString {
		return fmt.Errorf(constants.ErrEmptyCollectionName)
	}

	return c.validateItems(c.Item)
}

func (c *Collection) validateItems(items []Item) error {
	for _, item := range items {
		if item.Request != nil {
			req, err := CreateRequestFromJSON(item.Name, item.Request)
			if err != nil {
				return fmt.Errorf(constants.ErrInvalidRequestAt, item.ID, err)
			}
			if err := req.Validate(); err != nil {
				return fmt.Errorf(constants.ErrInvalidRequestAt, item.ID, err)
			}
		}
		if len(item.Item) > 0 {
			if err := c.validateItems(item.Item); err != nil {
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
		c.Item = append(c.Item, newItem)
	} else {
		if err := c.addItemToFolder(newItem, &c.Item, folderPath); err != nil {
			return err
		}
	}

	return nil
}

func (c *Collection) addItemToFolder(item Item, items *[]Item, folderPath []string) error {
	if len(folderPath) == 0 {
		return errors.NewAppError(http.StatusBadRequest, constants.ErrInvalidFolderPath, nil)
	}

	for i := range *items {
		if (*items)[i].Name == folderPath[0] && len((*items)[i].Item) > 0 {
			if len(folderPath) == 1 {
				(*items)[i].Item = append((*items)[i].Item, item)
				return nil
			}
			return c.addItemToFolder(item, &(*items)[i].Item, folderPath[1:])
		}
	}

	return errors.NewAppError(http.StatusNotFound, constants.ErrFolderNotFound, nil)
}

func (c *Collection) RemoveRequest(requestID string) bool {
	return c.removeRequestFromItems(requestID, &c.Item)
}

func (c *Collection) removeRequestFromItems(requestID string, items *[]Item) bool {
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
			if c.removeRequestFromItems(requestID, &(*items)[i].Item) {
				return true
			}
		}
	}
	return false
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

func (c *Collection) AddFolder(folderName string, parentPath ...string) error {
	newFolder := Item{
		ID:   uuid.New().String(),
		Name: folderName,
		Item: []Item{},
	}

	if len(parentPath) == 0 {
		c.Item = append(c.Item, newFolder)
	} else {
		if err := c.addItemToFolder(newFolder, &c.Item, parentPath); err != nil {
			return err
		}
	}

	return nil
}

func (c *Collection) DeleteFolder(folderPath []string) error {
	if len(folderPath) == 0 {
		return errors.NewAppError(http.StatusBadRequest, constants.ErrInvalidFolderPath, nil)
	}

	return c.deleteFolderFromItems(&c.Item, folderPath)
}

func (c *Collection) deleteFolderFromItems(items *[]Item, folderPath []string) error {
	for i := range *items {
		if (*items)[i].Name == folderPath[0] {
			if len(folderPath) == 1 {
				*items = append((*items)[:i], (*items)[i+1:]...)
				return nil
			}
			return c.deleteFolderFromItems(&(*items)[i].Item, folderPath[1:])
		}
	}
	return errors.NewAppError(http.StatusNotFound, constants.ErrFolderNotFound, nil)
}

func (c *Collection) ToExportable() *Collection {
	exportable := &Collection{
		Info: c.Info,
		Item: make([]Item, len(c.Item)),
	}

	for i, item := range c.Item {
		exportable.Item[i] = item.toExportableItem()
	}

	return exportable
}

func (i Item) toExportableItem() Item {
	exportable := Item{
		Name:    i.Name,
		Request: i.Request,
	}

	if i.Request != nil {
		var requestMap map[string]interface{}
		err := json.Unmarshal(i.Request, &requestMap)
		if err == nil {
			delete(requestMap, "id")
			exportable.Request, err = json.Marshal(requestMap)
			if err != nil {
				exportable.Request = i.Request
			}
		} else {
			exportable.Request = i.Request
		}
	}

	if len(i.Item) > 0 {
		exportable.Item = make([]Item, len(i.Item))
		for j, subItem := range i.Item {
			exportable.Item[j] = subItem.toExportableItem()
		}
	}

	return exportable
}
