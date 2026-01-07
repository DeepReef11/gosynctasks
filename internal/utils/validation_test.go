package utils

import (
	"testing"
	"time"
)

func TestValidatePriority(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		wantErr  bool
	}{
		{"valid priority 0", 0, false},
		{"valid priority 1", 1, false},
		{"valid priority 5", 5, false},
		{"valid priority 9", 9, false},
		{"invalid priority -1", -1, true},
		{"invalid priority 10", 10, true},
		{"invalid priority 100", 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePriority(tt.priority)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePriority(%d) error = %v, wantErr %v", tt.priority, err, tt.wantErr)
			}
		})
	}
}

func TestParseDateFlag(t *testing.T) {
	tests := []struct {
		name     string
		dateFlag string
		wantDate *time.Time
		wantErr  bool
	}{
		{
			name:     "empty string returns nil",
			dateFlag: "",
			wantDate: nil,
			wantErr:  false,
		},
		{
			name:     "valid ISO date",
			dateFlag: "2026-01-15",
			wantDate: ptrTime(time.Date(2026, 1, 15, 0, 0, 0, 0, time.Local)),
			wantErr:  false,
		},
		{
			name:     "another valid date",
			dateFlag: "2025-12-31",
			wantDate: ptrTime(time.Date(2025, 12, 31, 0, 0, 0, 0, time.Local)),
			wantErr:  false,
		},
		{
			name:     "invalid format - text",
			dateFlag: "not-a-date",
			wantErr:  true,
		},
		{
			name:     "invalid format - wrong separator",
			dateFlag: "2026/01/15",
			wantErr:  true,
		},
		{
			name:     "invalid month",
			dateFlag: "2026-13-15",
			wantErr:  true,
		},
		{
			name:     "invalid day",
			dateFlag: "2026-01-40",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDateFlag(tt.dateFlag)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDateFlag(%q) error = %v, wantErr %v", tt.dateFlag, err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if (result == nil) != (tt.wantDate == nil) {
					t.Errorf("ParseDateFlag(%q) nil mismatch: got %v, want %v", tt.dateFlag, result, tt.wantDate)
					return
				}
				if result != nil && tt.wantDate != nil && !result.Equal(*tt.wantDate) {
					t.Errorf("ParseDateFlag(%q) = %v, want %v", tt.dateFlag, result, tt.wantDate)
				}
			}
		})
	}
}

// ptrTime is a helper to create a time pointer
func ptrTime(t time.Time) *time.Time {
	return &t
}

func TestValidateDates(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)

	tests := []struct {
		name      string
		startDate *time.Time
		dueDate   *time.Time
		wantErr   bool
	}{
		{
			name:      "nil dates",
			startDate: nil,
			dueDate:   nil,
			wantErr:   false,
		},
		{
			name:      "only start date",
			startDate: &now,
			dueDate:   nil,
			wantErr:   false,
		},
		{
			name:      "only due date",
			startDate: nil,
			dueDate:   &now,
			wantErr:   false,
		},
		{
			name:      "valid: due after start",
			startDate: &now,
			dueDate:   &tomorrow,
			wantErr:   false,
		},
		{
			name:      "valid: same date",
			startDate: &now,
			dueDate:   &now,
			wantErr:   false,
		},
		{
			name:      "invalid: due before start",
			startDate: &now,
			dueDate:   &yesterday,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDates(tt.startDate, tt.dueDate)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDates() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
