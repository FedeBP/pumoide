package models

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/FedeBP/pumoide/backend/internal/domain"
	"github.com/FedeBP/pumoide/backend/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestRESTRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request models.RESTRequest
		wantErr bool
	}{
		{
			name: "Valid REST request",
			request: models.RESTRequest{
				Name:    "Test Request",
				Method:  domain.MethodGet,
				URL:     "https://api.example.com/test",
				Timeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "Invalid method",
			request: models.RESTRequest{
				Name:    "Test Request",
				Method:  "INVALID",
				URL:     "https://api.example.com/test",
				Timeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "Empty name",
			request: models.RESTRequest{
				Method:  domain.MethodGet,
				URL:     "https://api.example.com/test",
				Timeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "Invalid URL",
			request: models.RESTRequest{
				Name:    "Test Request",
				Method:  domain.MethodGet,
				URL:     "not-a-valid-url",
				Timeout: 30 * time.Second,
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
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"message": "Hello, World!"}`))
		if err != nil {
			t.Errorf("Error writing message: %v", err)
			return
		}
	}))
	defer ts.Close()

	req := models.RESTRequest{
		Name:    "Test Request",
		Method:  domain.MethodGet,
		URL:     ts.URL,
		Timeout: 30 * time.Second,
	}

	env := &domain.Environment{
		Variables: make(map[string]string),
	}

	response, err := req.Execute(env)
	assert.NoError(t, err)
	assert.NotNil(t, response)

	restResponse, ok := response.(*domain.RESTResponse)
	assert.True(t, ok)
	assert.Equal(t, http.StatusOK, restResponse.StatusCode)
	assert.Contains(t, restResponse.Body, "Hello, World!")
}
