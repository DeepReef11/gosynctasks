package config

import (
	"fmt"
	"strings"

	// "gosynctasks/backend"
	"gosynctasks/backend"
	// "gosynctasks/connectors"
	"gosynctasks/internal/utils"
	"gosynctasks/internal/views"
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

// expandPath expands ~ and $HOME in paths while respecting escaped versions.
// Escaping rules:
//   - \~ becomes literal ~
//   - \$HOME or \$ becomes literal $HOME or $
//   - ~ at start of path expands to user home directory
//   - $HOME anywhere in path expands to user home directory
func expandPath(path string) string {
	if path == "" {
		return path
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		// If we can't get home dir, return path unchanged
		return path
	}

	// Use a placeholder to protect escaped sequences during expansion
	const escapedTildePlaceholder = "\x00ESCAPED_TILDE\x00"
	const escapedDollarPlaceholder = "\x00ESCAPED_DOLLAR\x00"

	// Step 1: Replace escaped sequences with placeholders
	path = strings.ReplaceAll(path, `\~`, escapedTildePlaceholder)
	path = strings.ReplaceAll(path, `\$`, escapedDollarPlaceholder)

	// Step 2: Expand ~ at the start of the path
	if strings.HasPrefix(path, "~/") || path == "~" {
		if path == "~" {
			path = homeDir
		} else {
			path = filepath.Join(homeDir, path[2:])
		}
	}

	// Step 3: Expand $HOME anywhere in the path
	path = strings.ReplaceAll(path, "$HOME", homeDir)

	// Step 4: Restore escaped sequences (unescape them)
	path = strings.ReplaceAll(path, escapedTildePlaceholder, "~")
	path = strings.ReplaceAll(path, escapedDollarPlaceholder, "$")

	return path
}

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

// SyncConfig represents global sync settings that apply to ALL remote backends.
// When enabled, each remote backend (nextcloud, todoist, etc.) gets its own
// automatic cache database (e.g., ~/.local/share/gosynctasks/caches/nextcloud.db).
// Remote backends can opt-out by setting sync: false in their backend config.
type SyncConfig struct {
	Enabled            bool   `yaml:"enabled"`                       // Enable automatic caching for all remote backends
	LocalBackend       string `yaml:"local_backend,omitempty"`       // Type of cache backend: "sqlite" (default), "file", "git"
	ConflictResolution string `yaml:"conflict_resolution,omitempty"` // Conflict strategy: server_wins (default), local_wins, merge, keep_both
	AutoSync           bool   `yaml:"auto_sync,omitempty"`           // Auto-sync after write operations
	SyncInterval       int    `yaml:"sync_interval,omitempty"`       // Minutes between syncs (default: 5, 0=manual only)
	OfflineMode        string `yaml:"offline_mode,omitempty"`        // Offline mode: auto (default), online, offline
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
			// Accept either:
			// - Full URL with credentials (legacy)
			// - URL without credentials + username (keyring/env)
			// - Host + username (keyring/env)
			// - Neither (all from env vars)
			hasURL := backendConfig.URL != ""
			hasHost := backendConfig.Host != ""

			if !hasURL && !hasHost {
				// Must have username for env var lookup
				if backendConfig.Username == "" {
					return fmt.Errorf("backend %q: URL, host, or username is required for %s backend", name, backendConfig.Type)
				}
			}
		case "git":
			if backendConfig.File == "" {
				// Use default
				backendConfig.File = "TODO.md"
			}
		case "sqlite":
			// db_path is optional - empty string means use XDG default
			// No validation needed
		}

		// Per-backend sync validation is minimal now - just a boolean opt-out
		// No additional validation needed
	}

	// Validate default backend exists and is enabled
	if c.DefaultBackend != "" {
		backend, exists := c.Backends[c.DefaultBackend]
		if !exists {
			// Auto-fix: If default_backend ends with "-cache" (old architecture), strip it
			if strings.HasSuffix(c.DefaultBackend, "-cache") {
				baseBackend := strings.TrimSuffix(c.DefaultBackend, "-cache")
				if _, baseExists := c.Backends[baseBackend]; baseExists {
					utils.Warnf("Auto-fixing config: changing default_backend from %q to %q (cache backends are now automatic)", c.DefaultBackend, baseBackend)
					c.DefaultBackend = baseBackend
					backend = c.Backends[c.DefaultBackend]
					exists = true
				}
			}

			if !exists {
				return fmt.Errorf("default backend %q not found in configured backends", c.DefaultBackend)
			}
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

	// Validate global sync configuration
	if c.Sync != nil && c.Sync.Enabled {
		// Validate local backend type
		if c.Sync.LocalBackend == "" {
			c.Sync.LocalBackend = "sqlite" // Default to sqlite
		}

		validLocalBackends := map[string]bool{
			"sqlite": true,
			"file":   true,
			"git":    true,
		}
		if !validLocalBackends[c.Sync.LocalBackend] {
			return fmt.Errorf("sync.local_backend must be sqlite, file, or git, got %q", c.Sync.LocalBackend)
		}

		// Validate conflict resolution
		if c.Sync.ConflictResolution != "" {
			validStrategies := map[string]bool{
				"server_wins": true,
				"local_wins":  true,
				"merge":       true,
				"keep_both":   true,
			}
			if !validStrategies[c.Sync.ConflictResolution] {
				return fmt.Errorf("sync.conflict_resolution must be server_wins, local_wins, merge, or keep_both, got %q", c.Sync.ConflictResolution)
			}
		} else {
			c.Sync.ConflictResolution = "server_wins" // Default
		}

		// Validate offline mode
		if c.Sync.OfflineMode != "" {
			validModes := map[string]bool{
				"auto":    true,
				"online":  true,
				"offline": true,
			}
			if !validModes[c.Sync.OfflineMode] {
				return fmt.Errorf("sync.offline_mode must be auto, online, or offline, got %q", c.Sync.OfflineMode)
			}
		} else {
			c.Sync.OfflineMode = "auto" // Default
		}

		// Validate sync interval
		if c.Sync.SyncInterval < 0 {
			return fmt.Errorf("sync.sync_interval cannot be negative")
		}
		if c.Sync.SyncInterval == 0 {
			c.Sync.SyncInterval = 5 // Default to 5 minutes
		}
	}

	return nil
}

// GetSyncEnabledBackends returns all backends that have sync enabled
// GetBackendsToBeCached returns all backends that should be cached based on global sync settings.
// This includes:
//   - Remote backends (nextcloud, todoist) when global sync is enabled, unless they opt-out
//   - Local backends (sqlite, file, git) that explicitly opt-in with sync: true
func (c *Config) GetBackendsToBeCached() map[string]backend.BackendConfig {
	cached := make(map[string]backend.BackendConfig)

	globalSyncEnabled := c.Sync != nil && c.Sync.Enabled

	for name, backendConfig := range c.Backends {
		if !backendConfig.Enabled {
			continue
		}

		if backendConfig.ShouldBeCached(globalSyncEnabled) {
			cached[name] = backendConfig
		}
	}

	return cached
}

// GetCacheDatabasePath returns the shared cache database path for all remote backends.
// Format: ~/.local/share/gosynctasks/cache.db
// All backends are stored in the same database with backend_name column for separation.
func (c *Config) GetCacheDatabasePath() (string, error) {
	// Get XDG data dir or default
	var dataDir string
	if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
		dataDir = xdgDataHome
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		dataDir = filepath.Join(homeDir, ".local", "share")
	}

	// Single shared cache database for all backends
	cachePath := filepath.Join(dataDir, "gosynctasks", "cache.db")
	return cachePath, nil
}

