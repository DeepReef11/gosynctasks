package formatters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"gosynctasks/backend"
	"os/exec"
	"strings"
	"time"
)

// PluginFormatter executes external scripts to format field values
type PluginFormatter struct {
	ctx     *FormatContext
	command string
	args    []string
	timeout time.Duration
	env     map[string]string
}

// NewPluginFormatter creates a new plugin formatter
func NewPluginFormatter(ctx *FormatContext, command string, args []string, timeoutMs int, env map[string]string) *PluginFormatter {
	// Set default timeout to 1 second if not specified
	timeout := 1000 * time.Millisecond
	if timeoutMs > 0 {
		timeout = time.Duration(timeoutMs) * time.Millisecond
	}

	// Enforce maximum timeout of 5 seconds
	maxTimeout := 5000 * time.Millisecond
	if timeout > maxTimeout {
		timeout = maxTimeout
	}

	return &PluginFormatter{
		ctx:     ctx,
		command: command,
		args:    args,
		timeout: timeout,
		env:     env,
	}
}

// Format executes the plugin script and returns the formatted output
func (f *PluginFormatter) Format(task backend.Task, format string, width int, color bool) string {
	// Prepare task data as JSON
	taskData, err := f.prepareTaskData(task, format, width, color)
	if err != nil {
		return fmt.Sprintf("[plugin error: %v]", err)
	}

	// Execute the plugin
	output, err := f.executePlugin(taskData)
	if err != nil {
		return fmt.Sprintf("[plugin error: %v]", err)
	}

	// Trim trailing whitespace and return
	return strings.TrimRight(output, "\r\n")
}

// prepareTaskData converts the task to JSON for passing to the plugin
func (f *PluginFormatter) prepareTaskData(task backend.Task, format string, width int, color bool) ([]byte, error) {
	// Create a simplified task representation for JSON serialization
	data := map[string]interface{}{
		"uid":         task.UID,
		"summary":     task.Summary,
		"description": task.Description,
		"status":      task.Status,
		"priority":    task.Priority,
		"categories":  task.Categories,
		"format":      format,
		"width":       width,
		"color":       color,
	}

	// Add date fields (convert to ISO 8601 format)
	if !task.DueDate.IsZero() {
		data["due_date"] = task.DueDate.Format(time.RFC3339)
	}
	if !task.StartDate.IsZero() {
		data["start_date"] = task.StartDate.Format(time.RFC3339)
	}
	if !task.Created.IsZero() {
		data["created"] = task.Created.Format(time.RFC3339)
	}
	if !task.Modified.IsZero() {
		data["modified"] = task.Modified.Format(time.RFC3339)
	}
	if !task.Completed.IsZero() {
		data["completed"] = task.Completed.Format(time.RFC3339)
	}

	// Add parent UID if present
	if task.ParentUID != "" {
		data["parent_uid"] = task.ParentUID
	}

	return json.Marshal(data)
}

// executePlugin runs the external command with timeout and security measures
func (f *PluginFormatter) executePlugin(input []byte) (string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), f.timeout)
	defer cancel()

	// Build command
	cmd := exec.CommandContext(ctx, f.command, f.args...)

	// Set up input/output buffers
	cmd.Stdin = bytes.NewReader(input)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set environment variables if specified
	if len(f.env) > 0 {
		cmd.Env = append(cmd.Environ())
		for key, value := range f.env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	// Execute the command
	err := cmd.Run()

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("plugin timed out after %v", f.timeout)
	}

	// Check for execution errors
	if err != nil {
		// Include stderr if available
		if stderr.Len() > 0 {
			return "", fmt.Errorf("plugin failed: %v (stderr: %s)", err, stderr.String())
		}
		return "", fmt.Errorf("plugin failed: %v", err)
	}

	return stdout.String(), nil
}
