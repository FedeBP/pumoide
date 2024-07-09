package domain

import (
	"time"
)

type HistoryEntry struct {
	ID                 string                 `json:"id"`
	Timestamp          time.Time              `json:"timestamp"`
	Request            map[string]interface{} `json:"request"`
	Response           map[string]interface{} `json:"response"`
	ExecutionTime      time.Duration          `json:"executionTime"`
	PerformanceMetrics PerformanceMetrics     `json:"performance_metrics"`
}
