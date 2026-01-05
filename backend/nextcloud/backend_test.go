package nextcloud

import (
	"gosynctasks/backend"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// Helper function to create test URL (keeps http:// scheme for httptest server)
func createTestURL(t *testing.T, serverURL string) *url.URL {
	// Parse the httptest server URL and add credentials
	u, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("Failed to parse server URL: %v", err)
	}
	u.User = url.UserPassword("testuser", "testpass")
	u.Scheme = "nextcloud" // Change scheme to nextcloud for backend detection
	return u
}

// Helper function to create test connector config with InsecureSkipVerify
func createTestConnector(t *testing.T, serverURL string) backend.ConnectorConfig {
	return backend.ConnectorConfig{
		URL:                createTestURL(t, serverURL),
		InsecureSkipVerify: true, // Allow HTTP connections for testing
		SuppressSSLWarning: true,
	}
}

// Helper function to create test backend with proper HTTP URL
func createTestBackend(t *testing.T, serverURL string) *NextcloudBackend {
	nb := &NextcloudBackend{
		Connector: createTestConnector(t, serverURL),
	}
	// Override baseURL to use HTTP for httptest server
	nb.baseURL = serverURL
	return nb
}

// Mock CalDAV server responses
const mockTaskListsResponse = `<?xml version="1.0"?>
<d:multistatus xmlns:d="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav" xmlns:cs="http://calendarserver.org/ns/">
    <d:response>
        <d:href>/remote.php/dav/calendars/testuser/tasks/</d:href>
        <d:propstat>
            <d:prop>
                <d:displayname>My Tasks</d:displayname>
                <cal:supported-calendar-component-set>
                    <cal:comp name="VTODO"/>
                </cal:supported-calendar-component-set>
                <cs:getctag>12345</cs:getctag>
                <d:calendar-color>#0082c9</d:calendar-color>
            </d:prop>
            <d:status>HTTP/1.1 200 OK</d:status>
        </d:propstat>
    </d:response>
    <d:response>
        <d:href>/remote.php/dav/calendars/testuser/work/</d:href>
        <d:propstat>
            <d:prop>
                <d:displayname>Work Tasks</d:displayname>
                <cal:supported-calendar-component-set>
                    <cal:comp name="VTODO"/>
                </cal:supported-calendar-component-set>
                <cs:getctag>67890</cs:getctag>
            </d:prop>
            <d:status>HTTP/1.1 200 OK</d:status>
        </d:propstat>
    </d:response>
</d:multistatus>`

const mockTasksResponse = `<?xml version="1.0"?>
<d:multistatus xmlns:d="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav">
    <d:response>
        <d:href>/remote.php/dav/calendars/testuser/tasks/task1.ics</d:href>
        <d:propstat>
            <d:prop>
                <cal:calendar-data>BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Nextcloud Tasks//
BEGIN:VTODO
UID:task1
SUMMARY:Buy groceries
DESCRIPTION:Milk, eggs, bread
STATUS:NEEDS-ACTION
PRIORITY:5
CREATED:20230101T120000Z
LAST-MODIFIED:20230101T120000Z
DUE:20230110T120000Z
END:VTODO
END:VCALENDAR</cal:calendar-data>
            </d:prop>
            <d:status>HTTP/1.1 200 OK</d:status>
        </d:propstat>
    </d:response>
    <d:response>
        <d:href>/remote.php/dav/calendars/testuser/tasks/task2.ics</d:href>
        <d:propstat>
            <d:prop>
                <cal:calendar-data>BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Nextcloud Tasks//
BEGIN:VTODO
UID:task2
SUMMARY:Finish report
STATUS:COMPLETED
PRIORITY:1
CREATED:20230102T120000Z
LAST-MODIFIED:20230103T120000Z
COMPLETED:20230103T150000Z
END:VTODO
END:VCALENDAR</cal:calendar-data>
            </d:prop>
            <d:status>HTTP/1.1 200 OK</d:status>
        </d:propstat>
    </d:response>
</d:multistatus>`

