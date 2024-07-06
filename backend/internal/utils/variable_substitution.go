package utils

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/FedeBP/pumoide/backend/internal/models"
	"github.com/google/uuid"
)

var (
	variableRegex = regexp.MustCompile(`{{(.*?)}}`)
	functionRegex = regexp.MustCompile(`\$([a-zA-Z0-9]+)(?:\((.*?)\))?`)
)

func SubstituteVariables(input string, env *models.Environment) string {
	if env == nil {
		env = &models.Environment{Variables: make(map[string]string)}
	}

	result := variableRegex.ReplaceAllStringFunc(input, func(match string) string {
		key := strings.Trim(match, "{}")

		if strings.HasPrefix(key, "$") {
			expanded := handleFunctionExpansion(key, env)
			return expanded
		}

		if value, exists := env.Variables[key]; exists {
			return value
		}

		return match
	})

	return result
}

func handleFunctionExpansion(function string, env *models.Environment) string {

	parts := functionRegex.FindStringSubmatch(function)
	if len(parts) < 2 {
		return function
	}

	functionName := parts[1]
	var args []string
	if len(parts) > 2 && parts[2] != "" {
		rawArgs := strings.Split(parts[2], ",")
		for _, arg := range rawArgs {
			processedArg := getEnvOrArg(arg, env)
			args = append(args, processedArg)
		}
	}

	switch functionName {
	case "randomInt":
		return generateRandomInt(args, env)
	case "randomFloat":
		return generateRandomFloat(args, env)
	case "randomString":
		return generateRandomString(args, env)
	case "timestamp":
		return generateTimestamp(args, env)
	case "uuid":
		return generateUUID()
	case "base64":
		return handleBase64(args, env)
	case "md5":
		return generateMD5(args, env)
	case "sha1":
		return generateSHA1(args, env)
	case "sha256":
		return generateSHA256(args, env)
	case "lower":
		return strings.ToLower(getEnvOrArg(args[0], env))
	case "upper":
		return strings.ToUpper(getEnvOrArg(args[0], env))
	case "capitalize":
		return capitalize(getEnvOrArg(args[0], env))
	case "env":
		return getEnvVariable(args, env)
	default:
		return function
	}
}

func generateRandomInt(args []string, env *models.Environment) string {
	minimum, maximum := 0, 100
	var err error
	if len(args) >= 2 {
		minimumStr := getEnvOrArg(args[0], env)
		maxStr := getEnvOrArg(args[1], env)
		if _, err = fmt.Sscanf(minimumStr, "%d", &minimum); err != nil {
			return fmt.Sprintf("Error: Invalid minimum value - %v", err)
		}
		if _, err = fmt.Sscanf(maxStr, "%d", &maximum); err != nil {
			return fmt.Sprintf("Error: Invalid maximum value - %v", err)
		}
	}
	if minimum > maximum {
		minimum, maximum = maximum, minimum
	}
	return fmt.Sprintf("%d", rand.Intn(maximum-minimum+1)+minimum)
}

func generateRandomFloat(args []string, env *models.Environment) string {
	minimum, maximum := 0.0, 1.0
	var err error
	if len(args) >= 2 {
		minimumStr := getEnvOrArg(args[0], env)
		maxStr := getEnvOrArg(args[1], env)
		if _, err = fmt.Sscanf(minimumStr, "%f", &minimum); err != nil {
			return fmt.Sprintf("Error: Invalid minimum value - %v", err)
		}
		if _, err = fmt.Sscanf(maxStr, "%f", &maximum); err != nil {
			return fmt.Sprintf("Error: Invalid maximum value - %v", err)
		}
	}
	if minimum > maximum {
		minimum, maximum = maximum, minimum
	}
	return fmt.Sprintf("%.6f", minimum+rand.Float64()*(maximum-minimum))
}

func generateRandomString(args []string, env *models.Environment) string {
	length := 10
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var err error
	if len(args) >= 1 {
		lengthStr := getEnvOrArg(args[0], env)
		if _, err = fmt.Sscanf(lengthStr, "%d", &length); err != nil {
			return fmt.Sprintf("Error: Invalid length - %v", err)
		}
	}
	if len(args) >= 2 {
		charset = getEnvOrArg(args[1], env)
	}
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func generateTimestamp(args []string, env *models.Environment) string {
	format := "2006-01-02T15:04:05Z07:00"
	if len(args) > 0 {
		format = getEnvOrArg(args[0], env)
	}
	return time.Now().Format(format)
}

func generateUUID() string {
	return uuid.New().String()
}

func handleBase64(args []string, env *models.Environment) string {
	if len(args) == 0 {
		return "Error: No input provided for base64"
	}
	input := getEnvOrArg(args[0], env)
	if len(args) > 1 && args[1] == "decode" {
		decoded, err := base64.StdEncoding.DecodeString(input)
		if err != nil {
			return fmt.Sprintf("Error: Invalid base64 string - %v", err)
		}
		return string(decoded)
	}
	return base64.StdEncoding.EncodeToString([]byte(input))
}

func generateMD5(args []string, env *models.Environment) string {
	if len(args) == 0 {
		return "Error: No input provided for MD5"
	}
	input := getEnvOrArg(args[0], env)
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}

func generateSHA1(args []string, env *models.Environment) string {
	if len(args) == 0 {
		return "Error: No input provided for SHA1"
	}
	input := getEnvOrArg(args[0], env)
	hash := sha1.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}

func generateSHA256(args []string, env *models.Environment) string {
	if len(args) == 0 {
		return "Error: No input provided for SHA256"
	}
	input := getEnvOrArg(args[0], env)
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	return string(append([]rune{unicode.ToUpper(r[0])}, r[1:]...))
}

func getEnvOrArg(arg string, env *models.Environment) string {
	if strings.HasPrefix(arg, "$") {
		varName := strings.TrimPrefix(arg, "$")
		if value, exists := env.Variables[varName]; exists {
			return value
		}
	}
	return arg
}

func getEnvVariable(args []string, env *models.Environment) string {
	if len(args) == 0 {
		return "Error: No environment variable name provided"
	}
	varName := args[0]
	if value, exists := env.Variables[varName]; exists {
		return value
	}
	if len(args) > 1 {
		return args[1]
	}
	return fmt.Sprintf("Error: Environment variable %s not found", varName)
}
