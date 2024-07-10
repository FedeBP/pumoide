package domain

import (
	"encoding/json"
	"fmt"

	"github.com/FedeBP/pumoide/backend/pkg/validators"
	"github.com/xeipuuv/gojsonschema"
)

type WebSocketResponse struct {
	Messages         []interface{}
	ValidationErrors []string
}

func (r *WebSocketResponse) Validate(validation *ResponseValidation) []string {
	var errs []string

	if validation == nil {
		return errs
	}

	messageJSON, err := json.Marshal(r.Messages)
	if err != nil {
		errs = append(errs, fmt.Sprintf("Failed to marshal WebSocket message: %v", err))
		return errs
	}

	if validation.JSONSchema != "" {
		schemaLoader := gojsonschema.NewStringLoader(validation.JSONSchema)
		documentLoader := gojsonschema.NewBytesLoader(messageJSON)

		result, err := gojsonschema.Validate(schemaLoader, documentLoader)
		if err != nil {
			errs = append(errs, fmt.Sprintf("JSON Schema validation error: %v", err))
		} else if !result.Valid() {
			for _, desc := range result.Errors() {
				errs = append(errs, fmt.Sprintf("JSON Schema validation failed: %s", desc))
			}
		}
	}

	for _, assertion := range validation.CustomAssertions {
		if err := validators.EvaluateAssertion(assertion, string(messageJSON)); err != nil {
			errs = append(errs, fmt.Sprintf("Assertion failed: %v", err))
		}
	}

	r.ValidationErrors = errs
	return errs
}

func (r *WebSocketResponse) GetBody() interface{} {
	return r.Messages
}

func (r *WebSocketResponse) GetStatusCode() int {
	return 201
}

func (r *WebSocketResponse) GetHeaders() []Header {
	return make([]Header, 0)
}
