package app

import (
	"os"
	"strconv"
	"time"

	"github.com/FedeBP/pumoide/backend/utils"
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
		ClientTimeout:           time.Duration(getEnvAsInt("CLIENT_TIMEOUT", 30)) * time.Second,
		Workers:                 getEnvAsInt("WORKERS", 10),
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
