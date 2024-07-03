package app

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/FedeBP/pumoide/backend/utils"
	"github.com/sirupsen/logrus"
)

type Pumoide struct {
	Config *Config
	Logger *logrus.Logger
	Router *http.ServeMux
}

func NewPumoide() (*Pumoide, error) {
	config := LoadConfig()
	logger := initLogger(config)

	return &Pumoide{
		Config: config,
		Logger: logger,
		Router: http.NewServeMux(),
	}, nil
}

func initLogger(config *Config) *logrus.Logger {
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

	return logger
}

func (a *Pumoide) Start() error {
	a.Logger.Printf("Storage location: %s", utils.GetCurrentStorageLocation())

	if err := utils.EnsureDir(a.Config.DefaultCollectionsPath); err != nil {
		return fmt.Errorf("failed to create collections directory: %w", err)
	}

	if err := utils.EnsureDir(a.Config.DefaultEnvironmentsPath); err != nil {
		return fmt.Errorf("failed to create environments directory: %w", err)
	}

	listener, err := net.Listen("tcp", a.Config.Port)
	if err != nil {
		a.Logger.Fatalf("Failed to start listener: %v", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	log.Printf("Server starting on port %d...", port)
	a.Logger.Printf("Server starting on port %d...", port)

	if err := os.WriteFile("port.txt", []byte(fmt.Sprintf("%d", port)), 0644); err != nil {
		a.Logger.Printf("Failed to write port to file: %v", err)
	}

	return http.Serve(listener, a.Router)
}
