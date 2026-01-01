package utils

import (
	"errors"
	"strings"
	"testing"
)

func TestErrorWithSuggestion_Error(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		suggestion     string
		wantContains   []string
		wantNotContain string
	}{
		{
			name:         "with suggestion",
			err:          errors.New("task not found"),
			suggestion:   "Try searching with a different term",
			wantContains: []string{"task not found", "Suggestion:", "Try searching"},
		},
		{
			name:           "without suggestion",
			err:            errors.New("simple error"),
			suggestion:     "",
			wantContains:   []string{"simple error"},
			wantNotContain: "Suggestion:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &ErrorWithSuggestion{
				Err:        tt.err,
				Suggestion: tt.suggestion,
			}

			result := e.Error()

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("Error() = %q, want to contain %q", result, want)
				}
			}

			if tt.wantNotContain != "" && strings.Contains(result, tt.wantNotContain) {
				t.Errorf("Error() = %q, should not contain %q", result, tt.wantNotContain)
			}
		})
	}
}

func TestErrorWithSuggestion_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	wrapped := &ErrorWithSuggestion{
		Err:        originalErr,
		Suggestion: "do something",
	}

	unwrapped := wrapped.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("Unwrap() returned %v, want %v", unwrapped, originalErr)
	}

	// Test with errors.Is
	if !errors.Is(wrapped, originalErr) {
		t.Error("errors.Is should work with wrapped error")
	}
}

func TestErrTaskNotFound(t *testing.T) {
	err := ErrTaskNotFound("my task")

	errStr := err.Error()
	if !strings.Contains(errStr, "my task") {
		t.Errorf("Error should contain search term 'my task', got: %s", errStr)
	}
	if !strings.Contains(errStr, "Suggestion:") {
		t.Errorf("Error should contain suggestion, got: %s", errStr)
	}
	if !strings.Contains(errStr, "gosynctasks") {
		t.Errorf("Error should suggest gosynctasks command, got: %s", errStr)
	}
}

func TestErrListNotFound(t *testing.T) {
	err := ErrListNotFound("Work")

	errStr := err.Error()
	if !strings.Contains(errStr, "Work") {
		t.Errorf("Error should contain list name 'Work', got: %s", errStr)
	}
	if !strings.Contains(errStr, "gosynctasks list") {
		t.Errorf("Error should suggest 'gosynctasks list', got: %s", errStr)
	}
}

func TestErrNoListsAvailable(t *testing.T) {
	err := ErrNoListsAvailable()

	errStr := err.Error()
	if !strings.Contains(errStr, "no task lists") {
		t.Errorf("Error should mention no lists, got: %s", errStr)
	}
	if !strings.Contains(errStr, "list create") {
		t.Errorf("Error should suggest creating a list, got: %s", errStr)
	}
}

func TestErrSyncNotEnabled(t *testing.T) {
	err := ErrSyncNotEnabled()

	errStr := err.Error()
	if !strings.Contains(errStr, "sync is not enabled") {
		t.Errorf("Error should mention sync not enabled, got: %s", errStr)
	}
	if !strings.Contains(errStr, "config.yaml") {
		t.Errorf("Error should mention config file, got: %s", errStr)
	}
}

func TestErrBackendNotConfigured(t *testing.T) {
	err := ErrBackendNotConfigured("nextcloud")

	errStr := err.Error()
	if !strings.Contains(errStr, "nextcloud") {
		t.Errorf("Error should contain backend name 'nextcloud', got: %s", errStr)
	}
	if !strings.Contains(errStr, "backends.nextcloud") {
		t.Errorf("Error should suggest config location, got: %s", errStr)
	}
}

