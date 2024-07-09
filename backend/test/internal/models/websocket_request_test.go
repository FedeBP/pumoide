package models

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/FedeBP/pumoide/backend/internal/domain"
	"github.com/FedeBP/pumoide/backend/internal/models"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestWebSocketRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request models.WebSocketRequest
		wantErr bool
	}{
		{
			name: "Valid WebSocket request",
			request: models.WebSocketRequest{
				Name:        "Test WebSocket",
				URL:         "ws://example.com/socket",
				MessageType: "text",
				Message:     "Hello, WebSocket!",
				Timeout:     30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "Empty URL",
			request: models.WebSocketRequest{
				Name:        "Test WebSocket",
				MessageType: "text",
				Message:     "Hello, WebSocket!",
				Timeout:     30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "Invalid message type",
			request: models.WebSocketRequest{
				Name:        "Test WebSocket",
				URL:         "ws://example.com/socket",
				MessageType: "invalid",
				Message:     "Hello, WebSocket!",
				Timeout:     30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "Empty message",
			request: models.WebSocketRequest{
				Name:        "Test WebSocket",
				URL:         "ws://example.com/socket",
				MessageType: "text",
				Timeout:     30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "Invalid timeout",
			request: models.WebSocketRequest{
				Name:        "Test WebSocket",
				URL:         "ws://example.com/socket",
				MessageType: "text",
				Message:     "Hello, WebSocket!",
				Timeout:     0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWebSocketRequest_Execute(t *testing.T) {
	var upgrader = websocket.Upgrader{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() {
			err := c.Close()
			if err != nil {
				return
			}
		}()

		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				break
			}
			err = c.WriteMessage(mt, []byte("Echo: "+string(message)))
			if err != nil {
				break
			}
		}
	}))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	req := models.WebSocketRequest{
		Name:        "Test WebSocket",
		URL:         wsURL,
		MessageType: "text",
		Message:     "Hello, WebSocket!",
		Timeout:     30 * time.Second,
	}

	env := &domain.Environment{
		Variables: make(map[string]string),
	}

	response, err := req.Execute(env)
	assert.NoError(t, err)
	assert.NotNil(t, response)

	wsResponse, ok := response.(*domain.WebSocketResponse)
	assert.True(t, ok)
	assert.Contains(t, wsResponse.Message, "Echo: Hello, WebSocket!")
}
