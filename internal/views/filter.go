package views

import (
	"gosynctasks/backend"
	"sort"
	"strings"
	"time"
)

// ApplyFilters filters tasks based on view filter configuration
func ApplyFilters(tasks []backend.Task, filters *ViewFilters) []backend.Task {
	if filters == nil {
		return tasks
	}

	var filtered []backend.Task

	for _, task := range tasks {
		if matchesFilters(task, filters) {
			filtered = append(filtered, task)
		}
	}

	return filtered
}

// matchesFilters checks if a task matches all filter criteria
func matchesFilters(task backend.Task, filters *ViewFilters) bool {
	// Status filter
	if len(filters.Status) > 0 {
		matched := false
		for _, status := range filters.Status {
			if strings.EqualFold(task.Status, status) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Priority filter
	if filters.PriorityMin > 0 || filters.PriorityMax > 0 {
		// If only min is set, check >= min
		if filters.PriorityMin > 0 && filters.PriorityMax == 0 {
			if task.Priority < filters.PriorityMin {
				return false
			}
		}
		// If only max is set, check <= max
		if filters.PriorityMax > 0 && filters.PriorityMin == 0 {
			if task.Priority > filters.PriorityMax {
				return false
			}
		}
		// If both are set, check range
		if filters.PriorityMin > 0 && filters.PriorityMax > 0 {
			if task.Priority < filters.PriorityMin || task.Priority > filters.PriorityMax {
				return false
			}
		}
	}

	// Tags filter (task must have all specified tags)
	if len(filters.Tags) > 0 {
		taskTags := make(map[string]bool)
		for _, tag := range task.Categories {
			taskTags[strings.ToLower(tag)] = true
		}

		for _, requiredTag := range filters.Tags {
			if !taskTags[strings.ToLower(requiredTag)] {
				return false
			}
		}
	}

	// Due date filters
	if filters.DueBefore != nil {
		if task.DueDate == nil || !task.DueDate.Before(*filters.DueBefore) {
			return false
		}
	}

	if filters.DueAfter != nil {
		if task.DueDate == nil || !task.DueDate.After(*filters.DueAfter) {
			return false
		}
	}

	// Start date filters
	if filters.StartBefore != nil {
		if task.StartDate == nil || !task.StartDate.Before(*filters.StartBefore) {
			return false
		}
	}

	if filters.StartAfter != nil {
		if task.StartDate == nil || !task.StartDate.After(*filters.StartAfter) {
			return false
		}
	}

	return true
}

// ApplySort sorts tasks based on view sort configuration
func ApplySort(tasks []backend.Task, sortBy string, sortOrder string) {
	if sortBy == "" {
		return
	}

	// Normalize sort order
	ascending := true
	if strings.ToLower(sortOrder) == "desc" {
		ascending = false
	}

	sort.Slice(tasks, func(i, j int) bool {
		var less bool

		switch sortBy {
		case "status":
			less = tasks[i].Status < tasks[j].Status
		case "summary":
			less = strings.ToLower(tasks[i].Summary) < strings.ToLower(tasks[j].Summary)
		case "priority":
			// Lower priority number = higher priority (1 is highest)
			// 0 means undefined, should go last
			pi, pj := tasks[i].Priority, tasks[j].Priority
			if pi == 0 && pj == 0 {
				less = false
			} else if pi == 0 {
				less = false // undefined goes last
			} else if pj == 0 {
				less = true // undefined goes last
			} else {
				less = pi < pj
			}
		case "due_date":
			less = compareDates(tasks[i].DueDate, tasks[j].DueDate, true)
		case "start_date":
			less = compareDates(tasks[i].StartDate, tasks[j].StartDate, true)
		case "created":
			less = compareDates(&tasks[i].Created, &tasks[j].Created, true)
		case "modified":
			less = compareDates(&tasks[i].Modified, &tasks[j].Modified, true)
		default:
			less = false
		}

		if ascending {
			return less
		}
		return !less
	})
}

// compareDates compares two date pointers, handling nil values
// nilsLast determines whether nil values should be considered greater than non-nil
func compareDates(a, b *time.Time, nilsLast bool) bool {
	if a == nil && b == nil {
		return false
	}
	if a == nil {
		return !nilsLast
	}
	if b == nil {
		return nilsLast
	}
	return a.Before(*b)
}
