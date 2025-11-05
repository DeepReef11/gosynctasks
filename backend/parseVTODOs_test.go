package backend

import (
	"strings"
	"testing"
	"time"
)

func TestParseICalTime(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
		expected  string // Expected time in RFC3339 format for comparison
	}{
		{
			name:      "UTC time format",
			input:     "20240315T143000Z",
			wantError: false,
			expected:  "2024-03-15T14:30:00Z",
		},
		{
			name:      "Local time format",
			input:     "20240315T143000",
			wantError: false,
			expected:  "2024-03-15T14:30:00Z",
		},
		{
			name:      "Date only format",
			input:     "20240315",
			wantError: false,
			expected:  "2024-03-15T00:00:00Z",
		},
		{
			name:      "Invalid format",
			input:     "2024-03-15",
			wantError: true,
		},
		{
			name:      "Empty string",
			input:     "",
			wantError: true,
		},
		{
			name:      "Malformed date",
			input:     "20241332T143000Z",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseICalTime(tt.input)

			if (err != nil) != tt.wantError {
				t.Errorf("parseICalTime(%q) error = %v, wantError %v", tt.input, err, tt.wantError)
				return
			}

			if !tt.wantError {
				expected, _ := time.Parse(time.RFC3339, tt.expected)
				if !result.Equal(expected) {
					t.Errorf("parseICalTime(%q) = %v, want %v", tt.input, result, expected)
				}
			}
		})
	}
}

