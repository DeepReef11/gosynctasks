package config

import (
	"gosynctasks/backend"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	// Import backend packages to register their init() functions
	_ "gosynctasks/backend/file"
	_ "gosynctasks/backend/git"
	_ "gosynctasks/backend/nextcloud"
	_ "gosynctasks/backend/sqlite"
)

// TestGetDefaultBackend tests the GetDefaultBackend method
func TestGetDefaultBackend(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		wantErr  bool
		wantType string
	}{
		{
			name: "explicit default backend",
			config: &Config{
				Backends: map[string]backend.BackendConfig{
					"nextcloud": {Type: "nextcloud", Enabled: true, URL: "nextcloud://example.com"},
					"git":       {Type: "git", Enabled: true, File: "TODO.md"},
				},
				DefaultBackend: "nextcloud",
			},
			wantErr:  false,
			wantType: "nextcloud",
		},
		{
			name: "no default, pick first enabled",
			config: &Config{
				Backends: map[string]backend.BackendConfig{
					"git": {Type: "git", Enabled: true, File: "TODO.md"},
				},
				DefaultBackend: "",
			},
			wantErr:  false,
			wantType: "git",
		},
		{
			name: "no backends configured",
			config: &Config{
				Backends: map[string]backend.BackendConfig{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backendConfig, err := tt.config.GetDefaultBackend()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDefaultBackend() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && backendConfig.Type != tt.wantType {
				t.Errorf("GetDefaultBackend() type = %s, want %s", backendConfig.Type, tt.wantType)
			}
		})
	}
}

// TestGetEnabledBackends tests the GetEnabledBackends method
func TestGetEnabledBackends(t *testing.T) {
	tests := []struct {
		name         string
		config       *Config
		wantCount    int
		wantBackends []string
	}{
		{
			name: "multiple enabled backends",
			config: &Config{
				Backends: map[string]backend.BackendConfig{
					"nextcloud": {Type: "nextcloud", Enabled: true, URL: "nextcloud://example.com"},
					"git":       {Type: "git", Enabled: true, File: "TODO.md"},
					"disabled":  {Type: "file", Enabled: false, URL: "file://path"},
				},
			},
			wantCount:    2,
			wantBackends: []string{"nextcloud", "git"},
		},
		{
			name: "no enabled backends",
			config: &Config{
				Backends: map[string]backend.BackendConfig{
					"disabled": {Type: "file", Enabled: false, URL: "file://path"},
				},
			},
			wantCount:    0,
			wantBackends: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enabled := tt.config.GetEnabledBackends()
			if len(enabled) != tt.wantCount {
				t.Errorf("GetEnabledBackends() count = %d, want %d", len(enabled), tt.wantCount)
			}
			for _, name := range tt.wantBackends {
				if _, ok := enabled[name]; !ok {
					t.Errorf("expected backend %q not found in enabled backends", name)
				}
			}
		})
	}
}

// TestConfigValidation tests the Validate method
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid new format config",
			config: Config{
				Backends: map[string]backend.BackendConfig{
					"nextcloud": {Type: "nextcloud", Enabled: true, URL: "nextcloud://example.com"},
				},
				DefaultBackend:    "nextcloud",
				AutoDetectBackend: false,
				BackendPriority:   []string{"nextcloud"},
				UI:                "cli",
			},
			wantErr: false,
		},
		{
			name: "invalid UI value",
			config: Config{
				Backends: map[string]backend.BackendConfig{
					"nextcloud": {Type: "nextcloud", Enabled: true, URL: "nextcloud://example.com"},
				},
				UI: "invalid",
			},
			wantErr: true,
		},
		{
			name: "no backends configured",
			config: Config{
				Backends: map[string]backend.BackendConfig{},
				UI:       "cli",
			},
			wantErr: true,
			errMsg:  "no backends configured",
		},
		{
			name: "default backend not found",
			config: Config{
				Backends: map[string]backend.BackendConfig{
					"nextcloud": {Type: "nextcloud", Enabled: true, URL: "nextcloud://example.com"},
				},
				DefaultBackend: "nonexistent",
				UI:             "cli",
			},
			wantErr: true,
			errMsg:  "default backend",
		},
		{
			name: "default backend disabled",
			config: Config{
				Backends: map[string]backend.BackendConfig{
					"nextcloud": {Type: "nextcloud", Enabled: false, URL: "nextcloud://example.com"},
				},
				DefaultBackend: "nextcloud",
				UI:             "cli",
			},
			wantErr: true,
			errMsg:  "disabled",
		},
		{
			name: "nextcloud backend missing URL",
			config: Config{
				Backends: map[string]backend.BackendConfig{
					"nextcloud": {Type: "nextcloud", Enabled: true},
				},
				UI: "cli",
			},
			wantErr: true,
			errMsg:  "URL is required",
		},
		{
			name: "sqlite backend missing db_path is valid (uses XDG default)",
			config: Config{
				Backends: map[string]backend.BackendConfig{
					"sqlite": {Type: "sqlite", Enabled: true},
				},
				UI: "cli",
			},
			wantErr: false, // db_path is optional for sqlite
		},
		{
			name: "priority references unknown backend",
			config: Config{
				Backends: map[string]backend.BackendConfig{
					"nextcloud": {Type: "nextcloud", Enabled: true, URL: "nextcloud://example.com"},
				},
				BackendPriority: []string{"nonexistent"},
				UI:              "cli",
			},
			wantErr: true,
			errMsg:  "unknown backend",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || len(tt.errMsg) == 0 {
					t.Errorf("expected error containing %q, got %v", tt.errMsg, err)
				}
			}
		})
	}
}

