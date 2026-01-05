package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// ENABLE_BACKGROUND_LOGGING controls whether background sync logging is enabled
// Set to false to disable background sync logging entirely
//
// When enabled (true):
//   - Background sync logs to: /tmp/gosynctasks-_internal_background_sync-{PID}.log
//   - Each background sync process creates its own log file with unique PID
//   - Logs include sync operations, errors, and timing information
//
// When disabled (false):
//   - No log files are created
//   - All logging calls are no-ops (zero overhead)
//   - Background sync still runs, just without logging
const ENABLE_BACKGROUND_LOGGING = true

// Logger provides leveled logging with verbose mode support
type Logger struct {
	verbose bool
	mu      sync.RWMutex
}

// BackgroundLogger provides logging for background sync processes
type BackgroundLogger struct {
	logger   *log.Logger
	logFile  *os.File
	enabled  bool
	filePath string
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

// ============================================================================
// BackgroundLogger implementation
// ============================================================================

// NewBackgroundLogger creates a new background logger that writes to a PID-specific tmp file
// The log file will be at: /tmp/gosynctasks-_internal_background_sync-{PID}.log
func NewBackgroundLogger() (*BackgroundLogger, error) {
	bl := &BackgroundLogger{
		enabled: ENABLE_BACKGROUND_LOGGING,
	}

	// If logging is disabled, use a no-op logger
	if !bl.enabled {
		bl.logger = log.New(io.Discard, "", 0)
		return bl, nil
	}

	// Create PID-specific log file path
	pid := os.Getpid()
	bl.filePath = filepath.Join(os.TempDir(), fmt.Sprintf("gosynctasks-_internal_background_sync-%d.log", pid))

	// Open log file (create or append)
	logFile, err := os.OpenFile(bl.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// If we can't open the log file, fall back to discard
		bl.logger = log.New(io.Discard, "", 0)
		bl.enabled = false
		return bl, fmt.Errorf("failed to open log file %s: %w (logging disabled)", bl.filePath, err)
	}

	bl.logFile = logFile
	bl.logger = log.New(logFile, "[BackgroundSync] ", log.LstdFlags)

	return bl, nil
}

// Printf logs a formatted message (same signature as log.Printf)
func (bl *BackgroundLogger) Printf(format string, v ...interface{}) {
	if bl.logger != nil {
		bl.logger.Printf(format, v...)
	}
}

// Print logs a message (same signature as log.Print)
func (bl *BackgroundLogger) Print(v ...interface{}) {
	if bl.logger != nil {
		bl.logger.Print(v...)
	}
}

// Println logs a message with newline (same signature as log.Println)
func (bl *BackgroundLogger) Println(v ...interface{}) {
	if bl.logger != nil {
		bl.logger.Println(v...)
	}
}

// Close closes the log file
func (bl *BackgroundLogger) Close() error {
	if bl.logFile != nil {
		return bl.logFile.Close()
	}
	return nil
}

// GetLogPath returns the path to the log file
func (bl *BackgroundLogger) GetLogPath() string {
	return bl.filePath
}

// IsEnabled returns whether logging is enabled
func (bl *BackgroundLogger) IsEnabled() bool {
	return bl.enabled
}
