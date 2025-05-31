package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

func Init() {
	Log = logrus.New()
	Log.SetOutput(os.Stdout)
	Log.SetLevel(logrus.InfoLevel)
	Log.SetFormatter(&logrus.JSONFormatter{})
}

func GetLogger() *logrus.Logger {
	if Log == nil {
		Init()
	}
	return Log
}