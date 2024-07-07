package models

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/FedeBP/pumoide/backend/pkg/constants"
	"github.com/FedeBP/pumoide/backend/pkg/errors"
	"github.com/google/uuid"
)

type History struct {
	basePath   string
	maxAge     time.Duration
	maxEntries int
	mutex      sync.RWMutex
}

type HistoryEntry struct {
	ID                 string             `json:"id"`
	Timestamp          time.Time          `json:"timestamp"`
	Request            Request            `json:"request"`
	Response           Response           `json:"response"`
	ExecutionTime      time.Duration      `json:"executionTime"`
	PerformanceMetrics PerformanceMetrics `json:"performance_metrics"`
}

func NewHistoryManager(basePath string, maxAge time.Duration, maxEntries int) *History {
	return &History{
		basePath:   filepath.Join(basePath, "history"),
		maxAge:     maxAge,
		maxEntries: maxEntries,
	}
}

func (hm *History) AddEntry(entry HistoryEntry) error {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	entry.ID = uuid.New().String()

	err := os.MkdirAll(hm.basePath, 0755)
	if err != nil {
		return errors.NewAppError(http.StatusInternalServerError, "Failed to create history directory", err)
	}

	filename := filepath.Join(hm.basePath, fmt.Sprintf("%s.json", entry.ID))

	data, err := json.Marshal(entry)
	if err != nil {
		return errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToEncodeRequest, err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err)
	}

	go hm.cleanup()

	return nil
}

func (hm *History) GetEntry(id string) (*HistoryEntry, error) {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	filename := filepath.Join(hm.basePath, fmt.Sprintf("%s.json", id))
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToReadResponse, err)
	}

	var entry HistoryEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, constants.ErrInvalidRequestBody, err)
	}

	return &entry, nil
}

func (hm *History) GetEntries(page, pageSize int) ([]HistoryEntry, error) {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	files, err := filepath.Glob(filepath.Join(hm.basePath, "*.json"))
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToReadResponse, err)
	}

	var entries []HistoryEntry
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var entry HistoryEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}

		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= len(entries) {
		return []HistoryEntry{}, nil
	}
	if end > len(entries) {
		end = len(entries)
	}

	return entries[start:end], nil
}

func (hm *History) DeleteEntry(id string) error {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	filename := filepath.Join(hm.basePath, fmt.Sprintf("%s.json", id))
	if err := os.Remove(filename); err != nil {
		return errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToDeleteRequest, err)
	}

	return nil
}

func (hm *History) cleanup() {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	files, err := filepath.Glob(filepath.Join(hm.basePath, "*.json"))
	if err != nil {
		return
	}

	var entries []HistoryEntry
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var entry HistoryEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}

		if time.Since(entry.Timestamp) > hm.maxAge {
			err := os.Remove(file)
			if err != nil {
				return
			}
		} else {
			entries = append(entries, entry)
		}
	}

	if len(entries) > hm.maxEntries {
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Timestamp.Before(entries[j].Timestamp)
		})

		for i := 0; i < len(entries)-hm.maxEntries; i++ {
			filename := filepath.Join(hm.basePath, fmt.Sprintf("%s.json", entries[i].ID))
			err := os.Remove(filename)
			if err != nil {
				return
			}
		}
	}
}

func (hm *History) GetMutex() *sync.RWMutex {
	return &hm.mutex
}

func (hm *History) GetBasePath() string {
	return hm.basePath
}
