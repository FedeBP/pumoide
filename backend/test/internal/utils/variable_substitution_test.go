package utils

import (
	"testing"
	"time"

	"github.com/FedeBP/pumoide/backend/internal/models"
	"github.com/FedeBP/pumoide/backend/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestSubstituteVariables(t *testing.T) {
	env := &models.Environment{
		Variables: map[string]string{
			"API_KEY":   "secret_key_123",
			"BASE_URL":  "https://api.example.com",
			"MIN":       "1",
			"MAX":       "100",
			"LENGTH":    "8",
			"CHARSET":   "ABC123",
			"FORMAT":    "2006-01-02",
			"INPUT":     "Hello, World!",
			"TEXT":      "sample text",
			"ENCODE":    "encode",
			"TIMESTAMP": "2006-01-02T15:04:05Z07:00",
		},
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Simple variable", "{{API_KEY}}", "secret_key_123"},
		{"Nested variable", "{{BASE_URL}}/{{API_KEY}}", "https://api.example.com/secret_key_123"},
		{"Random integer", "{{$randomInt($MIN,$MAX)}}", "^[1-9][0-9]?$|^100$"},
		{"Random float", "{{$randomFloat($MIN,$MAX)}}", "^[1-9][0-9]?\\.[0-9]{6}$|^100\\.0{6}$"},
		{"Random string", "{{$randomString($LENGTH,$CHARSET)}}", "^[ABC123]{8}$"},
		{"Timestamp", "{{$timestamp($FORMAT)}}", time.Now().Format("2006-01-02")},
		{"UUID", "{{$uuid}}", "^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$"},
		{"Base64 encode", "{{$base64($INPUT)}}", "SGVsbG8sIFdvcmxkIQ=="},
		{"Base64 decode", "{{$base64(SGVsbG8sIFdvcmxkIQ==,decode)}}", "Hello, World!"},
		{"MD5", "{{$md5($INPUT)}}", "65a8e27d8879283831b664bd8b7f0ad4"},
		{"SHA1", "{{$sha1($INPUT)}}", "0a0a9f2a6772942557ab5355d76af442f8f65e01"},
		{"SHA256", "{{$sha256($INPUT)}}", "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f"},
		{"Lowercase", "{{$lower($TEXT)}}", "sample text"},
		{"Uppercase", "{{$upper($TEXT)}}", "SAMPLE TEXT"},
		{"Capitalize", "{{$capitalize($TEXT)}}", "Sample text"},
		{"Environment variable", "{{$env(API_KEY)}}", "secret_key_123"},
		{"Environment variable with default", "{{$env(NONEXISTENT,default)}}", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.SubstituteVariables(tt.input, env)
			if tt.name == "Random integer" || tt.name == "Random float" || tt.name == "Random string" || tt.name == "Timestamp" || tt.name == "UUID" {
				assert.Regexp(t, tt.expected, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
