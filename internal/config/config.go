package config

import (
	"fmt"

	// "gosynctasks/backend"
	"gosynctasks/backend"
	// "gosynctasks/connectors"
	"gosynctasks/internal/utils"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"

	_ "embed"
)

var configOnce sync.Once

var globalConfig *Config

var customConfigPath string // Custom config path set via --config flag

// var globalConnector *connectors.TaskConnector

//go:embed config.sample.yaml
var sampleConfig []byte

const (
	CONFIG_DIR_PATH  = "gosynctasks"
	CONFIG_FILE_PATH = "config.yaml"
	CONFIG_DIR_PERM  = 0755
	CONFIG_FILE_PERM = 0644
)

// Config represents the application configuration.
type Config struct {
	CanWriteConfig bool `yaml:"canWriteConfig"`

	// Backend configuration
	Backends          map[string]backend.BackendConfig `yaml:"backends,omitempty"`
	DefaultBackend    string                           `yaml:"default_backend,omitempty"`
	AutoDetectBackend bool                             `yaml:"auto_detect_backend,omitempty"`
	BackendPriority   []string                         `yaml:"backend_priority,omitempty"`

	// Common settings
	UI         string      `yaml:"ui" validate:"oneof=cli tui"`
	DateFormat string      `yaml:"date_format,omitempty"` // Go time format string, defaults to "2006-01-02"
	Sync       *SyncConfig `yaml:"sync,omitempty"`        // Sync configuration
}

// SyncConfig represents sync-related settings
type SyncConfig struct {
	Enabled            bool   `yaml:"enabled"`                       // Enable sync functionality
	ConflictResolution string `yaml:"conflict_resolution,omitempty"` // Strategy: server_wins, local_wins, merge, keep_both
	AutoSync           bool   `yaml:"auto_sync,omitempty"`           // Auto-sync on command execution
	SyncInterval       int    `yaml:"sync_interval,omitempty"`       // Minutes between auto-syncs (0=disabled)
	OfflineMode        string `yaml:"offline_mode,omitempty"`        // Mode: auto, online, offline
	LocalBackend       string `yaml:"local_backend,omitempty"`       // Name of local SQLite backend for sync
	RemoteBackend      string `yaml:"remote_backend,omitempty"`      // Name of remote backend to sync with
}

// GetBackend returns the backend configuration for the given name
func (c *Config) GetBackend(name string) (*backend.BackendConfig, error) {
	backendConfig, exists := c.Backends[name]
	if !exists {
		return nil, fmt.Errorf("backend %q not found in config", name)
	}

	return &backendConfig, nil
}

// GetDefaultBackend returns the default backend configuration
func (c *Config) GetDefaultBackend() (*backend.BackendConfig, error) {
	if c.DefaultBackend == "" {
		// Try to find the first enabled backend
		for _, backendConfig := range c.Backends {
			if backendConfig.Enabled {
				return &backendConfig, nil
			}
		}
		return nil, fmt.Errorf("no default backend specified and no enabled backends found")
	}

	return c.GetBackend(c.DefaultBackend)
}

// GetEnabledBackends returns all enabled backend configurations
func (c *Config) GetEnabledBackends() map[string]backend.BackendConfig {
	enabled := make(map[string]backend.BackendConfig)

	for name, backendConfig := range c.Backends {
		if backendConfig.Enabled {
			enabled[name] = backendConfig
		}
	}

	return enabled
}

func (c Config) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return err
	}

	// Validate that backends map is not empty
	if len(c.Backends) == 0 {
		return fmt.Errorf("no backends configured")
	}

	// Validate each backend config
	for name, backendConfig := range c.Backends {
		if err := validate.Struct(backendConfig); err != nil {
			return fmt.Errorf("backend %q validation failed: %w", name, err)
		}

		// Type-specific validation
		switch backendConfig.Type {
		case "nextcloud", "file":
			if backendConfig.URL == "" {
				return fmt.Errorf("backend %q: URL is required for %s backend", name, backendConfig.Type)
			}
		case "git":
			if backendConfig.File == "" {
				// Use default
				backendConfig.File = "TODO.md"
			}
		case "sqlite":
			if backendConfig.DBPath == "" {
				return fmt.Errorf("backend %q: db_path is required for sqlite backend", name)
			}
		}
	}

	// Validate default backend exists and is enabled
	if c.DefaultBackend != "" {
		backend, exists := c.Backends[c.DefaultBackend]
		if !exists {
			return fmt.Errorf("default backend %q not found in configured backends", c.DefaultBackend)
		}
		if !backend.Enabled {
			return fmt.Errorf("default backend %q is disabled", c.DefaultBackend)
		}
	}

	// Validate backend priority list references valid backends
	for _, name := range c.BackendPriority {
		if _, exists := c.Backends[name]; !exists {
			return fmt.Errorf("backend_priority references unknown backend %q", name)
		}
	}

	return nil
}

