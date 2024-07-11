package services_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/FedeBP/pumoide/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironment(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "env_test")
	require.NoError(t, err)
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			return
		}
	}(tempDir)

	t.Run("SaveAndLoadEnvironment", func(t *testing.T) {
		env := &domain.Environment{
			ID:   uuid.New().String(),
			Name: "Test Environment",
			Variables: map[string]string{
				"API_URL": "https://api.example.com",
				"API_KEY": "test-key",
			},
		}

		err := env.Save(tempDir)
		require.NoError(t, err)

		_, err = os.Stat(filepath.Join(tempDir, env.ID+".json"))
		assert.NoError(t, err)

		loadedEnv, err := domain.LoadEnvironment(tempDir, env.ID)
		require.NoError(t, err)
		assert.Equal(t, env.ID, loadedEnv.ID)
		assert.Equal(t, env.Name, loadedEnv.Name)
		assert.Equal(t, env.Variables, loadedEnv.Variables)
	})

	t.Run("LoadNonExistentEnvironment", func(t *testing.T) {
		_, err := domain.LoadEnvironment(tempDir, "non-existent-id")
		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err), "Expected a 'file not found' error")
	})

	t.Run("SaveEnvironmentWithExistingName", func(t *testing.T) {
		env1 := &domain.Environment{
			ID:   uuid.New().String(),
			Name: "Duplicate Name",
			Variables: map[string]string{
				"VAR1": "value1",
			},
		}

		err := env1.Save(tempDir)
		require.NoError(t, err)

		env2 := &domain.Environment{
			ID:   uuid.New().String(),
			Name: "Duplicate Name",
			Variables: map[string]string{
				"VAR2": "value2",
			},
		}

		err = env2.Save(tempDir)
		assert.NoError(t, err)

		_, err = os.Stat(filepath.Join(tempDir, env1.ID+".json"))
		assert.NoError(t, err)
		_, err = os.Stat(filepath.Join(tempDir, env2.ID+".json"))
		assert.NoError(t, err)
	})
}
