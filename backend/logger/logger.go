package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

func Init(logLevel string) {
	Log = logrus.New()

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	Log.SetLevel(level)

	Log.SetFormatter(&logrus.JSONFormatter{})

	Log.SetOutput(os.Stdout)
}

func GetLogger() *logrus.Logger {
	if Log == nil {
		Init("info")
	}
	return Log
}
