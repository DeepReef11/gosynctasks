package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// ExpandPath expands ~ and environment variables in file paths
// Examples:
//   - "~/data/file.txt" -> "/home/user/data/file.txt"
//   - "~/.config/app" -> "/home/user/.config/app"
//   - "$HOME/data" -> "/home/user/data"
//   - "/abs/path" -> "/abs/path" (unchanged)
func ExpandPath(path string) (string, error) {
	if path == "" {
		return path, nil
	}

	// Expand environment variables first
	path = os.ExpandEnv(path)

	// Handle tilde expansion
	if strings.HasPrefix(path, "~/") || path == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		if path == "~" {
			return homeDir, nil
		}

		// Replace ~ with home directory
		path = filepath.Join(homeDir, path[2:])
	}

	return path, nil
}
