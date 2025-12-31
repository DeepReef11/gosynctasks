package git

import (
	"gosynctasks/backend"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestMarkdownParser tests the markdown parser
func TestMarkdownParser(t *testing.T) {
	content := `<!-- gosynctasks:enabled -->

## Work Tasks
- [ ] Review PR #123 @uid:task-001 @priority:1 @due:2025-01-20
- [x] Deploy to staging @uid:task-002 @completed:2025-01-10
  This task has a description
  spanning multiple lines
- [>] Update documentation @uid:task-003 @priority:5
- [-] Cancelled task @uid:task-004

## Personal
- [ ] Buy groceries @uid:task-005 @priority:5 @due:2025-01-15
`

	parser := NewMarkdownParser()
	taskLists, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Check we have 2 lists
	if len(taskLists) != 2 {
		t.Errorf("expected 2 task lists, got %d", len(taskLists))
	}

	// Check Work Tasks
	workTasks, ok := taskLists["Work Tasks"]
	if !ok {
		t.Fatal("Work Tasks list not found")
	}
	if len(workTasks) != 4 {
		t.Errorf("expected 4 work tasks, got %d", len(workTasks))
	}

	// Check first task
	task := workTasks[0]
	if task.Summary != "Review PR #123" {
		t.Errorf("task summary = %q, want %q", task.Summary, "Review PR #123")
	}
	if task.UID != "task-001" {
		t.Errorf("task UID = %q, want %q", task.UID, "task-001")
	}
	if task.Priority != 1 {
		t.Errorf("task priority = %d, want 1", task.Priority)
	}
	if task.Status != "TODO" {
		t.Errorf("task status = %q, want TODO", task.Status)
	}

	// Check second task (completed with description)
	task = workTasks[1]
	if task.Status != "DONE" {
		t.Errorf("task status = %q, want DONE", task.Status)
	}
	if !strings.Contains(task.Description, "spanning multiple lines") {
		t.Errorf("task description = %q, missing expected content", task.Description)
	}

	// Check third task (processing)
	task = workTasks[2]
	if task.Status != "PROCESSING" {
		t.Errorf("task status = %q, want PROCESSING", task.Status)
	}

	// Check fourth task (cancelled)
	task = workTasks[3]
	if task.Status != "CANCELLED" {
		t.Errorf("task status = %q, want CANCELLED", task.Status)
	}

	// Check Personal tasks
	personalTasks, ok := taskLists["Personal"]
	if !ok {
		t.Fatal("Personal list not found")
	}
	if len(personalTasks) != 1 {
		t.Errorf("expected 1 personal task, got %d", len(personalTasks))
	}
}

// TestMarkdownWriter tests the markdown writer
func TestMarkdownWriter(t *testing.T) {
	taskLists := map[string][]backend.Task{
		"Work Tasks": {
			{
				UID:      "task-001",
				Summary:  "Review PR #123",
				Status:   "TODO",
				Priority: 1,
				DueDate:  &[]time.Time{time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)}[0],
				Created:  time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC),
			},
			{
				UID:         "task-002",
				Summary:     "Deploy to staging",
				Status:      "DONE",
				Description: "Completed successfully",
				Completed:   &[]time.Time{time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)}[0],
			},
		},
	}

	writer := NewMarkdownWriter()
	content := writer.Write(taskLists)

	// Check marker is present
	if !strings.Contains(content, gitBackendMarker) {
		t.Error("content should contain gosynctasks marker")
	}

	// Check header is present
	if !strings.Contains(content, "## Work Tasks") {
		t.Error("content should contain Work Tasks header")
	}

	// Check tasks are present
	if !strings.Contains(content, "Review PR #123") {
		t.Error("content should contain first task summary")
	}

	// Check checkboxes
	if !strings.Contains(content, "- [ ] Review PR #123") {
		t.Error("content should have TODO checkbox for first task")
	}
	if !strings.Contains(content, "- [x] Deploy to staging") {
		t.Error("content should have DONE checkbox for second task")
	}

	// Check tags
	if !strings.Contains(content, "@uid:task-001") {
		t.Error("content should contain UID tag")
	}
	if !strings.Contains(content, "@priority:1") {
		t.Error("content should contain priority tag")
	}
	if !strings.Contains(content, "@due:2025-01-20") {
		t.Error("content should contain due date tag")
	}

	// Check description is indented
	if !strings.Contains(content, "  Completed successfully") {
		t.Error("content should have indented description")
	}
}

