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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRESTRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request models.RESTRequest
		wantErr bool
	}{
		{
			name: "Valid request",
			request: models.RESTRequest{
				Name:   "Test Request",
				Method: domain.MethodGet,
				URL:    "https://api.example.com/test",
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			request: models.RESTRequest{
				Method: domain.MethodGet,
				URL:    "https://api.example.com/test",
			},
			wantErr: true,
		},
		{
			name: "Invalid method",
			request: models.RESTRequest{
				Name:   "Test Request",
				Method: "INVALID",
				URL:    "https://api.example.com/test",
			},
			wantErr: true,
		},
		{
			name: "Invalid URL",
			request: models.RESTRequest{
				Name:   "Test Request",
				Method: domain.MethodGet,
				URL:    "invalid-url",
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

func TestRESTRequest_Execute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "Hello, World!"}`))
	}))
	defer server.Close()

	timeout := 5 * time.Second
	req := &models.RESTRequest{
		Name:    "Test Request",
		Method:  domain.MethodGet,
		URL:     server.URL,
		Timeout: &timeout,
	}

	transport := &http.Transport{}
	resp, err := req.Execute(context.Background(), &domain.Environment{}, transport)
	require.NoError(t, err)
	require.NotNil(t, resp)

	restResp, ok := resp.(*domain.RESTResponse)
	require.True(t, ok)
	assert.Equal(t, http.StatusOK, restResp.StatusCode)

	bodyMap, ok := restResp.Body.(map[string]interface{})
	require.True(t, ok, "Response body should be a map")
	message, ok := bodyMap["message"].(string)
	require.True(t, ok, "Message should be a string")
	assert.Equal(t, "Hello, World!", message)
}

func TestRESTRequest_MarshalUnmarshalJSON(t *testing.T) {
	timeout := 5 * time.Second
	req := &models.RESTRequest{
		Name:    "Test Request",
		Method:  domain.MethodGet,
		URL:     "https://api.example.com/test",
		Timeout: &timeout,
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var jsonMap map[string]interface{}
	err = json.Unmarshal(data, &jsonMap)
	require.NoError(t, err)

	assert.Equal(t, "Test Request", jsonMap["name"])
	assert.Equal(t, "GET", jsonMap["method"])
	assert.Equal(t, "https://api.example.com/test", jsonMap["url"])
	assert.Equal(t, "5s", jsonMap["timeout"])

	var unmarshaled models.RESTRequest
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, req.Name, unmarshaled.Name)
	assert.Equal(t, req.Method, unmarshaled.Method)
	assert.Equal(t, req.URL, unmarshaled.URL)
	require.NotNil(t, unmarshaled.Timeout)
	assert.Equal(t, *req.Timeout, *unmarshaled.Timeout)

	reqNilTimeout := models.RESTRequest{
		Name:   "Test Request No Timeout",
		Method: domain.MethodGet,
		URL:    "https://api.example.com/test",
	}

	dataNilTimeout, err := json.Marshal(reqNilTimeout)
	require.NoError(t, err)

	var jsonMapNilTimeout map[string]interface{}
	err = json.Unmarshal(dataNilTimeout, &jsonMapNilTimeout)
	require.NoError(t, err)

	assert.Equal(t, "Test Request No Timeout", jsonMapNilTimeout["name"])
	assert.Equal(t, "GET", jsonMapNilTimeout["method"])
	assert.Equal(t, "https://api.example.com/test", jsonMapNilTimeout["url"])
	_, timeoutExists := jsonMapNilTimeout["timeout"]
	assert.False(t, timeoutExists)

	var unmarshaledNilTimeout models.RESTRequest
	err = json.Unmarshal(dataNilTimeout, &unmarshaledNilTimeout)
	require.NoError(t, err)

	assert.Equal(t, reqNilTimeout.Name, unmarshaledNilTimeout.Name)
	assert.Equal(t, reqNilTimeout.Method, unmarshaledNilTimeout.Method)
	assert.Equal(t, reqNilTimeout.URL, unmarshaledNilTimeout.URL)
	assert.Nil(t, unmarshaledNilTimeout.Timeout)
}
