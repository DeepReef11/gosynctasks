package backend

import (
	"strings"
	"testing"
	"time"
)

func TestFormatWithView_StartDate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		task          Task
		expectedColor string
		description   string
	}{
		{
			name: "past start date (cyan)",
			task: Task{
				UID:       "test-1",
				Summary:   "Task with past start",
				Status:    "NEEDS-ACTION",
				StartDate: timePtr(now.AddDate(0, 0, -5)),
			},
			expectedColor: "\033[36m", // Cyan
			description:   "Start date in the past should be cyan",
		},
		{
			name: "start date within 3 days (yellow)",
			task: Task{
				UID:       "test-2",
				Summary:   "Task starting soon",
				Status:    "NEEDS-ACTION",
				StartDate: timePtr(now.Add(48 * time.Hour)),
			},
			expectedColor: "\033[33m", // Yellow
			description:   "Start date within 3 days should be yellow",
		},
		{
			name: "future start date (gray)",
			task: Task{
				UID:       "test-3",
				Summary:   "Task starting later",
				Status:    "NEEDS-ACTION",
				StartDate: timePtr(now.AddDate(0, 0, 10)),
			},
			expectedColor: "\033[90m", // Gray
			description:   "Future start date (>3 days) should be gray",
		},
		{
			name: "no start date",
			task: Task{
				UID:     "test-4",
				Summary: "Task without start",
				Status:  "NEEDS-ACTION",
			},
			expectedColor: "",
			description:   "No start date should not display anything",
		},
		{
			name: "both start and due dates",
			task: Task{
				UID:       "test-5",
				Summary:   "Task with both dates",
				Status:    "NEEDS-ACTION",
				StartDate: timePtr(now.Add(48 * time.Hour)),
				DueDate:   timePtr(now.AddDate(0, 0, 7)),
			},
			expectedColor: "\033[33m", // Yellow (for start)
			description:   "Task with both dates should show both",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.task.FormatWithView("basic", nil, "2006-01-02")

			// Check if start date appears in output when expected
			if tt.task.StartDate != nil {
				if !strings.Contains(result, "(starts:") {
					t.Errorf("Expected '(starts:' in output, got:\n%s", result)
				}

				// Check for correct color code
				if tt.expectedColor != "" && !strings.Contains(result, tt.expectedColor) {
					t.Errorf("Expected color code %q in output for %s, got:\n%s",
						tt.expectedColor, tt.description, result)
				}
			} else {
				if strings.Contains(result, "(starts:") {
					t.Errorf("Did not expect '(starts:' in output when StartDate is nil, got:\n%s", result)
				}
			}

			// If both dates present, verify both appear
			if tt.task.StartDate != nil && tt.task.DueDate != nil {
				if !strings.Contains(result, "(starts:") || !strings.Contains(result, "(due:") {
					t.Errorf("Expected both start and due dates in output, got:\n%s", result)
				}

				// Verify start appears before due
				startIdx := strings.Index(result, "(starts:")
				dueIdx := strings.Index(result, "(due:")
				if startIdx > dueIdx {
					t.Errorf("Expected start date to appear before due date, got:\n%s", result)
				}
			}
		})
	}
}

func TestFormatWithView_StartDateBoundaries(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		hoursOffset   float64
		expectedColor string
		description   string
	}{
		{
			name:          "exactly 72 hours (3 days) future",
			hoursOffset:   72,
			expectedColor: "\033[90m", // Gray (>= 72 hours = future)
			description:   "Exactly 3 days should be gray",
		},
		{
			name:          "just under 72 hours",
			hoursOffset:   71.5,
			expectedColor: "\033[33m", // Yellow (< 72 hours)
			description:   "Just under 3 days should be yellow",
		},
		{
			name:          "exactly now",
			hoursOffset:   0,
			expectedColor: "\033[36m", // Cyan (past/present)
			description:   "Right now should be cyan",
		},
		{
			name:          "1 second ago",
			hoursOffset:   -1.0 / 3600.0,
			expectedColor: "\033[36m", // Cyan
			description:   "Past should be cyan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := Task{
				UID:       "test-boundary",
				Summary:   "Boundary test",
				Status:    "NEEDS-ACTION",
				StartDate: timePtr(now.Add(time.Duration(tt.hoursOffset * float64(time.Hour)))),
			}

			result := task.FormatWithView("basic", nil, "2006-01-02")

			if !strings.Contains(result, tt.expectedColor) {
				t.Errorf("Test '%s': Expected color %q for %s, got:\n%s",
					tt.name, tt.expectedColor, tt.description, result)
			}
		})
	}
}

// Helper function to create time pointer
func timePtr(t time.Time) *time.Time {
	return &t
}

