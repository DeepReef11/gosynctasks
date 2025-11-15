package backend

import (
	"testing"
)

// MockBackend is a mock implementation of TaskManager for testing
type MockBackend struct {
	name string
}

func (m *MockBackend) GetTaskLists() ([]TaskList, error)                     { return nil, nil }
func (m *MockBackend) GetTasks(string, *TaskFilter) ([]Task, error)          { return nil, nil }
func (m *MockBackend) FindTasksBySummary(string, string) ([]Task, error)     { return nil, nil }
func (m *MockBackend) AddTask(string, Task) error                            { return nil }
func (m *MockBackend) UpdateTask(string, Task) error                         { return nil }
func (m *MockBackend) DeleteTask(string, string) error                       { return nil }
func (m *MockBackend) CreateTaskList(string, string, string) (string, error) { return "", nil }
func (m *MockBackend) DeleteTaskList(string) error                           { return nil }
func (m *MockBackend) RenameTaskList(string, string) error                   { return nil }
func (m *MockBackend) GetDeletedTaskLists() ([]TaskList, error)              { return nil, nil }
func (m *MockBackend) RestoreTaskList(string) error                          { return nil }
func (m *MockBackend) PermanentlyDeleteTaskList(string) error                { return nil }
func (m *MockBackend) ParseStatusFlag(string) (string, error)                { return "", nil }
func (m *MockBackend) StatusToDisplayName(string) string                     { return "" }
func (m *MockBackend) SortTasks([]Task)                                      {}
func (m *MockBackend) GetPriorityColor(int) string                           { return "" }

// MockDetectableBackend is a mock that implements DetectableBackend
type MockDetectableBackend struct {
	MockBackend
	canDetect     bool
	detectionInfo string
}

func (m *MockDetectableBackend) CanDetect() (bool, error) {
	return m.canDetect, nil
}

func (m *MockDetectableBackend) DetectionInfo() string {
	return m.detectionInfo
}

// TestNewBackendRegistry tests creating a backend registry
func TestNewBackendRegistry(t *testing.T) {
	configs := map[string]BackendConfig{
		"nextcloud": {
			Type:    "nextcloud",
			Enabled: true,
			URL:     "nextcloud://example.com",
		},
		"disabled": {
			Type:    "git",
			Enabled: false,
			File:    "TODO.md",
		},
	}

	registry, err := NewBackendRegistry(configs)
	if err != nil {
		t.Fatalf("NewBackendRegistry() error = %v", err)
	}

	// Should have enabled backends initialized (nextcloud will fail but be skipped)
	// Disabled backends should not be in the registry
	if registry == nil {
		t.Fatal("registry should not be nil")
	}

	// Try to get disabled backend (should fail)
	_, err = registry.GetBackend("disabled")
	if err == nil {
		t.Error("GetBackend() should fail for disabled backend")
	}
}

