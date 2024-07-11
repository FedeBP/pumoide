package logger_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/FedeBP/pumoide/backend/pkg/logger"
	"github.com/sirupsen/logrus"
)

func TestInit(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "logtest")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to delete temp directory: %v", err)
		}
	}(tempDir)

	logFile := filepath.Join(tempDir, "test.log")

	logger.Init(logFile, "debug")

	if logger.Log == nil {
		t.Fatal("Logger is nil after initialization")
	}

	logger.Log.Info("Test log entry")

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Logger did not write to the file")
	}

	if logger.Log.Level != logrus.DebugLevel {
		t.Errorf("Expected debug level, got %v", logger.Log.Level)
	}

	logger.Close()
}

func TestGetLogger(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "logtest")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to delete temp directory: %v", err)
		}
	}(tempDir)

	logFile := filepath.Join(tempDir, "test.log")

	log1 := logger.GetLogger(logFile, "info")
	if log1 == nil {
		t.Fatal("GetLogger returned nil")
	}

	if log1.Level != logrus.InfoLevel {
		t.Errorf("Expected info level, got %v", log1.Level)
	}

	logger.Close()

	log2 := logger.GetLogger(logFile, "debug")
	if log2 == nil {
		t.Fatal("GetLogger returned nil after Close")
	}

	if log2.Level != logrus.DebugLevel {
		t.Errorf("Expected debug level after reinitialization, got %v", log2.Level)
	}

	logger.Close()
}