// TestMarkdownRoundTrip tests parse -> write -> parse consistency
func TestMarkdownRoundTrip(t *testing.T) {
	original := `<!-- gosynctasks:enabled -->

## Test List
- [ ] backend.Task 1 @uid:task-001 @priority:1
- [x] backend.Task 2 @uid:task-002
  With description
`

	// Parse
	parser := NewMarkdownParser()
	taskLists, err := parser.Parse(original)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Write
	writer := NewMarkdownWriter()
	written := writer.Write(taskLists)

	// Parse again
	taskLists2, err := parser.Parse(written)
	if err != nil {
		t.Fatalf("Parse() second time error = %v", err)
	}

	// Compare task counts
	if len(taskLists) != len(taskLists2) {
		t.Errorf("list count changed: %d -> %d", len(taskLists), len(taskLists2))
	}

	// Compare tasks
	for listName, tasks := range taskLists {
		tasks2, ok := taskLists2[listName]
		if !ok {
			t.Errorf("list %q missing after round trip", listName)
			continue
		}
		if len(tasks) != len(tasks2) {
			t.Errorf("task count in %q changed: %d -> %d", listName, len(tasks), len(tasks2))
		}

		for i := range tasks {
			if i >= len(tasks2) {
				break
			}
			if tasks[i].UID != tasks2[i].UID {
				t.Errorf("task UID changed: %q -> %q", tasks[i].UID, tasks2[i].UID)
			}
			if tasks[i].Summary != tasks2[i].Summary {
				t.Errorf("task summary changed: %q -> %q", tasks[i].Summary, tasks2[i].Summary)
			}
			if tasks[i].Status != tasks2[i].Status {
				t.Errorf("task status changed: %q -> %q", tasks[i].Status, tasks2[i].Status)
			}
		}
	}
}

// TestGitBackendFindRepo tests git repository detection
func TestGitBackendFindRepo(t *testing.T) {
	// This test requires being run in a git repository
	gb := &GitBackend{}
	repoPath, err := gb.findGitRepo()
	if err != nil {
		t.Skipf("Not in a git repository, skipping test: %v", err)
	}

	// Should return an absolute path
	if !filepath.IsAbs(repoPath) {
		t.Errorf("findGitRepo() returned relative path: %s", repoPath)
	}

	// Should contain .git directory
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		t.Errorf(".git not found in returned path: %s", repoPath)
	}
}