func TestUnescapeText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "newline escape",
			input:    "Line 1\\nLine 2",
			expected: "Line 1\nLine 2",
		},
		{
			name:     "comma escape",
			input:    "Item 1\\, Item 2",
			expected: "Item 1, Item 2",
		},
		{
			name:     "semicolon escape",
			input:    "Part 1\\; Part 2",
			expected: "Part 1; Part 2",
		},
		{
			name:     "backslash escape",
			input:    "Path\\\\to\\\\file",
			expected: "Path\\to\\file",
		},
		{
			name:     "multiple escapes",
			input:    "Text\\nwith\\, multiple\\; escapes\\\\here",
			expected: "Text\nwith, multiple; escapes\\here",
		},
		{
			name:     "no escapes",
			input:    "Plain text",
			expected: "Plain text",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := unescapeText(tt.input)
			if result != tt.expected {
				t.Errorf("unescapeText(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "valid positive number",
			input:    "42",
			expected: 42,
		},
		{
			name:     "valid zero",
			input:    "0",
			expected: 0,
		},
		{
			name:     "valid negative number",
			input:    "-5",
			expected: -5,
		},
		{
			name:     "invalid string",
			input:    "abc",
			expected: 0,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "decimal number",
			input:    "3.14",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseInt(tt.input)
			if result != tt.expected {
				t.Errorf("parseInt(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractVTODOBlocks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int // Number of blocks expected
		contains []string
	}{
		{
			name: "single VTODO block",
			input: `BEGIN:VTODO
UID:task-1
SUMMARY:Test Task
END:VTODO`,
			expected: 1,
			contains: []string{"UID:task-1", "SUMMARY:Test Task"},
		},
		{
			name: "multiple VTODO blocks",
			input: `BEGIN:VTODO
UID:task-1
SUMMARY:Task 1
END:VTODO
BEGIN:VTODO
UID:task-2
SUMMARY:Task 2
END:VTODO`,
			expected: 2,
			contains: []string{"UID:task-1", "UID:task-2"},
		},
		{
			name: "VTODO with extra whitespace",
			input: `
  BEGIN:VTODO
    UID:task-1
    SUMMARY:Test Task
  END:VTODO
`,
			expected: 1,
			contains: []string{"UID:task-1"},
		},
		{
			name: "nested in VCALENDAR",
			input: `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VTODO
UID:task-1
SUMMARY:Test Task
END:VTODO
END:VCALENDAR`,
			expected: 1,
			contains: []string{"UID:task-1"},
		},
		{
			name:     "no VTODO blocks",
			input:    `BEGIN:VCALENDAR\nVERSION:2.0\nEND:VCALENDAR`,
			expected: 0,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVTODOBlocks(tt.input)

			if len(result) != tt.expected {
				t.Errorf("extractVTODOBlocks() returned %d blocks, want %d", len(result), tt.expected)
			}

			for _, substr := range tt.contains {
				found := false
				for _, block := range result {
					if strings.Contains(block, substr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("extractVTODOBlocks() blocks don't contain %q", substr)
				}
			}
		})
	}
}

func TestParseVTODO(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
		checkFunc func(*testing.T, Task)
	}{
		{
			name: "complete VTODO",
			input: `BEGIN:VTODO
UID:task-123
SUMMARY:Complete task
DESCRIPTION:This is a test task
STATUS:IN-PROCESS
PRIORITY:5
CREATED:20240315T120000Z
LAST-MODIFIED:20240315T130000Z
DUE:20240320T180000Z
END:VTODO`,
			wantError: false,
			checkFunc: func(t *testing.T, task Task) {
				if task.UID != "task-123" {
					t.Errorf("UID = %q, want %q", task.UID, "task-123")
				}
				if task.Summary != "Complete task" {
					t.Errorf("Summary = %q, want %q", task.Summary, "Complete task")
				}
				if task.Description != "This is a test task" {
					t.Errorf("Description = %q, want %q", task.Description, "This is a test task")
				}
				if task.Status != "IN-PROCESS" {
					t.Errorf("Status = %q, want %q", task.Status, "IN-PROCESS")
				}
				if task.Priority != 5 {
					t.Errorf("Priority = %d, want %d", task.Priority, 5)
				}
				if task.DueDate == nil {
					t.Error("DueDate is nil, want non-nil")
				}
			},
		},
		{
			name: "minimal VTODO",
			input: `BEGIN:VTODO
UID:minimal-task
SUMMARY:Minimal
END:VTODO`,
			wantError: false,
			checkFunc: func(t *testing.T, task Task) {
				if task.UID != "minimal-task" {
					t.Errorf("UID = %q, want %q", task.UID, "minimal-task")
				}
				if task.Summary != "Minimal" {
					t.Errorf("Summary = %q, want %q", task.Summary, "Minimal")
				}
				if task.Status != "NEEDS-ACTION" {
					t.Errorf("Status = %q, want default %q", task.Status, "NEEDS-ACTION")
				}
			},
		},
		{
			name: "VTODO with escaped text",
			input: `BEGIN:VTODO
UID:escaped-task
SUMMARY:Task\\nwith\\, escapes
DESCRIPTION:Line 1\\nLine 2\\; etc
END:VTODO`,
			wantError: false,
			checkFunc: func(t *testing.T, task Task) {
				if !strings.Contains(task.Summary, "\n") {
					t.Errorf("Summary should contain newline, got %q", task.Summary)
				}
				if !strings.Contains(task.Description, "\n") {
					t.Errorf("Description should contain newline, got %q", task.Description)
				}
			},
		},
		{
			name: "VTODO with parameters",
			input: `BEGIN:VTODO
UID:param-task
SUMMARY:Task with params
DTSTART;VALUE=DATE:20240315
DUE;VALUE=DATE:20240320
END:VTODO`,
			wantError: false,
			checkFunc: func(t *testing.T, task Task) {
				if task.StartDate == nil {
					t.Error("StartDate is nil, want non-nil")
				}
				if task.DueDate == nil {
					t.Error("DueDate is nil, want non-nil")
				}
			},
		},
		{
			name: "VTODO with categories",
			input: `BEGIN:VTODO
UID:cat-task
SUMMARY:Categorized task
CATEGORIES:Work,Important
END:VTODO`,
			wantError: false,
			checkFunc: func(t *testing.T, task Task) {
				if len(task.Categories) != 2 {
					t.Errorf("len(Categories) = %d, want 2", len(task.Categories))
				}
			},
		},
		{
			name: "VTODO with parent (subtask)",
			input: `BEGIN:VTODO
UID:subtask-1
SUMMARY:Subtask
RELATED-TO:parent-task
END:VTODO`,
			wantError: false,
			checkFunc: func(t *testing.T, task Task) {
				if task.ParentUID != "parent-task" {
					t.Errorf("ParentUID = %q, want %q", task.ParentUID, "parent-task")
				}
			},
		},
		{
			name: "VTODO missing UID",
			input: `BEGIN:VTODO
SUMMARY:No UID task
END:VTODO`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseVTODO(tt.input)

			if (err != nil) != tt.wantError {
				t.Errorf("parseVTODO() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}

func TestExtractXMLValue(t *testing.T) {
	tests := []struct {
		name     string
		xml      string
		tag      string
		expected string
	}{
		{
			name:     "tag without namespace",
			xml:      "<displayname>My Calendar</displayname>",
			tag:      "displayname",
			expected: "My Calendar",
		},
		{
			name:     "tag with d: namespace",
			xml:      "<d:displayname>My Calendar</d:displayname>",
			tag:      "displayname",
			expected: "My Calendar",
		},
		{
			name:     "tag with cs: namespace",
			xml:      "<cs:getctag>abc123</cs:getctag>",
			tag:      "getctag",
			expected: "abc123",
		},
		{
			name:     "tag with ic: namespace",
			xml:      "<ic:calendar-color>#FF0000</ic:calendar-color>",
			tag:      "calendar-color",
			expected: "#FF0000",
		},
		{
			name:     "tag with whitespace",
			xml:      "<displayname>  My Calendar  </displayname>",
			tag:      "displayname",
			expected: "My Calendar",
		},
		{
			name:     "tag not found",
			xml:      "<displayname>My Calendar</displayname>",
			tag:      "notfound",
			expected: "",
		},
		{
			name:     "empty tag value",
			xml:      "<displayname></displayname>",
			tag:      "displayname",
			expected: "",
		},
		{
			name:     "nested tags",
			xml:      "<outer><displayname>Inner Value</displayname></outer>",
			tag:      "displayname",
			expected: "Inner Value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractXMLValue(tt.xml, tt.tag)
			if result != tt.expected {
				t.Errorf("extractXMLValue(%q, %q) = %q, want %q", tt.xml, tt.tag, result, tt.expected)
			}
		})
	}
}

func TestExtractResponses(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int // Number of responses expected
	}{
		{
			name: "single response with d: namespace",
			input: `<d:response>
				<d:href>/calendars/user/tasks/</d:href>
				<d:propstat>
					<d:prop><d:displayname>Tasks</d:displayname></d:prop>
				</d:propstat>
			</d:response>`,
			expected: 1,
		},
		{
			name: "multiple responses",
			input: `<d:response>
				<d:href>/calendars/user/tasks1/</d:href>
			</d:response>
			<d:response>
				<d:href>/calendars/user/tasks2/</d:href>
			</d:response>`,
			expected: 2,
		},
		{
			name: "response without namespace",
			input: `<response>
				<href>/calendars/user/tasks/</href>
			</response>`,
			expected: 1,
		},
		{
			name: "D: uppercase namespace",
			input: `<D:response>
				<D:href>/calendars/user/tasks/</D:href>
			</D:response>`,
			expected: 1,
		},
		{
			name:     "no responses",
			input:    `<multistatus><propstat></propstat></multistatus>`,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractResponses(tt.input)
			if len(result) != tt.expected {
				t.Errorf("extractResponses() returned %d responses, want %d", len(result), tt.expected)
			}
		})
	}
}

func TestParseTaskListResponse(t *testing.T) {
	tests := []struct {
		name      string
		response  string
		baseURL   string
		checkFunc func(*testing.T, TaskList)
	}{
		{
			name: "complete task list response",
			response: `<d:response>
				<d:href>/calendars/user/work-tasks/</d:href>
				<d:propstat>
					<d:prop>
						<d:displayname>Work Tasks</d:displayname>
						<cs:getctag>123abc</cs:getctag>
						<ic:calendar-color>#FF0000</ic:calendar-color>
					</d:prop>
				</d:propstat>
			</d:response>`,
			baseURL: "https://example.com",
			checkFunc: func(t *testing.T, tl TaskList) {
				if tl.ID != "work-tasks" {
					t.Errorf("ID = %q, want %q", tl.ID, "work-tasks")
				}
				if tl.Name != "Work Tasks" {
					t.Errorf("Name = %q, want %q", tl.Name, "Work Tasks")
				}
				if tl.CTags != "123abc" {
					t.Errorf("CTags = %q, want %q", tl.CTags, "123abc")
				}
				if tl.Color != "#FF0000" {
					t.Errorf("Color = %q, want %q", tl.Color, "#FF0000")
				}
			},
		},
		{
			name: "minimal task list response",
			response: `<d:response>
				<d:href>/calendars/user/simple/</d:href>
				<d:propstat>
					<d:prop>
						<d:displayname>Simple List</d:displayname>
					</d:prop>
				</d:propstat>
			</d:response>`,
			baseURL: "https://example.com",
			checkFunc: func(t *testing.T, tl TaskList) {
				if tl.ID != "simple" {
					t.Errorf("ID = %q, want %q", tl.ID, "simple")
				}
				if tl.Name != "Simple List" {
					t.Errorf("Name = %q, want %q", tl.Name, "Simple List")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTaskListResponse(tt.response, tt.baseURL)
			if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}
