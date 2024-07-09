package domain

import "time"

type HistoryEntry struct {
	ID                 string             `json:"id"`
	Timestamp          time.Time          `json:"timestamp"`
	Request            Request            `json:"request"`
	Response           Response           `json:"response"`
	ExecutionTime      time.Duration      `json:"executionTime"`
	PerformanceMetrics PerformanceMetrics `json:"performance_metrics"`
}
