package tests

import (
	"github.com/FedeBP/pumoide/backend/utils"
	"os"
	"path/filepath"
	"testing"
)

func TestGetDefaultCollectionsPath(t *testing.T) {
	path := utils.GetDefaultCollectionsPath()
	if !filepath.IsAbs(path) {
		t.Errorf("GetDefaultCollectionsPath should return an absolute path")
	}
	if filepath.Base(path) != "collections" {
		t.Errorf("GetDefaultCollectionsPath should end with 'collections'")
	}
}

func TestGetDefaultEnvironmentsPath(t *testing.T) {
	path := utils.GetDefaultEnvironmentsPath()
	if !filepath.IsAbs(path) {
		t.Errorf("GetDefaultEnvironmentsPath should return an absolute path")
	}
	if filepath.Base(path) != "environments" {
		t.Errorf("GetDefaultEnvironmentsPath should end with 'environments'")
	}
}

func TestEnsureDir(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "ensure_dir_test")
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Errorf("Failed to clean up temp dir: %v", err)
		}
	}(tempDir)

	testDir := filepath.Join(tempDir, "test_dir")
	err := utils.EnsureDir(testDir)
	if err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Errorf("EnsureDir did not create the directory")
	}
}

func TestGetCurrentStorageLocation(t *testing.T) {
	location := utils.GetCurrentStorageLocation()
	if !filepath.IsAbs(location) {
		t.Errorf("GetCurrentStorageLocation should return an absolute path")
	}
	if filepath.Base(location) != ".pumoide" {
		t.Errorf("GetCurrentStorageLocation should end with 'pumoide'")
	}
}
