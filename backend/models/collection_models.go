package models

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/FedeBP/pumoide/backend/apperrors"
	"github.com/google/uuid"
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
	if err := c.Validate(); err != nil {
		return apperrors.NewAppError(http.StatusBadRequest, "Invalid collection", err)
	}

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

func (c *Collection) AddRequest(request Request) error {
	if err := request.Validate(); err != nil {
		return apperrors.NewAppError(http.StatusBadRequest, "Invalid request", err)
	}

	if request.ID == "" {
		request.ID = uuid.New().String()
	}
	c.Requests = append(c.Requests, request)
	return nil
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

func NewCollectionFromImported(imported ImportedCollection) (Collection, error) {
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

		if err := newRequest.Validate(); err != nil {
			return Collection{}, apperrors.NewAppError(http.StatusBadRequest, "Invalid imported request", err)
		}

		newCollection.Requests = append(newCollection.Requests, newRequest)
	}

	if err := newCollection.Validate(); err != nil {
		return Collection{}, apperrors.NewAppError(http.StatusBadRequest, "Invalid imported collection", err)
	}

	return newCollection, nil
}

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

func (m Method) IsValid() bool {
	for _, validMethod := range GetValidMethods() {
		if m == validMethod {
			return true
		}
	}
	return false
}

func (r *Request) Validate() error {
	if r.Name == "" {
		return apperrors.NewAppError(http.StatusBadRequest, "Request name cannot be empty", nil)
	}

	if !r.Method.IsValid() {
		return apperrors.NewAppError(http.StatusMethodNotAllowed, fmt.Sprintf("Invalid HTTP method: %s", r.Method), nil)
	}

	if _, err := url.Parse(r.URL); err != nil {
		return apperrors.NewAppError(http.StatusBadRequest, fmt.Sprintf("invalid URL: %s", r.URL), err)
	}

	for _, header := range r.Headers {
		if header.Key == "" {
			return apperrors.NewAppError(http.StatusBadRequest, "Header key cannot be empty", nil)
		}
	}

	if r.Auth != nil {
		if err := r.Auth.Validate(); err != nil {
			return apperrors.NewAppError(http.StatusBadRequest, fmt.Sprintf("invalid authentication: %s", r.Auth), err)
		}
	}

	return nil
}

func (a *Auth) Validate() error {
	switch a.Type {
	case AuthNone:
		return nil

	case AuthBasic:
		if _, ok := a.Params["username"]; !ok {
			return apperrors.NewAppError(http.StatusBadRequest, "basic auth requires a username", nil)
		}
		if _, ok := a.Params["password"]; !ok {
			return apperrors.NewAppError(http.StatusBadRequest, "basic auth requires a password", nil)
		}

	case AuthBearer:
		if _, ok := a.Params["token"]; !ok {
			return apperrors.NewAppError(http.StatusBadRequest, "bearer auth requires a token", nil)
		}

	case AuthAPIKey:
		if _, ok := a.Params["key"]; !ok {
			return apperrors.NewAppError(http.StatusBadRequest, "API key auth requires a key", nil)
		}
		if _, ok := a.Params["value"]; !ok {
			return apperrors.NewAppError(http.StatusBadRequest, "API key auth requires a value", nil)
		}
		if in, ok := a.Params["in"]; !ok || (in != "header" && in != "query") {
			return apperrors.NewAppError(http.StatusBadRequest, "API key auth requires 'in' to be either 'header' or 'query'", nil)
		}

	case AuthOAuth2:
		if _, ok := a.Params["access_token"]; !ok {
			return apperrors.NewAppError(http.StatusBadRequest, "OAuth2 auth requires an access token", nil)
		}

	case AuthAWSSigV4:
		requiredParams := []string{"access_key", "secret_key", "region", "service"}
		for _, param := range requiredParams {
			if _, ok := a.Params[param]; !ok {
				return apperrors.NewAppError(http.StatusBadRequest, fmt.Sprintf("AWS SigV4 auth requires %s", param), nil)
			}
		}

	case AuthDigest:
		requiredParams := []string{"username", "password", "realm", "nonce", "qop", "nc", "cnonce"}
		for _, param := range requiredParams {
			if _, ok := a.Params[param]; !ok {
				return apperrors.NewAppError(http.StatusBadRequest, fmt.Sprintf("Digest auth requires %s", param), nil)
			}
		}

	default:
		return apperrors.NewAppError(http.StatusBadRequest, fmt.Sprintf("Unsupported auth type: %s", a.Type), nil)
	}

	return nil
}

func (c *Collection) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("collection name cannot be empty")
	}

	for i, req := range c.Requests {
		if err := req.Validate(); err != nil {
			return fmt.Errorf("invalid request at index %d: %w", i, err)
		}
	}

	return nil
}
