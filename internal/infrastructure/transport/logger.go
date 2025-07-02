package transport

import (
	"log"
)

// Logger interface for logging - deprecated, use port.Logger instead
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
}

// DefaultLogger implements the port.Logger interface using standard log package
// Implements port.Logger interface
type DefaultLogger struct{
	// Level could be used to filter logs based on severity
	level string
}

// Debug logs a debug message
func (l *DefaultLogger) Debug(format string, args ...interface{}) {
	log.Printf("[DEBUG] "+format, args...)
}

// Info logs an info message
func (l *DefaultLogger) Info(format string, args ...interface{}) {
	log.Printf("[INFO] "+format, args...)
}

// Warn logs a warning message
func (l *DefaultLogger) Warn(format string, args ...interface{}) {
	log.Printf("[WARN] "+format, args...)
}

// Error logs an error message
func (l *DefaultLogger) Error(format string, args ...interface{}) {
	log.Printf("[ERROR] "+format, args...)
}

// SetLevel mengubah level logging
func (l *DefaultLogger) SetLevel(level string) {
	l.level = level
	// Level implementation could be added here to filter logs
}

// Close menutup logger
func (l *DefaultLogger) Close() error {
	// Nothing to close for the standard logger
	return nil
}
