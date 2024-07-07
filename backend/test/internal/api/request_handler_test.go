package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/FedeBP/pumoide/backend/internal/api"
	"github.com/FedeBP/pumoide/backend/internal/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRequestHandler(client *http.Client, env string, testLogger *logrus.Logger, workers int) *api.RequestHandler {
	return &api.RequestHandler{
		Client:          client,
		EnvironmentPath: env,
		Logger:          testLogger,
		WorkerCount:     workers,
	}
}

func TestRequestHandler_ServeHTTP(t *testing.T) {
	mockClient := &http.Client{
		Transport: &mockTransport{},
	}

	testLogger := logrus.New()
	testLogger.SetOutput(&bytes.Buffer{})

	handler := newRequestHandler(mockClient, "test_env_path", testLogger, 5)

	tests := []struct {
		name           string
		requests       []models.Request
		expectedStatus int
	}{
		{
			name: "Single request",
			requests: []models.Request{
				{
					Method: "GET",
					URL:    "http://example.com",
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Multiple requests",
			requests: []models.Request{
				{
					Method: "GET",
					URL:    "http://example.com",
				},
				{
					Method: "POST",
					URL:    "http://example.com/post",
					Body:   "test body",
				},
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.requests)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/pumoide-api/execute", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			var results []map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &results)
			require.NoError(t, err)

			assert.Equal(t, len(tt.requests), len(results))

			for i, result := range results {
				assert.Contains(t, result, "request")
				assert.Contains(t, result, "response")
				assert.Contains(t, result, "performance_metrics")

				request := result["request"].(map[string]interface{})
				assert.Equal(t, string(tt.requests[i].Method), request["method"])
				assert.Equal(t, tt.requests[i].URL, request["url"])

				response := result["response"].(map[string]interface{})
				assert.Contains(t, response, "statusCode")
				assert.Contains(t, response, "headers")
				assert.Contains(t, response, "body")

				metrics := result["performance_metrics"].(map[string]interface{})
				assert.Contains(t, metrics, "total")
				assert.Contains(t, metrics, "dns_lookup")
				assert.Contains(t, metrics, "tcp_connection")
				assert.Contains(t, metrics, "tls_handshake")
				assert.Contains(t, metrics, "server_processing")
				assert.Contains(t, metrics, "content_transfer")
			}
		})
	}
}

func TestRequestHandler_ExecuteRequest_Timeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := &http.Client{
		Timeout: 1 * time.Second,
	}

	testLogger := logrus.New()
	testLogger.SetOutput(&bytes.Buffer{})

	handler := newRequestHandler(client, "test_env_path", testLogger, 5)

	req := models.Request{
		Method: "GET",
		URL:    ts.URL,
	}

	result := handler.ExecuteRequest(req, nil, nil)

	assert.Contains(t, result.Error, "context deadline exceeded")
}

type mockTransport struct{}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
		Header:     make(http.Header),
	}

	switch req.Method {
	case "GET":
		resp.StatusCode = http.StatusOK
	case "POST":
		resp.StatusCode = http.StatusCreated
	}

	return resp, nil
}