// TestGitBackendHasMarker tests marker detection
func TestGitBackendHasMarker(t *testing.T) {
	gb := &GitBackend{}

	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "with marker",
			content: "<!-- gosynctasks:enabled -->\n# Tasks",
			want:    true,
		},
		{
			name:    "without marker",
			content: "# Tasks\n- [ ] backend.Task 1",
			want:    false,
		},
		{
			name:    "marker in middle",
			content: "# Header\n<!-- gosynctasks:enabled -->\n- [ ] backend.Task",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gb.hasMarker(tt.content); got != tt.want {
				t.Errorf("hasMarker() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGitBackendParseStatusFlag tests status flag parsing
func TestGitBackendParseStatusFlag(t *testing.T) {
	gb := &GitBackend{}

	tests := []struct {
		flag    string
		want    string
		wantErr bool
	}{
		{"T", "TODO", false},
		{"D", "DONE", false},
		{"P", "PROCESSING", false},
		{"C", "CANCELLED", false},
		{"TODO", "TODO", false},
		{"DONE", "DONE", false},
		{"todo", "TODO", false},
		{"done", "DONE", false},
		{"INVALID", "", true},
		{"X", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			got, err := gb.ParseStatusFlag(tt.flag)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseStatusFlag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseStatusFlag() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestGitBackendGenerateUID tests UID generation
func TestGitBackendGenerateUID(t *testing.T) {
	gb := &GitBackend{}

	// Generate two UIDs
	uid1 := gb.generateUID()
	time.Sleep(10 * time.Millisecond) // Ensure different timestamp
	uid2 := gb.generateUID()

	// Should start with "task-"
	if !strings.HasPrefix(uid1, "task-") {
		t.Errorf("UID should start with 'task-': %s", uid1)
	}

	// Should be unique
	if uid1 == uid2 {
		t.Errorf("UIDs should be unique: %s == %s", uid1, uid2)
	}

	// Should have expected format: task-{timestamp}-{random}
	parts := strings.Split(uid1, "-")
	if len(parts) != 3 {
		t.Errorf("UID should have 3 parts: %s", uid1)
	}
}

// TestGitBackendSortTasks tests task sorting
func TestGitBackendSortTasks(t *testing.T) {
	gb := &GitBackend{}

	tasks := []backend.Task{
		{UID: "3", Priority: 0, Created: time.Now()},                     // undefined priority
		{UID: "1", Priority: 1, Created: time.Now().Add(-2 * time.Hour)}, // high priority, older
		{UID: "2", Priority: 1, Created: time.Now().Add(-1 * time.Hour)}, // high priority, newer
		{UID: "4", Priority: 5, Created: time.Now()},                     // medium priority
		{UID: "5", Priority: 9, Created: time.Now()},                     // low priority
	}

	gb.SortTasks(tasks)

	// Check order: should be sorted by priority (1, 1, 5, 9, 0)
	// Within same priority, older tasks first
	expectedOrder := []string{"1", "2", "4", "5", "3"}
	for i, expected := range expectedOrder {
		if tasks[i].UID != expected {
			t.Errorf("position %d: got UID %s, want %s", i, tasks[i].UID, expected)
		}
	}
}

// TestGitBackendGetPriorityColor tests priority color coding
func TestGitBackendGetPriorityColor(t *testing.T) {
	gb := &GitBackend{}

	tests := []struct {
		priority int
		want     string
	}{
		{0, ""},         // undefined - no color
		{1, "\033[31m"}, // high - red
		{4, "\033[31m"}, // high - red
		{5, "\033[33m"}, // medium - yellow
		{6, "\033[34m"}, // low - blue
		{9, "\033[34m"}, // low - blue
		{10, ""},        // out of range - no color
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.priority)), func(t *testing.T) {
			got := gb.GetPriorityColor(tt.priority)
			if got != tt.want {
				t.Errorf("GetPriorityColor(%d) = %q, want %q", tt.priority, got, tt.want)
			}
		})
	}
}

// TestGitBackendFilterTasks tests task filtering
func TestGitBackendFilterTasks(t *testing.T) {
	gb := &GitBackend{}

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)

	tasks := []backend.Task{
		{UID: "1", Status: "TODO", DueDate: &tomorrow, Created: yesterday},
		{UID: "2", Status: "DONE", DueDate: &yesterday, Created: yesterday},
		{UID: "3", Status: "TODO", DueDate: &now, Created: now},
	}

	tests := []struct {
		name   string
		filter backend.TaskFilter
		want   int
	}{
		{
			name: "filter by status",
			filter: backend.TaskFilter{
				Statuses: &[]string{"TODO"},
			},
			want: 2, // tasks 1 and 3
		},
		{
			name: "filter by due after",
			filter: backend.TaskFilter{
				DueAfter: &now,
			},
			want: 2, // tasks 1 and 3
		},
		{
			name: "filter by created after",
			filter: backend.TaskFilter{
				CreatedAfter: &now,
			},
			want: 1, // task 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := gb.filterTasks(tasks, &tt.filter)
			if len(filtered) != tt.want {
				t.Errorf("filterTasks() returned %d tasks, want %d", len(filtered), tt.want)
			}
		})
	}
}