// TestBackendRegistryGetBackend tests getting backends from registry
func TestBackendRegistryGetBackend(t *testing.T) {
	registry := &BackendRegistry{
		backends: map[string]TaskManager{
			"mock": &MockBackend{name: "mock"},
		},
		configs: map[string]BackendConfig{
			"mock": {Type: "mock", Enabled: true},
		},
	}

	tests := []struct {
		name        string
		backendName string
		wantErr     bool
	}{
		{
			name:        "existing backend",
			backendName: "mock",
			wantErr:     false,
		},
		{
			name:        "non-existing backend",
			backendName: "nonexistent",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := registry.GetBackend(tt.backendName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBackend() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestBackendRegistryListBackends tests listing all backends
func TestBackendRegistryListBackends(t *testing.T) {
	registry := &BackendRegistry{
		backends: map[string]TaskManager{
			"mock1": &MockBackend{name: "mock1"},
		},
		configs: map[string]BackendConfig{
			"mock1": {Type: "mock", Enabled: true},
			"mock2": {Type: "mock", Enabled: false},
		},
	}

	infos := registry.ListBackends()
	if len(infos) != 2 {
		t.Errorf("ListBackends() returned %d backends, want 2", len(infos))
	}

	// Check that we have info for both backends
	var foundMock1, foundMock2 bool
	for _, info := range infos {
		if info.Name == "mock1" {
			foundMock1 = true
			if !info.Ready {
				t.Error("mock1 should be ready")
			}
		}
		if info.Name == "mock2" {
			foundMock2 = true
			if info.Ready {
				t.Error("mock2 should not be ready (disabled)")
			}
		}
	}

	if !foundMock1 || !foundMock2 {
		t.Error("ListBackends() should return info for all configured backends")
	}
}

// TestBackendSelector tests the backend selector
func TestBackendSelector(t *testing.T) {
	tests := []struct {
		name           string
		registry       *BackendRegistry
		explicit       string
		autoDetect     bool
		defaultBackend string
		priority       []string
		wantName       string
		wantErr        bool
	}{
		{
			name: "explicit backend selection",
			registry: &BackendRegistry{
				backends: map[string]TaskManager{
					"explicit": &MockBackend{},
				},
				configs: map[string]BackendConfig{
					"explicit": {Type: "mock", Enabled: true},
				},
			},
			explicit: "explicit",
			wantName: "explicit",
			wantErr:  false,
		},
		{
			name: "explicit backend not found",
			registry: &BackendRegistry{
				backends: map[string]TaskManager{},
				configs:  map[string]BackendConfig{},
			},
			explicit: "nonexistent",
			wantErr:  true,
		},
		{
			name: "auto-detect backend",
			registry: &BackendRegistry{
				backends: map[string]TaskManager{
					"detectable": &MockDetectableBackend{
						MockBackend:   MockBackend{},
						canDetect:     true,
						detectionInfo: "Detected",
					},
				},
				configs: map[string]BackendConfig{
					"detectable": {Type: "mock", Enabled: true},
				},
			},
			autoDetect: true,
			priority:   []string{"detectable"},
			wantName:   "detectable",
			wantErr:    false,
		},
		{
			name: "default backend selection",
			registry: &BackendRegistry{
				backends: map[string]TaskManager{
					"default": &MockBackend{},
				},
				configs: map[string]BackendConfig{
					"default": {Type: "mock", Enabled: true},
				},
			},
			defaultBackend: "default",
			wantName:       "default",
			wantErr:        false,
		},
		{
			name: "first enabled backend",
			registry: &BackendRegistry{
				backends: map[string]TaskManager{
					"first": &MockBackend{},
				},
				configs: map[string]BackendConfig{
					"first": {Type: "mock", Enabled: true},
				},
			},
			wantName: "first",
			wantErr:  false,
		},
		{
			name: "no backends available",
			registry: &BackendRegistry{
				backends: map[string]TaskManager{},
				configs:  map[string]BackendConfig{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := NewBackendSelector(tt.registry)
			name, _, err := selector.Select(
				tt.explicit,
				tt.autoDetect,
				tt.defaultBackend,
				tt.priority,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("Select() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && name != tt.wantName {
				t.Errorf("Select() name = %s, want %s", name, tt.wantName)
			}
		})
	}
}

// TestBackendSelectorDetectAll tests detecting all backends
func TestBackendSelectorDetectAll(t *testing.T) {
	registry := &BackendRegistry{
		backends: map[string]TaskManager{
			"detectable": &MockDetectableBackend{
				MockBackend:   MockBackend{},
				canDetect:     true,
				detectionInfo: "Mock detection info",
			},
			"not-detectable": &MockBackend{},
		},
		configs: map[string]BackendConfig{
			"detectable":     {Type: "mock", Enabled: true},
			"not-detectable": {Type: "mock", Enabled: true},
		},
	}

	selector := NewBackendSelector(registry)
	detected := selector.DetectAll()

	// Should only return the detectable backend
	if len(detected) != 1 {
		t.Errorf("DetectAll() returned %d backends, want 1", len(detected))
	}

	if len(detected) > 0 && detected[0].Name != "detectable" {
		t.Errorf("DetectAll() returned %s, want detectable", detected[0].Name)
	}
}

// TestBackendInfoString tests the String() method of BackendInfo
func TestBackendInfoString(t *testing.T) {
	info := BackendInfo{
		Name:          "test",
		Type:          "mock",
		Enabled:       true,
		Ready:         true,
		Detected:      true,
		DetectionInfo: "Test detection",
	}

	str := info.String()
	if str == "" {
		t.Error("String() should not return empty string")
	}

	// Should contain key information
	if !contains(str, "test") || !contains(str, "mock") {
		t.Error("String() should contain name and type")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}
