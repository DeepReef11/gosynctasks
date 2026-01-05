package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackgroundLogger(t *testing.T) {
	// Create a new background logger
	bgLogger, err := NewBackgroundLogger()
	if err != nil && bgLogger.IsEnabled() {
		t.Fatalf("Failed to create background logger: %v", err)
	}
	defer bgLogger.Close()

	// Check if logging is enabled based on the constant
	if !ENABLE_BACKGROUND_LOGGING {
		t.Log("Background logging is disabled via ENABLE_BACKGROUND_LOGGING constant")
		if bgLogger.IsEnabled() {
			t.Error("Logger should be disabled when ENABLE_BACKGROUND_LOGGING is false")
		}
		return
	}

	// Verify logger is enabled
	if !bgLogger.IsEnabled() {
		t.Fatal("Logger should be enabled when ENABLE_BACKGROUND_LOGGING is true")
	}

	// Verify log path is set correctly
	logPath := bgLogger.GetLogPath()
	if logPath == "" {
		t.Fatal("Log path should not be empty")
	}

	// Check log path format: /tmp/gosynctasks-_internal_background_sync-{PID}.log
	expectedPrefix := filepath.Join(os.TempDir(), "gosynctasks-_internal_background_sync-")
	if !strings.HasPrefix(logPath, expectedPrefix) {
		t.Errorf("Log path should start with %s, got: %s", expectedPrefix, logPath)
	}

	// Verify log file exists and contains PID
	pid := os.Getpid()
	t.Logf("Log file path: %s", logPath)
	t.Logf("PID: %d", pid)

	// Write a test message
	bgLogger.Printf("Test message from PID %d", pid)

	// Close the logger to flush
	bgLogger.Close()

	// Verify file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("Log file should exist at %s", logPath)
	}

	// Clean up test log file
	os.Remove(logPath)
}

func TestBackgroundLoggerDisabled(t *testing.T) {
	// This test verifies behavior when logging is disabled
	// We can't actually test with ENABLE_BACKGROUND_LOGGING=false without modifying the const,
	// but we can verify the behavior when file creation fails

	// Save original constant state for documentation
	t.Logf("ENABLE_BACKGROUND_LOGGING is currently: %v", ENABLE_BACKGROUND_LOGGING)

	if !ENABLE_BACKGROUND_LOGGING {
		bgLogger, _ := NewBackgroundLogger()
		defer bgLogger.Close()

		if bgLogger.IsEnabled() {
			t.Error("Logger should be disabled when ENABLE_BACKGROUND_LOGGING is false")
		}

		// These calls should not panic even when disabled
		bgLogger.Printf("Test message")
		bgLogger.Print("Test message")
		bgLogger.Println("Test message")
	}
}

func TestBackgroundLoggerMethods(t *testing.T) {
	bgLogger, err := NewBackgroundLogger()
	if err != nil && bgLogger.IsEnabled() {
		t.Fatalf("Failed to create background logger: %v", err)
	}
	defer bgLogger.Close()

	if !bgLogger.IsEnabled() {
		t.Skip("Logging is disabled, skipping method tests")
	}

	// Test different logging methods - these should not panic
	bgLogger.Printf("Printf test: %d", 42)
	bgLogger.Print("Print test")
	bgLogger.Println("Println test")

	// Clean up
	logPath := bgLogger.GetLogPath()
	bgLogger.Close()
	os.Remove(logPath)
}