func TestErrBackendOffline(t *testing.T) {
	tests := []struct {
		name           string
		backend        string
		reason         string
		wantSuggestion string
	}{
		{
			name:           "DNS error",
			backend:        "nextcloud",
			reason:         "DNS resolution failed",
			wantSuggestion: "DNS settings",
		},
		{
			name:           "Connection refused",
			backend:        "nextcloud",
			reason:         "connection refused",
			wantSuggestion: "server is running",
		},
		{
			name:           "Timeout",
			backend:        "nextcloud",
			reason:         "connection timeout",
			wantSuggestion: "slow or unreachable",
		},
		{
			name:           "Generic error",
			backend:        "nextcloud",
			reason:         "unknown error",
			wantSuggestion: "internet connection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ErrBackendOffline(tt.backend, tt.reason)

			errStr := err.Error()
			if !strings.Contains(errStr, tt.backend) {
				t.Errorf("Error should contain backend name, got: %s", errStr)
			}
			if !strings.Contains(errStr, tt.reason) {
				t.Errorf("Error should contain reason, got: %s", errStr)
			}
			if !strings.Contains(errStr, tt.wantSuggestion) {
				t.Errorf("Error should contain suggestion about '%s', got: %s", tt.wantSuggestion, errStr)
			}
		})
	}
}

func TestErrInvalidPriority(t *testing.T) {
	err := ErrInvalidPriority(15)

	errStr := err.Error()
	if !strings.Contains(errStr, "15") {
		t.Errorf("Error should contain invalid value '15', got: %s", errStr)
	}
	if !strings.Contains(errStr, "0") || !strings.Contains(errStr, "9") {
		t.Errorf("Error should mention valid range 0-9, got: %s", errStr)
	}
}

func TestErrInvalidDate(t *testing.T) {
	err := ErrInvalidDate("01/15/2026")

	errStr := err.Error()
	if !strings.Contains(errStr, "01/15/2026") {
		t.Errorf("Error should contain invalid date, got: %s", errStr)
	}
	if !strings.Contains(errStr, "YYYY-MM-DD") {
		t.Errorf("Error should suggest correct format, got: %s", errStr)
	}
}

func TestErrInvalidStatus(t *testing.T) {
	validStatuses := []string{"TODO", "DONE", "PROCESSING"}
	err := ErrInvalidStatus("INVALID", validStatuses)

	errStr := err.Error()
	if !strings.Contains(errStr, "INVALID") {
		t.Errorf("Error should contain invalid status, got: %s", errStr)
	}
	for _, status := range validStatuses {
		if !strings.Contains(errStr, status) {
			t.Errorf("Error should list valid status '%s', got: %s", status, errStr)
		}
	}
}

func TestErrCredentialsNotFound(t *testing.T) {
	err := ErrCredentialsNotFound("nextcloud", "user@example.com")

	errStr := err.Error()
	if !strings.Contains(errStr, "nextcloud") {
		t.Errorf("Error should contain backend name, got: %s", errStr)
	}
	if !strings.Contains(errStr, "user@example.com") {
		t.Errorf("Error should contain username, got: %s", errStr)
	}
	if !strings.Contains(errStr, "credentials set") {
		t.Errorf("Error should suggest storing credentials, got: %s", errStr)
	}
}

func TestErrAuthenticationFailed(t *testing.T) {
	err := ErrAuthenticationFailed("nextcloud")

	errStr := err.Error()
	if !strings.Contains(errStr, "authentication failed") {
		t.Errorf("Error should mention authentication failure, got: %s", errStr)
	}
	if !strings.Contains(errStr, "credentials get") {
		t.Errorf("Error should suggest checking credentials, got: %s", errStr)
	}
}

func TestWrapWithSuggestion(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		suggestion string
		wantNil    bool
	}{
		{
			name:       "wrap error",
			err:        errors.New("original error"),
			suggestion: "try this instead",
			wantNil:    false,
		},
		{
			name:       "wrap nil",
			err:        nil,
			suggestion: "this should not appear",
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapWithSuggestion(tt.err, tt.suggestion)

			if tt.wantNil {
				if result != nil {
					t.Errorf("WrapWithSuggestion(nil, _) should return nil, got %v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("WrapWithSuggestion() returned nil for non-nil error")
			}

			errStr := result.Error()
			if !strings.Contains(errStr, "original error") {
				t.Errorf("Wrapped error should contain original message, got: %s", errStr)
			}
			if !strings.Contains(errStr, tt.suggestion) {
				t.Errorf("Wrapped error should contain suggestion, got: %s", errStr)
			}
		})
	}
}
