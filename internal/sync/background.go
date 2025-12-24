package sync

import (
	"os"
	"os/exec"
	"path/filepath"
)

// SpawnBackgroundSync spawns a detached background process to handle sync
// This allows the main CLI to exit immediately while sync continues
func SpawnBackgroundSync() error {
	// Get the current executable path
	executable, err := os.Executable()
	if err != nil {
		return err
	}

	// Resolve symlinks
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return err
	}

	// Spawn a detached background process
	// Use _internal_background_sync as a hidden command
	cmd := exec.Command(executable, "_internal_background_sync")

	// Detach from parent process
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// Start the process and don't wait for it
	if err := cmd.Start(); err != nil {
		return err
	}

	// Detach - don't wait for the process to complete
	// The parent process can exit immediately
	return nil
}
