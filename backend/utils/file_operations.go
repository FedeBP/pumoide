package utils

import (
	"os"
	"path/filepath"
)

var BaseDir string

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic("Unable to determine user home directory")
	}

	BaseDir = filepath.Join(homeDir, ".pumoide")

	err = os.MkdirAll(BaseDir, os.ModePerm)
	if err != nil {
		panic("Unable to create base directory for storage")
	}
}

func GetDefaultCollectionsPath() string {
	return filepath.Join(BaseDir, "collections")
}

func GetDefaultEnvironmentsPath() string {
	return filepath.Join(BaseDir, "environments")
}

func GetDefaultLogsPath() string {
	return filepath.Join(BaseDir, "logs")
}

func EnsureDir(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

func GetCurrentStorageLocation() string {
	return BaseDir
}
