package models

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/FedeBP/pumoide/backend/pkg/constants"
	"github.com/FedeBP/pumoide/backend/pkg/errors"
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
	OAuth2 *OAuth2Config     `json:"oauth2,omitempty"`
}

type OAuth2Config struct {
	ClientID     string   `json:"clientId"`
	ClientSecret string   `json:"clientSecret"`
	TokenURL     string   `json:"tokenUrl"`
	AuthURL      string   `json:"authUrl"`
	RedirectURL  string   `json:"redirectUrl"`
	Scopes       []string `json:"scopes"`
	GrantType    string   `json:"grantType"`
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

type PerformanceMetrics struct {
	DNSLookup        time.Duration `json:"dns_lookup"`
	TCPConnection    time.Duration `json:"tcp_connection"`
	TLSHandshake     time.Duration `json:"tls_handshake"`
	ServerProcessing time.Duration `json:"server_processing"`
	ContentTransfer  time.Duration `json:"content_transfer"`
	Total            time.Duration `json:"total"`
}

type Request struct {
	ID                 string              `json:"id"`
	Name               string              `json:"name"`
	Method             Method              `json:"method"`
	URL                string              `json:"url"`
	Headers            []Header            `json:"headers"`
	QueryParams        map[string]string   `json:"queryParams"`
	Body               string              `json:"body"`
	Auth               *Auth               `json:"auth,omitempty"`
	DependsOn          []string            `json:"dependsOn,omitempty"`
	ExtractVariables   map[string]string   `json:"extractVariables,omitempty"`
	ResponseValidation *ResponseValidation `json:"responseValidation,omitempty"`
}

type RequestResult struct {
	Request            Request            `json:"request"`
	Response           Response           `json:"response"`
	Error              string             `json:"error,omitempty"`
	ValidationErrors   []string           `json:"validationErrors,omitempty"`
	PerformanceMetrics PerformanceMetrics `json:"performance_metrics"`
}

type Response struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

type ResponseValidation struct {
	ExpectedStatusCode int               `json:"expectedStatusCode,omitempty"`
	ExpectedHeaders    map[string]string `json:"expectedHeaders,omitempty"`
	JSONSchema         string            `json:"jsonSchema,omitempty"`
	CustomAssertions   []string          `json:"customAssertions,omitempty"`
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
		return errors.NewAppError(http.StatusBadRequest, constants.ErrInvalidCollection, err)
	}

	if c.ID == constants.EmptyString {
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
		return errors.NewAppError(http.StatusBadRequest, constants.ErrInvalidRequest, err)
	}

	if request.ID == constants.EmptyString {
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
			return Collection{}, errors.NewAppError(http.StatusBadRequest, constants.ErrInvalidRequestAt, err)
		}

		newCollection.Requests = append(newCollection.Requests, newRequest)
	}

	if err := newCollection.Validate(); err != nil {
		return Collection{}, errors.NewAppError(http.StatusBadRequest, constants.ErrInvalidCollection, err)
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
	if r.Name == constants.EmptyString {
		return errors.NewAppError(http.StatusBadRequest, constants.ErrEmptyRequestName, nil)
	}

	if !r.Method.IsValid() {
		return errors.NewAppError(http.StatusMethodNotAllowed, fmt.Sprintf(constants.ErrInvalidHTTPMethod+": %s", r.Method), nil)
	}

	if _, err := url.Parse(r.URL); err != nil {
		return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf(constants.ErrInvalidURL+": %s", r.URL), err)
	}

	for _, header := range r.Headers {
		if header.Key == constants.EmptyString {
			return errors.NewAppError(http.StatusBadRequest, constants.ErrEmptyHeaderKey, nil)
		}
	}

	if r.Auth != nil {
		if err := r.Auth.Validate(); err != nil {
			return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf(constants.ErrInvalidAuth, r.Auth), err)
		}
	}

	return nil
}

func (a *Auth) Validate() error {
	switch a.Type {
	case AuthNone:
		return nil

	case AuthBasic:
		if _, ok := a.Params[constants.Username]; !ok {
			return errors.NewAppError(http.StatusBadRequest, constants.ErrBasicAuthUser, nil)
		}
		if _, ok := a.Params[constants.Password]; !ok {
			return errors.NewAppError(http.StatusBadRequest, constants.ErrBasicAuthPass, nil)
		}

	case AuthBearer:
		if _, ok := a.Params[constants.Token]; !ok {
			return errors.NewAppError(http.StatusBadRequest, constants.ErrBearerAuth, nil)
		}

	case AuthAPIKey:
		if _, ok := a.Params[constants.Key]; !ok {
			return errors.NewAppError(http.StatusBadRequest, constants.ErrAPIKey, nil)
		}
		if _, ok := a.Params[constants.Value]; !ok {
			return errors.NewAppError(http.StatusBadRequest, constants.ErrAPIKeyValue, nil)
		}
		if in, ok := a.Params[constants.In]; !ok || (in != constants.Header && in != constants.Query) {
			return errors.NewAppError(http.StatusBadRequest, constants.ErrAPIKeyIn, nil)
		}

	case AuthOAuth2:
		if _, ok := a.Params[constants.AccessToken]; !ok {
			return errors.NewAppError(http.StatusBadRequest, constants.ErrOAuth2Token, nil)
		}

	case AuthAWSSigV4:
		requiredParams := []string{constants.AccessKey, constants.SecretKey, constants.Region, constants.Service}
		for _, param := range requiredParams {
			if _, ok := a.Params[param]; !ok {
				return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf(constants.ErrAuthAWS, param), nil)
			}
		}

	case AuthDigest:
		requiredParams := []string{constants.Username, constants.Password, constants.Realm, constants.Nonce, constants.Qop, constants.NC, constants.Cnonce}
		for _, param := range requiredParams {
			if _, ok := a.Params[param]; !ok {
				return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf(constants.ErrDigestAuth, param), nil)
			}
		}

	default:
		return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf(constants.ErrUnsupportedType, a.Type), nil)
	}

	return nil
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
