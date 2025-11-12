package backend

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// MarkdownParser parses markdown task files into Task structures.
type MarkdownParser struct {
	// Regex patterns for parsing
	checkboxPattern *regexp.Regexp
	tagPattern      *regexp.Regexp
}

// NewMarkdownParser creates a new markdown parser.
func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{
		// Matches: - [ ] Task summary @tag:value @tag2:value2
		checkboxPattern: regexp.MustCompile(`^-\s+\[([ xX>\-])\]\s+(.+)$`),
		// Matches: @tag:value
		tagPattern: regexp.MustCompile(`@(\w+):([^\s]+)`),
	}
}

// Parse parses markdown content into task lists.
func (p *MarkdownParser) Parse(content string) (map[string][]Task, error) {
	lines := strings.Split(content, "\n")
	taskLists := make(map[string][]Task)
	currentList := "Default"
	var currentTask *Task
	var descriptionLines []string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for list header (## Header)
		if strings.HasPrefix(trimmed, "## ") {
			// Save any pending task description
			if currentTask != nil && len(descriptionLines) > 0 {
				currentTask.Description = strings.Join(descriptionLines, "\n")
				descriptionLines = nil
			}
			currentTask = nil

			currentList = strings.TrimSpace(trimmed[3:])
			if currentList == "" {
				currentList = fmt.Sprintf("List-%d", i)
			}
			continue
		}

		// Check for task checkbox
		if matches := p.checkboxPattern.FindStringSubmatch(trimmed); matches != nil {
			// Save any pending task description
			if currentTask != nil && len(descriptionLines) > 0 {
				currentTask.Description = strings.Join(descriptionLines, "\n")
				descriptionLines = nil
			}

			// Parse task
			statusChar := matches[1]
			rest := matches[2]

			task := Task{
				Status:   p.parseStatus(statusChar),
				Created:  time.Now(),
				Modified: time.Now(),
			}

			// Extract tags and summary
			summary, tags := p.extractTags(rest)
			task.Summary = summary

			// Apply tags
			for key, value := range tags {
				switch key {
				case "uid":
					task.UID = value
				case "priority":
					fmt.Sscanf(value, "%d", &task.Priority)
				case "due":
					if t, err := time.Parse("2006-01-02", value); err == nil {
						task.DueDate = &t
					}
				case "start":
					if t, err := time.Parse("2006-01-02", value); err == nil {
						task.StartDate = &t
					}
				case "created":
					if t, err := time.Parse("2006-01-02", value); err == nil {
						task.Created = t
					}
				case "completed":
					if t, err := time.Parse("2006-01-02", value); err == nil {
						task.Completed = &t
					}
				case "status":
					task.Status = value
				}
			}

			// Add to current list
			taskLists[currentList] = append(taskLists[currentList], task)
			currentTask = &taskLists[currentList][len(taskLists[currentList])-1]
			continue
		}

		// Check for task description (indented lines following a task)
		if currentTask != nil && (strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t")) {
			descLine := strings.TrimSpace(line)
			if descLine != "" {
				descriptionLines = append(descriptionLines, descLine)
			}
			continue
		}

		// Empty line or other content ends current task
		if currentTask != nil && len(descriptionLines) > 0 {
			currentTask.Description = strings.Join(descriptionLines, "\n")
			descriptionLines = nil
			currentTask = nil
		}
	}

	// Save any final task description
	if currentTask != nil && len(descriptionLines) > 0 {
		currentTask.Description = strings.Join(descriptionLines, "\n")
	}

	return taskLists, nil
}

// parseStatus converts markdown checkbox status to task status.
func (p *MarkdownParser) parseStatus(statusChar string) string {
	switch statusChar {
	case "x", "X":
		return "DONE"
	case ">":
		return "PROCESSING"
	case "-":
		return "CANCELLED"
	default:
		return "TODO"
	}
}

// extractTags extracts @tag:value pairs from text and returns cleaned summary and tags.
func (p *MarkdownParser) extractTags(text string) (string, map[string]string) {
	tags := make(map[string]string)

	// Find all tags
	matches := p.tagPattern.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) == 3 {
			key := strings.ToLower(match[1])
			value := match[2]
			tags[key] = value
		}
	}

	// Remove tags from summary
	summary := p.tagPattern.ReplaceAllString(text, "")
	summary = strings.TrimSpace(summary)

	return summary, tags
}
