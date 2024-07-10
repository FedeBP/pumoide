package models

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
			name: "Empty URL",
			request: models.WebSocketRequest{
				Name:        "Test WebSocket",
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
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
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
	defer server.Close()

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

	reqNilTimeout := &models.WebSocketRequest{
		Name:        "Test WebSocket No Timeout",
		URL:         "ws://example.com/socket",
		MessageType: "text",
		Message:     "Hello, WebSocket!",
	}

	dataNilTimeout, err := json.Marshal(reqNilTimeout)
	require.NoError(t, err)

	var jsonMapNilTimeout map[string]interface{}
	err = json.Unmarshal(dataNilTimeout, &jsonMapNilTimeout)
	require.NoError(t, err)

	assert.Equal(t, "Test WebSocket No Timeout", jsonMapNilTimeout["name"])
	assert.Equal(t, "ws://example.com/socket", jsonMapNilTimeout["url"])
	assert.Equal(t, "text", jsonMapNilTimeout["messageType"])
	assert.Equal(t, "Hello, WebSocket!", jsonMapNilTimeout["message"])
	_, timeoutExists := jsonMapNilTimeout["timeout"]
	assert.False(t, timeoutExists)

	var unmarshaledNilTimeout models.WebSocketRequest
	err = json.Unmarshal(dataNilTimeout, &unmarshaledNilTimeout)
	require.NoError(t, err)

	assert.Equal(t, reqNilTimeout.Name, unmarshaledNilTimeout.Name)
	assert.Equal(t, reqNilTimeout.URL, unmarshaledNilTimeout.URL)
	assert.Equal(t, reqNilTimeout.MessageType, unmarshaledNilTimeout.MessageType)
	assert.Equal(t, reqNilTimeout.Message, unmarshaledNilTimeout.Message)
	assert.Nil(t, unmarshaledNilTimeout.Timeout)
}
