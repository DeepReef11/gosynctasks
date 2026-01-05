//go:build windows

package operations

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// spawnBackgroundSync spawns a completely detached background process to sync (Windows)
func spawnBackgroundSync(configPath string) {
	// Get current executable path
	executable, err := os.Executable()
	if err != nil {
		return // Silent fail - will sync on next operation
	}

	// Check if we're running from a test binary
	// Test binaries don't have sync commands, so spawning would cause issues
	if isTestBinary(executable) {
		return // Don't spawn from test binaries
	}

	// Build command args with config path
	args := []string{"sync", "--quiet"}
	if configPath != "" {
		args = append(args, "--config", configPath)
	}

	// Spawn detached process: gosynctasks sync --quiet --config <path>
	cmd := exec.Command(executable, args...)

	// Windows-specific: create new process group
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}

	// Redirect all I/O
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Start and immediately detach
	_ = cmd.Start()
	// Don't wait - process runs independently
}

// isTestBinary checks if the executable is a test binary
func isTestBinary(path string) bool {
	// Test binaries typically have names like:
	// - *.test (Linux/macOS)
	// - *.test.exe (Windows)
	// - Contain ".test" in the path
	return strings.Contains(path, ".test")
}
