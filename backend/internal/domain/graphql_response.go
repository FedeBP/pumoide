package domain

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FedeBP/pumoide/backend/pkg/validators"
	"github.com/xeipuuv/gojsonschema"
)

type GraphQLResponse struct {
	StatusCode       int
	Headers          []Header
	Body             interface{}
	ValidationErrors []string
}

func (r *GraphQLResponse) Validate(validation *ResponseValidation) []string {
	var errs []string

	if validation == nil {
		return errs
	}

	if validation.ExpectedStatusCode != 0 && r.StatusCode != validation.ExpectedStatusCode {
		errs = append(errs, fmt.Sprintf("Expected status code %d, but got %d", validation.ExpectedStatusCode, r.StatusCode))
	}

	for key, expectedValue := range validation.ExpectedHeaders {
		actualValue := r.getHeaderValue(key)
		if actualValue != expectedValue {
			errs = append(errs, fmt.Sprintf("Expected header %s to be %s, but got %s", key, expectedValue, actualValue))
		}
	}

	bodyJSON, err := json.Marshal(r.Body)
	if err != nil {
		errs = append(errs, fmt.Sprintf("Failed to marshal GraphQL response body: %v", err))
		return errs
	}

	if validation.JSONSchema != "" {
		schemaLoader := gojsonschema.NewStringLoader(validation.JSONSchema)
		documentLoader := gojsonschema.NewBytesLoader(bodyJSON)

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
		if err := validators.EvaluateAssertion(assertion, string(bodyJSON)); err != nil {
			errs = append(errs, fmt.Sprintf("Assertion failed: %v", err))
		}
	}

	if graphQLErrors, ok := r.Body.(map[string]interface{})["errors"]; ok {
		errs = append(errs, fmt.Sprintf("GraphQL errors: %v", graphQLErrors))
	}

	r.ValidationErrors = errs
	return errs
}

func (r *GraphQLResponse) getHeaderValue(key string) string {
	for _, header := range r.Headers {
		if strings.EqualFold(header.Key, key) {
			return header.Value
		}
	}
	return ""
}

func (r *GraphQLResponse) GetBody() interface{} {
	return r.Body
}

func (r *GraphQLResponse) GetStatusCode() int {
	return r.StatusCode
}

func (r *GraphQLResponse) GetHeaders() []Header {
	return r.Headers
}
