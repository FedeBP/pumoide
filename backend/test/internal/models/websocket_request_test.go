package models_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/FedeBP/pumoide/backend/internal/domain"
	"github.com/FedeBP/pumoide/backend/internal/models"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebSocketRequest_Validate(t *testing.T) {
	timeout := 5 * time.Second
	tests := []struct {
		name    string
		request models.WebSocketRequest
		wantErr bool
	}{
		{
			name: "Valid request",
			request: models.WebSocketRequest{
				Name:        "Test WebSocket",
				URL:         "ws://example.com/socket",
				MessageType: "text",
				Message:     "Hello, WebSocket!",
				Timeout:     &timeout,
			},
			wantErr: false,
		},
		{
			name: "Invalid URL",
			request: models.WebSocketRequest{
				Name:        "Test WebSocket",
				URL:         "invalid-url",
				MessageType: "text",
				Message:     "Hello, WebSocket!",
				Timeout:     &timeout,
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
				Timeout:     &timeout,
			},
			wantErr: true,
		},
		{
			name: "Empty message",
			request: models.WebSocketRequest{
				Name:        "Test WebSocket",
				URL:         "ws://example.com/socket",
				MessageType: "text",
				Timeout:     &timeout,
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
	var serverClosed bool
	var serverCloseError error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("Failed to upgrade connection: %v", err)
			return
		}
		defer func() {
			if err := c.Close(); err != nil {
				serverCloseError = err
			}
			serverClosed = true
		}()
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					t.Logf("ReadMessage error: %v", err)
				}
				break
			}
			err = c.WriteMessage(mt, []byte("Echo: "+string(message)))
			if err != nil {
				t.Logf("WriteMessage error: %v", err)
				break
			}
		}
	}))
	defer func() {
		server.Close()
		if serverCloseError != nil {
			t.Logf("Server WebSocket close error: %v", serverCloseError)
		}
	}()

	timeout := 5 * time.Second
	wsURL := "ws" + server.URL[4:]
	req := models.WebSocketRequest{
		Name:        "Test WebSocket",
		URL:         wsURL,
		MessageType: "text",
		Message:     "Hello, WebSocket!",
		Timeout:     &timeout,
	}

	resp, err := req.Execute(context.Background(), &domain.Environment{}, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)

	wsResp, ok := resp.(*domain.WebSocketResponse)
	require.True(t, ok)
	assert.Len(t, wsResp.Messages, 1)
	assert.Contains(t, wsResp.Messages[0], "Echo: Hello, WebSocket!")

	time.Sleep(100 * time.Millisecond)
	assert.True(t, serverClosed, "Server should have closed the WebSocket connection")
	assert.NoError(t, serverCloseError, "Server should not have encountered an error while closing the WebSocket connection")
}

func TestWebSocketRequest_MarshalUnmarshalJSON(t *testing.T) {
	timeout := 5 * time.Second
	req := &models.WebSocketRequest{
		Name:        "Test WebSocket",
		URL:         "ws://example.com/socket",
		MessageType: "text",
		Message:     "Hello, WebSocket!",
		Timeout:     &timeout,
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var jsonMap map[string]interface{}
	err = json.Unmarshal(data, &jsonMap)
	require.NoError(t, err)

	assert.Equal(t, "Test WebSocket", jsonMap["name"])
	assert.Equal(t, "ws://example.com/socket", jsonMap["url"])
	assert.Equal(t, "text", jsonMap["messageType"])
	assert.Equal(t, "Hello, WebSocket!", jsonMap["message"])
	assert.Equal(t, "5s", jsonMap["timeout"])

	var unmarshaled models.WebSocketRequest
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, req.Name, unmarshaled.Name)
	assert.Equal(t, req.URL, unmarshaled.URL)
	assert.Equal(t, req.MessageType, unmarshaled.MessageType)
	assert.Equal(t, req.Message, unmarshaled.Message)
	require.NotNil(t, unmarshaled.Timeout)
	assert.Equal(t, *req.Timeout, *unmarshaled.Timeout)
}
