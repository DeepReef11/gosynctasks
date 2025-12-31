package credentials

import (
	"os"
	"testing"
)

func TestNormalizeBackendName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "nextcloud",
			expected: "NEXTCLOUD",
		},
		{
			name:     "name with hyphen",
			input:    "nextcloud-work",
			expected: "NEXTCLOUD_WORK",
		},
		{
			name:     "name with multiple hyphens",
			input:    "my-cloud-backend",
			expected: "MY_CLOUD_BACKEND",
		},
		{
			name:     "already uppercase",
			input:    "NEXTCLOUD",
			expected: "NEXTCLOUD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeBackendName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeBackendName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetEnvVarName(t *testing.T) {
	tests := []struct {
		name        string
		backendName string
		field       string
		expected    string
	}{
		{
			name:        "username field",
			backendName: "nextcloud",
			field:       "USERNAME",
			expected:    "GOSYNCTASKS_NEXTCLOUD_USERNAME",
		},
		{
			name:        "password field",
			backendName: "nextcloud-work",
			field:       "PASSWORD",
			expected:    "GOSYNCTASKS_NEXTCLOUD_WORK_PASSWORD",
		},
		{
			name:        "host field",
			backendName: "my-backend",
			field:       "HOST",
			expected:    "GOSYNCTASKS_MY_BACKEND_HOST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEnvVarName(tt.backendName, tt.field)
			if result != tt.expected {
				t.Errorf("getEnvVarName(%q, %q) = %q, want %q", tt.backendName, tt.field, result, tt.expected)
			}
		})
	}
}

func TestGetUsername(t *testing.T) {
	// Set test environment variable
	os.Setenv("GOSYNCTASKS_TESTBACKEND_USERNAME", "testuser")
	defer os.Unsetenv("GOSYNCTASKS_TESTBACKEND_USERNAME")

	tests := []struct {
		name        string
		backendName string
		expected    string
	}{
		{
			name:        "existing env var",
			backendName: "testbackend",
			expected:    "testuser",
		},
		{
			name:        "non-existing env var",
			backendName: "nonexistent",
			expected:    "",
		},
		{
			name:        "empty backend name",
			backendName: "",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetUsername(tt.backendName)
			if result != tt.expected {
				t.Errorf("GetUsername(%q) = %q, want %q", tt.backendName, result, tt.expected)
			}
		})
	}
}

func TestGetPassword(t *testing.T) {
	// Set test environment variable
	os.Setenv("GOSYNCTASKS_TESTBACKEND_PASSWORD", "testpass")
	defer os.Unsetenv("GOSYNCTASKS_TESTBACKEND_PASSWORD")

	tests := []struct {
		name        string
		backendName string
		expected    string
	}{
		{
			name:        "existing env var",
			backendName: "testbackend",
			expected:    "testpass",
		},
		{
			name:        "non-existing env var",
			backendName: "nonexistent",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPassword(tt.backendName)
			if result != tt.expected {
				t.Errorf("GetPassword(%q) = %q, want %q", tt.backendName, result, tt.expected)
			}
		})
	}
}

func TestGetHost(t *testing.T) {
	// Set test environment variable
	os.Setenv("GOSYNCTASKS_TESTBACKEND_HOST", "example.com")
	defer os.Unsetenv("GOSYNCTASKS_TESTBACKEND_HOST")

	tests := []struct {
		name        string
		backendName string
		expected    string
	}{
		{
			name:        "existing env var",
			backendName: "testbackend",
			expected:    "example.com",
		},
		{
			name:        "non-existing env var",
			backendName: "nonexistent",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetHost(tt.backendName)
			if result != tt.expected {
				t.Errorf("GetHost(%q) = %q, want %q", tt.backendName, result, tt.expected)
			}
		})
	}
}

func TestHasCredentials(t *testing.T) {
	// Set up test environment
	os.Setenv("GOSYNCTASKS_COMPLETE_USERNAME", "user")
	os.Setenv("GOSYNCTASKS_COMPLETE_PASSWORD", "pass")
	os.Setenv("GOSYNCTASKS_PARTIAL_USERNAME", "user")
	defer func() {
		os.Unsetenv("GOSYNCTASKS_COMPLETE_USERNAME")
		os.Unsetenv("GOSYNCTASKS_COMPLETE_PASSWORD")
		os.Unsetenv("GOSYNCTASKS_PARTIAL_USERNAME")
	}()

	tests := []struct {
		name        string
		backendName string
		expected    bool
	}{
		{
			name:        "both username and password exist",
			backendName: "complete",
			expected:    true,
		},
		{
			name:        "only username exists",
			backendName: "partial",
			expected:    false,
		},
		{
			name:        "neither exists",
			backendName: "nonexistent",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasCredentials(tt.backendName)
			if result != tt.expected {
				t.Errorf("HasCredentials(%q) = %v, want %v", tt.backendName, result, tt.expected)
			}
		})
	}
}
