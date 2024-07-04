package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FedeBP/pumoide/backend/api"
	"github.com/FedeBP/pumoide/backend/models"
)

func TestRequestValidation(t *testing.T) {
	handler := &api.RequestHandler{
		Client: &http.Client{},
		Logger: logger,
	}

	tests := []struct {
		name            string
		request         models.Request
		env             *models.Environment
		previousResults []models.RequestResult
		mockResponse    string
		mockStatusCode  int
		expectError     bool
		errorMessage    string
	}{
		{
			name: "Valid status code with environment variable",
			request: models.Request{
				Method: "GET",
				URL:    "http://example.com/api/{{apiVersion}}",
				ResponseValidation: &models.ResponseValidation{
					ExpectedStatusCode: 200,
				},
			},
			env: &models.Environment{
				Variables: map[string]string{"apiVersion": "v1"},
			},
			mockResponse:   `{"status": "ok"}`,
			mockStatusCode: 200,
			expectError:    false,
		},
		{
			name: "Valid status code with previous result variable",
			request: models.Request{
				Method: "GET",
				URL:    "http://example.com/users/{{userId}}",
				ResponseValidation: &models.ResponseValidation{
					ExpectedStatusCode: 200,
				},
			},
			previousResults: []models.RequestResult{
				{
					Request: models.Request{
						ExtractVariables: map[string]string{"userId": "$.id"},
					},
					Response: models.Response{
						Body: `{"id": "123"}`,
					},
				},
			},
			mockResponse:   `{"status": "ok"}`,
			mockStatusCode: 200,
			expectError:    false,
		},
		{
			name: "Valid status code",
			request: models.Request{
				Method: "GET",
				URL:    "http://example.com",
				ResponseValidation: &models.ResponseValidation{
					ExpectedStatusCode: 200,
				},
			},
			mockResponse:   `{"status": "ok"}`,
			mockStatusCode: 200,
			expectError:    false,
		},
		{
			name: "Invalid status code",
			request: models.Request{
				Method: "GET",
				URL:    "http://example.com",
				ResponseValidation: &models.ResponseValidation{
					ExpectedStatusCode: 200,
				},
			},
			mockResponse:   `{"status": "error"}`,
			mockStatusCode: 400,
			expectError:    true,
			errorMessage:   "Expected status code 200, but got 400",
		},
		{
			name: "Valid JSON schema",
			request: models.Request{
				Method: "GET",
				URL:    "http://example.com",
				ResponseValidation: &models.ResponseValidation{
					JSONSchema: `{"type": "object", "properties": {"status": {"type": "string"}}, "required": ["status"]}`,
				},
			},
			mockResponse:   `{"status": "ok"}`,
			mockStatusCode: 200,
			expectError:    false,
		},
		{
			name: "Invalid JSON schema",
			request: models.Request{
				Method: "GET",
				URL:    "http://example.com",
				ResponseValidation: &models.ResponseValidation{
					JSONSchema: `{"type": "object", "properties": {"status": {"type": "number"}}, "required": ["status"]}`,
				},
			},
			mockResponse:   `{"status": "ok"}`,
			mockStatusCode: 200,
			expectError:    true,
			errorMessage:   "JSON Schema validation failed",
		},
		{
			name: "Valid custom assertion",
			request: models.Request{
				Method: "GET",
				URL:    "http://example.com",
				ResponseValidation: &models.ResponseValidation{
					CustomAssertions: []string{"status == ok"},
				},
			},
			mockResponse:   `{"status": "ok"}`,
			mockStatusCode: 200,
			expectError:    false,
		},
		{
			name: "Invalid custom assertion",
			request: models.Request{
				Method: "GET",
				URL:    "http://example.com",
				ResponseValidation: &models.ResponseValidation{
					CustomAssertions: []string{"status == error"},
				},
			},
			mockResponse:   `{"status": "ok"}`,
			mockStatusCode: 200,
			expectError:    true,
			errorMessage:   "Assertion failed: Expected 'error', but got 'ok'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.env != nil && tt.env.Variables["apiVersion"] != "" {
					if !strings.Contains(r.URL.Path, tt.env.Variables["apiVersion"]) {
						t.Errorf("Expected URL to contain %s, but got %s", tt.env.Variables["apiVersion"], r.URL.Path)
					}
				}
				if len(tt.previousResults) > 0 {
					userId := "123"
					if !strings.Contains(r.URL.Path, userId) {
						t.Errorf("Expected URL to contain %s, but got %s", userId, r.URL.Path)
					}
				}

				w.WriteHeader(tt.mockStatusCode)
				_, err := w.Write([]byte(tt.mockResponse))
				if err != nil {
					t.Errorf("Error writing header")
					return
				}
			}))
			defer server.Close()

			tt.request.URL = server.URL + strings.TrimPrefix(tt.request.URL, "http://example.com")

			result := handler.ExecuteRequest(tt.request, tt.env, tt.previousResults)

			if tt.expectError {
				if len(result.ValidationErrors) == 0 {
					t.Errorf("Expected validation error, but got none")
				} else {
					errorFound := false
					for _, err := range result.ValidationErrors {
						if strings.Contains(err, tt.errorMessage) {
							errorFound = true
							break
						}
					}
					if !errorFound {
						t.Errorf("Expected error message '%s', but it was not found in the validation errors", tt.errorMessage)
					}
				}
			} else {
				if len(result.ValidationErrors) > 0 {
					t.Errorf("Unexpected validation errors: %v", result.ValidationErrors)
				}
			}
		})
	}
}
