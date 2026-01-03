package backend

import (
	"fmt"
	"strings"
)

// BackendRegistry manages configured backends and provides access to them.
// It stores backend instances keyed by their configuration name.
type BackendRegistry struct {
	backends map[string]TaskManager
	configs  map[string]BackendConfig
}

// NewBackendRegistry creates a new BackendRegistry from a map of backend configurations.
func NewBackendRegistry(configs map[string]BackendConfig) (*BackendRegistry, error) {
	registry := &BackendRegistry{
		backends: make(map[string]TaskManager),
		configs:  configs,
	}

	// Initialize all enabled backends
	for name, config := range configs {
		if !config.Enabled {
			continue
		}

		taskManager, err := config.TaskManager()
		if err != nil {
			// Skip backends that can't be initialized (e.g., git backend when not in a git repo)
			continue
		}

		registry.backends[name] = taskManager
	}

	return registry, nil
}

// GetBackend returns the TaskManager for the specified backend name.
// Returns an error if the backend doesn't exist or isn't initialized.
func (r *BackendRegistry) GetBackend(name string) (TaskManager, error) {
	backend, exists := r.backends[name]
	if !exists {
		return nil, fmt.Errorf("backend %q not found or not initialized", name)
	}
	return backend, nil
}

// ListBackends returns information about all configured backends.
func (r *BackendRegistry) ListBackends() []BackendInfo {
	var infos []BackendInfo

	for name, config := range r.configs {
		info := BackendInfo{
			Name:    name,
			Type:    config.Type,
			Enabled: config.Enabled,
			Ready:   false,
		}

		// Check if backend is actually initialized
		if _, exists := r.backends[name]; exists {
			info.Ready = true
		}

		// Add detection info if backend supports it
		if backend, exists := r.backends[name]; exists {
			if detectable, ok := backend.(DetectableBackend); ok {
				if detected, err := detectable.CanDetect(); err == nil && detected {
					info.Detected = true
					info.DetectionInfo = detectable.DetectionInfo()
				}
			}
		}

		infos = append(infos, info)
	}

	return infos
}

// GetEnabledBackends returns a slice of names of all enabled backends.
func (r *BackendRegistry) GetEnabledBackends() []string {
	var enabled []string
	for name, config := range r.configs {
		if config.Enabled {
			if _, exists := r.backends[name]; exists {
				enabled = append(enabled, name)
			}
		}
	}
	return enabled
}

// BackendInfo contains information about a backend for display purposes.
type BackendInfo struct {
	Name          string
	Type          string
	Enabled       bool
	Ready         bool   // Whether the backend is initialized
	Detected      bool   // Whether the backend was auto-detected
	DetectionInfo string // Human-readable detection information
}

// String returns a formatted string representation of the backend info.
func (bi BackendInfo) String() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Name: %s", bi.Name))
	parts = append(parts, fmt.Sprintf("Type: %s", bi.Type))

	var status []string
	if bi.Enabled {
		status = append(status, "enabled")
	} else {
		status = append(status, "disabled")
	}
	if bi.Ready {
		status = append(status, "ready")
	} else {
		status = append(status, "not ready")
	}
	if bi.Detected {
		status = append(status, "detected")
	}

	parts = append(parts, fmt.Sprintf("Status: %s", strings.Join(status, ", ")))

	if bi.DetectionInfo != "" {
		parts = append(parts, fmt.Sprintf("Detection: %s", bi.DetectionInfo))
	}

	return strings.Join(parts, " | ")
}

// BackendSelector implements the backend selection logic based on priority.
type BackendSelector struct {
	registry *BackendRegistry
}

// NewBackendSelector creates a new BackendSelector with the given registry.
func NewBackendSelector(registry *BackendRegistry) *BackendSelector {
	return &BackendSelector{
		registry: registry,
	}
}

