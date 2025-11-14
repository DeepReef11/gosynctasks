package backend

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (nB *NextcloudBackend) parseVTODOs(xmlData string) ([]Task, error) {
	var tasks []Task

	// Extract VTODO blocks from XML
	vtodoBlocks := extractVTODOBlocks(xmlData)

	for _, vtodo := range vtodoBlocks {
		task, err := parseVTODO(vtodo)
		if err != nil {
			continue // Skip invalid tasks
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func extractVTODOBlocks(xmlData string) []string {
	var blocks []string
	lines := strings.Split(xmlData, "\n")

	var currentBlock strings.Builder
	inVTODO := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "BEGIN:VTODO") {
			inVTODO = true
			currentBlock.Reset()
			currentBlock.WriteString(line + "\n")
		} else if strings.HasPrefix(line, "END:VTODO") && inVTODO {
			currentBlock.WriteString(line + "\n")
			blocks = append(blocks, currentBlock.String())
			inVTODO = false
		} else if inVTODO {
			currentBlock.WriteString(line + "\n")
		}
	}

	return blocks
}

func parseVTODO(vtodo string) (Task, error) {
	task := Task{
		Status:   "NEEDS-ACTION",
		Priority: 0,
		Created:  time.Now(),
		Modified: time.Now(),
	}

	lines := strings.SplitSeq(vtodo, "\n")

	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		// Handle parameters (e.g., DTSTART;VALUE=DATE:20240101)
		if strings.Contains(key, ";") {
			key = strings.Split(key, ";")[0]
		}

		switch key {
		case "UID":
			task.UID = value
		case "SUMMARY":
			task.Summary = unescapeText(value)
		case "DESCRIPTION":
			task.Description = unescapeText(value)
		case "STATUS":
			task.Status = value
		case "PRIORITY":
			if p := parseInt(value); p >= 0 && p <= 9 {
				task.Priority = p
			}
		case "CREATED":
			if t, err := parseICalTime(value); err == nil {
				task.Created = t
			}
		case "LAST-MODIFIED":
			if t, err := parseICalTime(value); err == nil {
				task.Modified = t
			}
		case "DUE":
			if t, err := parseICalTime(value); err == nil {
				task.DueDate = &t
			}
		case "DTSTART":
			if t, err := parseICalTime(value); err == nil {
				task.StartDate = &t
			}
		case "COMPLETED":
			if t, err := parseICalTime(value); err == nil {
				task.Completed = &t
			}
		case "CATEGORIES":
			task.Categories = strings.Split(unescapeText(value), ",")
		case "RELATED-TO":
			task.ParentUID = value
		}
	}

	if task.UID == "" {
		return task, fmt.Errorf("missing UID")
	}

	return task, nil
}

func parseICalTime(value string) (time.Time, error) {
	// Handle different iCal time formats
	formats := []string{
		"20060102T150405Z", // UTC
		"20060102T150405",  // Local
		"20060102",         // Date only
	}

	for _, format := range formats {
		if t, err := time.Parse(format, value); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid time format: %s", value)
}

func unescapeText(text string) string {
	text = strings.ReplaceAll(text, "\\n", "\n")
	text = strings.ReplaceAll(text, "\\,", ",")
	text = strings.ReplaceAll(text, "\\;", ";")
	text = strings.ReplaceAll(text, "\\\\", "\\")
	return text
}

func parseInt(s string) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return 0
}

func (nB *NextcloudBackend) parseTaskLists(xmlData, baseURL string) ([]TaskList, error) {
	var taskLists []TaskList

	responses := extractResponses(xmlData)

	for _, response := range responses {
		// Only include calendars with 200 OK status and VTODO support
		if !strings.Contains(response, "HTTP/1.1 200 OK") {
			continue
		}

		taskList := parseTaskListResponse(response, baseURL)

		// Skip trashbin, inbox, outbox, and other special collections
		if taskList.ID == "trashbin" || taskList.ID == "inbox" || taskList.ID == "outbox" {
			continue
		}

		// Skip trashed/deleted calendars (have nc:deleted-at set)
		if taskList.DeletedAt != "" {
			continue
		}

		// Only include calendars that actually support VTODO
		if taskList.ID != "" && strings.Contains(response, `<cal:comp name="VTODO"/>`) {
			taskLists = append(taskLists, taskList)
		}
	}

	return taskLists, nil
}

func containsVTODO(response string) bool {
	return strings.Contains(response, `<cal:comp name="VTODO"/>`)
}

func extractResponses(xmlData string) []string {
	var responses []string

	// Try different response tag patterns
	patterns := []string{
		"<d:response>",
		"<response>",
		"<D:response>",
	}

	for _, startTag := range patterns {
		endTag := strings.Replace(startTag, "<", "</", 1)

		data := xmlData
		for {
			start := strings.Index(data, startTag)
			if start == -1 {
				break
			}

			end := strings.Index(data[start:], endTag)
			if end == -1 {
				break
			}

			response := data[start : start+end+len(endTag)]
			responses = append(responses, response)
			data = data[start+end+len(endTag):]
		}

		if len(responses) > 0 {
			break
		}
	}

	// fmt.Printf("extractResponses found %d responses\n", len(responses))
	return responses
}

func parseTaskListResponse(response, baseURL string) TaskList {
	taskList := TaskList{}

	// Extract href (calendar ID)
	if href := extractXMLValue(response, "href"); href != "" {
		// Extract calendar ID from href path
		parts := strings.Split(strings.Trim(href, "/"), "/")
		if len(parts) > 0 {
			taskList.ID = parts[len(parts)-1]
		}
		taskList.URL = href
	}

	// Extract displayname
	taskList.Name = extractXMLValue(response, "displayname")

	// Extract ctag
	taskList.CTags = extractXMLValue(response, "getctag")

	// Extract color
	taskList.Color = extractXMLValue(response, "calendar-color")

	// Extract deleted-at timestamp (Nextcloud trash)
	taskList.DeletedAt = extractXMLValue(response, "deleted-at")

	return taskList
}

func extractXMLValue(xml, tag string) string {
	// Try without namespace prefix first
	if start := strings.Index(xml, fmt.Sprintf("<%s>", tag)); start != -1 {
		start += len(tag) + 2
		if end := strings.Index(xml[start:], fmt.Sprintf("</%s>", tag)); end != -1 {
			return strings.TrimSpace(xml[start : start+end])
		}
	}

	// Try with namespace prefixes
	for _, prefix := range []string{"d:", "cs:", "ic:"} {
		fullTag := prefix + tag
		if start := strings.Index(xml, fmt.Sprintf("<%s>", fullTag)); start != -1 {
			start += len(fullTag) + 2
			if end := strings.Index(xml[start:], fmt.Sprintf("</%s>", fullTag)); end != -1 {
				return strings.TrimSpace(xml[start : start+end])
			}
		}
	}

	return ""
}
