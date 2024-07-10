package models

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/FedeBP/pumoide/backend/internal/domain"
	"github.com/FedeBP/pumoide/backend/internal/middleware"
	"github.com/FedeBP/pumoide/backend/internal/utils"
	"github.com/FedeBP/pumoide/backend/pkg/constants"
	"github.com/FedeBP/pumoide/backend/pkg/errors"
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
	Timeout            *time.Duration             `json:"-"`
	ResponseValidation *domain.ResponseValidation `json:"responseValidation,omitempty"`
}

func (r *WebSocketRequest) Execute(ctx context.Context, env *domain.Environment, _ *http.Transport) (domain.Response, error) {
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = *r.Timeout

	header := r.prepareHeaders()

	if err := r.applyAuthentication(header, env); err != nil {
		return nil, err
	}

	conn, err := r.establishConnection(ctx, dialer, header)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			log.Printf("Error closing WebSocket connection: %v", closeErr)
			if err == nil {
				err = errors.NewAppError(http.StatusInternalServerError, "Failed to close WebSocket connection", closeErr)
			}
		}
	}()

	if err := conn.WriteMessage(websocket.TextMessage, []byte(r.Message)); err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, "Failed to send ws message", err)
	}

	messages, err := r.readMessages(ctx, conn)
	if err != nil {
		return nil, err
	}

	return r.createResponse(messages), nil
}

func (r *WebSocketRequest) prepareHeaders() http.Header {
	header := http.Header{}
	for _, h := range r.Headers {
		header.Set(h.Key, h.Value)
	}
	return header
}

func (r *WebSocketRequest) applyAuthentication(header http.Header, env *domain.Environment) error {
	if r.Auth == nil {
		return nil
	}

	dummyReq, err := http.NewRequest("GET", r.URL, nil)
	if err != nil {
		return errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToCreateRequest, err)
	}
	dummyReq.Header = header

	if r.Auth.Type == domain.AuthOAuth2 {
		if err := domain.RefreshOAuth2TokenIfNeeded(dummyReq, r.Auth); err != nil {
			return errors.NewAppError(http.StatusInternalServerError, "Failed to refresh token", err)
		}
	}

	if err := middleware.ApplyAuthentication(dummyReq, r.Auth, env); err != nil {
		return errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToAuthenticate, err)
	}

	header = dummyReq.Header
	return nil
}

func (r *WebSocketRequest) establishConnection(ctx context.Context, dialer *websocket.Dialer, header http.Header) (*websocket.Conn, error) {
	dialCtx, cancel := context.WithTimeout(ctx, *r.Timeout)
	defer cancel()

	conn, _, err := dialer.DialContext(dialCtx, r.URL, header)
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, "Failed to establish ws connection", err)
	}

	return conn, nil
}

func (r *WebSocketRequest) readMessages(ctx context.Context, conn *websocket.Conn) ([]interface{}, error) {
	messages := make([]interface{}, 0)
	for i := 0; i < 5; i++ {
		select {
		case <-ctx.Done():
			if len(messages) == 0 {
				return nil, errors.NewAppError(http.StatusRequestTimeout, "Timed out", ctx.Err())
			}
			return messages, nil
		default:
			if err := conn.SetReadDeadline(time.Now().Add(*r.Timeout)); err != nil {
				return nil, errors.NewAppError(http.StatusInternalServerError, "Failed to set read deadline", err)
			}

			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					return nil, errors.NewAppError(http.StatusInternalServerError, "ws closed", err)
				}
				return messages, nil
			}

			prettyMessage, err := utils.PrettyJSON(string(message))
			if err != nil {
				messages = append(messages, string(message))
			} else {
				messages = append(messages, prettyMessage)
			}
		}
	}
	return messages, nil
}

func (r *WebSocketRequest) createResponse(messages []interface{}) domain.Response {
	response := &domain.WebSocketResponse{
		Messages: messages,
	}

	if r.ResponseValidation != nil {
		response.ValidationErrors = response.Validate(r.ResponseValidation)
	}

	return response
}

func (r *WebSocketRequest) Validate() error {
	if r == nil {
		return fmt.Errorf("WebSocket request is nil")
	}

	if err := validateURL(r.URL); err != nil {
		return err
	}

	if r.MessageType != "text" && r.MessageType != "binary" {
		return fmt.Errorf("invalid message type: must be 'text' or 'binary'")
	}

	if r.Message == "" {
		return fmt.Errorf("WebSocket message is required")
	}

	if *r.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than 0")
	}

	return nil
}

func (r *WebSocketRequest) MarshalJSON() ([]byte, error) {
	type Alias WebSocketRequest
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

func (r *WebSocketRequest) UnmarshalJSON(data []byte) error {
	type Alias WebSocketRequest
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
func (r *WebSocketRequest) GetTimeout() *time.Duration {
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
func (r *WebSocketRequest) GetContext() context.Context {
	return context.Background()
}
