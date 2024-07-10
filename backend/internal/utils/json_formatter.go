package utils

import (
	"encoding/json"
	"net/http"

	"github.com/FedeBP/pumoide/backend/internal/domain"
)

func PrettyJSON(input string) (interface{}, error) {
	var parsed interface{}
	err := json.Unmarshal([]byte(input), &parsed)
	return parsed, err
}

func ConvertHeaders(httpHeaders http.Header) []domain.Header {
	var headers []domain.Header
	for key, values := range httpHeaders {
		for _, value := range values {
			headers = append(headers, domain.Header{Key: key, Value: value})
		}
	}
	return headers
}
