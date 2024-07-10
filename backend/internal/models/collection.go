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
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Requests    []domain.Request `json:"requests"`
}

type ImportedCollection struct {
	Info struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"info"`
	Item []struct {
		Name    string `json:"name"`
		Request struct {
			Method string          `json:"method"`
			URL    string          `json:"url"`
			Header []domain.Header `json:"header"`
			Body   struct {
				Mode string `json:"mode"`
				Raw  string `json:"raw"`
			} `json:"body"`
			GraphQL *struct {
				Query     string                 `json:"query"`
				Variables map[string]interface{} `json:"variables"`
			} `json:"graphql,omitempty"`
		} `json:"request"`
	} `json:"item"`
}

type ExportedCollection struct {
	Info struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Schema      string `json:"schema"`
	} `json:"info"`
	Item []struct {
		Name    string `json:"name"`
		Request struct {
			Method  string            `json:"method"`
			URL     string            `json:"url"`
			Header  []domain.Header   `json:"header"`
			Body    map[string]string `json:"body,omitempty"`
			GraphQL *struct {
				Query     string                 `json:"query"`
				Variables map[string]interface{} `json:"variables"`
			} `json:"graphql,omitempty"`
		} `json:"request"`
	} `json:"item"`
}

func (c *Collection) Save(path string) error {
	if err := c.Validate(); err != nil {
		return errors.NewAppError(http.StatusBadRequest, constants.ErrInvalidCollection, err)
	}

	if c.ID == constants.EmptyString {
		c.ID = uuid.New().String()
	}

	for _, req := range c.Requests {
		if req.GetID() == constants.EmptyString {
			req.SetID(uuid.New().String())
		}
	}

	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(path, c.ID+".json"), data, 0644)
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

func (c *Collection) AddRequest(request domain.Request) error {
	if err := request.Validate(); err != nil {
		return errors.NewAppError(http.StatusBadRequest, constants.ErrInvalidRequest, err)
	}

	if request.GetID() == constants.EmptyString {
		request.SetID(uuid.New().String())
	}
	c.Requests = append(c.Requests, request)
	return nil
}

func (c *Collection) RemoveRequest(requestID string) bool {
	for i, req := range c.Requests {
		if req.GetID() == requestID {
			c.Requests = append(c.Requests[:i], c.Requests[i+1:]...)
			return true
		}
	}
	return false
}

func (c *Collection) ToExportedCollection() ExportedCollection {
	exported := ExportedCollection{}
	exported.Info.Name = c.Name
	exported.Info.Description = c.Description
	exported.Info.Schema = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"

	for _, req := range c.Requests {
		item := struct {
			Name    string `json:"name"`
			Request struct {
				Method  string            `json:"method"`
				URL     string            `json:"url"`
				Header  []domain.Header   `json:"header"`
				Body    map[string]string `json:"body,omitempty"`
				GraphQL *struct {
					Query     string                 `json:"query"`
					Variables map[string]interface{} `json:"variables"`
				} `json:"graphql,omitempty"`
			} `json:"request"`
		}{
			Name: req.GetName(),
		}

		switch r := req.(type) {
		case *RESTRequest:
			item.Request.Method = string(r.Method)
			item.Request.URL = r.URL
			item.Request.Header = r.GetHeaders()
			item.Request.Body = map[string]string{
				"mode": "raw",
				"raw":  r.Body,
			}
		case *WebSocketRequest:
			item.Request.Method = "websocket"
			item.Request.URL = r.URL
			for _, header := range r.Headers {
				item.Request.Header = append(item.Request.Header, domain.Header{Key: header.Key, Value: header.Value})
			}
			item.Request.Body = map[string]string{
				"mode": "raw",
				"raw":  r.Message,
			}
		case *GraphQLRequest:
			item.Request.Method = "POST"
			item.Request.URL = r.URL
			item.Request.Header = r.GetHeaders()
			item.Request.GraphQL = &struct {
				Query     string                 `json:"query"`
				Variables map[string]interface{} `json:"variables"`
			}{
				Query:     r.Query,
				Variables: r.Variables,
			}
		}

		exported.Item = append(exported.Item, item)
	}

	return exported
}

func NewCollectionFromImported(imported ImportedCollection) (*Collection, error) {
	newCollection := &Collection{
		ID:          uuid.New().String(),
		Name:        imported.Info.Name,
		Description: imported.Info.Description,
	}

	for _, item := range imported.Item {
		var newRequest domain.Request

		if item.Request.GraphQL != nil {
			graphqlReq := &GraphQLRequest{
				ID:        uuid.New().String(),
				Name:      item.Name,
				Type:      domain.RequestTypeGraphQL,
				URL:       item.Request.URL,
				Query:     item.Request.GraphQL.Query,
				Variables: item.Request.GraphQL.Variables,
			}
			for _, header := range item.Request.Header {
				graphqlReq.Headers = append(graphqlReq.Headers, domain.Header{Key: header.Key, Value: header.Value})
			}
			newRequest = graphqlReq
		} else if item.Request.Method == "websocket" {
			wsReq := &WebSocketRequest{
				ID:          uuid.New().String(),
				Name:        item.Name,
				URL:         item.Request.URL,
				MessageType: "text",
				Message:     item.Request.Body.Raw,
			}
			for _, header := range item.Request.Header {
				wsReq.Headers = append(wsReq.Headers, domain.Header{Key: header.Key, Value: header.Value})
			}
			newRequest = wsReq
		} else {
			restReq := &RESTRequest{
				ID:      uuid.New().String(),
				Name:    item.Name,
				Headers: item.Request.Header,
				Method:  domain.Method(item.Request.Method),
				URL:     item.Request.URL,
			}
			if item.Request.Body.Mode == "raw" {
				restReq.Body = item.Request.Body.Raw
			}
			newRequest = restReq
		}

		if err := newRequest.Validate(); err != nil {
			return nil, errors.NewAppError(http.StatusBadRequest, constants.ErrInvalidRequestAt, err)
		}

		newCollection.Requests = append(newCollection.Requests, newRequest)
	}

	if err := newCollection.Validate(); err != nil {
		return nil, errors.NewAppError(http.StatusBadRequest, constants.ErrInvalidCollection, err)
	}

	return newCollection, nil
}

func (c *Collection) Validate() error {
	if c.Name == constants.EmptyString {
		return fmt.Errorf(constants.ErrEmptyCollectionName)
	}

	for i, req := range c.Requests {
		if err := req.Validate(); err != nil {
			return fmt.Errorf(constants.ErrInvalidRequestAt, i, err)
		}
	}

	return nil
}

func (c *Collection) UnmarshalJSON(data []byte) error {
	type Alias Collection
	aux := &struct {
		Requests []json.RawMessage `json:"requests"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	c.Requests = make([]domain.Request, len(aux.Requests))
	for i, raw := range aux.Requests {
		var requestType struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &requestType); err != nil {
			return err
		}

		var request domain.Request
		switch requestType.Type {
		case "rest":
			request = &RESTRequest{}
		case "websocket":
			request = &WebSocketRequest{}
		case "graphql":
			request = &GraphQLRequest{}
		default:
			return fmt.Errorf("unknown request type: %s", requestType.Type)
		}

		if err := json.Unmarshal(raw, request); err != nil {
			return err
		}

		c.Requests[i] = request
	}

	return nil
}
