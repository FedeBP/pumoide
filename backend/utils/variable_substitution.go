package utils

import (
	"github.com/FedeBP/pumoide/backend/models"
	"strings"
)

func SubstituteVariables(input string, env *models.Environment) string {
	if env == nil {
		return input
	}
	for key, value := range env.Variables {
		input = strings.ReplaceAll(input, "{{"+key+"}}", value)
	}
	return input
}
