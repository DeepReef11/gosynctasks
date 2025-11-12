package config

import (
	"encoding/json"
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

	_ "embed"
)

var configOnce sync.Once

var globalConfig *Config

var customConfigPath string // Custom config path set via --config flag

// var globalConnector *connectors.TaskConnector

//go:embed config.sample.json
var sampleConfig []byte

const (
	CONFIG_DIR_PATH  = "gosynctasks"
	CONFIG_FILE_PATH = "config.json"
	CONFIG_DIR_PERM  = 0755
	CONFIG_FILE_PERM = 0644
)

// Config represents the application configuration.
// It supports both old single-backend format (deprecated) and new multi-backend format.
type Config struct {
	// Old format (deprecated - will be migrated automatically)
	Connector      *backend.ConnectorConfig `json:"connector,omitempty"`
	CanWriteConfig bool                     `json:"canWriteConfig"`

	// New multi-backend format
	Backends          map[string]backend.BackendConfig `json:"backends,omitempty"`
	DefaultBackend    string                           `json:"default_backend,omitempty"`
	AutoDetectBackend bool                             `json:"auto_detect_backend,omitempty"`
	BackendPriority   []string                         `json:"backend_priority,omitempty"`

	// Common settings
	UI         string `json:"ui" validate:"oneof=cli tui"`
	DateFormat string `json:"date_format,omitempty"` // Go time format string, defaults to "2006-01-02"
}

// IsOldFormat returns true if this config uses the old single-backend format
func (c *Config) IsOldFormat() bool {
	return c.Connector != nil && len(c.Backends) == 0
}

// GetBackend returns the backend configuration for the given name
func (c *Config) GetBackend(name string) (*backend.BackendConfig, error) {
	if c.IsOldFormat() {
		return nil, fmt.Errorf("config is in old format, please migrate first")
	}

	backendConfig, exists := c.Backends[name]
	if !exists {
		return nil, fmt.Errorf("backend %q not found in config", name)
	}

	return &backendConfig, nil
}

// GetDefaultBackend returns the default backend configuration
func (c *Config) GetDefaultBackend() (*backend.BackendConfig, error) {
	if c.IsOldFormat() {
		// For backward compatibility, create a BackendConfig from old Connector
		if c.Connector == nil {
			return nil, fmt.Errorf("no connector configured in old format")
		}

		return &backend.BackendConfig{
			Type:               c.Connector.URL.Scheme,
			Enabled:            true,
			URL:                c.Connector.URL.String(),
			InsecureSkipVerify: c.Connector.InsecureSkipVerify,
			SuppressSSLWarning: c.Connector.SuppressSSLWarning,
		}, nil
	}

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

	if c.IsOldFormat() {
		// For backward compatibility, return the old connector as a backend
		if c.Connector != nil {
			enabled[c.Connector.URL.Scheme] = backend.BackendConfig{
				Type:               c.Connector.URL.Scheme,
				Enabled:            true,
				URL:                c.Connector.URL.String(),
				InsecureSkipVerify: c.Connector.InsecureSkipVerify,
				SuppressSSLWarning: c.Connector.SuppressSSLWarning,
			}
		}
		return enabled
	}

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

	// Additional validation for multi-backend config
	if !c.IsOldFormat() {
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
// If path is empty or ".", it uses "./gosynctasks/config.json" (current directory).
// If path is a directory, it looks for "config.json" inside it.
// If path is a file, it uses that file directly.
// This must be called before GetConfig() is called for the first time.
func SetCustomConfigPath(path string) {
	if path == "" || path == "." {
		customConfigPath = filepath.Join(".", CONFIG_DIR_PATH, CONFIG_FILE_PATH)
	} else {
		// Check if path is a directory
		info, err := os.Stat(path)
		if err == nil && info.IsDir() {
			customConfigPath = filepath.Join(path, CONFIG_FILE_PATH)
		} else {
			customConfigPath = path
		}
	}
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
	// configObj, err := UnmarshalJSON(configData)
	err := json.Unmarshal(configData, &configObj)

	if err != nil {
		log.Fatalf("Invalid JSON in config file %s: %v", configPath, err)
	}

	// Check if migration is needed
	if configObj.IsOldFormat() {
		fmt.Println("\n=== Config Migration ===")
		fmt.Println("Detected old config format. Migrating to new multi-backend format...")

		// Create backup
		backupPath := configPath + ".backup"
		if err := createBackup(configPath, backupPath); err != nil {
			log.Fatalf("Failed to create config backup: %v", err)
		}
		fmt.Printf("✓ Backup created: %s\n", backupPath)

		// Migrate to new format
		migratedConfig, err := migrateConfig(&configObj)
		if err != nil {
			log.Fatalf("Failed to migrate config: %v", err)
		}

		// Validate migrated config
		if err = migratedConfig.Validate(); err != nil {
			log.Fatalf("Migrated config validation failed: %v", err)
		}

		// Write migrated config
		migratedData, err := json.MarshalIndent(migratedConfig, "", "  ")
		if err != nil {
			log.Fatalf("Failed to serialize migrated config: %v", err)
		}

		if err := WriteConfigFile(configPath, migratedData); err != nil {
			log.Fatalf("Failed to write migrated config: %v", err)
		}

		fmt.Printf("✓ Config migrated successfully\n")
		fmt.Printf("✓ New config written to: %s\n", configPath)
		fmt.Println("======================")

		return migratedConfig, nil
	}

	if err = configObj.Validate(); err != nil {
		log.Fatalf("Missing field(s) in JSON config file %s: %v", configPath, err)
	}
	return &configObj, err
}

// createBackup creates a timestamped backup of the config file
func createBackup(configPath, backupPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config for backup: %w", err)
	}

	// Add timestamp to backup
	timestamp := fmt.Sprintf(".%d", os.Getpid())
	backupPathWithTime := backupPath + timestamp

	if err := os.WriteFile(backupPathWithTime, data, CONFIG_FILE_PERM); err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}

	return nil
}

// migrateConfig converts old format config to new multi-backend format
func migrateConfig(oldConfig *Config) (*Config, error) {
	if oldConfig.Connector == nil {
		return nil, fmt.Errorf("old config has no connector field")
	}

	// Determine backend type from URL scheme
	scheme := oldConfig.Connector.URL.Scheme
	backendType := scheme // Default to scheme as type

	// Create the backend name (use scheme as default name)
	backendName := scheme

	// Build the new backend config
	newBackendConfig := backend.BackendConfig{
		Type:               backendType,
		Enabled:            true,
		URL:                oldConfig.Connector.URL.String(),
		InsecureSkipVerify: oldConfig.Connector.InsecureSkipVerify,
		SuppressSSLWarning: oldConfig.Connector.SuppressSSLWarning,
	}

	// Create new config with migrated backend
	newConfig := &Config{
		Backends: map[string]backend.BackendConfig{
			backendName: newBackendConfig,
		},
		DefaultBackend:    backendName,
		AutoDetectBackend: false, // Conservative default
		BackendPriority:   []string{backendName},
		CanWriteConfig:    oldConfig.CanWriteConfig,
		UI:                oldConfig.UI,
		DateFormat:        oldConfig.DateFormat,
	}

	return newConfig, nil
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