// Select chooses a backend based on the selection criteria.
// Selection priority:
// 1. Explicit backend name (if provided)
// 2. Sync local backend (if sync is enabled and no explicit backend)
// 3. Auto-detection (if enabled)
// 4. Default backend
// 5. First enabled backend
func (s *BackendSelector) Select(explicitBackend string, autoDetect bool, defaultBackend string, priority []string, syncEnabled bool, syncLocalBackend string, cachePath string) (string, TaskManager, error) {
	// Priority 1: Explicit backend name
	// If sync is enabled and explicit backend is a remote, use cache for it
	if explicitBackend != "" {
		// Check if sync is enabled and this is a remote backend
		if syncEnabled && syncLocalBackend != "" && cachePath != "" && s.isRemoteBackend(explicitBackend) {
			// Use cache for this explicit remote backend
			cacheBackend, err := s.createCacheBackend(syncLocalBackend, explicitBackend, cachePath)
			if err == nil {
				return syncLocalBackend + "-cache", cacheBackend, nil
			}
			// If cache creation fails, fall through to use explicit backend directly
		}

		// Use explicit backend directly (no sync or cache creation failed)
		backend, err := s.registry.GetBackend(explicitBackend)
		if err != nil {
			return "", nil, fmt.Errorf("explicitly specified backend %q: %w", explicitBackend, err)
		}
		return explicitBackend, backend, nil
	}

	// Priority 2: Sync local backend (when sync is enabled)
	// This ensures that when sync is active, operations use the local cache backend
	// and sync happens in the background to the remote backend
	if syncEnabled && syncLocalBackend != "" {
		// Determine which remote backend to cache
		var remoteBackendName string
		if defaultBackend != "" {
			remoteBackendName = defaultBackend
		} else if len(priority) > 0 {
			remoteBackendName = priority[0]
		}

		if remoteBackendName != "" && cachePath != "" {
			// Create a cache backend for this remote backend
			cacheBackend, err := s.createCacheBackend(syncLocalBackend, remoteBackendName, cachePath)
			if err == nil {
				return syncLocalBackend + "-cache", cacheBackend, nil
			}
		}
		// If sync cache creation fails, fall through to next priority
	}

	// Priority 3: Auto-detection
	if autoDetect {
		name, backend, err := s.autoDetect(priority)
		if err == nil && backend != nil {
			return name, backend, nil
		}
		// If auto-detection fails, fall through to next priority
	}

	// Priority 4: Default backend
	if defaultBackend != "" {
		backend, err := s.registry.GetBackend(defaultBackend)
		if err == nil {
			return defaultBackend, backend, nil
		}
		// If default backend fails, fall through to next priority
	}

	// Priority 5: First enabled backend
	enabled := s.registry.GetEnabledBackends()
	if len(enabled) > 0 {
		name := enabled[0]
		backend, err := s.registry.GetBackend(name)
		if err != nil {
			return "", nil, fmt.Errorf("failed to get first enabled backend: %w", err)
		}
		return name, backend, nil
	}

	return "", nil, fmt.Errorf("no backends available")
}

// autoDetect attempts to detect a usable backend based on the environment.
// It checks backends in priority order and returns the first detected one.
func (s *BackendSelector) autoDetect(priority []string) (string, TaskManager, error) {
	// Try backends in priority order first
	for _, name := range priority {
		backend, err := s.registry.GetBackend(name)
		if err != nil {
			continue
		}

		// Check if backend supports detection
		if detectable, ok := backend.(DetectableBackend); ok {
			detected, err := detectable.CanDetect()
			if err == nil && detected {
				return name, backend, nil
			}
		}
	}

	// If no priority backend detected, try all detectable backends
	for name, backend := range s.registry.backends {
		if detectable, ok := backend.(DetectableBackend); ok {
			detected, err := detectable.CanDetect()
			if err == nil && detected {
				return name, backend, nil
			}
		}
	}

	return "", nil, fmt.Errorf("no backend detected")
}

// DetectAll returns information about all detected backends.
func (s *BackendSelector) DetectAll() []BackendInfo {
	var detected []BackendInfo

	for name, backend := range s.registry.backends {
		if detectable, ok := backend.(DetectableBackend); ok {
			if canDetect, err := detectable.CanDetect(); err == nil && canDetect {
				config := s.registry.configs[name]
				info := BackendInfo{
					Name:          name,
					Type:          config.Type,
					Enabled:       config.Enabled,
					Ready:         true,
					Detected:      true,
					DetectionInfo: detectable.DetectionInfo(),
				}
				detected = append(detected, info)
			}
		}
	}

	return detected
}

// isRemoteBackend checks if a backend is a remote backend (nextcloud, todoist)
func (s *BackendSelector) isRemoteBackend(backendName string) bool {
	config, exists := s.registry.configs[backendName]
	if !exists {
		return false
	}
	remoteTypes := map[string]bool{
		"nextcloud": true,
		"todoist":   true,
	}
	return remoteTypes[config.Type]
}

// createCacheBackend creates a cache backend instance for a remote backend
func (s *BackendSelector) createCacheBackend(cacheType string, remoteBackendName string, cachePath string) (TaskManager, error) {
	if cacheType != "sqlite" {
		return nil, fmt.Errorf("only sqlite cache backend is currently supported, got %s", cacheType)
	}

	// Create cache backend configuration
	// Important: Name must match remote backend name for backend_name filtering
	cacheConfig := BackendConfig{
		Name:    remoteBackendName, // Use remote backend name for backend_name column
		Type:    "sqlite",
		Enabled: true,
		DBPath:  cachePath, // Shared database for all backends
	}

	// Create cache backend using TaskManager() method which calls registered constructor
	cacheBackend, err := cacheConfig.TaskManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create cache backend: %w", err)
	}

	return cacheBackend, nil
}
