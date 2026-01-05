//go:build unix || darwin || linux

package operations

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	internalSync "gosynctasks/internal/sync"
)

// spawnBackgroundSync spawns a completely detached background process to sync (Unix/Linux/macOS)
// In test mode (when running from test binary), it calls runSyncSynchronously instead
func spawnBackgroundSync(configPath string) {
	// Debug: Log that we're attempting to spawn
	debugLog := "/tmp/gosynctasks-spawn-debug.log"
	if f, err := os.OpenFile(debugLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		_, _ = f.WriteString(fmt.Sprintf("[%s] spawnBackgroundSync called with config: %s\n",
			time.Now().Format(time.RFC3339), configPath))
		_ = f.Close()
	}

	// Get current executable path
	executable, err := os.Executable()
	if err != nil {
		if f, err := os.OpenFile(debugLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			_, _ = f.WriteString(fmt.Sprintf("[%s] ERROR: Failed to get executable: %v\n",
				time.Now().Format(time.RFC3339), err))
			_ = f.Close()
		}
		return // Silent fail - will sync on next operation
	}

	// Check if we're running from a test binary (e.g., nextcloud.test)
	// Test binaries don't have _internal_background_sync command, so we run sync synchronously instead
	if isTestBinary(executable) {
		if f, err := os.OpenFile(debugLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			_, _ = f.WriteString(fmt.Sprintf("[%s] Test mode detected: running sync synchronously instead of spawning\n",
				time.Now().Format(time.RFC3339)))
			_ = f.Close()
		}
		// Run sync in a goroutine to not block the operation
		go runSyncSynchronously(configPath)
		return
	}

	// Build command args with config path
	args := []string{"_internal_background_sync"}
	if configPath != "" {
		args = append(args, "--config", configPath)
	}

	if f, err := os.OpenFile(debugLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		_, _ = f.WriteString(fmt.Sprintf("[%s] Spawning: %s %v\n",
			time.Now().Format(time.RFC3339), executable, args))
		_ = f.Close()
	}

	// Spawn detached process: gosynctasks _internal_background_sync --config <path>
	cmd := exec.Command(executable, args...)

	// Completely detach from parent process (Unix-specific)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // New process group
		Pgid:    0,
	}

	// Redirect all I/O to /dev/null
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Start and immediately detach
	err = cmd.Start()
	if f, err2 := os.OpenFile(debugLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
		if err != nil {
			_, _ = f.WriteString(fmt.Sprintf("[%s] ERROR starting process: %v\n",
				time.Now().Format(time.RFC3339), err))
		} else {
			_, _ = f.WriteString(fmt.Sprintf("[%s] Process spawned successfully, PID: %d\n",
				time.Now().Format(time.RFC3339), cmd.Process.Pid))
		}
		_ = f.Close()
	}
	// Don't wait - process runs independently
}

// isTestBinary checks if the executable is a test binary
func isTestBinary(path string) bool {
	// Test binaries typically have names like:
	// - *.test (Linux/macOS)
	// - *.test.exe (Windows, but this is Unix file)
	// - Contain ".test" in the path
	return strings.Contains(path, ".test")
}

// runSyncSynchronously runs sync in the current process (used in test mode)
// This mimics what _internal_background_sync does but without spawning a new process
func runSyncSynchronously(configPath string) {
	debugLog := "/tmp/gosynctasks-spawn-debug.log"
	if f, err := os.OpenFile(debugLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		_, _ = f.WriteString(fmt.Sprintf("[%s] runSyncSynchronously: running in-process sync\n",
			time.Now().Format(time.RFC3339)))
		_ = f.Close()
	}

	// Import and call the in-process sync from internal/sync
	// This is the same logic as _internal_background_sync but runs in the current process
	if err := internalSync.RunBackgroundSyncInProcess(); err != nil {
		if f, err := os.OpenFile(debugLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			_, _ = f.WriteString(fmt.Sprintf("[%s] Error running in-process sync: %v\n",
				time.Now().Format(time.RFC3339), err))
			_ = f.Close()
		}
	}
}