func TestNextcloudBackend_GetTaskLists(t *testing.T) {
	// Create mock CalDAV server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		if r.Method != "PROPFIND" {
			t.Errorf("Expected PROPFIND, got %s", r.Method)
		}

		depth := r.Header.Get("Depth")
		if depth != "1" {
			t.Errorf("Expected Depth: 1, got %s", depth)
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		w.Write([]byte(mockTaskListsResponse))
	}))
	defer server.Close()

	// Create backend with test server
	nb := createTestBackend(t, server.URL)

	// Test GetTaskLists
	lists, err := nb.GetTaskLists()
	if err != nil {
		t.Fatalf("GetTaskLists failed: %v", err)
	}

	// Verify results
	if len(lists) != 2 {
		t.Fatalf("Expected 2 lists, got %d", len(lists))
	}

	// Check first list
	if lists[0].Name != "My Tasks" {
		t.Errorf("Expected 'My Tasks', got '%s'", lists[0].Name)
	}
	if lists[0].ID == "" {
		t.Error("Expected non-empty list ID")
	}
	if lists[0].CTags != "12345" {
		t.Errorf("Expected CTags '12345', got '%s'", lists[0].CTags)
	}
	if lists[0].Color != "#0082c9" {
		t.Errorf("Expected color '#0082c9', got '%s'", lists[0].Color)
	}

	// Check second list
	if lists[1].Name != "Work Tasks" {
		t.Errorf("Expected 'Work Tasks', got '%s'", lists[1].Name)
	}
	if lists[1].CTags != "67890" {
		t.Errorf("Expected CTags '67890', got '%s'", lists[1].CTags)
	}
}

func TestNextcloudBackend_GetTaskLists_AuthenticationError(t *testing.T) {
	// Create mock CalDAV server that returns 401
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	}))
	defer server.Close()

	// Create backend with test server
	nb := createTestBackend(t, server.URL)

	// Test GetTaskLists with authentication failure
	_, err := nb.GetTaskLists()
	if err == nil {
		t.Fatal("Expected error for 401 response, got nil")
	}

	// Verify it's a backend.BackendError
	backendErr, ok := err.(*backend.BackendError)
	if !ok {
		t.Fatalf("Expected backend.BackendError, got %T", err)
	}

	// Verify it's recognized as unauthorized
	if !backendErr.IsUnauthorized() {
		t.Errorf("Expected IsUnauthorized() to be true for 401 error")
	}

	// Verify error message contains helpful information
	errMsg := err.Error()
	if !strings.Contains(errMsg, "Authentication failed") {
		t.Errorf("Expected error message to mention authentication failure, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "username and password") {
		t.Errorf("Expected error message to mention username and password, got: %s", errMsg)
	}
}

func TestNextcloudBackend_GetTasks(t *testing.T) {
	// Create mock CalDAV server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != "REPORT" {
			t.Errorf("Expected REPORT, got %s", r.Method)
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		w.Write([]byte(mockTasksResponse))
	}))
	defer server.Close()

	// Create backend with test server
	nb := createTestBackend(t, server.URL)

	// Test GetTasks
	tasks, err := nb.GetTasks("/calendars/testuser/tasks/", nil)
	if err != nil {
		t.Fatalf("GetTasks failed: %v", err)
	}

	// Verify results
	if len(tasks) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(tasks))
	}

	// Check first task
	task1 := tasks[0]
	if task1.UID != "task1" {
		t.Errorf("Expected UID 'task1', got '%s'", task1.UID)
	}
	if task1.Summary != "Buy groceries" {
		t.Errorf("Expected 'Buy groceries', got '%s'", task1.Summary)
	}
	if task1.Description != "Milk, eggs, bread" {
		t.Errorf("Expected description, got '%s'", task1.Description)
	}
	if task1.Status != "NEEDS-ACTION" {
		t.Errorf("Expected status 'NEEDS-ACTION', got '%s'", task1.Status)
	}
	if task1.Priority != 5 {
		t.Errorf("Expected priority 5, got %d", task1.Priority)
	}

	// Check second task
	task2 := tasks[1]
	if task2.UID != "task2" {
		t.Errorf("Expected UID 'task2', got '%s'", task2.UID)
	}
	if task2.Summary != "Finish report" {
		t.Errorf("Expected 'Finish report', got '%s'", task2.Summary)
	}
	if task2.Status != "COMPLETED" {
		t.Errorf("Expected status 'COMPLETED', got '%s'", task2.Status)
	}
	if task2.Priority != 1 {
		t.Errorf("Expected priority 1, got %d", task2.Priority)
	}
}

