package todoist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// Todoist REST API v2 base URL
	APIBaseURL = "https://api.todoist.com/rest/v2"

	// API rate limit: ~450 requests per 15 minutes
	// We'll implement basic retry logic with exponential backoff
)

// APIClient handles HTTP communication with Todoist REST API v2
type APIClient struct {
	baseURL    string
	apiToken   string
	httpClient *http.Client
}

// NewAPIClient creates a new Todoist API client
func NewAPIClient(apiToken string) *APIClient {
	return &APIClient{
		baseURL:  APIBaseURL,
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Project represents a Todoist project (maps to TaskList)
type Project struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	CommentCount   int    `json:"comment_count"`
	Order          int    `json:"order"`
	Color          string `json:"color"`
	IsShared       bool   `json:"is_shared"`
	IsFavorite     bool   `json:"is_favorite"`
	IsInboxProject bool   `json:"is_inbox_project"`
	IsTeamInbox    bool   `json:"is_team_inbox"`
	ViewStyle      string `json:"view_style"`
	URL            string `json:"url"`
	ParentID       string `json:"parent_id,omitempty"`
}

// TodoistTask represents a task from Todoist API
type TodoistTask struct {
	ID            string   `json:"id"`
	ProjectID     string   `json:"project_id"`
	SectionID     string   `json:"section_id,omitempty"`
	Content       string   `json:"content"`
	Description   string   `json:"description"`
	IsCompleted   bool     `json:"is_completed"`
	Labels        []string `json:"labels"`
	ParentID      string   `json:"parent_id,omitempty"`
	Order         int      `json:"order"`
	Priority      int      `json:"priority"` // 1=normal, 2, 3, 4=urgent
	Due           *Due     `json:"due,omitempty"`
	URL           string   `json:"url"`
	CommentCount  int      `json:"comment_count"`
	CreatedAt     string   `json:"created_at"` // RFC3339 format
	CreatorID     string   `json:"creator_id"`
	AssigneeID    string   `json:"assignee_id,omitempty"`
	AssignerID    string   `json:"assigner_id,omitempty"`
	Duration      *Duration `json:"duration,omitempty"`
}

// Due represents task due date information
type Due struct {
	Date        string `json:"date"`           // YYYY-MM-DD
	String      string `json:"string"`         // Human-readable (e.g., "tomorrow")
	Lang        string `json:"lang,omitempty"` // Language code
	IsRecurring bool   `json:"is_recurring"`
	Datetime    string `json:"datetime,omitempty"` // RFC3339 with time
	Timezone    string `json:"timezone,omitempty"`
}

// Duration represents task duration
type Duration struct {
	Amount int    `json:"amount"`
	Unit   string `json:"unit"` // "minute" or "day"
}

// CreateTaskRequest represents request body for creating a task
type CreateTaskRequest struct {
	Content     string   `json:"content"`
	Description string   `json:"description,omitempty"`
	ProjectID   string   `json:"project_id,omitempty"`
	SectionID   string   `json:"section_id,omitempty"`
	ParentID    string   `json:"parent_id,omitempty"`
	Order       int      `json:"order,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	Priority    int      `json:"priority,omitempty"` // 1-4
	DueString   string   `json:"due_string,omitempty"`
	DueDate     string   `json:"due_date,omitempty"` // YYYY-MM-DD
	DueDatetime string   `json:"due_datetime,omitempty"` // RFC3339
	DueLang     string   `json:"due_lang,omitempty"`
	AssigneeID  string   `json:"assignee_id,omitempty"`
}

// UpdateTaskRequest represents request body for updating a task
type UpdateTaskRequest struct {
	Content     string   `json:"content,omitempty"`
	Description string   `json:"description,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	Priority    int      `json:"priority,omitempty"`
	DueString   string   `json:"due_string,omitempty"`
	DueDate     string   `json:"due_date,omitempty"`
	DueDatetime string   `json:"due_datetime,omitempty"`
	DueLang     string   `json:"due_lang,omitempty"`
	AssigneeID  string   `json:"assignee_id,omitempty"`
}

// CreateProjectRequest represents request body for creating a project
type CreateProjectRequest struct {
	Name       string `json:"name"`
	ParentID   string `json:"parent_id,omitempty"`
	Color      string `json:"color,omitempty"`
	IsFavorite bool   `json:"is_favorite,omitempty"`
	ViewStyle  string `json:"view_style,omitempty"` // "list" or "board"
}

// UpdateProjectRequest represents request body for updating a project
type UpdateProjectRequest struct {
	Name       string `json:"name,omitempty"`
	Color      string `json:"color,omitempty"`
	IsFavorite *bool  `json:"is_favorite,omitempty"`
	ViewStyle  string `json:"view_style,omitempty"`
}

// doRequest performs an HTTP request with authentication
func (c *APIClient) doRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := c.baseURL + endpoint
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// GetProjects retrieves all projects
func (c *APIClient) GetProjects() ([]Project, error) {
	resp, err := c.doRequest("GET", "/projects", nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var projects []Project
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return projects, nil
}

// GetProject retrieves a single project by ID
func (c *APIClient) GetProject(projectID string) (*Project, error) {
	resp, err := c.doRequest("GET", "/projects/"+projectID, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var project Project
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &project, nil
}

// CreateProject creates a new project
func (c *APIClient) CreateProject(req CreateProjectRequest) (*Project, error) {
	resp, err := c.doRequest("POST", "/projects", req)
	if err != nil {
		return nil, err
	}
	
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var project Project
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &project, nil
}

// UpdateProject updates an existing project
func (c *APIClient) UpdateProject(projectID string, req UpdateProjectRequest) error {
	resp, err := c.doRequest("POST", "/projects/"+projectID, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("project not found: %s", projectID)
	}
	// Todoist returns either 200 (with updated project) or 204 (no content)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteProject deletes a project
func (c *APIClient) DeleteProject(projectID string) error {
	resp, err := c.doRequest("DELETE", "/projects/"+projectID, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("project not found: %s", projectID)
	}
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetTasks retrieves all tasks, optionally filtered by project
func (c *APIClient) GetTasks(projectID string) ([]TodoistTask, error) {
	endpoint := "/tasks"
	if projectID != "" {
		endpoint += "?project_id=" + projectID
	}

	resp, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var tasks []TodoistTask
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return tasks, nil
}

// GetTask retrieves a single task by ID
func (c *APIClient) GetTask(taskID string) (*TodoistTask, error) {
	resp, err := c.doRequest("GET", "/tasks/"+taskID, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var task TodoistTask
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &task, nil
}

// CreateTask creates a new task
func (c *APIClient) CreateTask(req CreateTaskRequest) (*TodoistTask, error) {
	resp, err := c.doRequest("POST", "/tasks", req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var task TodoistTask
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &task, nil
}

// UpdateTask updates an existing task
func (c *APIClient) UpdateTask(taskID string, req UpdateTaskRequest) error {
	resp, err := c.doRequest("POST", "/tasks/"+taskID, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("task not found: %s", taskID)
	}
	// Todoist returns either 200 (with updated task) or 204 (no content)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// CloseTask marks a task as completed
func (c *APIClient) CloseTask(taskID string) error {
	resp, err := c.doRequest("POST", "/tasks/"+taskID+"/close", nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("task not found: %s", taskID)
	}
	// Todoist returns either 200 (with updated task) or 204 (no content)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// ReopenTask marks a completed task as not completed
func (c *APIClient) ReopenTask(taskID string) error {
	resp, err := c.doRequest("POST", "/tasks/"+taskID+"/reopen", nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("task not found: %s", taskID)
	}
	// Todoist returns either 200 (with updated task) or 204 (no content)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteTask deletes a task
func (c *APIClient) DeleteTask(taskID string) error {
	resp, err := c.doRequest("DELETE", "/tasks/"+taskID, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("task not found: %s", taskID)
	}
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
