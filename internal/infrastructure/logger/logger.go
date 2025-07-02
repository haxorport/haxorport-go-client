package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/haxorport/haxorport-go-client/internal/domain/port"
)

// Level mendefinisikan level logging
type Level int

const (
	// LevelDebug is the level for debug messages
	LevelDebug Level = iota
	// LevelInfo is the level for informational messages
	LevelInfo
	// LevelWarn is the level for warning messages
	LevelWarn
	// LevelError is the level for error messages
	LevelError
)

// String returns the string representation of the level
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel converts a string to Level
func ParseLevel(level string) Level {
	switch strings.ToLower(level) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// Logger is an implementation of port.Logger
type Logger struct {
	logger *log.Logger
	level  Level
	writer io.Writer
}

// NewLogger creates a new Logger instance
func NewLogger(writer io.Writer, level string) *Logger {
	return &Logger{
		logger: log.New(writer, "", 0),
		level:  ParseLevel(level),
		writer: writer,
	}
}

// SetLevel changes the logging level
func (l *Logger) SetLevel(level string) {
	l.level = ParseLevel(level)
}

// log records a message with a specific level
func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	// Format time
	now := time.Now().Format("2006-01-02 15:04:05.000")
	
	// Format message
	var message string
	if len(args) > 0 {
		message = fmt.Sprintf(format, args...)
	} else {
		message = format
	}

	// Log message
	l.logger.Printf("[%s] %s %s", now, level.String(), message)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Info logs an informational message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// Close closes the writer if it implements io.Closer
func (l *Logger) Close() error {
	if closer, ok := l.writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// NewFileLogger creates a logger that writes to a file
func NewFileLogger(filePath string, level string) (*Logger, error) {
	// Create directory if it doesn't exist
	dir := strings.TrimSuffix(filePath, "/"+strings.Split(filePath, "/")[len(strings.Split(filePath, "/"))-1])
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}
	
	// Open file
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}
	
	return NewLogger(file, level), nil
}

// Ensure Logger implements port.Logger
var _ port.Logger = (*Logger)(nil)
