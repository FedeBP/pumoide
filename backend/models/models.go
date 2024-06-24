package models

import (
	"encoding/json"
	"github.com/google/uuid"
	"os"
	"path/filepath"
)

type Collection struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Requests []Request `json:"requests"`
}

type Request struct {
	ID          string            `json:"id"`
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	QueryParams map[string]string `json:"queryParams"`
	Headers     map[string]string `json:"headers"`
	Body        string            `json:"body"`
	AuthType    string            `json:"authType"`
	AuthToken   string            `json:"authToken"`
}

func (c *Collection) ensureRequestIDs() {
	for i := range c.Requests {
		if c.Requests[i].ID == "" {
			c.Requests[i].ID = uuid.New().String()
		}
	}
}

func (c *Collection) Save(path string) error {
	c.ensureRequestIDs()
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(path, c.ID+".json"), data, 0644)
}

func LoadCollection(path string, id string) (*Collection, error) {
	data, err := os.ReadFile(filepath.Join(path, id+".json"))
	if err != nil {
		return nil, err
	}
	var collection Collection
	err = json.Unmarshal(data, &collection)
	if err != nil {
		return nil, err
	}
	collection.ensureRequestIDs()
	return &collection, nil
}

func (c *Collection) AddRequest(request Request) {
	if request.ID == "" {
		request.ID = uuid.New().String()
	}
	c.Requests = append(c.Requests, request)
}

func (c *Collection) RemoveRequest(requestID string) bool {
	for i, req := range c.Requests {
		if req.ID == requestID {
			c.Requests = append(c.Requests[:i], c.Requests[i+1:]...)
			return true
		}
	}
	return false
}
