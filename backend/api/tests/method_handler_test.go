package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FedeBP/pumoide/backend/api"
	"github.com/FedeBP/pumoide/backend/models"
)

func TestMethodHandler(t *testing.T) {
	handler := api.NewMethodHandler(logger)

	req, _ := http.NewRequest(http.MethodGet, "/pumoide-api/methods", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var methods []models.Method
	if err := json.Unmarshal(rr.Body.Bytes(), &methods); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	expectedMethods := models.GetValidMethods()
	if len(methods) != len(expectedMethods) {
		t.Errorf("Handler returned wrong number of methods: got %v want %v", len(methods), len(expectedMethods))
	}

	for i, method := range methods {
		if method != expectedMethods[i] {
			t.Errorf("Handler returned wrong method at index %d: got %v want %v", i, method, expectedMethods[i])
		}
	}
}
