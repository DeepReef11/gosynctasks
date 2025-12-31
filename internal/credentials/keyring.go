package credentials

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const (
	// KeyringServicePrefix is the prefix for all gosynctasks keyring entries
	KeyringServicePrefix = "gosynctasks"
)

// KeyringEntry represents a credential stored in the keyring
type KeyringEntry struct {
	BackendName string
	Username    string
}

// getServiceName returns the keyring service name for a backend
func getServiceName(backendName string) string {
	return fmt.Sprintf("%s-%s", KeyringServicePrefix, backendName)
}

// Set stores credentials in the OS keyring
func Set(backendName, username, password string) error {
	if backendName == "" {
		return fmt.Errorf("backend name cannot be empty")
	}
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	serviceName := getServiceName(backendName)
	err := keyring.Set(serviceName, username, password)
	if err != nil {
		return fmt.Errorf("failed to store credentials in keyring: %w", err)
	}

	return nil
}

// Get retrieves a password from the OS keyring
func Get(backendName, username string) (string, error) {
	if backendName == "" {
		return "", fmt.Errorf("backend name cannot be empty")
	}
	if username == "" {
		return "", fmt.Errorf("username cannot be empty")
	}

	serviceName := getServiceName(backendName)
	password, err := keyring.Get(serviceName, username)
	if err != nil {
		// Check if it's a "not found" error
		if err == keyring.ErrNotFound {
			return "", fmt.Errorf("no credentials found in keyring for backend %q and user %q", backendName, username)
		}
		return "", fmt.Errorf("failed to retrieve credentials from keyring: %w", err)
	}

	return password, nil
}

// Delete removes credentials from the OS keyring
func Delete(backendName, username string) error {
	if backendName == "" {
		return fmt.Errorf("backend name cannot be empty")
	}
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	serviceName := getServiceName(backendName)
	err := keyring.Delete(serviceName, username)
	if err != nil {
		if err == keyring.ErrNotFound {
			return fmt.Errorf("no credentials found in keyring for backend %q and user %q", backendName, username)
		}
		return fmt.Errorf("failed to delete credentials from keyring: %w", err)
	}

	return nil
}

// IsAvailable checks if the keyring is accessible
// This is useful for providing helpful error messages when keyring is not available
func IsAvailable() bool {
	// Try to access the keyring with a test operation
	// Use a unique service name that won't conflict with real credentials
	testService := "gosynctasks-keyring-test"
	testUser := "test"

	// Try to get a non-existent item - if keyring is available, we'll get ErrNotFound
	// If keyring is not available, we'll get a different error
	_, err := keyring.Get(testService, testUser)

	// Keyring is available if we get ErrNotFound (meaning keyring works but item doesn't exist)
	// or if we get no error (meaning the test item exists, which is unlikely but means keyring works)
	return err == nil || err == keyring.ErrNotFound
}

// List returns all keyring entries for gosynctasks
// Note: go-keyring doesn't provide a native list function, so this is a stub
// that would require platform-specific implementation or storing metadata separately
func List() ([]KeyringEntry, error) {
	// Unfortunately, go-keyring doesn't provide a way to list all entries
	// This would require platform-specific code or maintaining a separate registry
	// For now, we'll return an error indicating this limitation
	return nil, fmt.Errorf("listing keyring entries is not supported by the current keyring library")
}
