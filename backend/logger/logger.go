package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

func Init(logFilePath string, logLevel string) {
	Log = logrus.New()

	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		Log.Out = file
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
