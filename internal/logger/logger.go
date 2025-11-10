package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

// Init initializes the logger with specified configuration
func Init(level string) error {
	Log = logrus.New()

	// Set output to stdout
	Log.SetOutput(os.Stdout)

	// Set log format to JSON for easy parsing by log aggregation tools
	Log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

	// Parse and set log level
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		Log.Warn("Invalid log level provided, defaulting to INFO")
		logLevel = logrus.InfoLevel
	}
	Log.SetLevel(logLevel)

	Log.WithFields(logrus.Fields{
		"level": logLevel.String(),
	}).Info("Logger initialized successfully")

	return nil
}

// WithFields creates a new log entry with the specified fields
func WithFields(fields logrus.Fields) *logrus.Entry {
	return Log.WithFields(fields)
}

// Info logs an info message
func Info(msg string) {
	Log.Info(msg)
}

// Warn logs a warning message
func Warn(msg string) {
	Log.Warn(msg)
}

// Error logs an error message
func Error(msg string) {
	Log.Error(msg)
}

// Fatal logs a fatal message and exits
func Fatal(msg string) {
	Log.Fatal(msg)
}

// Debug logs a debug message
func Debug(msg string) {
	Log.Debug(msg)
}

