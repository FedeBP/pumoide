package models

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/FedeBP/pumoide/backend/internal/domain"
	"github.com/FedeBP/pumoide/backend/internal/middleware"
	"github.com/gorilla/websocket"
)

type WebSocketRequest struct {
	ID                 string                     `json:"id,omitempty"`
	Name               string                     `json:"name,omitempty"`
	Type               domain.RequestType         `json:"type"`
	URL                string                     `json:"url"`
	Auth               *domain.Auth               `json:"auth,omitempty"`
	Headers            []domain.Header            `json:"headers,omitempty"`
	QueryParams        map[string]string          `json:"queryParams,omitempty"`
	DependsOn          []string                   `json:"dependsOn,omitempty"`
	ExtractVariables   map[string]string          `json:"extractVariables,omitempty"`
	Protocols          []string                   `json:"protocols,omitempty"`
	MessageType        string                     `json:"messageType,omitempty"`
	Message            string                     `json:"message,omitempty"`
	Timeout            time.Duration              `json:"timeout,omitempty"`
	ResponseValidation *domain.ResponseValidation `json:"responseValidation,omitempty"`
}

func (r *WebSocketRequest) Execute(env *domain.Environment) (domain.Response, error) {
	dialer := websocket.DefaultDialer

	header := http.Header{}
	for _, h := range r.Headers {
		header.Set(h.Key, h.Value)
	}

	dummyReq, err := http.NewRequest("GET", r.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create dummy request: %v", err)
	}
	dummyReq.Header = header

	if r.Auth != nil {
		if r.Auth.Type == domain.AuthOAuth2 {
			err = domain.RefreshOAuth2TokenIfNeeded(dummyReq, r.Auth)
			if err != nil {
				return nil, err
			}
		}
		err = middleware.ApplyAuthentication(dummyReq, r.Auth, env)
		if err != nil {
			return nil, err
		}
	}

	header = dummyReq.Header

	conn, _, err := dialer.Dial(r.URL, header)
	if err != nil {
		return nil, fmt.Errorf("failed to establish WebSocket connection: %v", err)
	}
	defer func() {
		err := conn.Close()
		if err != nil {
			return
		}
	}()

	err = conn.WriteMessage(websocket.TextMessage, []byte(r.Message))
	if err != nil {
		return nil, fmt.Errorf("failed to send WebSocket message: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout)
	defer cancel()

	var message []byte
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, message, err = conn.ReadMessage()
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout waiting for WebSocket response")
	case <-done:
		if err != nil {
			return nil, fmt.Errorf("error reading WebSocket message: %v", err)
		}
	}

	response := &domain.WebSocketResponse{
		Message: string(message),
	}

	if r.ResponseValidation != nil {
		response.ValidationErrors = response.Validate(r.ResponseValidation)
	}

	return response, nil
}

func (r *WebSocketRequest) Validate() error {
	if r == nil {
		return fmt.Errorf("WebSocket request is nil")
	}

	if r.URL == "" {
		return fmt.Errorf("WebSocket URL is required")
	}

	_, err := url.Parse(r.URL)
	if err != nil {
		return fmt.Errorf("invalid WebSocket URL: %v", err)
	}

	if r.MessageType != "text" && r.MessageType != "binary" {
		return fmt.Errorf("invalid message type: must be 'text' or 'binary'")
	}

	if r.Message == "" {
		return fmt.Errorf("WebSocket message is required")
	}

	if r.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than 0")
	}

	return nil
}

func (r *WebSocketRequest) GetID() string {
	return r.ID
}

func (r *WebSocketRequest) SetID(ID string) {
	r.ID = ID
}

func (r *WebSocketRequest) GetName() string {
	return r.Name
}

func (r *WebSocketRequest) GetType() domain.RequestType {
	return r.Type
}

func (r *WebSocketRequest) GetAuth() *domain.Auth {
	return r.Auth
}

func (r *WebSocketRequest) GetHeaders() []domain.Header {
	return r.Headers
}

func (r *WebSocketRequest) SetHeaders(headers []domain.Header) {
	r.Headers = headers
}

func (r *WebSocketRequest) GetQueryParams() map[string]string {
	return r.QueryParams
}

func (r *WebSocketRequest) SetQueryParams(params map[string]string) {
	r.QueryParams = params
}

func (r *WebSocketRequest) GetDependsOn() []string {
	return r.DependsOn
}

func (r *WebSocketRequest) GetExtractVariables() map[string]string {
	return r.ExtractVariables
}

func (r *WebSocketRequest) GetTimeout() time.Duration {
	return r.Timeout
}

func (r *WebSocketRequest) GetURL() string {
	return r.URL
}

func (r *WebSocketRequest) SetURL(url string) {
	r.URL = url
}

func (r *WebSocketRequest) GetBodyOrMessage() string {
	return r.Message
}

func (r *WebSocketRequest) SetBodyOrMessage(s string) {
	r.Message = s
}