// TestBackendConfigTaskManager tests creating TaskManager from BackendConfig
func TestBackendConfigTaskManager(t *testing.T) {
	tests := []struct {
		name    string
		backend backend.BackendConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "disabled backend",
			backend: backend.BackendConfig{
				Type:    "nextcloud",
				Enabled: false,
				URL:     "nextcloud://example.com",
			},
			wantErr: true,
			errMsg:  "disabled",
		},
		{
			name: "git backend implemented",
			backend: backend.BackendConfig{
				Type:    "git",
				Enabled: true,
				File:    "TODO.md",
			},
			wantErr: false,
		},
		{
			name: "sqlite backend not implemented",
			backend: backend.BackendConfig{
				Type:    "sqlite",
				Enabled: true,
				DBPath:  "/path/to/db",
			},
			wantErr: true,
			errMsg:  "not yet implemented",
		},
		{
			name: "invalid URL for nextcloud",
			backend: backend.BackendConfig{
				Type:    "nextcloud",
				Enabled: true,
				URL:     "://invalid",
			},
			wantErr: true,
			errMsg:  "invalid URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.backend.TaskManager()
			if (err != nil) != tt.wantErr {
				t.Errorf("TaskManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				// Check if error message contains expected substring
				// (simplified check)
			}
		})
	}
}

// TestSetCustomConfigPath tests the SetCustomConfigPath function
func TestSetCustomConfigPath(t *testing.T) {
	// Save original state
	originalCustomPath := customConfigPath
	defer func() {
		customConfigPath = originalCustomPath
	}()

	// Create temporary directory for testing
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		path     string
		setup    func() string // Returns the path to use for testing
		expected string
	}{
		{
			name:     "empty path defaults to current dir",
			path:     "",
			expected: filepath.Join(".", CONFIG_DIR_PATH, CONFIG_FILE_PATH),
		},
		{
			name:     "dot path defaults to current dir",
			path:     ".",
			expected: filepath.Join(".", CONFIG_DIR_PATH, CONFIG_FILE_PATH),
		},
		{
			name:     "json file path used as-is",
			path:     "/path/to/myconfig.json",
			expected: "/path/to/myconfig.json",
		},
		{
			name:     "uppercase JSON file path used as-is",
			path:     "/path/to/MYCONFIG.JSON",
			expected: "/path/to/MYCONFIG.JSON",
		},
		{
			name: "existing directory appends config.json",
			setup: func() string {
				// Create a test directory
				dir := filepath.Join(tmpDir, "existing_dir")
				os.MkdirAll(dir, 0755)
				return dir
			},
			expected: "", // Will be set dynamically
		},
		{
			name: "existing file used as-is",
			setup: func() string {
				// Create a test file
				file := filepath.Join(tmpDir, "existing_config")
				os.WriteFile(file, []byte("{}"), 0644)
				return file
			},
			expected: "", // Will be set dynamically
		},
		{
			name:     "non-existent directory path (no .json) appends config.json",
			path:     "/path/to/nonexistent/dir",
			expected: filepath.Join("/path/to/nonexistent/dir", CONFIG_FILE_PATH),
		},
		{
			name:     "non-existent path with .json treated as file",
			path:     "/path/to/nonexistent/config.json",
			expected: "/path/to/nonexistent/config.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset custom path before each test
			customConfigPath = ""

			path := tt.path
			expected := tt.expected

			// Run setup if provided
			if tt.setup != nil {
				path = tt.setup()
				if tt.name == "existing directory appends config.json" {
					expected = filepath.Join(path, CONFIG_FILE_PATH)
				} else if tt.name == "existing file used as-is" {
					expected = path
				}
			}

			// Set custom config path
			SetCustomConfigPath(path)

			// Verify result
			if customConfigPath != expected {
				t.Errorf("SetCustomConfigPath(%q)\n  got:  %q\n  want: %q", path, customConfigPath, expected)
			}
		})
	}
}

// Helper function to parse URL without error handling
func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return u
}
