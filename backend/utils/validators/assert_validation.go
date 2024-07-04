package validators

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/FedeBP/pumoide/backend/constants"
	"github.com/FedeBP/pumoide/backend/errors"
	"github.com/tidwall/gjson"
)

type AssertionError struct {
	Message string
}

func (e AssertionError) Error() string {
	return e.Message
}

func EvaluateAssertion(assertion string, responseBody string) error {
	parts := strings.SplitN(assertion, " ", 3)
	if len(parts) < 2 {
		return errors.NewAppError(http.StatusBadRequest, constants.ErrInvalidAssertionFormat, nil)
	}

	path, operator := parts[0], parts[1]
	expectedValue := ""
	if len(parts) == 3 {
		expectedValue = parts[2]
	}

	result := gjson.Get(responseBody, path)

	switch operator {
	case constants.OpExists:
		if !result.Exists() {
			return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf(constants.ErrPathNotFound, path), nil)
		}
	case constants.OpNotExists:
		if result.Exists() {
			return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf(constants.ErrPathExists, path), nil)
		}
	case constants.OpEquals:
		if result.String() != expectedValue {
			return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf("Expected '%s', but got '%s'", expectedValue, result.String()), nil)
		}
	case constants.OpNotEquals:
		if result.String() == expectedValue {
			return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf(constants.ErrExpectedNotEquals, expectedValue), nil)
		}
	case constants.OpContains:
		if !strings.Contains(result.String(), expectedValue) {
			return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf(constants.ErrExpectedContains, result.String(), expectedValue), nil)
		}
	case constants.OpNotContains:
		if strings.Contains(result.String(), expectedValue) {
			return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf(constants.ErrExpectedNotContains, result.String(), expectedValue), nil)
		}
	case constants.OpGreaterThan, constants.OpGreaterThanOrEquals, constants.OpLessThan, constants.OpLessThanOrEquals:
		return compareNumbers(result.String(), expectedValue, operator)
	case constants.OpMatches:
		matched, err := regexp.MatchString(expectedValue, result.String())
		if err != nil {
			return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf(constants.ErrInvalidRegexPattern, err), err)
		}
		if !matched {
			return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf(constants.ErrExpectedMatch, result.String(), expectedValue), nil)
		}
	case constants.OpType:
		if !checkType(result, expectedValue) {
			return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf(constants.ErrExpectedType, expectedValue, result.Type), nil)
		}
	default:
		return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf(constants.ErrUnknownOperator, operator), nil)
	}

	return nil
}

func compareNumbers(actualValue, expectedValue, operator string) error {
	actual, err := strconv.ParseFloat(actualValue, 64)
	if err != nil {
		return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf(constants.ErrTypeConversion, actualValue), err)
	}

	expected, err := strconv.ParseFloat(expectedValue, 64)
	if err != nil {
		return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf(constants.ErrTypeConversion, expectedValue), err)
	}

	var result bool
	switch operator {
	case constants.OpGreaterThan:
		result = actual > expected
	case constants.OpGreaterThanOrEquals:
		result = actual >= expected
	case constants.OpLessThan:
		result = actual < expected
	case constants.OpLessThanOrEquals:
		result = actual <= expected
	}

	if !result {
		return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf(constants.ErrComparisonFailed, actualValue, operator, expectedValue), nil)
	}

	return nil
}

func checkType(result gjson.Result, expectedType string) bool {
	switch expectedType {
	case "string":
		return result.Type == gjson.String
	case "number":
		return result.Type == gjson.Number
	case "boolean":
		return result.Type == gjson.True || result.Type == gjson.False
	case "object":
		return result.Type == gjson.JSON && result.IsObject()
	case "array":
		return result.Type == gjson.JSON && result.IsArray()
	case "null":
		return result.Type == gjson.Null
	default:
		return false
	}
}