// GetSyncPairs returns all sync pairs for remote backends that should be cached.
// Each remote backend (nextcloud, todoist, etc.) gets its own automatic cache database.
func (c *Config) GetSyncPairs() []SyncPair {
	var pairs []SyncPair

	globalSyncEnabled := c.Sync != nil && c.Sync.Enabled
	if !globalSyncEnabled {
		return pairs
	}

	for name, backendConfig := range c.Backends {
		if !backendConfig.Enabled {
			continue
		}

		// Only cache remote backends
		if !backendConfig.ShouldBeCached(globalSyncEnabled) {
			continue
		}

		pairs = append(pairs, SyncPair{
			RemoteBackendName:  name,                          // The remote backend to cache
			CacheDatabasePath:  "",                            // Will be set by GetCacheDatabasePath
			ConflictResolution: c.Sync.ConflictResolution,     // From global config
			AutoSync:           c.Sync.AutoSync,               // From global config
			SyncInterval:       c.Sync.SyncInterval,           // From global config
			OfflineMode:        c.Sync.OfflineMode,            // From global config
		})
	}

	return pairs
}

// SyncPair represents an automatic cache for a remote backend.
// Each remote backend gets its own SQLite cache database.
type SyncPair struct {
	RemoteBackendName  string // Name of the remote backend to cache (e.g., "nextcloud", "todoist")
	CacheDatabasePath  string // Path to the cache database (e.g., ~/.local/share/gosynctasks/caches/nextcloud.db)
	ConflictResolution string // Conflict resolution strategy
	AutoSync           bool   // Whether to auto-sync after write operations
	SyncInterval       int    // Minutes between syncs
	OfflineMode        string // Offline mode: auto, online, offline
}

