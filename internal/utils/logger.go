package utils

import (
	"fmt"
	"log"
	"os"
	"sync"
)

// Logger provides leveled logging with verbose mode support
type Logger struct {
	verbose bool
	mu      sync.RWMutex
}

var (
	globalLogger *Logger
	loggerOnce   sync.Once
)

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	loggerOnce.Do(func() {
		globalLogger = &Logger{
			verbose: false,
		}
	})
	return globalLogger
}

// SetVerbose enables or disables verbose logging
func (l *Logger) SetVerbose(verbose bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.verbose = verbose
}

// IsVerbose returns whether verbose logging is enabled
func (l *Logger) IsVerbose() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.verbose
}

// Debug logs a debug message (only when verbose is enabled)
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.IsVerbose() {
		log.Printf("[DEBUG] "+format, args...)
	}
}

// Info logs an informational message
func (l *Logger) Info(format string, args ...interface{}) {
	log.Printf("[INFO] "+format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	log.Printf("[WARN] "+format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	log.Printf("[ERROR] "+format, args...)
}

// Debugf is a convenience function for debug logging
func Debugf(format string, args ...interface{}) {
	GetLogger().Debug(format, args...)
}

// Infof is a convenience function for info logging
func Infof(format string, args ...interface{}) {
	GetLogger().Info(format, args...)
}

// Warnf is a convenience function for warning logging
func Warnf(format string, args ...interface{}) {
	GetLogger().Warn(format, args...)
}

// Errorf is a convenience function for error logging
func Errorf(format string, args ...interface{}) {
	GetLogger().Error(format, args...)
}

// SetVerboseMode is a convenience function to set global verbose mode
func SetVerboseMode(verbose bool) {
	GetLogger().SetVerbose(verbose)
	if verbose {
		// Also set log flags to include timestamp and file info
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
		log.SetOutput(os.Stderr)
	} else {
		// Simplified output for normal mode
		log.SetFlags(0)
		log.SetOutput(os.Stderr)
	}
}

// LogOperation logs the start and end of an operation
func LogOperation(operation string, fn func() error) error {
	logger := GetLogger()
	logger.Debug("Starting operation: %s", operation)

	err := fn()

	if err != nil {
		logger.Debug("Operation failed: %s - %v", operation, err)
	} else {
		logger.Debug("Operation completed: %s", operation)
	}

	return err
}

// LogOperationf logs the start and end of an operation with formatted message
func LogOperationf(format string, fn func() error, args ...interface{}) error {
	operation := fmt.Sprintf(format, args...)
	return LogOperation(operation, fn)
}
