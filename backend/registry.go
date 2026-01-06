package backend

import (
	"fmt"
	"sync"
)

// BackendConstructor is a function that creates a new backend instance
type BackendConstructor func(config ConnectorConfig) (TaskManager, error)

// BackendConfigConstructor is a function that creates a backend from BackendConfig
type BackendConfigConstructor func(config BackendConfig) (TaskManager, error)

// Registry holds registered backend constructors
type Registry struct {
	mu                     sync.RWMutex
	schemeConstructors     map[string]BackendConstructor
	typeConstructors       map[string]BackendConfigConstructor
	detectableConstructors map[string]BackendConfigConstructor
}

var globalRegistry = &Registry{
	schemeConstructors:     make(map[string]BackendConstructor),
	typeConstructors:       make(map[string]BackendConfigConstructor),
	detectableConstructors: make(map[string]BackendConfigConstructor),
}

// RegisterScheme registers a backend constructor for a URL scheme
func RegisterScheme(scheme string, constructor BackendConstructor) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.schemeConstructors[scheme] = constructor
}

// RegisterType registers a backend constructor for a config type
func RegisterType(backendType string, constructor BackendConfigConstructor) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.typeConstructors[backendType] = constructor
}

// RegisterDetectable registers a backend as detectable with auto-detection capability
func RegisterDetectable(backendType string, constructor BackendConfigConstructor) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.detectableConstructors[backendType] = constructor
	// Also register as a regular type
	globalRegistry.typeConstructors[backendType] = constructor
}

// GetSchemeConstructor returns the constructor for a URL scheme
func GetSchemeConstructor(scheme string) (BackendConstructor, error) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	constructor, ok := globalRegistry.schemeConstructors[scheme]
	if !ok {
		return nil, fmt.Errorf("unsupported URL scheme: %s", scheme)
	}
	return constructor, nil
}

// GetTypeConstructor returns the constructor for a backend type
func GetTypeConstructor(backendType string) (BackendConfigConstructor, error) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	constructor, ok := globalRegistry.typeConstructors[backendType]
	if !ok {
		return nil, fmt.Errorf("unsupported backend type: %s", backendType)
	}
	return constructor, nil
}

// GetDetectableConstructors returns all detectable backend constructors
func GetDetectableConstructors() map[string]BackendConfigConstructor {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	// Return a copy to avoid concurrent access issues
	result := make(map[string]BackendConfigConstructor)
	for k, v := range globalRegistry.detectableConstructors {
		result[k] = v
	}
	return result
}
