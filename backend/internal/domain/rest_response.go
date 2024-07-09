package domain

import (
	"encoding/json"
	"fmt"

	"github.com/FedeBP/pumoide/backend/pkg/validators"
	"github.com/xeipuuv/gojsonschema"
)

type RESTResponse struct {
	StatusCode       int
	Headers          []Header
	Body             string
	ValidationErrors []string
}

func (r *RESTResponse) Validate(validation *ResponseValidation) []string {
	var errs []string

	if validation == nil {
		return errs
	}

	if validation.ExpectedStatusCode != 0 && r.StatusCode != validation.ExpectedStatusCode {
		errs = append(errs, fmt.Sprintf("Expected status code %d, but got %d", validation.ExpectedStatusCode, r.StatusCode))
	}

	headers := make(map[string]string)
	for _, h := range r.Headers {
		headers[h.Key] = h.Value
	}

	for key, expectedValue := range validation.ExpectedHeaders {
		if actualValue, exists := headers[key]; !exists || actualValue != expectedValue {
			errs = append(errs, fmt.Sprintf("Expected header %s to be %s, but got %s", key, expectedValue, actualValue))
		}
	}

	if validation.JSONSchema != "" {
		schemaLoader := gojsonschema.NewStringLoader(validation.JSONSchema)
		documentLoader := gojsonschema.NewStringLoader(r.Body)

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
	err := json.Unmarshal([]byte(r.Body), &responseBody)
	if err != nil {
		errs = append(errs, fmt.Sprintf("Failed to parse response body as JSON: %v", err))
	}

	for _, assertion := range validation.CustomAssertions {
		if err := validators.EvaluateAssertion(assertion, r.Body); err != nil {
			errs = append(errs, fmt.Sprintf("Assertion failed: %v", err))
		}
	}

	r.ValidationErrors = errs
	return errs
}

func (r *RESTResponse) GetBody() string {
	return r.Body
}

func (r *RESTResponse) GetStatusCode() int {
	return r.StatusCode
}

func (r *RESTResponse) GetHeaders() []Header {
	return r.Headers
}
