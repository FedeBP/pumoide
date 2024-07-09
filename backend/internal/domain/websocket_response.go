package domain

import (
	"encoding/json"
	"fmt"

	"github.com/FedeBP/pumoide/backend/internal/validators"
	"github.com/xeipuuv/gojsonschema"
)

type WebSocketResponse struct {
	Message          string
	ValidationErrors []string
}

func (r *WebSocketResponse) Validate(validation *ResponseValidation) []string {
	var errs []string

	if validation == nil {
		return errs
	}

	if validation.JSONSchema != "" {
		schemaLoader := gojsonschema.NewStringLoader(validation.JSONSchema)
		documentLoader := gojsonschema.NewStringLoader(r.Message)

		result, err := gojsonschema.Validate(schemaLoader, documentLoader)
		if err != nil {
			errs = append(errs, fmt.Sprintf("JSON Schema validation error: %v", err))
		} else if !result.Valid() {
			for _, desc := range result.Errors() {
				errs = append(errs, fmt.Sprintf("JSON Schema validation failed: %s", desc))
			}
		}
	}

	var responseBody map[string]interface{}
	err := json.Unmarshal([]byte(r.Message), &responseBody)
	if err != nil {
		errs = append(errs, fmt.Sprintf("Failed to parse response body as JSON: %v", err))
	}

	for _, assertion := range validation.CustomAssertions {
		if err := validators.EvaluateAssertion(assertion, r.Message); err != nil {
			errs = append(errs, fmt.Sprintf("Assertion failed: %v", err))
		}
	}

	r.ValidationErrors = errs
	return errs
}

func (r *WebSocketResponse) GetBody() string {
	return r.Message
}

func (r *WebSocketResponse) GetStatusCode() int {
	return 201
}

func (r *WebSocketResponse) GetHeaders() []Header {
	return make([]Header, 0)
}