func TestNextcloudBackend_GetTasks_WithFilter(t *testing.T) {
	requestCount := 0
	var capturedRequestBody string

	// Create mock CalDAV server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// Capture request body to verify filter
		buf, _ := io.ReadAll(r.Body)
		capturedRequestBody = string(buf)

		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		w.Write([]byte(mockTasksResponse))
	}))
	defer server.Close()

	nb := createTestBackend(t, server.URL)

	// Test with status filter (use CalDAV status name, not app status name)
	needsActionStatus := "NEEDS-ACTION"
	filter := &backend.TaskFilter{
		Statuses: &[]string{needsActionStatus},
	}

	_, err := nb.GetTasks("/calendars/testuser/tasks/", filter)
	if err != nil {
		t.Fatalf("GetTasks with filter failed: %v", err)
	}

	// Verify request was made
	if requestCount != 1 {
		t.Errorf("Expected 1 request, got %d", requestCount)
	}

	// Verify filter was included in calendar query
	// NEEDS-ACTION status is filtered by checking that COMPLETED is not defined
	if !strings.Contains(capturedRequestBody, "COMPLETED") || !strings.Contains(capturedRequestBody, "is-not-defined") {
		t.Errorf("Expected filter to check COMPLETED is-not-defined for NEEDS-ACTION status, got: %s", capturedRequestBody)
	}
}

func TestNextcloudBackend_FindTasksBySummary(t *testing.T) {
	// Create mock CalDAV server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		w.Write([]byte(mockTasksResponse))
	}))
	defer server.Close()

	nb := createTestBackend(t, server.URL)

	tests := []struct {
		name          string
		searchSummary string
		expectedCount int
		expectedUID   string
	}{
		{
			name:          "Exact match",
			searchSummary: "Buy groceries",
			expectedCount: 1,
			expectedUID:   "task1",
		},
		{
			name:          "Partial match",
			searchSummary: "groceries",
			expectedCount: 1,
			expectedUID:   "task1",
		},
		{
			name:          "Case insensitive",
			searchSummary: "FINISH REPORT",
			expectedCount: 1,
			expectedUID:   "task2",
		},
		{
			name:          "No match",
			searchSummary: "nonexistent task",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := nb.FindTasksBySummary("/calendars/testuser/tasks/", tt.searchSummary)
			if err != nil {
				t.Fatalf("FindTasksBySummary failed: %v", err)
			}

			if len(matches) != tt.expectedCount {
				t.Errorf("Expected %d matches, got %d", tt.expectedCount, len(matches))
			}

			if tt.expectedCount > 0 && matches[0].UID != tt.expectedUID {
				t.Errorf("Expected UID '%s', got '%s'", tt.expectedUID, matches[0].UID)
			}
		})
	}
}