func (c *Config) GetDateFormat() string {
	if c.DateFormat == "" {
		return "2006-01-02" // Default to yyyy-mm-dd
	}
	return c.DateFormat
}

// expandAllPaths expands ~ and $HOME in all path fields throughout the config
func (c *Config) expandAllPaths() {
	// Expand paths in each backend config
	for name, backendCfg := range c.Backends {
		// Expand DBPath (sqlite)
		if backendCfg.DBPath != "" {
			backendCfg.DBPath = expandPath(backendCfg.DBPath)
		}

		// Expand File (git)
		if backendCfg.File != "" {
			backendCfg.File = expandPath(backendCfg.File)
		}

		// Expand FallbackFiles (git)
		if len(backendCfg.FallbackFiles) > 0 {
			for i, file := range backendCfg.FallbackFiles {
				backendCfg.FallbackFiles[i] = expandPath(file)
			}
		}

		// Expand URL if it looks like a file path (file:// scheme)
		if backendCfg.URL != "" && strings.HasPrefix(backendCfg.URL, "file://") {
			// Extract path part after file://
			pathPart := strings.TrimPrefix(backendCfg.URL, "file://")
			expandedPath := expandPath(pathPart)
			backendCfg.URL = "file://" + expandedPath
		}

		// Update the backend config in the map
		c.Backends[name] = backendCfg
	}
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

// migrateGlobalSyncConfig is no longer needed - global sync config is the primary approach now.
// This function is kept as a placeholder for potential future migrations.
func (c *Config) migrateGlobalSyncConfig() {
	// No migration needed - global sync config is the standard approach
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

	// Copy built-in views to user config on first run
	copied, err := views.CopyBuiltInViewsToUserConfig()
	if err != nil {
		log.Printf("Warning: Failed to copy built-in views: %v", err)
	} else if copied {
		fmt.Println("Built-in views copied to user config directory")
	}

	return configData
}

func parseConfig(configData []byte, configPath string) (*Config, error) {
	var configObj Config
	err := yaml.Unmarshal(configData, &configObj)

	if err != nil {
		log.Fatalf("Invalid YAML in config file %s: %v", configPath, err)
	}

	// Set backend names from map keys
	for name, backendConfig := range configObj.Backends {
		backendConfig.Name = name
		configObj.Backends[name] = backendConfig
	}

	// Expand ~ and $HOME in all path fields
	configObj.expandAllPaths()

	// Migrate old global sync config to per-backend sync (if needed)
	configObj.migrateGlobalSyncConfig()

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
