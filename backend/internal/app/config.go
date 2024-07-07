package app

import (
	"os"
	"strconv"
	"time"

	"github.com/FedeBP/pumoide/backend/internal/utils"
	"golang.org/x/time/rate"
)

type Config struct {
	Port                    string
	RateLimit               rate.Limit
	RateLimitBurst          int
	DefaultCollectionsPath  string
	DefaultEnvironmentsPath string
	LogFilePath             string
	LogFileName             string
	LogLevel                string
	ClientTimeout           time.Duration
	Workers                 int
	HistoryEnabled          bool
	HistoryMaxAge           time.Duration
	HistoryMaxEntries       int
}

func LoadConfig() *Config {
	return &Config{
		Port:                    getEnv("PORT", ":0"),
		RateLimit:               rate.Limit(getEnvAsFloat("RATE_LIMIT", 10)),
		RateLimitBurst:          getEnvAsInt("RATE_LIMIT_BURST", 30),
		DefaultCollectionsPath:  getEnv("COLLECTIONS_PATH", utils.GetDefaultCollectionsPath()),
		DefaultEnvironmentsPath: getEnv("ENVIRONMENTS_PATH", utils.GetDefaultEnvironmentsPath()),
		LogFilePath:             getEnv("LOG_FILE_PATH", utils.GetDefaultLogsPath()),
		LogFileName:             getEnv("LOG_FILE_NAME", "pumoide.log"),
		LogLevel:                getEnv("LOG_LEVEL", "info"),
		ClientTimeout:           getEnvAsDuration("CLIENT_TIMEOUT", 0*time.Second),
		Workers:                 getEnvAsInt("WORKERS", 10),
		HistoryEnabled:          getEnvAsBool("HISTORY_ENABLED", true),
		HistoryMaxAge:           getEnvAsDuration("HISTORY_MAX_AGE", 30*24*time.Hour),
		HistoryMaxEntries:       getEnvAsInt("HISTORY_MAX_ENTRIES", 1000),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	strValue := getEnv(key, "")
	if value, err := strconv.Atoi(strValue); err == nil {
		return value
	}
	return fallback
}

func getEnvAsFloat(key string, fallback float64) float64 {
	strValue := getEnv(key, "")
	if value, err := strconv.ParseFloat(strValue, 64); err == nil {
		return value
	}
	return fallback
}

func getEnvAsBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		v, err := strconv.ParseBool(value)
		if err == nil {
			return v
		}
	}
	return fallback
}

func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	if value, ok := os.LookupEnv(key); ok {
		v, err := time.ParseDuration(value)
		if err == nil {
			return v
		}
	}
	return fallback
}