func TestNextcloudBackend_AddTask(t *testing.T) {
	var capturedMethod string
	var capturedBody string

	// Create mock CalDAV server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method

		buf, _ := io.ReadAll(r.Body)
		capturedBody = string(buf)

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	nb := createTestBackend(t, server.URL)

	// Create test task
	task := backend.Task{
		Summary:     "New test task",
		Description: "backend.Task description",
		Status:      "NEEDS-ACTION",
		Priority:    3,
	}

	_, err := nb.AddTask("/calendars/testuser/tasks/", task)
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Verify HTTP method
	if capturedMethod != "PUT" {
		t.Errorf("Expected PUT request, got %s", capturedMethod)
	}

	// Verify iCalendar content
	if !strings.Contains(capturedBody, "BEGIN:VCALENDAR") {
		t.Error("Expected iCalendar content to contain BEGIN:VCALENDAR")
	}
	if !strings.Contains(capturedBody, "BEGIN:VTODO") {
		t.Error("Expected iCalendar content to contain BEGIN:VTODO")
	}
	if !strings.Contains(capturedBody, "SUMMARY:New test task") {
		t.Error("Expected iCalendar content to contain task summary")
	}
	if !strings.Contains(capturedBody, "DESCRIPTION:backend.Task description") {
		t.Error("Expected iCalendar content to contain task description")
	}
	if !strings.Contains(capturedBody, "STATUS:NEEDS-ACTION") {
		t.Error("Expected iCalendar content to contain status")
	}
	if !strings.Contains(capturedBody, "PRIORITY:3") {
		t.Error("Expected iCalendar content to contain priority")
	}
}

func TestNextcloudBackend_AddTask_PendingUIDReplacement(t *testing.T) {
	var capturedUID string

	// Create mock CalDAV server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract UID from URL path
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) > 0 {
			capturedUID = strings.TrimSuffix(parts[len(parts)-1], ".ics")
		}

		buf, _ := io.ReadAll(r.Body)
		body := string(buf)

		// Extract UID from iCalendar content
		for _, line := range strings.Split(body, "\n") {
			if strings.HasPrefix(line, "UID:") {
				capturedUID = strings.TrimSpace(strings.TrimPrefix(line, "UID:"))
				break
			}
		}

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	nb := createTestBackend(t, server.URL)

	// Test 1: Task with pending UID should get replaced
	taskWithPendingUID := backend.Task{
		UID:     "pending-123",
		Summary: "Task with pending UID",
		Status:  "NEEDS-ACTION",
	}

	returnedUID, err := nb.AddTask("/calendars/testuser/tasks/", taskWithPendingUID)
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	if strings.HasPrefix(returnedUID, "pending-") {
		t.Errorf("Expected pending UID to be replaced, but got: %s", returnedUID)
	}

	if !strings.HasPrefix(capturedUID, "task-") {
		t.Errorf("Expected generated UID to start with 'task-', but got: %s", capturedUID)
	}

	if returnedUID != capturedUID {
		t.Errorf("Returned UID (%s) doesn't match captured UID (%s)", returnedUID, capturedUID)
	}

	// Test 2: Task with empty UID should get generated
	taskWithEmptyUID := backend.Task{
		UID:     "",
		Summary: "Task with empty UID",
		Status:  "NEEDS-ACTION",
	}

	returnedUID2, err := nb.AddTask("/calendars/testuser/tasks/", taskWithEmptyUID)
	if err != nil {
		t.Fatalf("AddTask failed for empty UID: %v", err)
	}

	if !strings.HasPrefix(returnedUID2, "task-") {
		t.Errorf("Expected generated UID to start with 'task-', but got: %s", returnedUID2)
	}

	// Test 3: Task with normal UID should be preserved
	taskWithNormalUID := backend.Task{
		UID:     "my-custom-uid-456",
		Summary: "Task with normal UID",
		Status:  "NEEDS-ACTION",
	}

	returnedUID3, err := nb.AddTask("/calendars/testuser/tasks/", taskWithNormalUID)
	if err != nil {
		t.Fatalf("AddTask failed for normal UID: %v", err)
	}

	if returnedUID3 != "my-custom-uid-456" {
		t.Errorf("Expected normal UID to be preserved, but got: %s", returnedUID3)
	}
}

