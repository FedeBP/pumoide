package app

import (
	"os"
	"testing"

	"github.com/FedeBP/pumoide/backend/internal/app"
)

func TestLoadConfig(t *testing.T) {
	err := os.Setenv("PORT", ":8080")
	if err != nil {
		t.Errorf("Failed to create env variables")
		return
	}
	err = os.Setenv("LOG_LEVEL", "debug")
	if err != nil {
		t.Errorf("Failed to create env variables")
		return
	}

	config := app.LoadConfig()

	if config.Port != ":8080" {
		t.Errorf("Expected port :8080, got %s", config.Port)
	}

	if config.LogLevel != "debug" {
		t.Errorf("Expected log level debug, got %s", config.LogLevel)
	}

	err = os.Unsetenv("PORT")
	if err != nil {
		t.Errorf("Failed to remove env variables")
		return
	}
	err = os.Unsetenv("LOG_LEVEL")
	if err != nil {
		t.Errorf("Failed to remove env variables")
		return
	}

	config = app.LoadConfig()

	if config.Port != ":0" {
		t.Errorf("Expected default port :0, got %s", config.Port)
	}

	if config.LogLevel != "info" {
		t.Errorf("Expected default log level info, got %s", config.LogLevel)
	}
}
