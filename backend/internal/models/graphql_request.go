package models

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/FedeBP/pumoide/backend/internal/domain"
	"github.com/FedeBP/pumoide/backend/internal/middleware"
	"github.com/FedeBP/pumoide/backend/internal/utils"
	"github.com/FedeBP/pumoide/backend/pkg/constants"
	"github.com/FedeBP/pumoide/backend/pkg/errors"
)

type GraphQLRequest struct {
	ID                 string                     `json:"id,omitempty"`
	Name               string                     `json:"name,omitempty"`
	Type               domain.RequestType         `json:"type"`
	URL                string                     `json:"url"`
	Query              string                     `json:"query"`
	Variables          map[string]interface{}     `json:"variables,omitempty"`
	OperationName      string                     `json:"operationName,omitempty"`
	Auth               *domain.Auth               `json:"auth,omitempty"`
	Headers            []domain.Header            `json:"headers,omitempty"`
	DependsOn          []string                   `json:"dependsOn,omitempty"`
	ExtractVariables   map[string]string          `json:"extractVariables,omitempty"`
	Timeout            *time.Duration             `json:"-"`
	ResponseValidation *domain.ResponseValidation `json:"responseValidation,omitempty"`
}

func (r *GraphQLRequest) Execute(ctx context.Context, env *domain.Environment, transport *http.Transport) (domain.Response, error) {
	client := &http.Client{
		Timeout:   *r.Timeout,
		Transport: transport,
	}

	payload := map[string]interface{}{
		"query":         r.Query,
		"variables":     r.Variables,
		"operationName": r.OperationName,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToEncodeRequest, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.URL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToCreateRequest, err)
	}

	req.Header.Set(constants.ContentType, constants.AppJson)
	for _, header := range r.Headers {
		req.Header.Set(header.Key, header.Value)
	}

	if r.Auth != nil {
		if err := r.applyAuthentication(req, env); err != nil {
			return nil, err
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToExecuteRequest, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToReadResponse, err)
	}

	prettyBody, err := utils.PrettyJSON(string(body))
	if err != nil {
		prettyBody = string(body)
	}

	response := &domain.GraphQLResponse{
		StatusCode: resp.StatusCode,
		Headers:    utils.ConvertHeaders(resp.Header),
		Body:       prettyBody,
	}

	if r.ResponseValidation != nil {
		response.ValidationErrors = response.Validate(r.ResponseValidation)
	}

	return response, nil
}

func (r *GraphQLRequest) applyAuthentication(req *http.Request, env *domain.Environment) error {
	if r.Auth.Type == domain.AuthOAuth2 {
		if err := domain.RefreshOAuth2TokenIfNeeded(req, r.Auth); err != nil {
			return errors.NewAppError(http.StatusInternalServerError, "Failed to refresh token", err)
		}
	}
	return middleware.ApplyAuthentication(req, r.Auth, env)
}

func (r *GraphQLRequest) Validate() error {
	if r.Name == constants.EmptyString {
		return errors.NewAppError(http.StatusBadRequest, constants.ErrEmptyRequestName, nil)
	}

	if err := validateURL(r.URL); err != nil {
		return err
	}

	if r.Query == constants.EmptyString {
		return errors.NewAppError(http.StatusBadRequest, constants.ErrEmptyGraphQLQuery, nil)
	}

	if err := validateHeaders(r.Headers); err != nil {
		return err
	}

	if r.Auth != nil {
		if err := r.Auth.Validate(); err != nil {
			return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf(constants.ErrInvalidAuth, r.Auth.Type), err)
		}
	}

	return nil
}

func validateURL(urlString string) error {
	u, err := url.Parse(urlString)
	if err != nil || (u.Scheme == constants.EmptyString && u.Host == constants.EmptyString) {
		return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf(constants.ErrInvalidURL+": %s", urlString), err)
	}
	return nil
}

func validateHeaders(headers []domain.Header) error {
	for _, header := range headers {
		if header.Key == constants.EmptyString {
			return errors.NewAppError(http.StatusBadRequest, constants.ErrEmptyHeaderKey, nil)
		}
	}
	return nil
}

func (r *GraphQLRequest) SetQuery(query string) {
	r.Query = strings.Join(strings.Fields(strings.TrimSpace(query)), " ")
}

func (r *GraphQLRequest) MarshalJSON() ([]byte, error) {
	type Alias GraphQLRequest
	aux := struct {
		*Alias
		Timeout string `json:"timeout,omitempty"`
	}{
		Alias: (*Alias)(r),
	}
	if r.Timeout != nil {
		aux.Timeout = r.Timeout.String()
	}
	return json.Marshal(aux)
}

func (r *GraphQLRequest) UnmarshalJSON(data []byte) error {
	type Alias GraphQLRequest
	aux := struct {
		*Alias
		Timeout string `json:"timeout,omitempty"`
	}{
		Alias: (*Alias)(r),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Timeout != "" {
		duration, err := time.ParseDuration(aux.Timeout)
		if err != nil {
			return err
		}
		r.Timeout = &duration
	}
	return nil
}

func (r *GraphQLRequest) GetID() string {
	return r.ID
}
func (r *GraphQLRequest) SetID(id string) {
	r.ID = id
}
func (r *GraphQLRequest) GetName() string {
	return r.Name
}
func (r *GraphQLRequest) GetType() domain.RequestType {
	return r.Type
}
func (r *GraphQLRequest) GetAuth() *domain.Auth {
	return r.Auth
}
func (r *GraphQLRequest) GetHeaders() []domain.Header {
	return r.Headers
}
func (r *GraphQLRequest) SetHeaders(headers []domain.Header) {
	r.Headers = headers
}
func (r *GraphQLRequest) GetQueryParams() map[string]string {
	return nil
}
func (r *GraphQLRequest) SetQueryParams(params map[string]string) {}

func (r *GraphQLRequest) GetDependsOn() []string {
	return r.DependsOn
}
func (r *GraphQLRequest) GetExtractVariables() map[string]string {
	return r.ExtractVariables
}
func (r *GraphQLRequest) GetTimeout() *time.Duration {
	return r.Timeout
}
func (r *GraphQLRequest) GetURL() string {
	return r.URL
}
func (r *GraphQLRequest) SetURL(url string) {
	r.URL = url
}
func (r *GraphQLRequest) GetBodyOrMessage() string {
	return r.Query
}
func (r *GraphQLRequest) SetBodyOrMessage(s string) {
	r.Query = s
}

func (r *GraphQLRequest) GetContext() context.Context {
	return context.Background()
}