func (c *Config) GetDateFormat() string {
	if c.DateFormat == "" {
		return "2006-01-02" // Default to yyyy-mm-dd
	}
	return c.DateFormat
}

// SetCustomConfigPath sets a custom config path to use instead of the default user config directory.
// If path is empty or ".", it uses "./gosynctasks/config.yaml" (current directory).
// If path is a directory (or looks like one), it looks for "config.yaml" inside it.
// If path is a file, it uses that file directly.
// This must be called before GetConfig() is called for the first time.
// If GetConfig() was already called, this function will reset it to allow reloading with the new path.
func SetCustomConfigPath(path string) {
	if path == "" || path == "." {
		customConfigPath = filepath.Join(".", CONFIG_DIR_PATH, CONFIG_FILE_PATH)
	} else {
		// Check if path exists and is a directory
		info, err := os.Stat(path)
		if err == nil && info.IsDir() {
			// Path exists and is a directory
			customConfigPath = filepath.Join(path, CONFIG_FILE_PATH)
		} else if err != nil {
			// Path doesn't exist - determine intent from path structure
			// If path ends with config file extension, treat as file path
			// Otherwise, assume it's a directory path
			ext := filepath.Ext(path)
			if ext == ".yaml" || ext == ".yml" || ext == ".YAML" || ext == ".YML" || ext == ".json" || ext == ".JSON" {
				customConfigPath = path
			} else {
				// Assume directory, join with config.yaml
				customConfigPath = filepath.Join(path, CONFIG_FILE_PATH)
			}
		} else {
			// Path exists and is a file
			customConfigPath = path
		}
	}

	// Reset the sync.Once to force config reload with new path
	// This is necessary if GetConfig() was already called before this function
	configOnce = sync.Once{}
	globalConfig = nil
}

func GetConfig() *Config {
	configOnce.Do(func() {
		config, err := loadUserOrSampleConfig()
		if err != nil {
			log.Fatal(err)
		}
		globalConfig = config
	})
	return globalConfig
}

func loadUserOrSampleConfig() (*Config, error) {

	configPath, err := GetConfigPath()
	if err != nil {
		log.Fatalf("Config path couldn't be retrieved")
		return nil, err
	}
	configData, err := configDataFromPath(configPath)
	if err != nil {
		log.Fatalf("Config data couldn't be retrieved")
		return nil, err
	}
	configObj, err := parseConfig(configData, configPath)
	return configObj, err
}

func GetConfigPath() (string, error) {
	// If a custom config path was set, check if it exists
	if customConfigPath != "" {
		if _, err := os.Stat(customConfigPath); err == nil {
			return customConfigPath, nil
		}
		// Custom path was set but doesn't exist, still return it
		// (allows creation of config in custom location)
		return customConfigPath, nil
	}

	// Otherwise, use the default user config directory
	dir, err := os.UserConfigDir()

	if err != nil {
		return "", fmt.Errorf("failed to get user config dir: %w", err)
	}
	return filepath.Join(dir, CONFIG_DIR_PATH, CONFIG_FILE_PATH), nil
}

func createConfigDir(configPath string) error {
	return os.MkdirAll(filepath.Dir(configPath), CONFIG_DIR_PERM)
}
func WriteConfigFile(configPath string, data []byte) error {
	return os.WriteFile(configPath, data, CONFIG_FILE_PERM)
}

func createConfigFromSample(configPath string) []byte {
	var (
		configData []byte
		err        error
	)
	err = createConfigDir(configPath)
	if err != nil {
		log.Fatal(err)
	}
	configData = sampleConfig

	err = WriteConfigFile(configPath, configData)
	if err != nil {
		log.Fatal(err)
	}
	return configData
}

func parseConfig(configData []byte, configPath string) (*Config, error) {
	var configObj Config
	err := yaml.Unmarshal(configData, &configObj)

	if err != nil {
		log.Fatalf("Invalid YAML in config file %s: %v", configPath, err)
	}

	if err = configObj.Validate(); err != nil {
		log.Fatalf("Missing field(s) in YAML config file %s: %v", configPath, err)
	}
	return &configObj, err
}

func configDataFromPath(configPath string) ([]byte, error) {
	var (
		configData []byte
		err        error
	)

	configData, err = os.ReadFile(configPath)
	if os.IsNotExist(err) {
		fmt.Println("No config exist at ", configPath)

		shouldCopySample := utils.PromptYesNo("Do you want to copy config sample to " + configPath + "?")
		if shouldCopySample {
			configData = createConfigFromSample(configPath)

		} else {
			configData = sampleConfig
		}
	}

	return configData, nil

}
