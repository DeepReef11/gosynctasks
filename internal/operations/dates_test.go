package operations

import (
	"gosynctasks/internal/utils"
	"strings"
	"testing"
	"time"
)

func TestParseDateFlag(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectNil bool
		expectErr bool
		expected  string // Expected date in YYYY-MM-DD format
	}{
		{
			name:      "valid ISO date",
			input:     "2025-01-31",
			expectNil: false,
			expectErr: false,
			expected:  "2025-01-31",
		},
		{
			name:      "valid date with leading zeros",
			input:     "2025-01-05",
			expectNil: false,
			expectErr: false,
			expected:  "2025-01-05",
		},
		{
			name:      "empty string returns nil",
			input:     "",
			expectNil: true,
			expectErr: false,
		},
		{
			name:      "invalid format - missing day",
			input:     "2025-01",
			expectNil: false,
			expectErr: true,
		},
		{
			name:      "invalid format - wrong separator",
			input:     "2025/01/31",
			expectNil: false,
			expectErr: true,
		},
		{
			name:      "invalid format - no separators",
			input:     "20250131",
			expectNil: false,
			expectErr: true,
		},
		{
			name:      "invalid date - month out of range",
			input:     "2025-13-01",
			expectNil: false,
			expectErr: true,
		},
		{
			name:      "invalid date - day out of range",
			input:     "2025-02-30",
			expectNil: false,
			expectErr: true,
		},
		{
			name:      "invalid format - text",
			input:     "tomorrow",
			expectNil: false,
			expectErr: true,
		},
		{
			name:      "leap year date",
			input:     "2024-02-29",
			expectNil: false,
			expectErr: false,
			expected:  "2024-02-29",
		},
		{
			name:      "non-leap year Feb 29",
			input:     "2025-02-29",
			expectNil: false,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := utils.ParseDateFlag(tt.input)

			// Check error expectation
			if tt.expectErr && err == nil {
				t.Errorf("utils.ParseDateFlag(%q) expected error, got nil", tt.input)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("utils.ParseDateFlag(%q) unexpected error: %v", tt.input, err)
			}

			// Check nil expectation
			if tt.expectNil && result != nil {
				t.Errorf("utils.ParseDateFlag(%q) expected nil, got %v", tt.input, result)
			}
			if !tt.expectNil && !tt.expectErr && result == nil {
				t.Errorf("utils.ParseDateFlag(%q) expected non-nil result, got nil", tt.input)
			}

			// Check date value
			if !tt.expectNil && !tt.expectErr && result != nil {
				actual := result.Format("2006-01-02")
				if actual != tt.expected {
					t.Errorf("utils.ParseDateFlag(%q) = %s, want %s", tt.input, actual, tt.expected)
				}
			}

			// Check error message format
			if tt.expectErr && err != nil {
				errMsg := err.Error()
				if !strings.Contains(errMsg, "invalid date format") {
					t.Errorf("utils.ParseDateFlag(%q) error should mention 'invalid date format', got: %v", tt.input, err)
				}
				if !strings.Contains(errMsg, "YYYY-MM-DD") {
					t.Errorf("utils.ParseDateFlag(%q) error should mention expected format 'YYYY-MM-DD', got: %v", tt.input, err)
				}
			}
		})
	}
}

func TestValidateDates(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)
	nextWeek := today.AddDate(0, 0, 7)
	yesterday := today.AddDate(0, 0, -1)

	tests := []struct {
		name      string
		startDate *time.Time
		dueDate   *time.Time
		expectErr bool
		errMsg    string
	}{
		{
			name:      "both nil - valid",
			startDate: nil,
			dueDate:   nil,
			expectErr: false,
		},
		{
			name:      "only start date - valid",
			startDate: &today,
			dueDate:   nil,
			expectErr: false,
		},
		{
			name:      "only due date - valid",
			startDate: nil,
			dueDate:   &nextWeek,
			expectErr: false,
		},
		{
			name:      "start before due - valid",
			startDate: &today,
			dueDate:   &nextWeek,
			expectErr: false,
		},
		{
			name:      "start equals due - valid",
			startDate: &today,
			dueDate:   &today,
			expectErr: false,
		},
		{
			name:      "start after due - invalid",
			startDate: &nextWeek,
			dueDate:   &today,
			expectErr: true,
			errMsg:    "start date",
		},
		{
			name:      "start tomorrow, due yesterday - invalid",
			startDate: &tomorrow,
			dueDate:   &yesterday,
			expectErr: true,
			errMsg:    "cannot be after",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := utils.ValidateDates(tt.startDate, tt.dueDate)

			if tt.expectErr && err == nil {
				t.Errorf("utils.ValidateDates() expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("utils.ValidateDates() unexpected error: %v", err)
			}

			// Check error message contains expected substring
			if tt.expectErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("utils.ValidateDates() error should contain %q, got: %v", tt.errMsg, err)
				}
			}
		})
	}
}

func TestValidateDates_ErrorMessageFormat(t *testing.T) {
	// Test that error message includes both dates in readable format
	start := time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC)
	due := time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC)

	err := utils.ValidateDates(&start, &due)

	if err == nil {
		t.Fatal("Expected error for start > due")
	}

	errMsg := err.Error()

	// Should contain both dates
	if !strings.Contains(errMsg, "2025-02-15") {
		t.Errorf("Error message should contain start date '2025-02-15', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "2025-02-10") {
		t.Errorf("Error message should contain due date '2025-02-10', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "cannot be after") {
		t.Errorf("Error message should explain the issue, got: %s", errMsg)
	}
}
