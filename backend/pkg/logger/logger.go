package logger

import (
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	Log     *logrus.Logger
	logFile *os.File
	mu      sync.Mutex
)

func Init(logFilePath string, logLevel string) {
	mu.Lock()
	defer mu.Unlock()

	if Log != nil {
		return
	}

	Log = logrus.New()

	var err error
	logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		Log.Out = logFile
	} else {
		Log.Info("Failed to log to file, using default stderr")
	}

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	Log.SetLevel(level)

	Log.SetFormatter(&logrus.JSONFormatter{})
}

func GetLogger(logFilePath string, logLevel string) *logrus.Logger {
	if Log == nil {
		Init(logFilePath, logLevel)
	}
	return Log
}

func Close() {
	mu.Lock()
	defer mu.Unlock()

	if logFile != nil {
		err := logFile.Close()
		if err != nil {
			Log.Info("Failed to close log file")
			return
		}
		logFile = nil
	}
	Log = nil
}
