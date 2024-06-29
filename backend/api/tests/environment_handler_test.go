package tests

import (
	"bytes"
	"encoding/json"
	"github.com/FedeBP/pumoide/backend/api"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/FedeBP/pumoide/backend/models"
)

func setupEnvironmentTest(t *testing.T) (string, *api.EnvironmentHandler, func()) {
	tempDir, err := os.MkdirTemp("", "env_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	handler := api.NewEnvironmentHandler(tempDir, logger)

	cleanup := func() {
		err := os.RemoveAll(tempDir)
		if err != nil {
			return
		}
	}

	return tempDir, handler, cleanup
}

func TestEnvironmentHandler(t *testing.T) {
	_, handler, cleanup := setupEnvironmentTest(t)
	defer cleanup()

	var createdEnvID string

	t.Run("CreateEnvironment", func(t *testing.T) {
		env := models.Environment{
			Name:      "Test Env",
			Variables: map[string]string{"KEY": "VALUE"},
		}
		body, _ := json.Marshal(env)
		req, _ := http.NewRequest(http.MethodPost, "/pumoide-api/environments", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusCreated)
		}

		var responseEnv models.Environment
		err := json.Unmarshal(rr.Body.Bytes(), &responseEnv)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if responseEnv.Name != "Test Env" {
			t.Errorf("Handler returned wrong environment name: got %v want %v", responseEnv.Name, "Test Env")
		}
		createdEnvID = responseEnv.ID // Store the created environment ID
	})

	t.Run("GetEnvironments", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/pumoide-api/environments", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var environments []models.Environment
		err := json.Unmarshal(rr.Body.Bytes(), &environments)
		if err != nil {
			t.Fatalf("Failed to unmarshal environment: %v", err)
		}
		if len(environments) != 1 {
			t.Errorf("Handler returned wrong number of environments: got %v want %v", len(environments), 1)
		}
	})

	t.Run("UpdateEnvironment", func(t *testing.T) {
		updatedEnv := models.Environment{
			Name:      "Updated Test Env",
			Variables: map[string]string{"KEY": "NEW_VALUE"},
		}
		body, _ := json.Marshal(updatedEnv)
		req, _ := http.NewRequest(http.MethodPut, "/pumoide-api/environments?id="+createdEnvID, bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var responseEnv models.Environment
		err := json.Unmarshal(rr.Body.Bytes(), &responseEnv)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if responseEnv.Name != "Updated Test Env" {
			t.Errorf("Handler returned wrong environment name: got %v want %v", responseEnv.Name, "Updated Test Env")
		}
	})

	t.Run("DeleteEnvironment", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, "/pumoide-api/environments?id="+createdEnvID, nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})
}
