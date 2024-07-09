package models

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/FedeBP/pumoide/backend/internal/domain"
	"github.com/FedeBP/pumoide/backend/internal/middleware"
	"github.com/FedeBP/pumoide/backend/pkg/constants"
	"github.com/FedeBP/pumoide/backend/pkg/errors"
)

type RESTRequest struct {
	ID                 string                     `json:"id,omitempty"`
	Name               string                     `json:"name,omitempty"`
	Type               domain.RequestType         `json:"type"`
	Method             domain.Method              `json:"method"`
	URL                string                     `json:"url"`
	Body               string                     `json:"body,omitempty"`
	Auth               *domain.Auth               `json:"auth,omitempty"`
	Headers            []domain.Header            `json:"headers,omitempty"`
	QueryParams        map[string]string          `json:"queryParams,omitempty"`
	DependsOn          []string                   `json:"dependsOn,omitempty"`
	ExtractVariables   map[string]string          `json:"extractVariables,omitempty"`
	Timeout            time.Duration              `json:"timeout,omitempty"`
	ResponseValidation *domain.ResponseValidation `json:"responseValidation,omitempty"`
}

func (r *RESTRequest) Execute(env *domain.Environment) (domain.Response, error) {
	client := &http.Client{
		Timeout: r.Timeout,
	}

	req, err := http.NewRequest(string(r.Method), r.URL, bytes.NewBufferString(r.Body))
	if err != nil {
		return nil, err
	}

	for _, header := range r.Headers {
		req.Header.Set(header.Key, header.Value)
	}

	q := req.URL.Query()
	for key, value := range r.QueryParams {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()

	if r.Auth != nil {
		if r.Auth.Type == domain.AuthOAuth2 {
			err = domain.RefreshOAuth2TokenIfNeeded(req, r.Auth)
			if err != nil {
				return nil, err
			}
		}
		err = middleware.ApplyAuthentication(req, r.Auth, env)
		if err != nil {
			return nil, err
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	response := &domain.RESTResponse{
		StatusCode: resp.StatusCode,
		Headers:    convertHeaders(resp.Header),
		Body:       string(body),
	}

	if r.ResponseValidation != nil {
		response.ValidationErrors = response.Validate(r.ResponseValidation)
	}

	return response, nil
}

func convertHeaders(httpHeaders http.Header) []domain.Header {
	var headers []domain.Header
	for key, values := range httpHeaders {
		for _, value := range values {
			headers = append(headers, domain.Header{Key: key, Value: value})
		}
	}
	return headers
}

func (r *RESTRequest) Validate() error {
	if r.Name == constants.EmptyString {
		return errors.NewAppError(http.StatusBadRequest, constants.ErrEmptyRequestName, nil)
	}

	if !r.Method.Validate() {
		return errors.NewAppError(http.StatusMethodNotAllowed, fmt.Sprintf(constants.ErrInvalidHTTPMethod+": %s", r.Method), nil)
	}

	u, err := url.Parse(r.URL)
	if err != nil || (u.Scheme == constants.EmptyString && u.Host == constants.EmptyString) {
		return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf(constants.ErrInvalidURL+": %s", r.URL), err)
	}

	for _, header := range r.Headers {
		if header.Key == constants.EmptyString {
			return errors.NewAppError(http.StatusBadRequest, constants.ErrEmptyHeaderKey, nil)
		}
	}

	if r.Auth != nil {
		if err := r.Auth.Validate(); err != nil {
			return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf(constants.ErrInvalidAuth, r.Auth.Type), err)
		}
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

func (r *RESTRequest) GetTimeout() time.Duration {
	return r.Timeout
}

func (r *RESTRequest) GetURL() string {
	return r.URL
}

func (r *RESTRequest) SetURL(url string) {
	r.URL = url
}

func (r *RESTRequest) GetBodyOrMessage() string {
	return r.Body
}

func (r *RESTRequest) SetBodyOrMessage(s string) {
	r.Body = s
}
