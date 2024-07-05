package validators

import (
	"encoding/json"
	"fmt"

	"github.com/FedeBP/pumoide/backend/internal/models"
	"github.com/xeipuuv/gojsonschema"
)

func ValidateResponse(response models.Response, validation *models.ResponseValidation) []string {
	var errs []string

	if validation == nil {
		return errs
	}

	if validation.ExpectedStatusCode != 0 && response.StatusCode != validation.ExpectedStatusCode {
		errs = append(errs, fmt.Sprintf("Expected status code %d, but got %d", validation.ExpectedStatusCode, response.StatusCode))
	}

	for key, expectedValue := range validation.ExpectedHeaders {
		if actualValue, exists := response.Headers[key]; !exists || actualValue != expectedValue {
			errs = append(errs, fmt.Sprintf("Expected header %s to be %s, but got %s", key, expectedValue, actualValue))
		}
	}

	if validation.JSONSchema != "" {
		schemaLoader := gojsonschema.NewStringLoader(validation.JSONSchema)
		documentLoader := gojsonschema.NewStringLoader(response.Body)

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
	err := json.Unmarshal([]byte(response.Body), &responseBody)
	if err != nil {
		return nil
	}

	for _, assertion := range validation.CustomAssertions {
		if err := EvaluateAssertion(assertion, response.Body); err != nil {
			errs = append(errs, fmt.Sprintf("Assertion failed: %v", err))
		}
	}

	return errs
}
