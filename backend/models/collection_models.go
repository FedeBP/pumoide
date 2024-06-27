package models

import (
	"encoding/json"
	"github.com/google/uuid"
	"os"
	"path/filepath"
)

type AuthType string

const (
	AuthNone     AuthType = "none"
	AuthBasic    AuthType = "basic"
	AuthBearer   AuthType = "bearer"
	AuthAPIKey   AuthType = "apiKey"
	AuthOAuth2   AuthType = "oauth2"
	AuthAWSSigV4 AuthType = "awsSigV4"
	AuthDigest   AuthType = "digest"
	AuthNTLM     AuthType = "ntlm"
)

type Auth struct {
	Type   AuthType          `json:"type"`
	Params map[string]string `json:"params"`
}

type Method string

const (
	MethodGet     Method = "GET"
	MethodPost    Method = "POST"
	MethodPut     Method = "PUT"
	MethodDelete  Method = "DELETE"
	MethodPatch   Method = "PATCH"
	MethodHead    Method = "HEAD"
	MethodOptions Method = "OPTIONS"
	MethodTrace   Method = "TRACE"
	MethodConnect Method = "CONNECT"
)

func GetValidMethods() []Method {
	return []Method{
		MethodGet,
		MethodPost,
		MethodPut,
		MethodDelete,
		MethodPatch,
		MethodHead,
		MethodOptions,
		MethodTrace,
		MethodConnect,
	}
}

type Header struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Request struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Method      Method            `json:"method"`
	URL         string            `json:"url"`
	Headers     []Header          `json:"headers"`
	QueryParams map[string]string `json:"queryParams"`
	Body        string            `json:"body"`
	Auth        *Auth             `json:"auth,omitempty"`
}

type Collection struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Requests    []Request `json:"requests"`
}

type ImportedCollection struct {
	Info struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"info"`
	Item []struct {
		Name    string `json:"name"`
		Request struct {
			Method string   `json:"method"`
			URL    string   `json:"url"`
			Header []Header `json:"header"`
			Body   struct {
				Mode string `json:"mode"`
				Raw  string `json:"raw"`
			} `json:"body"`
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
			Method string            `json:"method"`
			URL    string            `json:"url"`
			Header []Header          `json:"header"`
			Body   map[string]string `json:"body"`
		} `json:"request"`
	} `json:"item"`
}

func (c *Collection) Save(path string) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
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

func (c *Collection) AddRequest(request Request) {
	if request.ID == "" {
		request.ID = uuid.New().String()
	}
	c.Requests = append(c.Requests, request)
}

func (c *Collection) RemoveRequest(requestID string) bool {
	for i, req := range c.Requests {
		if req.ID == requestID {
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
				Method string            `json:"method"`
				URL    string            `json:"url"`
				Header []Header          `json:"header"`
				Body   map[string]string `json:"body"`
			} `json:"request"`
		}{
			Name: req.Name,
			Request: struct {
				Method string            `json:"method"`
				URL    string            `json:"url"`
				Header []Header          `json:"header"`
				Body   map[string]string `json:"body"`
			}{
				Method: string(req.Method),
				URL:    req.URL,
				Header: req.Headers,
				Body: map[string]string{
					"mode": "raw",
					"raw":  req.Body,
				},
			},
		}
		exported.Item = append(exported.Item, item)
	}

	return exported
}

func NewCollectionFromImported(imported ImportedCollection) Collection {
	newCollection := Collection{
		ID:          uuid.New().String(),
		Name:        imported.Info.Name,
		Description: imported.Info.Description,
	}

	for _, item := range imported.Item {
		newRequest := Request{
			ID:      uuid.New().String(),
			Name:    item.Name,
			Method:  Method(item.Request.Method),
			URL:     item.Request.URL,
			Headers: item.Request.Header,
		}

		if item.Request.Body.Mode == "raw" {
			newRequest.Body = item.Request.Body.Raw
		}
		newCollection.Requests = append(newCollection.Requests, newRequest)
	}

	return newCollection
}

func (m Method) IsValid() bool {
	for _, validMethod := range GetValidMethods() {
		if m == validMethod {
			return true
		}
	}
	return false
}