func TestNextcloudBackend_UpdateTask(t *testing.T) {
	var capturedMethod string
	var capturedBody string

	// Create mock CalDAV server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method

		buf, _ := io.ReadAll(r.Body)
		capturedBody = string(buf)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	nb := createTestBackend(t, server.URL)

	// Create test task with UID (required for update)
	task := backend.Task{
		UID:         "existing-task-123",
		Summary:     "Updated task",
		Description: "Updated description",
		Status:      "COMPLETED",
		Priority:    1,
	}

	err := nb.UpdateTask("/calendars/testuser/tasks/", task)
	if err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}

	// Verify HTTP method
	if capturedMethod != "PUT" {
		t.Errorf("Expected PUT request, got %s", capturedMethod)
	}

	// Verify updated content
	if !strings.Contains(capturedBody, "UID:existing-task-123") {
		t.Error("Expected iCalendar content to contain existing UID")
	}
	if !strings.Contains(capturedBody, "SUMMARY:Updated task") {
		t.Error("Expected iCalendar content to contain updated summary")
	}
	if !strings.Contains(capturedBody, "STATUS:COMPLETED") {
		t.Error("Expected iCalendar content to contain updated status")
	}
}

func TestNextcloudBackend_SortTasks(t *testing.T) {
	nb := &NextcloudBackend{}

	tasks := []backend.Task{
		{Summary: "backend.Task A", Priority: 0}, // Undefined (should be last)
		{Summary: "backend.Task B", Priority: 5}, // Medium
		{Summary: "backend.Task C", Priority: 1}, // Highest
		{Summary: "backend.Task D", Priority: 9}, // Lowest
		{Summary: "backend.Task E", Priority: 3}, // High
	}

	nb.SortTasks(tasks)

	// Expected order: 1, 3, 5, 9, 0 (undefined last)
	expected := []int{1, 3, 5, 9, 0}
	for i, task := range tasks {
		if task.Priority != expected[i] {
			t.Errorf("Position %d: expected priority %d, got %d", i, expected[i], task.Priority)
		}
	}
}

func TestNextcloudBackend_GetPriorityColor(t *testing.T) {
	nb := &NextcloudBackend{}

	tests := []struct {
		priority int
		expected string
	}{
		{1, "\033[31m"}, // Red (highest)
		{4, "\033[31m"}, // Red
		{5, "\033[33m"}, // Yellow
		{6, "\033[34m"}, // Blue
		{9, "\033[34m"}, // Blue (lowest)
		{0, ""},         // No color (undefined)
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.priority)), func(t *testing.T) {
			color := nb.GetPriorityColor(tt.priority)
			if color != tt.expected {
				t.Errorf("Priority %d: expected color %q, got %q", tt.priority, tt.expected, color)
			}
		})
	}
}

