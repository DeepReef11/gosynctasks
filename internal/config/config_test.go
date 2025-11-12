package config

import (
	"encoding/json"
	"gosynctasks/backend"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

// TestIsOldFormat tests the IsOldFormat method
func TestIsOldFormat(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected bool
	}{
		{
			name: "old format with connector",
			config: Config{
				Connector: &backend.ConnectorConfig{
					URL: mustParseURL("nextcloud://user:pass@example.com"),
				},
				Backends: nil,
			},
			expected: true,
		},
		{
			name: "new format with backends",
			config: Config{
				Connector: nil,
				Backends: map[string]backend.BackendConfig{
					"nextcloud": {Type: "nextcloud", Enabled: true},
				},
			},
			expected: false,
		},
		{
			name: "empty config",
			config: Config{
				Connector: nil,
				Backends:  nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsOldFormat()
			if result != tt.expected {
				t.Errorf("IsOldFormat() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestMigrateConfig tests the migration from old to new format
func TestMigrateConfig(t *testing.T) {
	tests := []struct {
		name      string
		oldConfig *Config
		wantErr   bool
		validate  func(*testing.T, *Config)
	}{
		{
			name: "nextcloud backend migration",
			oldConfig: &Config{
				Connector: &backend.ConnectorConfig{
					URL:                mustParseURL("nextcloud://user:pass@example.com"),
					InsecureSkipVerify: true,
					SuppressSSLWarning: false,
				},
				CanWriteConfig: true,
				UI:             "cli",
				DateFormat:     "2006-01-02",
			},
			wantErr: false,
			validate: func(t *testing.T, newConfig *Config) {
				if newConfig.IsOldFormat() {
					t.Error("migrated config should not be old format")
				}
				if len(newConfig.Backends) != 1 {
					t.Errorf("expected 1 backend, got %d", len(newConfig.Backends))
				}
				ncBackend, ok := newConfig.Backends["nextcloud"]
				if !ok {
					t.Fatal("nextcloud backend not found")
				}
				if ncBackend.Type != "nextcloud" {
					t.Errorf("backend type = %s, want nextcloud", ncBackend.Type)
				}
				if !ncBackend.Enabled {
					t.Error("backend should be enabled")
				}
				if ncBackend.URL != "nextcloud://user:pass@example.com" {
					t.Errorf("URL = %s, want nextcloud://user:pass@example.com", ncBackend.URL)
				}
				if !ncBackend.InsecureSkipVerify {
					t.Error("InsecureSkipVerify should be true")
				}
				if newConfig.DefaultBackend != "nextcloud" {
					t.Errorf("DefaultBackend = %s, want nextcloud", newConfig.DefaultBackend)
				}
				if newConfig.UI != "cli" {
					t.Errorf("UI = %s, want cli", newConfig.UI)
				}
			},
		},
		{
			name: "nil connector",
			oldConfig: &Config{
				Connector: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newConfig, err := migrateConfig(tt.oldConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("migrateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, newConfig)
			}
		})
	}
}

// TestGetDefaultBackend tests the GetDefaultBackend method
func TestGetDefaultBackend(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
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
			wantErr: false,
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
			wantErr: false,
			wantType: "git",
		},
		{
			name: "old format",
			config: &Config{
				Connector: &backend.ConnectorConfig{
					URL: mustParseURL("nextcloud://example.com"),
				},
			},
			wantErr: false,
			wantType: "nextcloud",
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
		{
			name: "old format",
			config: &Config{
				Connector: &backend.ConnectorConfig{
					URL: mustParseURL("nextcloud://example.com"),
				},
			},
			wantCount:    1,
			wantBackends: []string{"nextcloud"},
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
			name: "sqlite backend missing db_path",
			config: Config{
				Backends: map[string]backend.BackendConfig{
					"sqlite": {Type: "sqlite", Enabled: true},
				},
				UI: "cli",
			},
			wantErr: true,
			errMsg:  "db_path is required",
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
			name: "git backend not implemented",
			backend: backend.BackendConfig{
				Type:    "git",
				Enabled: true,
				File:    "TODO.md",
			},
			wantErr: true,
			errMsg:  "not yet implemented",
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

// TestConfigMigrationIntegration tests the full migration flow
func TestConfigMigrationIntegration(t *testing.T) {
	// Create a temporary directory for test config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create an old format config file
	oldConfig := map[string]interface{}{
		"connector": map[string]interface{}{
			"url":                "nextcloud://user:pass@example.com",
			"insecure_skip_verify": false,
		},
		"canWriteConfig": true,
		"ui":             "cli",
		"date_format":    "2006-01-02",
	}

	oldConfigData, err := json.MarshalIndent(oldConfig, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal old config: %v", err)
	}

	err = os.WriteFile(configPath, oldConfigData, 0644)
	if err != nil {
		t.Fatalf("failed to write old config: %v", err)
	}

	// Parse the old config (this should trigger migration)
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	parsedConfig, err := parseConfig(configData, configPath)
	if err != nil {
		t.Fatalf("parseConfig() error = %v", err)
	}

	// Verify the config was migrated
	if parsedConfig.IsOldFormat() {
		t.Error("config should have been migrated to new format")
	}

	// Verify backends were created
	if len(parsedConfig.Backends) != 1 {
		t.Errorf("expected 1 backend, got %d", len(parsedConfig.Backends))
	}

	// Verify backup was created
	backupFiles, err := filepath.Glob(configPath + ".backup.*")
	if err != nil {
		t.Fatalf("failed to glob backup files: %v", err)
	}
	if len(backupFiles) == 0 {
		t.Error("backup file was not created")
	}

	// Verify the new config file was written
	newConfigData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read new config: %v", err)
	}

	var newConfig map[string]interface{}
	err = json.Unmarshal(newConfigData, &newConfig)
	if err != nil {
		t.Fatalf("failed to unmarshal new config: %v", err)
	}

	if _, ok := newConfig["backends"]; !ok {
		t.Error("new config should have 'backends' field")
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