// TestFormatWithView_UniformStatusNames tests that FormatWithView correctly handles
// status names across different backends using StatusToDisplayName translation.
func TestFormatWithView_UniformStatusNames(t *testing.T) {
	// Create mock backends
	nextcloudBackend := &NextcloudBackend{}
	fileBackend := &FileBackend{}

	tests := []struct {
		name             string
		nextcloudStatus  string // CalDAV status used by Nextcloud
		fileStatus       string // Canonical status used by File backend
		expectedSymbol   string
		expectedColor    string
		description      string
	}{
		{
			name:            "TODO status",
			nextcloudStatus: "NEEDS-ACTION",
			fileStatus:      "TODO",
			expectedSymbol:  "○",
			expectedColor:   "\033[37m", // White
			description:     "Both backends should display TODO status the same way",
		},
		{
			name:            "DONE status",
			nextcloudStatus: "COMPLETED",
			fileStatus:      "DONE",
			expectedSymbol:  "✓",
			expectedColor:   "\033[32m", // Green
			description:     "Both backends should display DONE status the same way",
		},
		{
			name:            "PROCESSING status",
			nextcloudStatus: "IN-PROCESS",
			fileStatus:      "PROCESSING",
			expectedSymbol:  "●",
			expectedColor:   "\033[33m", // Yellow
			description:     "Both backends should display PROCESSING status the same way",
		},
		{
			name:            "CANCELLED status",
			nextcloudStatus: "CANCELLED",
			fileStatus:      "CANCELLED",
			expectedSymbol:  "✗",
			expectedColor:   "\033[31m", // Red
			description:     "Both backends should display CANCELLED status the same way",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with Nextcloud backend (CalDAV status)
			nextcloudTask := Task{
				UID:     "test-nextcloud",
				Summary: "Test task",
				Status:  tt.nextcloudStatus,
			}
			nextcloudResult := nextcloudTask.FormatWithView("basic", nextcloudBackend, "2006-01-02")

			// Test with File backend (canonical status)
			fileTask := Task{
				UID:     "test-file",
				Summary: "Test task",
				Status:  tt.fileStatus,
			}
			fileResult := fileTask.FormatWithView("basic", fileBackend, "2006-01-02")

			// Both should contain the expected symbol
			if !strings.Contains(nextcloudResult, tt.expectedSymbol) {
				t.Errorf("Nextcloud backend: Expected symbol %q in output for %s, got:\n%s",
					tt.expectedSymbol, tt.description, nextcloudResult)
			}
			if !strings.Contains(fileResult, tt.expectedSymbol) {
				t.Errorf("File backend: Expected symbol %q in output for %s, got:\n%s",
					tt.expectedSymbol, tt.description, fileResult)
			}

			// Both should contain the expected color
			if !strings.Contains(nextcloudResult, tt.expectedColor) {
				t.Errorf("Nextcloud backend: Expected color %q in output for %s, got:\n%s",
					tt.expectedColor, tt.description, nextcloudResult)
			}
			if !strings.Contains(fileResult, tt.expectedColor) {
				t.Errorf("File backend: Expected color %q in output for %s, got:\n%s",
					tt.expectedColor, tt.description, fileResult)
			}
		})
	}
}

// TestStatusToDisplayName_Consistency tests that all backends return
// the same canonical display names for equivalent statuses.
func TestStatusToDisplayName_Consistency(t *testing.T) {
	nextcloudBackend := &NextcloudBackend{}
	fileBackend := &FileBackend{}

	tests := []struct {
		name            string
		nextcloudStatus string
		fileStatus      string
		expectedDisplay string
	}{
		{
			name:            "TODO mapping",
			nextcloudStatus: "NEEDS-ACTION",
			fileStatus:      "TODO",
			expectedDisplay: "TODO",
		},
		{
			name:            "DONE mapping",
			nextcloudStatus: "COMPLETED",
			fileStatus:      "DONE",
			expectedDisplay: "DONE",
		},
		{
			name:            "PROCESSING mapping",
			nextcloudStatus: "IN-PROCESS",
			fileStatus:      "PROCESSING",
			expectedDisplay: "PROCESSING",
		},
		{
			name:            "CANCELLED mapping",
			nextcloudStatus: "CANCELLED",
			fileStatus:      "CANCELLED",
			expectedDisplay: "CANCELLED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextcloudDisplay := nextcloudBackend.StatusToDisplayName(tt.nextcloudStatus)
			fileDisplay := fileBackend.StatusToDisplayName(tt.fileStatus)

			if nextcloudDisplay != tt.expectedDisplay {
				t.Errorf("Nextcloud backend: Expected display name %q, got %q",
					tt.expectedDisplay, nextcloudDisplay)
			}

			if fileDisplay != tt.expectedDisplay {
				t.Errorf("File backend: Expected display name %q, got %q",
					tt.expectedDisplay, fileDisplay)
			}

			// Most importantly, both should return the same display name
			if nextcloudDisplay != fileDisplay {
				t.Errorf("Inconsistent display names: Nextcloud returned %q, File returned %q",
					nextcloudDisplay, fileDisplay)
			}
		})
	}
}
