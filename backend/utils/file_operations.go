package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var BaseDir string

func init() {
	var err error
	BaseDir, err = getDocumentsFolder()
	if err != nil {
		// If we can't get the Documents folder, fall back to the current working directory
		BaseDir, err = os.Getwd()
		if err != nil {
			panic("Unable to determine base directory for storage")
		}
	}

	// Set the base directory for our application
	BaseDir = filepath.Join(BaseDir, "pumoide")

	// Ensure the base directory exists
	err = os.MkdirAll(BaseDir, os.ModePerm)
	if err != nil {
		panic("Unable to create base directory for storage")
	}
}

func getDocumentsFolder() (string, error) {
	var docDir string
	var _ error

	switch runtime.GOOS {
	case "windows":
		// On Windows, we need to navigate from the AppData folder to Documents
		configDir, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		// AppData is typically two levels up from Documents
		docDir = filepath.Join(filepath.Dir(filepath.Dir(configDir)), "Documents")
	case "darwin":
		// On macOS, the Documents folder is directly under the home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		docDir = filepath.Join(homeDir, "Documents")
	default:
		// On Linux and other systems, we'll use the XDG user directories specification
		out, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".config", "user-dirs.dirs"))
		if err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				if strings.HasPrefix(line, "XDG_DOCUMENTS_DIR") {
					docDir = strings.Trim(strings.SplitN(line, "=", 2)[1], "\"")
					docDir = os.ExpandEnv(docDir)
					break
				}
			}
		}
		// If we couldn't find it, fall back to a reasonable default
		if docDir == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			docDir = filepath.Join(homeDir, "Documents")
		}
	}

	return docDir, nil
}

func GetDefaultCollectionsPath() string {
	return filepath.Join(BaseDir, "collections")
}

func GetDefaultEnvironmentsPath() string {
	return filepath.Join(BaseDir, "environments")
}

func EnsureDir(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

func GetCurrentStorageLocation() string {
	return BaseDir
}
