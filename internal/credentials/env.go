package credentials

import (
	"os"
	"strings"
)

// normalizeBackendName converts a backend name to the format used in environment variables
// Example: "nextcloud-work" becomes "NEXTCLOUD_WORK"
func normalizeBackendName(backendName string) string {
	// Convert to uppercase
	normalized := strings.ToUpper(backendName)
	// Replace hyphens with underscores
	normalized = strings.ReplaceAll(normalized, "-", "_")
	return normalized
}

// getEnvVarName returns the environment variable name for a backend field
func getEnvVarName(backendName, field string) string {
	return "GOSYNCTASKS_" + normalizeBackendName(backendName) + "_" + strings.ToUpper(field)
}

// GetUsername retrieves the username from environment variables
// Looks for: GOSYNCTASKS_{BACKEND_NAME}_USERNAME
func GetUsername(backendName string) string {
	if backendName == "" {
		return ""
	}
	return os.Getenv(getEnvVarName(backendName, "USERNAME"))
}

// GetPassword retrieves the password from environment variables
// Looks for: GOSYNCTASKS_{BACKEND_NAME}_PASSWORD
func GetPassword(backendName string) string {
	if backendName == "" {
		return ""
	}
	return os.Getenv(getEnvVarName(backendName, "PASSWORD"))
}

// GetHost retrieves the host from environment variables
// Looks for: GOSYNCTASKS_{BACKEND_NAME}_HOST
func GetHost(backendName string) string {
	if backendName == "" {
		return ""
	}
	return os.Getenv(getEnvVarName(backendName, "HOST"))
}

// HasCredentials checks if credentials exist in environment variables
func HasCredentials(backendName string) bool {
	username := GetUsername(backendName)
	password := GetPassword(backendName)
	return username != "" && password != ""
}
