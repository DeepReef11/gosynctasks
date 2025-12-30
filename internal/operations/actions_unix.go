//go:build unix || darwin || linux

package operations

import (
	"os"
	"os/exec"
	"syscall"
)

// spawnBackgroundSync spawns a completely detached background process to sync (Unix/Linux/macOS)
func spawnBackgroundSync(configPath string) {
	// Get current executable path
	executable, err := os.Executable()
	if err != nil {
		return // Silent fail - will sync on next operation
	}

	// Build command args with config path
	args := []string{"sync", "--quiet"}
	if configPath != "" {
		args = append(args, "--config", configPath)
	}

	// Spawn detached process: gosynctasks sync --quiet --config <path>
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
	_ = cmd.Start()
	// Don't wait - process runs independently
}
