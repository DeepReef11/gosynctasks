package todoist

import (
	"encoding/json"
	"gosynctasks/backend"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

)

// mockTodoistServer creates a test HTTP server that mimics Todoist API
func mockTodoistServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authorization header
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch {
		// GET /projects - List projects
		case r.Method == "GET" && r.URL.Path == "/projects":
			projects := []Project{
				{
					ID:             "project1",
					Name:           "Test Project 1",
					CommentCount:   5,
					Order:          1,
					Color:          "red",
					IsFavorite:     true,
					IsInboxProject: false,
				},
				{
					ID:             "project2",
					Name:           "Test Project 2",
					CommentCount:   2,
					Order:          2,
					Color:          "blue",
					IsFavorite:     false,
					IsInboxProject: false,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(projects)

		// GET /projects/{id} - Get project
		case r.Method == "GET" && r.URL.Path == "/projects/project1":
			project := Project{
				ID:             "project1",
				Name:           "Test Project 1",
				CommentCount:   5,
				Order:          1,
				Color:          "red",
				IsFavorite:     true,
				IsInboxProject: false,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(project)

		// POST /projects - Create project
		case r.Method == "POST" && r.URL.Path == "/projects":
			var req CreateProjectRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			project := Project{
				ID:             "new-project-id",
				Name:           req.Name,
				Color:          req.Color,
				CommentCount:   0,
				Order:          1,
				IsFavorite:     req.IsFavorite,
				IsInboxProject: false,
			}
			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(project)

		// POST /projects/{id} - Update project
		case r.Method == "POST" && r.URL.Path == "/projects/project1":
			var req UpdateProjectRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)

		// DELETE /projects/{id} - Delete project
		case r.Method == "DELETE" && r.URL.Path == "/projects/project1":
			w.WriteHeader(http.StatusNoContent)

		// GET /tasks - List tasks
		case r.Method == "GET" && r.URL.Path == "/tasks":
			projectID := r.URL.Query().Get("project_id")
			tasks := []TodoistTask{
				{
					ID:          "task1",
					ProjectID:   projectID,
					Content:     "Test Task 1",
					Description: "Description 1",
					IsCompleted: false,
					Labels:      []string{"label1", "label2"},
					Priority:    4, // Urgent
					CreatedAt:   time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
					Due: &Due{
						Date:     "2026-01-15",
						String:   "Jan 15",
						Datetime: "2026-01-15T12:00:00Z",
					},
				},
				{
					ID:          "task2",
					ProjectID:   projectID,
					Content:     "Test Task 2",
					Description: "Description 2",
					IsCompleted: true,
					Labels:      []string{"label3"},
					Priority:    2, // Medium
					CreatedAt:   time.Now().Add(-48 * time.Hour).Format(time.RFC3339),
					ParentID:    "task1",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tasks)

		// GET /tasks/{id} - Get task
		case r.Method == "GET" && r.URL.Path == "/tasks/task1":
			task := TodoistTask{
				ID:          "task1",
				ProjectID:   "project1",
				Content:     "Test Task 1",
				Description: "Description 1",
				IsCompleted: false,
				Labels:      []string{"label1", "label2"},
				Priority:    4,
				CreatedAt:   time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(task)

		// POST /tasks - Create task
		case r.Method == "POST" && r.URL.Path == "/tasks":
			var req CreateTaskRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			task := TodoistTask{
				ID:          "new-task-id",
				ProjectID:   req.ProjectID,
				Content:     req.Content,
				Description: req.Description,
				IsCompleted: false,
				Labels:      req.Labels,
				Priority:    req.Priority,
				ParentID:    req.ParentID,
				CreatedAt:   time.Now().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(task)

		// POST /tasks/{id} - Update task
		case r.Method == "POST" && r.URL.Path == "/tasks/task1":
			var req UpdateTaskRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)

		// POST /tasks/{id}/close - Close task
		case r.Method == "POST" && r.URL.Path == "/tasks/task1/close":
			w.WriteHeader(http.StatusNoContent)

		// POST /tasks/{id}/reopen - Reopen task
		case r.Method == "POST" && r.URL.Path == "/tasks/task1/reopen":
			w.WriteHeader(http.StatusNoContent)

		// DELETE /tasks/{id} - Delete task
		case r.Method == "DELETE" && r.URL.Path == "/tasks/task1":
			w.WriteHeader(http.StatusNoContent)

		case r.Method == "DELETE" && r.URL.Path == "/tasks/nonexistent":
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"Task not found"}`))

		default:
			t.Logf("Unhandled request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestTodoistBackend_GetTaskLists_Mock(t *testing.T) {
	server := mockTodoistServer(t)
	defer server.Close()

	// Create backend with mocked server
	config := backend.BackendConfig{
		Type:     "todoist",
		Enabled:  true,
		APIToken: "test-token",
	}

	tb := &TodoistBackend{
		config:   config,
		apiToken: "test-token",
		apiClient: &APIClient{
			baseURL:    server.URL,
			apiToken:   "test-token",
			httpClient: &http.Client{},
		},
	}

	lists, err := tb.GetTaskLists()
	if err != nil {
		t.Fatalf("GetTaskLists() error = %v", err)
	}

	if len(lists) != 2 {
		t.Errorf("GetTaskLists() returned %d lists, want 2", len(lists))
	}

	if lists[0].Name != "Test Project 1" {
		t.Errorf("First list name = %q, want %q", lists[0].Name, "Test Project 1")
	}

	if lists[1].Color != "blue" {
		t.Errorf("Second list color = %q, want %q", lists[1].Color, "blue")
	}
}

func TestTodoistBackend_GetTasks_Mock(t *testing.T) {
	server := mockTodoistServer(t)
	defer server.Close()

	config := backend.BackendConfig{
		Type:     "todoist",
		Enabled:  true,
		APIToken: "test-token",
	}

	tb := &TodoistBackend{
		config:   config,
		apiToken: "test-token",
		apiClient: &APIClient{
			baseURL:    server.URL,
			apiToken:   "test-token",
			httpClient: &http.Client{},
		},
	}

	tasks, err := tb.GetTasks("project1", nil)
	if err != nil {
		t.Fatalf("GetTasks() error = %v", err)
	}

	if len(tasks) != 2 {
		t.Fatalf("GetTasks() returned %d tasks, want 2", len(tasks))
	}

	// Check task 1
	if tasks[0].Summary != "Test Task 1" {
		t.Errorf("Task 1 summary = %q, want %q", tasks[0].Summary, "Test Task 1")
	}
	if tasks[0].Status != "TODO" {
		t.Errorf("Task 1 status = %q, want %q", tasks[0].Status, "TODO")
	}
	if tasks[0].Priority != 1 { // Todoist priority 4 maps to 1
		t.Errorf("Task 1 priority = %d, want %d", tasks[0].Priority, 1)
	}
	if tasks[0].DueDate == nil {
		t.Error("Task 1 due date is nil, want non-nil")
	}

	// Check task 2 (subtask)
	if tasks[1].Status != "DONE" {
		t.Errorf("Task 2 status = %q, want %q", tasks[1].Status, "DONE")
	}
	if tasks[1].ParentUID != "task1" {
		t.Errorf("Task 2 parent = %q, want %q", tasks[1].ParentUID, "task1")
	}
}

func TestTodoistBackend_GetTasks_WithFilter(t *testing.T) {
	server := mockTodoistServer(t)
	defer server.Close()

	config := backend.BackendConfig{
		Type:     "todoist",
		Enabled:  true,
		APIToken: "test-token",
	}

	tb := &TodoistBackend{
		config:   config,
		apiToken: "test-token",
		apiClient: &APIClient{
			baseURL:    server.URL,
			apiToken:   "test-token",
			httpClient: &http.Client{},
		},
	}

	// Filter for TODO tasks only
	statuses := []string{"TODO"}
	filter := &backend.TaskFilter{
		Statuses: &statuses,
	}

	tasks, err := tb.GetTasks("project1", filter)
	if err != nil {
		t.Fatalf("GetTasks() error = %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("GetTasks() with TODO filter returned %d tasks, want 1", len(tasks))
	}

	if tasks[0].Status != "TODO" {
		t.Errorf("Filtered task status = %q, want %q", tasks[0].Status, "TODO")
	}
}

func TestTodoistBackend_FindTasksBySummary_Mock(t *testing.T) {
	server := mockTodoistServer(t)
	defer server.Close()

	config := backend.BackendConfig{
		Type:     "todoist",
		Enabled:  true,
		APIToken: "test-token",
	}

	tb := &TodoistBackend{
		config:   config,
		apiToken: "test-token",
		apiClient: &APIClient{
			baseURL:    server.URL,
			apiToken:   "test-token",
			httpClient: &http.Client{},
		},
	}

	// Search for "Task 1"
	tasks, err := tb.FindTasksBySummary("project1", "Task 1")
	if err != nil {
		t.Fatalf("FindTasksBySummary() error = %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("FindTasksBySummary() returned %d tasks, want 1", len(tasks))
	}

	if tasks[0].Summary != "Test Task 1" {
		t.Errorf("Found task summary = %q, want %q", tasks[0].Summary, "Test Task 1")
	}
}

func TestTodoistBackend_AddTask_Mock(t *testing.T) {
	server := mockTodoistServer(t)
	defer server.Close()

	config := backend.BackendConfig{
		Type:     "todoist",
		Enabled:  true,
		APIToken: "test-token",
	}

	tb := &TodoistBackend{
		config:   config,
		apiToken: "test-token",
		apiClient: &APIClient{
			baseURL:    server.URL,
			apiToken:   "test-token",
			httpClient: &http.Client{},
		},
	}

	dueDate := time.Now().Add(24 * time.Hour)
	newTask := backend.Task{
		Summary:     "New Task",
		Description: "New Description",
		Status:      "TODO",
		Priority:    1, // High priority
		Categories:  []string{"urgent"},
		DueDate:     &dueDate,
	}

	_, err := tb.AddTask("project1", newTask)
	if err != nil {
		t.Fatalf("AddTask() error = %v", err)
	}
}

func TestTodoistBackend_UpdateTask_Mock(t *testing.T) {
	server := mockTodoistServer(t)
	defer server.Close()

	config := backend.BackendConfig{
		Type:     "todoist",
		Enabled:  true,
		APIToken: "test-token",
	}

	tb := &TodoistBackend{
		config:   config,
		apiToken: "test-token",
		apiClient: &APIClient{
			baseURL:    server.URL,
			apiToken:   "test-token",
			httpClient: &http.Client{},
		},
	}

	updatedTask := backend.Task{
		UID:         "task1",
		Summary:     "Updated Task",
		Description: "Updated Description",
		Status:      "DONE",
		Priority:    3,
	}

	err := tb.UpdateTask("project1", updatedTask)
	if err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
}

func TestTodoistBackend_DeleteTask_Mock(t *testing.T) {
	server := mockTodoistServer(t)
	defer server.Close()

	config := backend.BackendConfig{
		Type:     "todoist",
		Enabled:  true,
		APIToken: "test-token",
	}

	tb := &TodoistBackend{
		config:   config,
		apiToken: "test-token",
		apiClient: &APIClient{
			baseURL:    server.URL,
			apiToken:   "test-token",
			httpClient: &http.Client{},
		},
	}

	err := tb.DeleteTask("project1", "task1")
	if err != nil {
		t.Fatalf("DeleteTask() error = %v", err)
	}
}

func TestTodoistBackend_DeleteTask_NotFound(t *testing.T) {
	server := mockTodoistServer(t)
	defer server.Close()

	config := backend.BackendConfig{
		Type:     "todoist",
		Enabled:  true,
		APIToken: "test-token",
	}

	tb := &TodoistBackend{
		config:   config,
		apiToken: "test-token",
		apiClient: &APIClient{
			baseURL:    server.URL,
			apiToken:   "test-token",
			httpClient: &http.Client{},
		},
	}

	err := tb.DeleteTask("project1", "nonexistent")
	if err == nil {
		t.Fatal("DeleteTask() expected error for nonexistent task, got nil")
	}
}

func TestTodoistBackend_CreateTaskList_Mock(t *testing.T) {
	server := mockTodoistServer(t)
	defer server.Close()

	config := backend.BackendConfig{
		Type:     "todoist",
		Enabled:  true,
		APIToken: "test-token",
	}

	tb := &TodoistBackend{
		config:   config,
		apiToken: "test-token",
		apiClient: &APIClient{
			baseURL:    server.URL,
			apiToken:   "test-token",
			httpClient: &http.Client{},
		},
	}

	listID, err := tb.CreateTaskList("New Project", "Project description", "green")
	if err != nil {
		t.Fatalf("CreateTaskList() error = %v", err)
	}

	if listID == "" {
		t.Error("CreateTaskList() returned empty list ID")
	}
}

func TestTodoistBackend_DeleteTaskList_Mock(t *testing.T) {
	server := mockTodoistServer(t)
	defer server.Close()

	config := backend.BackendConfig{
		Type:     "todoist",
		Enabled:  true,
		APIToken: "test-token",
	}

	tb := &TodoistBackend{
		config:   config,
		apiToken: "test-token",
		apiClient: &APIClient{
			baseURL:    server.URL,
			apiToken:   "test-token",
			httpClient: &http.Client{},
		},
	}

	err := tb.DeleteTaskList("project1")
	if err != nil {
		t.Fatalf("DeleteTaskList() error = %v", err)
	}
}

func TestTodoistBackend_RenameTaskList_Mock(t *testing.T) {
	server := mockTodoistServer(t)
	defer server.Close()

	config := backend.BackendConfig{
		Type:     "todoist",
		Enabled:  true,
		APIToken: "test-token",
	}

	tb := &TodoistBackend{
		config:   config,
		apiToken: "test-token",
		apiClient: &APIClient{
			baseURL:    server.URL,
			apiToken:   "test-token",
			httpClient: &http.Client{},
		},
	}

	err := tb.RenameTaskList("project1", "Renamed Project")
	if err != nil {
		t.Fatalf("RenameTaskList() error = %v", err)
	}
}

func TestTodoistBackend_SortTasks(t *testing.T) {
	tb := &TodoistBackend{}

	now := time.Now()
	tasks := []backend.Task{
		{Summary: "Low priority", Priority: 7, Created: now},
		{Summary: "High priority", Priority: 1, Created: now.Add(-time.Hour)},
		{Summary: "Medium priority", Priority: 5, Created: now.Add(-2 * time.Hour)},
		{Summary: "No priority", Priority: 0, Created: now.Add(-3 * time.Hour)},
	}

	tb.SortTasks(tasks)

	// After sorting: priority 1, 5, 7, 0 (0 goes last)
	if tasks[0].Priority != 1 {
		t.Errorf("First task priority = %d, want 1", tasks[0].Priority)
	}
	if tasks[1].Priority != 5 {
		t.Errorf("Second task priority = %d, want 5", tasks[1].Priority)
	}
	if tasks[2].Priority != 7 {
		t.Errorf("Third task priority = %d, want 7", tasks[2].Priority)
	}
	if tasks[3].Priority != 0 {
		t.Errorf("Fourth task priority = %d, want 0", tasks[3].Priority)
	}
}

func TestTodoistBackend_GetDeletedTaskLists(t *testing.T) {
	tb := &TodoistBackend{}

	lists, err := tb.GetDeletedTaskLists()
	if err != nil {
		t.Fatalf("GetDeletedTaskLists() error = %v", err)
	}

	if len(lists) != 0 {
		t.Errorf("GetDeletedTaskLists() returned %d lists, want 0 (not supported)", len(lists))
	}
}

func TestTodoistBackend_RestoreTaskList(t *testing.T) {
	tb := &TodoistBackend{}

	err := tb.RestoreTaskList("project1")
	if err == nil {
		t.Error("RestoreTaskList() expected error (not supported), got nil")
	}
}
