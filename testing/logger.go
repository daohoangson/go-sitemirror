package testing

import (
	"os"

	"github.com/Sirupsen/logrus"
)

var logger *logrus.Logger

// Logger returns an instance of logger
func Logger() *logrus.Logger {
	if logger == nil {
		logger = logrus.New()
		logger.Level = logrus.DebugLevel

		levelStr := os.Getenv("TESTING_LOGGER_LEVEL")
		if level, err := logrus.ParseLevel(levelStr); err == nil {
			logger.Level = level
		}
	}

	return logger
}
