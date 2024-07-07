package errors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FedeBP/pumoide/backend/pkg/errors"
	"github.com/sirupsen/logrus"
)

type discardWriter struct{}

func (dw discardWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func TestRespondWithError(t *testing.T) {
	w := httptest.NewRecorder()
	logger := logrus.New()
	logger.SetOutput(discardWriter{})
	logger.SetLevel(logrus.ErrorLevel)

	errors.RespondWithError(w, http.StatusBadRequest, "Test error", nil, logger)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if msg, ok := response["message"].(string); !ok || msg != "Test error" {
		t.Errorf("Expected message 'Test error', got '%v'", response["message"])
	}

	w = httptest.NewRecorder()
	errors.RespondWithError(w, http.StatusInternalServerError, "Another test error", nil, nil)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, w.Code)
	}

	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if msg, ok := response["message"].(string); !ok || msg != "Another test error" {
		t.Errorf("Expected message 'Another test error', got '%v'", response["message"])
	}
}
