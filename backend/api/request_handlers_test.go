package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FedeBP/pumoide/backend/models"
)

func TestRequestHandler_Handle(t *testing.T) {
	// Create a test server to mock external API
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the request method
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		// Check the headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}
		// Check query parameters
		if r.URL.Query().Get("param1") != "value1" {
			t.Errorf("Expected query param 'param1=value1', got '%s'", r.URL.Query().Get("param1"))
		}
		// Send a response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"message": "Test response"}`))
		if err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
		}
	}))
	defer testServer.Close()

	// Create a request to test
	testRequest := models.Request{
		Method: http.MethodGet,
		URL:    testServer.URL,
		Headers: []models.Header{
			{Key: "Content-Type", Value: "application/json"},
		},
		QueryParams: map[string]string{"param1": "value1"},
	}

	// Convert the request to JSON
	requestBody, err := json.Marshal(testRequest)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Create a request to our handler
	req, err := http.NewRequest(http.MethodPost, "/pumoide-api/execute", bytes.NewBuffer(requestBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Create our handler and serve the request
	handler := NewRequestHandler()
	handler.Handle(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body
	var response struct {
		StatusCode int               `json:"statusCode"`
		Headers    map[string]string `json:"headers"`
		Body       string            `json:"body"`
	}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, response.StatusCode)
	}

	expectedBody := `{"message": "Test response"}`
	if response.Body != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, response.Body)
	}
}
