package tests

import (
	"bytes"
	"encoding/json"
	"github.com/FedeBP/pumoide/backend/api"
	"github.com/FedeBP/pumoide/backend/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
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
		expectedBody   string
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
			expectedBody:   `[{"request":{"id":"","name":"","method":"GET","url":"http://example.com","headers":null,"queryParams":null,"body":""},"response":{"statusCode":200,"headers":{},"body":""}}]`,
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
			expectedBody:   `[{"request":{"id":"","name":"","method":"GET","url":"http://example.com","headers":null,"queryParams":null,"body":""},"response":{"statusCode":200,"headers":{},"body":""}},{"request":{"id":"","name":"","method":"POST","url":"http://example.com/post","headers":null,"queryParams":null,"body":"test body"},"response":{"statusCode":201,"headers":{},"body":""}}]`,
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

			assert.JSONEq(t, tt.expectedBody, rr.Body.String())
		})
	}
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
