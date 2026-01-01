package credentials

import (
	"testing"
)

func TestGetServiceName(t *testing.T) {
	tests := []struct {
		name        string
		backendName string
		want        string
	}{
		{
			name:        "simple backend name",
			backendName: "todoist",
			want:        "gosynctasks-todoist",
		},
		{
			name:        "backend with hyphen",
			backendName: "my-backend",
			want:        "gosynctasks-my-backend",
		},
		{
			name:        "nextcloud backend",
			backendName: "nextcloud",
			want:        "gosynctasks-nextcloud",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getServiceName(tt.backendName)
			if got != tt.want {
				t.Errorf("getServiceName(%q) = %q, want %q", tt.backendName, got, tt.want)
			}
		})
	}
}

func TestSet_Validation(t *testing.T) {
	tests := []struct {
		name        string
		backendName string
		username    string
		password    string
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty backend name",
			backendName: "",
			username:    "user",
			password:    "pass",
			wantErr:     true,
			errContains: "backend name cannot be empty",
		},
		{
			name:        "empty username",
			backendName: "todoist",
			username:    "",
			password:    "pass",
			wantErr:     true,
			errContains: "username cannot be empty",
		},
		{
			name:        "empty password",
			backendName: "todoist",
			username:    "user",
			password:    "",
			wantErr:     true,
			errContains: "password cannot be empty",
		},
		{
			name:        "all fields empty",
			backendName: "",
			username:    "",
			password:    "",
			wantErr:     true,
			errContains: "backend name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Set(tt.backendName, tt.username, tt.password)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Set() expected error containing %q, got nil", tt.errContains)
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Set() error = %q, want error containing %q", err.Error(), tt.errContains)
				}
			} else if err != nil {
				t.Errorf("Set() unexpected error = %v", err)
			}
		})
	}
}

func TestGet_Validation(t *testing.T) {
	tests := []struct {
		name        string
		backendName string
		username    string
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty backend name",
			backendName: "",
			username:    "user",
			wantErr:     true,
			errContains: "backend name cannot be empty",
		},
		{
			name:        "empty username",
			backendName: "todoist",
			username:    "",
			wantErr:     true,
			errContains: "username cannot be empty",
		},
		{
			name:        "both empty",
			backendName: "",
			username:    "",
			wantErr:     true,
			errContains: "backend name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Get(tt.backendName, tt.username)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Get() expected error containing %q, got nil", tt.errContains)
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Get() error = %q, want error containing %q", err.Error(), tt.errContains)
				}
			} else if err != nil {
				t.Errorf("Get() unexpected error = %v", err)
			}
		})
	}
}

func TestDelete_Validation(t *testing.T) {
	tests := []struct {
		name        string
		backendName string
		username    string
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty backend name",
			backendName: "",
			username:    "user",
			wantErr:     true,
			errContains: "backend name cannot be empty",
		},
		{
			name:        "empty username",
			backendName: "todoist",
			username:    "",
			wantErr:     true,
			errContains: "username cannot be empty",
		},
		{
			name:        "both empty",
			backendName: "",
			username:    "",
			wantErr:     true,
			errContains: "backend name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Delete(tt.backendName, tt.username)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Delete() expected error containing %q, got nil", tt.errContains)
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Delete() error = %q, want error containing %q", err.Error(), tt.errContains)
				}
			} else if err != nil {
				t.Errorf("Delete() unexpected error = %v", err)
			}
		})
	}
}

func TestIsAvailable(t *testing.T) {
	// This test just verifies the function runs without panicking
	// The actual result depends on the system's keyring availability
	t.Run("runs without error", func(t *testing.T) {
		available := IsAvailable()
		// Just log the result, don't assert on it since it's system-dependent
		t.Logf("Keyring available: %v", available)
	})
}

func TestList(t *testing.T) {
	// List() doesn't take parameters and currently returns an error
	// because go-keyring doesn't support listing entries
	t.Run("not supported", func(t *testing.T) {
		entries, err := List()

		// Should return an error indicating it's not supported
		if err == nil {
			t.Error("List() expected error (not supported), got nil")
		}

		if entries != nil {
			t.Errorf("List() returned %d entries, want nil on error", len(entries))
		}

		if !contains(err.Error(), "not supported") {
			t.Errorf("List() error = %q, want error containing 'not supported'", err.Error())
		}
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
