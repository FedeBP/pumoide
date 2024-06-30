package main

import (
	"fmt"
	"golang.org/x/time/rate"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/FedeBP/pumoide/backend/utils"
	"github.com/sirupsen/logrus"
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
}

type Pumoide struct {
	config *Config
	logger *logrus.Logger
	router *http.ServeMux
}

func (a *Pumoide) Start() error {
	log.Printf("Storage location: %s", utils.GetCurrentStorageLocation())

	if err := utils.EnsureDir(a.config.DefaultCollectionsPath); err != nil {
		return fmt.Errorf("failed to create collections directory: %w", err)
	}

	if err := utils.EnsureDir(a.config.DefaultEnvironmentsPath); err != nil {
		return fmt.Errorf("failed to create environments directory: %w", err)
	}

	listener, err := net.Listen("tcp", a.config.Port)
	if err != nil {
		a.logger.Fatalf("Failed to start listener: %v", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	log.Printf("Server starting on port %d...", port)
	a.logger.Printf("Server starting on port %d...", port)

	// Write the port for Tauri and Svelte to read...
	if err := os.WriteFile("port.txt", []byte(fmt.Sprintf("%d", port)), 0644); err != nil {
		a.logger.Printf("Failed to write port to file: %v", err)
	}

	return http.Serve(listener, a.router)
}

func InitPumoide() (*Pumoide, error) {
	config := &Config{
		Port:                    ":0",
		RateLimit:               rate.Limit(10),
		RateLimitBurst:          30,
		DefaultCollectionsPath:  utils.GetDefaultCollectionsPath(),
		DefaultEnvironmentsPath: utils.GetDefaultEnvironmentsPath(),
		LogFilePath:             utils.GetDefaultLogsPath(),
		LogFileName:             "pumoide.log",
		LogLevel:                "info",
		ClientTimeout:           30 * time.Second,
	}

	logger := logrus.New()

	logFilePath := filepath.Join(config.LogFilePath, config.LogFileName)

	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		logger.Out = file
	} else {
		logger.Info("Failed to log to file, using default stderr")
	}

	level, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	logger.SetFormatter(&logrus.JSONFormatter{})

	return &Pumoide{
		config: config,
		logger: logger,
		router: http.NewServeMux(),
	}, nil
}
