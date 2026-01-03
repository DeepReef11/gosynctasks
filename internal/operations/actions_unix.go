//go:build unix || darwin || linux

package operations

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// spawnBackgroundSync spawns a completely detached background process to sync (Unix/Linux/macOS)
func spawnBackgroundSync(configPath string) {
	// Debug: Log that we're attempting to spawn
	debugLog := "/tmp/gosynctasks-spawn-debug.log"
	if f, err := os.OpenFile(debugLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		f.WriteString(fmt.Sprintf("[%s] spawnBackgroundSync called with config: %s\n",
			time.Now().Format(time.RFC3339), configPath))
		f.Close()
	}

	// Get current executable path
	executable, err := os.Executable()
	if err != nil {
		if f, err := os.OpenFile(debugLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			f.WriteString(fmt.Sprintf("[%s] ERROR: Failed to get executable: %v\n",
				time.Now().Format(time.RFC3339), err))
			f.Close()
		}
		return // Silent fail - will sync on next operation
	}

	// Build command args with config path
	args := []string{"_internal_background_sync"}
	if configPath != "" {
		args = append(args, "--config", configPath)
	}

	if f, err := os.OpenFile(debugLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		f.WriteString(fmt.Sprintf("[%s] Spawning: %s %v\n",
			time.Now().Format(time.RFC3339), executable, args))
		f.Close()
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
			f.WriteString(fmt.Sprintf("[%s] ERROR starting process: %v\n",
				time.Now().Format(time.RFC3339), err))
		} else {
			f.WriteString(fmt.Sprintf("[%s] Process spawned successfully, PID: %d\n",
				time.Now().Format(time.RFC3339), cmd.Process.Pid))
		}
		f.Close()
	}
	// Don't wait - process runs independently
}
