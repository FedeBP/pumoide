package tests

import (
	"os"
	"testing"

	"github.com/FedeBP/pumoide/backend/models"
)

func TestEnvironmentSaveAndLoad(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "env_test")
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Errorf("Failed to clean up temp dir: %v", err)
		}
	}(tempDir)

	env := &models.Environment{
		ID:        "test-env",
		Name:      "Test Environment",
		Variables: map[string]string{"KEY": "VALUE"},
	}

	err := env.Save(tempDir)
	if err != nil {
		t.Fatalf("Failed to save environment: %v", err)
	}

	loadedEnv, err := models.LoadEnvironment(tempDir, "test-env")
	if err != nil {
		t.Fatalf("Failed to load environment: %v", err)
	}

	if loadedEnv.ID != env.ID || loadedEnv.Name != env.Name {
		t.Errorf("Loaded environment does not match saved environment")
	}

	if loadedEnv.Variables["KEY"] != "VALUE" {
		t.Errorf("Loaded environment variables do not match")
	}
}
