package backend

import (
	"testing"
)

func TestStatusStringTranslateToStandardStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    *[]string
		expected *[]string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty slice",
			input:    &[]string{},
			expected: &[]string{},
		},
		{
			name:     "TODO to NEEDS-ACTION",
			input:    &[]string{"TODO"},
			expected: &[]string{"NEEDS-ACTION"},
		},
		{
			name:     "DONE to COMPLETED",
			input:    &[]string{"DONE"},
			expected: &[]string{"COMPLETED"},
		},
		{
			name:     "PROCESSING to IN-PROCESS",
			input:    &[]string{"PROCESSING"},
			expected: &[]string{"IN-PROCESS"},
		},
		{
			name:     "CANCELLED stays CANCELLED",
			input:    &[]string{"CANCELLED"},
			expected: &[]string{"CANCELLED"},
		},
		{
			name:     "lowercase todo to NEEDS-ACTION",
			input:    &[]string{"todo"},
			expected: &[]string{"NEEDS-ACTION"},
		},
		{
			name:     "mixed case Done to COMPLETED",
			input:    &[]string{"Done"},
			expected: &[]string{"COMPLETED"},
		},
		{
			name:     "multiple statuses",
			input:    &[]string{"TODO", "DONE", "PROCESSING"},
			expected: &[]string{"NEEDS-ACTION", "COMPLETED", "IN-PROCESS"},
		},
		{
			name:     "mixed known and unknown statuses",
			input:    &[]string{"TODO", "UNKNOWN", "DONE"},
			expected: &[]string{"NEEDS-ACTION", "UNKNOWN", "COMPLETED"},
		},
		{
			name:     "unknown status passes through",
			input:    &[]string{"CUSTOM-STATUS"},
			expected: &[]string{"CUSTOM-STATUS"},
		},
		{
			name:     "already standard status",
			input:    &[]string{"NEEDS-ACTION"},
			expected: &[]string{"NEEDS-ACTION"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StatusStringTranslateToStandardStatus(tt.input)

			// Check if both are nil
			if result == nil && tt.expected == nil {
				return
			}

			// Check if one is nil and the other isn't
			if (result == nil) != (tt.expected == nil) {
				t.Errorf("StatusStringTranslateToStandardStatus() = %v, want %v", result, tt.expected)
				return
			}

			// Check slice lengths
			if len(*result) != len(*tt.expected) {
				t.Errorf("StatusStringTranslateToStandardStatus() length = %d, want %d", len(*result), len(*tt.expected))
				return
			}

			// Check each element
			for i := range *result {
				if (*result)[i] != (*tt.expected)[i] {
					t.Errorf("StatusStringTranslateToStandardStatus()[%d] = %q, want %q", i, (*result)[i], (*tt.expected)[i])
				}
			}
		})
	}
}

func TestStatusStringTranslateToAppStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    *[]string
		expected *[]string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty slice",
			input:    &[]string{},
			expected: &[]string{},
		},
		{
			name:     "NEEDS-ACTION to TODO",
			input:    &[]string{"NEEDS-ACTION"},
			expected: &[]string{"TODO"},
		},
		{
			name:     "COMPLETED to DONE",
			input:    &[]string{"COMPLETED"},
			expected: &[]string{"DONE"},
		},
		{
			name:     "IN-PROCESS to PROCESSING",
			input:    &[]string{"IN-PROCESS"},
			expected: &[]string{"PROCESSING"},
		},
		{
			name:     "CANCELLED stays CANCELLED",
			input:    &[]string{"CANCELLED"},
			expected: &[]string{"CANCELLED"},
		},
		{
			name:     "lowercase needs-action to TODO",
			input:    &[]string{"needs-action"},
			expected: &[]string{"TODO"},
		},
		{
			name:     "mixed case Completed to DONE",
			input:    &[]string{"Completed"},
			expected: &[]string{"DONE"},
		},
		{
			name:     "multiple statuses",
			input:    &[]string{"NEEDS-ACTION", "COMPLETED", "IN-PROCESS"},
			expected: &[]string{"TODO", "DONE", "PROCESSING"},
		},
		{
			name:     "mixed known and unknown statuses",
			input:    &[]string{"NEEDS-ACTION", "UNKNOWN", "COMPLETED"},
			expected: &[]string{"TODO", "UNKNOWN", "DONE"},
		},
		{
			name:     "unknown status passes through",
			input:    &[]string{"CUSTOM-STATUS"},
			expected: &[]string{"CUSTOM-STATUS"},
		},
		{
			name:     "already app status",
			input:    &[]string{"TODO"},
			expected: &[]string{"TODO"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StatusStringTranslateToAppStatus(tt.input)

			// Check if both are nil
			if result == nil && tt.expected == nil {
				return
			}

			// Check if one is nil and the other isn't
			if (result == nil) != (tt.expected == nil) {
				t.Errorf("StatusStringTranslateToAppStatus() = %v, want %v", result, tt.expected)
				return
			}

			// Check slice lengths
			if len(*result) != len(*tt.expected) {
				t.Errorf("StatusStringTranslateToAppStatus() length = %d, want %d", len(*result), len(*tt.expected))
				return
			}

			// Check each element
			for i := range *result {
				if (*result)[i] != (*tt.expected)[i] {
					t.Errorf("StatusStringTranslateToAppStatus()[%d] = %q, want %q", i, (*result)[i], (*tt.expected)[i])
				}
			}
		})
	}
}

func TestStatusTranslationRoundTrip(t *testing.T) {
	// Test that translating from app -> standard -> app returns the same values
	tests := []struct {
		name  string
		input []string
	}{
		{
			name:  "all app statuses",
			input: []string{"TODO", "DONE", "PROCESSING", "CANCELLED"},
		},
		{
			name:  "mixed case",
			input: []string{"todo", "Done", "PROCESSING"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Normalize to uppercase first
			normalized := make([]string, len(tt.input))
			for i, s := range tt.input {
				normalized[i] = normalizeStatus(s)
			}

			// App -> Standard
			standard := StatusStringTranslateToStandardStatus(&normalized)

			// Standard -> App
			result := StatusStringTranslateToAppStatus(standard)

			// Should match normalized input
			for i := range normalized {
				if (*result)[i] != normalized[i] {
					t.Errorf("Round trip failed: input[%d] = %q, got %q", i, normalized[i], (*result)[i])
				}
			}
		})
	}
}

// Helper function to normalize status (used in round-trip test)
func normalizeStatus(status string) string {
	statusMap := map[string]string{
		"TODO":       "TODO",
		"DONE":       "DONE",
		"PROCESSING": "PROCESSING",
		"CANCELLED":  "CANCELLED",
	}

	// Convert to uppercase and check map
	upper := ""
	for _, c := range status {
		if c >= 'a' && c <= 'z' {
			upper += string(c - 32)
		} else {
			upper += string(c)
		}
	}

	if normalized, ok := statusMap[upper]; ok {
		return normalized
	}
	return status
}
