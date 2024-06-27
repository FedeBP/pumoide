package models

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

type Environment struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Variables map[string]string `json:"variables"`
}

func (e *Environment) Save(path string) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(path, e.ID+".json"), data, 0644)
}

func LoadEnvironment(path string, id string) (*Environment, error) {
	data, err := os.ReadFile(filepath.Join(path, id+".json"))
	if err != nil {
		return nil, err
	}
	var environment Environment
	err = json.Unmarshal(data, &environment)
	return &environment, err
}
