package port

// Logger is an interface for logging
type Logger interface {
	// Debug logs a debug message
	Debug(format string, args ...interface{})
	
	// Info logs an informational message
	Info(format string, args ...interface{})
	
	// Warn logs a warning message
	Warn(format string, args ...interface{})
	
	// Error logs an error message
	Error(format string, args ...interface{})
	
	// SetLevel changes the logging level
	SetLevel(level string)
	
	// Close closes the logger
	Close() error
}
