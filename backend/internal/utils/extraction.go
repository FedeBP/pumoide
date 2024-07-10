package utils

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/FedeBP/pumoide/backend/internal/domain"
	"github.com/oliveagle/jsonpath"
)

func ExtractValueFromResponse(response domain.Response, extractPath string) (string, error) {
	body := response.GetBody()

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response body: %v", err)
	}

	if strings.HasPrefix(extractPath, "$") {
		return extractJSONValue(string(bodyJSON), extractPath)
	} else {
		return extractRegexValue(string(bodyJSON), extractPath)
	}
}

func extractJSONValue(jsonString string, jsonPath string) (string, error) {
	var data interface{}
	err := json.Unmarshal([]byte(jsonString), &data)
	if err != nil {
		return "", fmt.Errorf("failed to parse JSON: %v", err)
	}

	result, err := jsonpath.JsonPathLookup(data, jsonPath)
	if err != nil {
		return "", fmt.Errorf("failed to extract value using JSON path: %v", err)
	}

	switch v := result.(type) {
	case string:
		return v, nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(v), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

func extractRegexValue(responseBody string, regexPattern string) (string, error) {
	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %v", err)
	}

	matches := re.FindStringSubmatch(responseBody)
	if len(matches) > 1 {
		return matches[1], nil
	} else if len(matches) == 1 {
		return matches[0], nil
	}

	return "", fmt.Errorf("no match found for regex pattern")
}