func TestNextcloudBackend_ParseStatusFlag(t *testing.T) {
	nb := &NextcloudBackend{}

	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		// Abbreviations
		{name: "abbreviation T", input: "T", expected: "NEEDS-ACTION", expectError: false},
		{name: "abbreviation D", input: "D", expected: "COMPLETED", expectError: false},
		{name: "abbreviation P", input: "P", expected: "IN-PROCESS", expectError: false},
		{name: "abbreviation C", input: "C", expected: "CANCELLED", expectError: false},

		// App status names
		{name: "TODO", input: "TODO", expected: "NEEDS-ACTION", expectError: false},
		{name: "DONE", input: "DONE", expected: "COMPLETED", expectError: false},
		{name: "PROCESSING", input: "PROCESSING", expected: "IN-PROCESS", expectError: false},
		{name: "CANCELLED", input: "CANCELLED", expected: "CANCELLED", expectError: false},

		// CalDAV status names (already in target format)
		{name: "NEEDS-ACTION", input: "NEEDS-ACTION", expected: "NEEDS-ACTION", expectError: false},
		{name: "COMPLETED", input: "COMPLETED", expected: "COMPLETED", expectError: false},
		{name: "IN-PROCESS", input: "IN-PROCESS", expected: "IN-PROCESS", expectError: false},

		// Case insensitive
		{name: "lowercase todo", input: "todo", expected: "NEEDS-ACTION", expectError: false},
		{name: "mixed case Done", input: "Done", expected: "COMPLETED", expectError: false},
		{name: "lowercase t", input: "t", expected: "NEEDS-ACTION", expectError: false},

		// Error cases
		{name: "empty string", input: "", expected: "", expectError: true},
		{name: "invalid status", input: "INVALID", expected: "", expectError: true},
		{name: "invalid abbreviation", input: "X", expected: "", expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := nb.ParseStatusFlag(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("ParseStatusFlag(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseStatusFlag(%q) unexpected error: %v", tt.input, err)
				return
			}

			if result != tt.expected {
				t.Errorf("ParseStatusFlag(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNextcloudBackend_StatusToDisplayName(t *testing.T) {
	nb := &NextcloudBackend{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// CalDAV to Display
		{name: "NEEDS-ACTION to TODO", input: "NEEDS-ACTION", expected: "TODO"},
		{name: "COMPLETED to DONE", input: "COMPLETED", expected: "DONE"},
		{name: "IN-PROCESS to PROCESSING", input: "IN-PROCESS", expected: "PROCESSING"},
		{name: "CANCELLED to CANCELLED", input: "CANCELLED", expected: "CANCELLED"},

		// Case insensitive
		{name: "lowercase needs-action", input: "needs-action", expected: "TODO"},
		{name: "mixed case Completed", input: "Completed", expected: "DONE"},

		// Unknown status passes through
		{name: "unknown status", input: "CUSTOM-STATUS", expected: "CUSTOM-STATUS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nb.StatusToDisplayName(tt.input)
			if result != tt.expected {
				t.Errorf("StatusToDisplayName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNextcloudBackend_DeleteTask(t *testing.T) {
	tests := []struct {
		name           string
		taskUID        string
		responseStatus int
		responseBody   string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "successful delete with 204",
			taskUID:        "task-123",
			responseStatus: 204,
			responseBody:   "",
			expectError:    false,
		},
		{
			name:           "successful delete with 200",
			taskUID:        "task-456",
			responseStatus: 200,
			responseBody:   "",
			expectError:    false,
		},
		{
			name:           "task not found",
			taskUID:        "nonexistent",
			responseStatus: 404,
			responseBody:   "Not Found",
			expectError:    true,
			errorContains:  "task not found",
		},
		{
			name:           "unauthorized",
			taskUID:        "task-789",
			responseStatus: 401,
			responseBody:   "Unauthorized",
			expectError:    true,
			errorContains:  "failed with status 401",
		},
		{
			name:           "server error",
			taskUID:        "task-error",
			responseStatus: 500,
			responseBody:   "Internal Server Error",
			expectError:    true,
			errorContains:  "failed with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != "DELETE" {
					t.Errorf("Expected DELETE request, got %s", r.Method)
				}

				// Verify URL contains task UID
				if !strings.Contains(r.URL.Path, tt.taskUID) {
					t.Errorf("Expected URL to contain %s, got %s", tt.taskUID, r.URL.Path)
				}

				// Send mock response
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			// Create backend with mock server
			nb := createTestBackend(t, server.URL)

			// Execute DeleteTask
			err := nb.DeleteTask("test-list", tt.taskUID)

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("DeleteTask() expected error, got nil")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("DeleteTask() error = %q, want error containing %q", err.Error(), tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("DeleteTask() unexpected error: %v", err)
				}
			}
		})
	}
}
func TestNextcloudBackend_CreateTaskList(t *testing.T) {
	tests := []struct {
		name           string
		listName       string
		description    string
		color          string
		responseStatus int
		responseBody   string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "successful creation",
			listName:       "New List",
			description:    "Test description",
			color:          "#ff0000",
			responseStatus: 201,
			responseBody:   "",
			expectError:    false,
		},
		{
			name:           "creation without description",
			listName:       "Simple List",
			description:    "",
			color:          "",
			responseStatus: 201,
			responseBody:   "",
			expectError:    false,
		},
		{
			name:           "list already exists",
			listName:       "Existing",
			description:    "",
			color:          "",
			responseStatus: 405,
			responseBody:   "Method not allowed",
			expectError:    true,
			errorContains:  "list already exists",
		},
		{
			name:           "server error",
			listName:       "Failed",
			description:    "",
			color:          "",
			responseStatus: 500,
			responseBody:   "Internal server error",
			expectError:    true,
			errorContains:  "500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != "MKCOL" {
					t.Errorf("Expected MKCOL request, got %s", r.Method)
				}

				// Read and verify request body contains the list name
				body, _ := io.ReadAll(r.Body)
				bodyStr := string(body)
				if !strings.Contains(bodyStr, tt.listName) {
					t.Errorf("Expected request body to contain %q, got %q", tt.listName, bodyStr)
				}

				// Verify description if provided
				if tt.description != "" && !strings.Contains(bodyStr, tt.description) {
					t.Errorf("Expected request body to contain description %q", tt.description)
				}

				// Verify color if provided
				if tt.color != "" && !strings.Contains(bodyStr, tt.color) {
					t.Errorf("Expected request body to contain color %q", tt.color)
				}

				// Send mock response
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			// Create backend with mock server
			nb := createTestBackend(t, server.URL)

			// Execute CreateTaskList
			listID, err := nb.CreateTaskList(tt.listName, tt.description, tt.color)

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("CreateTaskList() expected error, got nil")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("CreateTaskList() error = %q, want error containing %q", err.Error(), tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("CreateTaskList() unexpected error: %v", err)
				}
				if listID == "" {
					t.Errorf("CreateTaskList() returned empty listID")
				}
			}
		})
	}
}

