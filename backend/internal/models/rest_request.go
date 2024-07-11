package models

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/FedeBP/pumoide/backend/internal/domain"
	"github.com/FedeBP/pumoide/backend/internal/middleware"
	"github.com/FedeBP/pumoide/backend/internal/utils"
	"github.com/FedeBP/pumoide/backend/pkg/constants"
	"github.com/FedeBP/pumoide/backend/pkg/errors"
)

type RESTRequest struct {
	ID                 string                     `json:"id,omitempty"`
	Name               string                     `json:"name,omitempty"`
	Type               domain.RequestType         `json:"type"`
	Method             domain.Method              `json:"method"`
	URL                string                     `json:"url"`
	Body               json.RawMessage            `json:"body,omitempty"`
	Auth               *domain.Auth               `json:"auth,omitempty"`
	Headers            []domain.Header            `json:"header,omitempty"`
	QueryParams        map[string]string          `json:"queryParams,omitempty"`
	DependsOn          []string                   `json:"dependsOn,omitempty"`
	ExtractVariables   map[string]string          `json:"extractVariables,omitempty"`
	Timeout            *time.Duration             `json:"-"`
	ResponseValidation *domain.ResponseValidation `json:"responseValidation,omitempty"`
}

func (r *RESTRequest) Execute(ctx context.Context, env *domain.Environment, transport *http.Transport) (domain.Response, error) {
	var client *http.Client
	if r.Timeout == nil {
		client = &http.Client{
			Timeout:   0,
			Transport: transport,
		}
	} else {
		client = &http.Client{
			Timeout:   *r.Timeout,
			Transport: transport,
		}
	}

	req, err := r.createRequest(ctx)
	if err != nil {
		return nil, err
	}

	if err := r.applyAuthentication(req, env); err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToExecuteRequest, err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			if err == nil {
				err = errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToCloseBody, closeErr)
			}
		}
	}()

	return r.processResponse(resp)
}

func (r *RESTRequest) createRequest(ctx context.Context) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, string(r.Method), r.URL, nil)
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToCreateRequest, err)
	}

	for _, header := range r.Headers {
		req.Header.Set(header.Key, header.Value)
	}

	q := req.URL.Query()
	for key, value := range r.QueryParams {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()

	return req, nil
}

func (r *RESTRequest) applyAuthentication(req *http.Request, env *domain.Environment) error {
	if r.Auth == nil {
		return nil
	}

	if r.Auth.Type == domain.AuthOAuth2 {
		if err := domain.RefreshOAuth2TokenIfNeeded(req, r.Auth); err != nil {
			return errors.NewAppError(http.StatusInternalServerError, constants.ErrRefreshToken, err)
		}
	}

	return middleware.ApplyAuthentication(req, r.Auth, env)
}

func (r *RESTRequest) processResponse(resp *http.Response) (domain.Response, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToReadResponse, err)
	}

	prettyBody, err := utils.PrettyJSON(string(body))
	if err != nil {
		prettyBody = string(body)
	}

	response := &domain.RESTResponse{
		StatusCode: resp.StatusCode,
		Headers:    utils.ConvertHeaders(resp.Header),
		Body:       prettyBody,
	}

	if r.ResponseValidation != nil {
		response.ValidationErrors = response.Validate(r.ResponseValidation)
	}

	return response, nil
}

func (r *RESTRequest) Validate() error {
	if err := validateURL(r.URL); err != nil {
		return err
	}

	if !r.Method.Validate() {
		return errors.NewAppError(http.StatusMethodNotAllowed, fmt.Sprintf(constants.ErrInvalidHTTPMethod+": %s", r.Method), nil)
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

func (r *RESTRequest) MarshalJSON() ([]byte, error) {
	type Alias RESTRequest
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

func (r *RESTRequest) UnmarshalJSON(data []byte) error {
	type Alias RESTRequest
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

func (r *RESTRequest) GetID() string {
	return r.ID
}
func (r *RESTRequest) SetID(ID string) {
	r.ID = ID
}
func (r *RESTRequest) GetName() string {
	return r.Name
}
func (r *RESTRequest) GetType() domain.RequestType {
	return r.Type
}
func (r *RESTRequest) GetAuth() *domain.Auth {
	return r.Auth
}
func (r *RESTRequest) GetHeaders() []domain.Header {
	return r.Headers
}
func (r *RESTRequest) SetHeaders(headers []domain.Header) {
	r.Headers = headers
}
func (r *RESTRequest) GetQueryParams() map[string]string {
	return r.QueryParams
}
func (r *RESTRequest) SetQueryParams(params map[string]string) {
	r.QueryParams = params
}
func (r *RESTRequest) GetDependsOn() []string {
	return r.DependsOn
}
func (r *RESTRequest) GetExtractVariables() map[string]string {
	return r.ExtractVariables
}
func (r *RESTRequest) GetTimeout() *time.Duration {
	return r.Timeout
}
func (r *RESTRequest) GetURL() string {
	return r.URL
}
func (r *RESTRequest) SetURL(url string) {
	r.URL = url
}
func (r *RESTRequest) GetBodyOrMessage() string {
	return string(r.Body)
}
func (r *RESTRequest) SetBodyOrMessage(s string) {
	r.Body = json.RawMessage(s)
}
func (r *RESTRequest) GetContext() context.Context {
	return context.Background()
}