func TestNextcloudBackend_DeleteTaskList(t *testing.T) {
	tests := []struct {
		name           string
		listID         string
		responseStatus int
		responseBody   string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "successful deletion",
			listID:         "test-list",
			responseStatus: 204,
			responseBody:   "",
			expectError:    false,
		},
		{
			name:           "list not found",
			listID:         "nonexistent",
			responseStatus: 404,
			responseBody:   "Not found",
			expectError:    true,
			errorContains:  "list not found",
		},
		{
			name:           "server error",
			listID:         "error-list",
			responseStatus: 500,
			responseBody:   "Internal server error",
			expectError:    true,
			errorContains:  "500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != "DELETE" {
					t.Errorf("Expected DELETE request, got %s", r.Method)
				}

				// Verify URL contains list ID
				if !strings.Contains(r.URL.Path, tt.listID) {
					t.Errorf("Expected URL to contain %s, got %s", tt.listID, r.URL.Path)
				}

				// Send mock response
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			// Create backend with mock server
			nb := createTestBackend(t, server.URL)

			// Execute DeleteTaskList
			err := nb.DeleteTaskList(tt.listID)

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("DeleteTaskList() expected error, got nil")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("DeleteTaskList() error = %q, want error containing %q", err.Error(), tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("DeleteTaskList() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestNextcloudBackend_RenameTaskList(t *testing.T) {
	tests := []struct {
		name           string
		listID         string
		newName        string
		responseStatus int
		responseBody   string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "successful rename",
			listID:         "test-list",
			newName:        "Renamed List",
			responseStatus: 207,
			responseBody:   `<?xml version="1.0"?><d:multistatus xmlns:d="DAV:"><d:response><d:status>HTTP/1.1 200 OK</d:status></d:response></d:multistatus>`,
			expectError:    false,
		},
		{
			name:           "list not found",
			listID:         "nonexistent",
			newName:        "New Name",
			responseStatus: 404,
			responseBody:   "Not found",
			expectError:    true,
			errorContains:  "list not found",
		},
		{
			name:           "server error",
			listID:         "error-list",
			newName:        "Error Name",
			responseStatus: 500,
			responseBody:   "Internal server error",
			expectError:    true,
			errorContains:  "500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != "PROPPATCH" {
					t.Errorf("Expected PROPPATCH request, got %s", r.Method)
				}

				// Verify URL contains list ID
				if !strings.Contains(r.URL.Path, tt.listID) {
					t.Errorf("Expected URL to contain %s, got %s", tt.listID, r.URL.Path)
				}

				// Read and verify request body contains new name
				body, _ := io.ReadAll(r.Body)
				bodyStr := string(body)
				if !strings.Contains(bodyStr, tt.newName) {
					t.Errorf("Expected request body to contain %q, got %q", tt.newName, bodyStr)
				}

				// Send mock response
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			// Create backend with mock server
			nb := createTestBackend(t, server.URL)

			// Execute RenameTaskList
			err := nb.RenameTaskList(tt.listID, tt.newName)

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("RenameTaskList() expected error, got nil")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("RenameTaskList() error = %q, want error containing %q", err.Error(), tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("RenameTaskList() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestHTTPSEnforcement tests that HTTPS is enforced by default
func TestHTTPSEnforcement(t *testing.T) {
	tests := []struct {
		name          string
		host          string
		allowHTTP     bool
		expectedProto string
	}{
		{
			name:          "Default HTTPS for standard port",
			host:          "nextcloud.example.com",
			allowHTTP:     false,
			expectedProto: "https",
		},
		{
			name:          "Default HTTPS for port 443",
			host:          "nextcloud.example.com:443",
			allowHTTP:     false,
			expectedProto: "https",
		},
		{
			name:          "Default HTTPS even for port 80 when AllowHTTP is false",
			host:          "nextcloud.example.com:80",
			allowHTTP:     false,
			expectedProto: "https",
		},
		{
			name:          "Default HTTPS even for port 8080 when AllowHTTP is false",
			host:          "localhost:8080",
			allowHTTP:     false,
			expectedProto: "https",
		},
		{
			name:          "HTTP allowed for port 80 when AllowHTTP is true",
			host:          "localhost:80",
			allowHTTP:     true,
			expectedProto: "http",
		},
		{
			name:          "HTTP allowed for port 8080 when AllowHTTP is true",
			host:          "localhost:8080",
			allowHTTP:     true,
			expectedProto: "http",
		},
		{
			name:          "HTTP allowed for port 8000 when AllowHTTP is true",
			host:          "localhost:8000",
			allowHTTP:     true,
			expectedProto: "http",
		},
		{
			name:          "HTTPS for non-standard port even with AllowHTTP",
			host:          "localhost:9090",
			allowHTTP:     true,
			expectedProto: "https",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create URL
			u, err := url.Parse("nextcloud://user:pass@" + tt.host)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			// Create connector config
			config := backend.ConnectorConfig{
				URL:                u,
				AllowHTTP:          tt.allowHTTP,
				InsecureSkipVerify: true,
				SuppressSSLWarning: true,
			}

			// Create backend
			nb := &NextcloudBackend{
				Connector: config,
			}

			// Get base URL (this triggers protocol selection)
			baseURL := nb.getBaseURL()

			// Verify the protocol
			if !strings.HasPrefix(baseURL, tt.expectedProto+"://") {
				t.Errorf("Expected protocol %s://, got %s", tt.expectedProto, baseURL)
			}

			// Verify the host is included
			if !strings.Contains(baseURL, tt.host) {
				t.Errorf("Expected baseURL to contain host %s, got %s", tt.host, baseURL)
			}
		})
	}
}

// TestHTTPSEnforcementDefault verifies HTTPS is the default without any config
func TestHTTPSEnforcementDefault(t *testing.T) {
	// Create URL without AllowHTTP (defaults to false)
	u, _ := url.Parse("nextcloud://user:pass@localhost:8080")

	config := backend.ConnectorConfig{
		URL:                u,
		InsecureSkipVerify: true,
		SuppressSSLWarning: true,
		// AllowHTTP is not set, defaults to false
	}

	nb := &NextcloudBackend{
		Connector: config,
	}

	baseURL := nb.getBaseURL()

	// Should use HTTPS even though port is 8080 (common HTTP port)
	if !strings.HasPrefix(baseURL, "https://") {
		t.Errorf("Expected HTTPS by default, got %s", baseURL)
	}
}
