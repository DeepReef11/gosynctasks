package nextcloud

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"gosynctasks/backend"
)

func init() {
	// Register Nextcloud backend for URL scheme "nextcloud://"
	backend.RegisterScheme("nextcloud", NewNextcloudBackend)

	// Register Nextcloud backend for config type "nextcloud"
	backend.RegisterType("nextcloud", newNextcloudBackendFromBackendConfig)
}

type NextcloudBackend struct {
	Connector backend.ConnectorConfig
	username  string
	password  string
	baseURL   string
	client    *http.Client
}

// Status mapping: user-friendly names and abbreviations to CalDAV standard
var statusToCalDAV = map[string]string{
	"T":            "NEEDS-ACTION",
	"TODO":         "NEEDS-ACTION",
	"D":            "COMPLETED",
	"DONE":         "COMPLETED",
	"P":            "IN-PROCESS",
	"PROCESSING":   "IN-PROCESS",
	"C":            "CANCELLED",
	"CANCELLED":    "CANCELLED",
	"NEEDS-ACTION": "NEEDS-ACTION",
	"COMPLETED":    "COMPLETED",
	"IN-PROCESS":   "IN-PROCESS",
}

// CalDAV status to display name mapping
var calDAVToDisplay = map[string]string{
	"NEEDS-ACTION": "TODO",
	"COMPLETED":    "DONE",
	"IN-PROCESS":   "PROCESSING",
	"CANCELLED":    "CANCELLED",
}

func (nB *NextcloudBackend) getClient() *http.Client {
	if nB.client == nil {
		nB.client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig:     &tls.Config{InsecureSkipVerify: nB.Connector.InsecureSkipVerify},
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 2,
				IdleConnTimeout:     30 * time.Second,
			},
			Timeout: 30 * time.Second,
		}
	}
	return nB.client
}

func (nB *NextcloudBackend) getUsername() string {
	if nB.username == "" {
		if nB.Connector.URL != nil && nB.Connector.URL.User != nil {
			nB.username = nB.Connector.URL.User.Username()
		}
	}
	return nB.username
}

func (nB *NextcloudBackend) getPassword() string {
	if nB.password == "" {
		if nB.Connector.URL != nil && nB.Connector.URL.User != nil {
			nB.password, _ = nB.Connector.URL.User.Password()
		}
	}
	return nB.password
}

func (nB *NextcloudBackend) getBaseURL() string {
	if nB.baseURL == "" {
		if nB.Connector.URL != nil {
			// SECURITY: Always use HTTPS by default for Nextcloud connections
			// HTTP is only allowed if explicitly configured via AllowHTTP flag
			protocol := "https"
			host := nB.Connector.URL.Host

			// Only use HTTP if explicitly allowed in config
			if nB.Connector.AllowHTTP {
				// Check if port is specified and is a common HTTP port
				if strings.Contains(host, ":80") || strings.Contains(host, ":8080") || strings.Contains(host, ":8000") {
					protocol = "http"
				}
			}
			// Otherwise, always use HTTPS regardless of port

			nB.baseURL = fmt.Sprintf("%s://%s", protocol, host)
		}
	}
	return nB.baseURL
}

// buildCalendarURL constructs the CalDAV calendar collection URL
func (nB *NextcloudBackend) buildCalendarURL() string {
	return fmt.Sprintf("%s/remote.php/dav/calendars/%s/", nB.getBaseURL(), nB.getUsername())
}

// buildListURL constructs the CalDAV URL for a specific task list
func (nB *NextcloudBackend) buildListURL(listID string) string {
	return fmt.Sprintf("%s/remote.php/dav/calendars/%s/%s/", nB.getBaseURL(), nB.getUsername(), listID)
}

// buildTaskURL constructs the CalDAV URL for a specific task
func (nB *NextcloudBackend) buildTaskURL(listID, taskUID string) string {
	return fmt.Sprintf("%s/remote.php/dav/calendars/%s/%s/%s.ics", nB.getBaseURL(), nB.getUsername(), listID, taskUID)
}

// makeAuthenticatedRequest creates and executes an authenticated HTTP request
func (nB *NextcloudBackend) makeAuthenticatedRequest(method, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set basic auth
	req.SetBasicAuth(nB.getUsername(), nB.getPassword())

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Send request
	client := nB.getClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	return resp, nil
}

// checkHTTPResponse checks HTTP response status and returns appropriate errors
func (nB *NextcloudBackend) checkHTTPResponse(resp *http.Response, operation string, allowedStatuses ...int) error {
	// If specific statuses are allowed, check against them
	if len(allowedStatuses) > 0 {
		for _, status := range allowedStatuses {
			if resp.StatusCode == status {
				return nil
			}
		}
	} else {
		// Default: allow 2xx status codes
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
	}

	// Read response body for error details
	body, _ := io.ReadAll(resp.Body)

	// Common error cases
	switch resp.StatusCode {
	case 401, 403:
		return backend.NewBackendError(operation, resp.StatusCode, "Authentication failed. Please check your username and password in the config file").
			WithBody(string(body))
	case 404:
		return backend.NewBackendError(operation, resp.StatusCode, "Resource not found. Please check your configuration").
			WithBody(string(body))
	case 405:
		return backend.NewBackendError(operation, resp.StatusCode, "Operation not allowed or resource already exists").
			WithBody(string(body))
	default:
		return backend.NewBackendError(operation, resp.StatusCode, resp.Status).
			WithBody(string(body))
	}
}

func (nB *NextcloudBackend) buildCalendarQuery(filter *backend.TaskFilter) string {
	query := `<?xml version="1.0" encoding="utf-8" ?>
<c:calendar-query xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:prop>
    <d:getetag />
    <c:calendar-data />
  </d:prop>
  <c:filter>
    <c:comp-filter name="VCALENDAR">
      <c:comp-filter name="VTODO">`

	if filter != nil {
		if filter.Statuses != nil {
			// Statuses are already in CalDAV format from BuildFilter
			for _, status := range *filter.Statuses {
				if status == "NEEDS-ACTION" {
					query += `<c:prop-filter name="COMPLETED">
  <c:is-not-defined/>
</c:prop-filter>`
				} else {
					query += fmt.Sprintf(`<c:prop-filter name="STATUS">
          <c:text-match><![CDATA[%s]]></c:text-match>
        </c:prop-filter>`, status)

				}
			}
		}

		if filter.DueAfter != nil || filter.DueBefore != nil {
			query += `
        <c:prop-filter name="DUE">`
			if filter.DueAfter != nil {
				query += fmt.Sprintf(`
          <c:time-range start="%s"/>`, filter.DueAfter.Format("20060102T150405Z"))
			}
			if filter.DueBefore != nil {
				query += fmt.Sprintf(`
          <c:time-range end="%s"/>`, filter.DueBefore.Format("20060102T150405Z"))
			}
			query += `
        </c:prop-filter>`
		}
	}

	query += `
      </c:comp-filter>
    </c:comp-filter>
  </c:filter>
</c:calendar-query>`

	return query
}
func (nB *NextcloudBackend) GetTasks(listID string, taskFilter *backend.TaskFilter) ([]backend.Task, error) {
	if nB.Connector.URL.User == nil {
		return nil, fmt.Errorf("no user credentials in URL")
	}

	// Build request body
	queryBody := nB.buildCalendarQuery(taskFilter)

	// Make authenticated request
	headers := map[string]string{
		"Content-Type": "application/xml",
		"Depth":        "1",
	}
	resp, err := nB.makeAuthenticatedRequest("REPORT", nB.buildListURL(listID), strings.NewReader(queryBody), headers)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if err := nB.checkHTTPResponse(resp, "GetTasks"); err != nil {
		return nil, err
	}

	// Parse response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	tasks, err := nB.parseVTODOs(string(respBody))
	if err != nil {
		return nil, err
	}

	// Apply client-side ExcludeStatuses filter (CalDAV doesn't support NOT IN queries easily)
	if taskFilter != nil && taskFilter.ExcludeStatuses != nil && len(*taskFilter.ExcludeStatuses) > 0 {
		filtered := make([]backend.Task, 0, len(tasks))
		excludeMap := make(map[string]bool)
		for _, status := range *taskFilter.ExcludeStatuses {
			excludeMap[status] = true
		}
		for _, task := range tasks {
			if !excludeMap[task.Status] {
				filtered = append(filtered, task)
			}
		}
		return filtered, nil
	}

	return tasks, nil
}

func (nB *NextcloudBackend) FindTasksBySummary(listID string, summary string) ([]backend.Task, error) {
	// For now, implement client-side filtering
	// Future optimization: could use CalDAV text-match query for server-side search

	// Get all tasks from the list
	allTasks, err := nB.GetTasks(listID, nil)
	if err != nil {
		return nil, err
	}

	// Filter by summary (case-insensitive partial match)
	var matches []backend.Task
	summaryLower := strings.ToLower(summary)

	for _, task := range allTasks {
		taskSummaryLower := strings.ToLower(task.Summary)

		// Include both exact and partial matches
		if strings.Contains(taskSummaryLower, summaryLower) {
			matches = append(matches, task)
		}
	}

	return matches, nil
}

func (nB *NextcloudBackend) GetTaskLists() ([]backend.TaskList, error) {
	calendarURL := nB.buildCalendarURL()

	// Build request body
	propfindBody := `<?xml version="1.0" encoding="utf-8" ?>
<d:propfind xmlns:d="DAV:" xmlns:cs="http://calendarserver.org/ns/" xmlns:c="urn:ietf:params:xml:ns:caldav" xmlns:ic="http://apple.com/ns/ical/" xmlns:nc="http://nextcloud.com/ns">
  <d:prop>
    <d:resourcetype />
    <d:displayname />
    <cs:getctag />
    <c:supported-calendar-component-set />
    <ic:calendar-color />
    <nc:deleted-at />
  </d:prop>
</d:propfind>`

	// Make authenticated request
	headers := map[string]string{
		"Content-Type": "application/xml",
		"Depth":        "1",
	}
	resp, err := nB.makeAuthenticatedRequest("PROPFIND", calendarURL, strings.NewReader(propfindBody), headers)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if err := nB.checkHTTPResponse(resp, "GetTaskLists"); err != nil {
		return nil, err
	}

	// Parse response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return nB.parseTaskLists(string(respBody), calendarURL)
}

func (nB *NextcloudBackend) GetDeletedTaskLists() ([]backend.TaskList, error) {
	calendarURL := nB.buildCalendarURL()

	// Build request body (same as GetTaskLists)
	propfindBody := `<?xml version="1.0" encoding="utf-8" ?>
<d:propfind xmlns:d="DAV:" xmlns:cs="http://calendarserver.org/ns/" xmlns:c="urn:ietf:params:xml:ns:caldav" xmlns:ic="http://apple.com/ns/ical/" xmlns:nc="http://nextcloud.com/ns">
  <d:prop>
    <d:resourcetype />
    <d:displayname />
    <cs:getctag />
    <c:supported-calendar-component-set />
    <ic:calendar-color />
    <nc:deleted-at />
  </d:prop>
</d:propfind>`

	// Make authenticated request
	headers := map[string]string{
		"Content-Type": "application/xml",
		"Depth":        "1",
	}
	resp, err := nB.makeAuthenticatedRequest("PROPFIND", calendarURL, strings.NewReader(propfindBody), headers)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if err := nB.checkHTTPResponse(resp, "GetDeletedTaskLists"); err != nil {
		return nil, err
	}

	// Parse response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return nB.parseDeletedTaskLists(string(respBody), calendarURL)
}

func (nB *NextcloudBackend) AddTask(listID string, task backend.Task) error {
	// Set defaults
	if task.UID == "" {
		task.UID = fmt.Sprintf("task-%d", time.Now().Unix())
	}
	if task.Created.IsZero() {
		task.Created = time.Now()
	}
	if task.Status == "" {
		task.Status = "NEEDS-ACTION"
	}

	// Build the iCalendar content
	icalContent := nB.buildICalContent(task)

	// Make authenticated request
	headers := map[string]string{
		"Content-Type": "text/calendar; charset=utf-8",
	}
	resp, err := nB.makeAuthenticatedRequest("PUT", nB.buildTaskURL(listID, task.UID), bytes.NewBufferString(icalContent), headers)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if err := nB.checkHTTPResponse(resp, "AddTask"); err != nil {
		if backendErr, ok := err.(*backend.BackendError); ok {
			return backendErr.WithTaskUID(task.UID).WithListID(listID)
		}
		return err
	}

	return nil
}

func (nB *NextcloudBackend) UpdateTask(listID string, task backend.Task) error {
	// Set modified time to now
	task.Modified = time.Now()

	// If status is COMPLETED and Completed time not set, set it now
	if task.Status == "COMPLETED" && task.Completed == nil {
		now := time.Now()
		task.Completed = &now
	}

	// Build the iCalendar content
	icalContent := nB.buildICalContent(task)

	// Make authenticated request (CalDAV uses PUT for both create and update)
	headers := map[string]string{
		"Content-Type": "text/calendar; charset=utf-8",
	}
	resp, err := nB.makeAuthenticatedRequest("PUT", nB.buildTaskURL(listID, task.UID), bytes.NewBufferString(icalContent), headers)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if err := nB.checkHTTPResponse(resp, "UpdateTask"); err != nil {
		if backendErr, ok := err.(*backend.BackendError); ok {
			return backendErr.WithTaskUID(task.UID).WithListID(listID)
		}
		return err
	}

	return nil
}

func (nB *NextcloudBackend) DeleteTask(listID string, taskUID string) error {
	// Make authenticated DELETE request
	// 204 No Content is the typical success status for DELETE
	resp, err := nB.makeAuthenticatedRequest("DELETE", nB.buildTaskURL(listID, taskUID), nil, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status - handle 404 specifically for task not found
	if resp.StatusCode == 404 {
		return backend.NewBackendError("DeleteTask", 404, "task not found (may have been already deleted)").
			WithTaskUID(taskUID).
			WithListID(listID)
	}

	// Check for other errors
	if err := nB.checkHTTPResponse(resp, "DeleteTask", 200, 204); err != nil {
		if backendErr, ok := err.(*backend.BackendError); ok {
			return backendErr.WithTaskUID(taskUID).WithListID(listID)
		}
		return err
	}

	return nil
}

func (nB *NextcloudBackend) CreateTaskList(name, description, color string) (string, error) {
	// Generate a unique list ID from the name (lowercase, replace spaces with dashes)
	listID := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	// Add timestamp to ensure uniqueness
	listID = fmt.Sprintf("%s-%d", listID, time.Now().Unix())

	// Build the MKCALENDAR request body
	mkcolBody := `<?xml version="1.0" encoding="utf-8" ?>
<d:mkcol xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav" xmlns:ic="http://apple.com/ns/ical/">
  <d:set>
    <d:prop>
      <d:resourcetype>
        <d:collection/>
        <c:calendar/>
      </d:resourcetype>
      <d:displayname>` + name + `</d:displayname>
      <c:supported-calendar-component-set>
        <c:comp name="VTODO"/>
      </c:supported-calendar-component-set>`

	if description != "" {
		mkcolBody += `
      <c:calendar-description>` + description + `</c:calendar-description>`
	}

	if color != "" {
		mkcolBody += `
      <ic:calendar-color>` + color + `</ic:calendar-color>`
	}

	mkcolBody += `
    </d:prop>
  </d:set>
</d:mkcol>`

	// Make authenticated request
	// 201 Created is the success status for MKCOL
	headers := map[string]string{
		"Content-Type": "application/xml; charset=utf-8",
	}
	resp, err := nB.makeAuthenticatedRequest("MKCOL", nB.buildListURL(listID), bytes.NewBufferString(mkcolBody), headers)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status - handle 405 specifically for list already exists
	if resp.StatusCode == 405 {
		body, _ := io.ReadAll(resp.Body)
		return "", backend.NewBackendError("CreateTaskList", 405, "list already exists or name conflict").
			WithBody(string(body))
	}

	// Check for other errors
	if err := nB.checkHTTPResponse(resp, "CreateTaskList", 201); err != nil {
		return "", err
	}

	return listID, nil
}

func (nB *NextcloudBackend) DeleteTaskList(listID string) error {
	// Make authenticated DELETE request
	// 204 No Content is the success status for DELETE
	resp, err := nB.makeAuthenticatedRequest("DELETE", nB.buildListURL(listID), nil, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status - handle 404 specifically for list not found
	if resp.StatusCode == 404 {
		return backend.NewBackendError("DeleteTaskList", 404, "list not found").
			WithListID(listID)
	}

	// Check for other errors
	if err := nB.checkHTTPResponse(resp, "DeleteTaskList", 200, 204); err != nil {
		if backendErr, ok := err.(*backend.BackendError); ok {
			return backendErr.WithListID(listID)
		}
		return err
	}

	return nil
}

func (nB *NextcloudBackend) RenameTaskList(listID, newName string) error {
	// Build the PROPPATCH request body to update displayname
	proppatchBody := `<?xml version="1.0" encoding="utf-8" ?>
<d:propertyupdate xmlns:d="DAV:">
  <d:set>
    <d:prop>
      <d:displayname>` + newName + `</d:displayname>
    </d:prop>
  </d:set>
</d:propertyupdate>`

	// Make authenticated request
	// 207 Multi-Status is the typical success status for PROPPATCH
	headers := map[string]string{
		"Content-Type": "application/xml; charset=utf-8",
	}
	resp, err := nB.makeAuthenticatedRequest("PROPPATCH", nB.buildListURL(listID), bytes.NewBufferString(proppatchBody), headers)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status - handle 404 specifically for list not found
	if resp.StatusCode == 404 {
		return backend.NewBackendError("RenameTaskList", 404, "list not found").
			WithListID(listID)
	}

	// Check for other errors
	if err := nB.checkHTTPResponse(resp, "RenameTaskList", 200, 207); err != nil {
		if backendErr, ok := err.(*backend.BackendError); ok {
			return backendErr.WithListID(listID)
		}
		return err
	}

	return nil
}

func (nB *NextcloudBackend) RestoreTaskList(listID string) error {
	// Build the MOVE request to restore from trash
	// Nextcloud uses MOVE method to restore deleted calendars
	// Source: deleted calendar URL, Destination: restored calendar URL

	// Build source URL (current location in trash)
	sourceURL := nB.buildListURL(listID)

	// Build destination URL (where to restore - use the deleted-suffix format)
	// Nextcloud appends a suffix to deleted calendars, we need to remove it
	restoredListID := strings.TrimSuffix(listID, "-deleted")
	if restoredListID == listID {
		// If no -deleted suffix, try restoring to same location
		restoredListID = listID
	}
	destURL := nB.buildListURL(restoredListID)

	// Make authenticated MOVE request
	headers := map[string]string{
		"Destination": destURL,
		"Overwrite":   "F", // Don't overwrite existing calendars
	}
	resp, err := nB.makeAuthenticatedRequest("MOVE", sourceURL, nil, headers)
	if err != nil {
		return fmt.Errorf("failed to restore list: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status - 201 Created or 204 No Content are success statuses
	if resp.StatusCode == 404 {
		return backend.NewBackendError("RestoreTaskList", 404, "list not found in trash").
			WithListID(listID)
	}

	if err := nB.checkHTTPResponse(resp, "RestoreTaskList", 201, 204); err != nil {
		if backendErr, ok := err.(*backend.BackendError); ok {
			return backendErr.WithListID(listID)
		}
		return err
	}

	return nil
}

func (nB *NextcloudBackend) PermanentlyDeleteTaskList(listID string) error {
	// Build DELETE request with special header to permanently delete from trash
	// For Nextcloud, we need to delete the calendar completely

	// Make authenticated DELETE request
	resp, err := nB.makeAuthenticatedRequest("DELETE", nB.buildListURL(listID), nil, nil)
	if err != nil {
		return fmt.Errorf("failed to permanently delete list: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status - handle 404 specifically for list not found
	if resp.StatusCode == 404 {
		return backend.NewBackendError("PermanentlyDeleteTaskList", 404, "list not found in trash").
			WithListID(listID)
	}

	// Check for other errors - 200 OK or 204 No Content are success statuses
	if err := nB.checkHTTPResponse(resp, "PermanentlyDeleteTaskList", 200, 204); err != nil {
		if backendErr, ok := err.(*backend.BackendError); ok {
			return backendErr.WithListID(listID)
		}
		return err
	}

	return nil
}

func (nb *NextcloudBackend) buildICalContent(task backend.Task) string {
	var icalContent bytes.Buffer

	// Format timestamps
	now := time.Now().UTC()
	dtstamp := now.Format("20060102T150405Z")
	created := task.Created.UTC().Format("20060102T150405Z")

	icalContent.WriteString("BEGIN:VCALENDAR\r\n")
	icalContent.WriteString("VERSION:2.0\r\n")
	icalContent.WriteString("PRODID:-//Go CalDAV Client//EN\r\n")
	icalContent.WriteString("BEGIN:VTODO\r\n")
	icalContent.WriteString(fmt.Sprintf("UID:%s\r\n", task.UID))
	icalContent.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", dtstamp))
	icalContent.WriteString(fmt.Sprintf("CREATED:%s\r\n", created))

	// Add LAST-MODIFIED if task has been modified
	if !task.Modified.IsZero() {
		modified := task.Modified.UTC().Format("20060102T150405Z")
		icalContent.WriteString(fmt.Sprintf("LAST-MODIFIED:%s\r\n", modified))
	}

	icalContent.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", task.Summary))

	if task.Description != "" {
		icalContent.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", task.Description))
	}

	icalContent.WriteString(fmt.Sprintf("STATUS:%s\r\n", task.Status))

	if task.Priority > 0 {
		icalContent.WriteString(fmt.Sprintf("PRIORITY:%d\r\n", task.Priority))
	}

	if task.DueDate != nil {
		due := task.DueDate.UTC().Format("20060102T150405Z")
		icalContent.WriteString(fmt.Sprintf("DUE:%s\r\n", due))
	}

	if task.StartDate != nil {
		start := task.StartDate.UTC().Format("20060102T150405Z")
		icalContent.WriteString(fmt.Sprintf("DTSTART:%s\r\n", start))
	}

	// Add COMPLETED timestamp if status is COMPLETED
	if task.Status == "COMPLETED" && task.Completed != nil {
		completed := task.Completed.UTC().Format("20060102T150405Z")
		icalContent.WriteString(fmt.Sprintf("COMPLETED:%s\r\n", completed))
	}

	// Add RELATED-TO for parent-child relationships
	if task.ParentUID != "" {
		icalContent.WriteString(fmt.Sprintf("RELATED-TO:%s\r\n", task.ParentUID))
	}

	icalContent.WriteString("END:VTODO\r\n")
	icalContent.WriteString("END:VCALENDAR\r\n")

	return icalContent.String()
}

func (nB *NextcloudBackend) SortTasks(tasks []backend.Task) {
	// Nextcloud priority: 1 is highest, 0 is undefined (goes last)
	sort.Slice(tasks, func(i, j int) bool {
		pi, pj := tasks[i].Priority, tasks[j].Priority

		// Priority 0 (undefined) goes to the end
		if pi == 0 && pj != 0 {
			return false
		}
		if pj == 0 && pi != 0 {
			return true
		}

		// Otherwise sort ascending (1, 2, 3, ...)
		return pi < pj
	})
}

func (nB *NextcloudBackend) ParseStatusFlag(statusFlag string) (string, error) {
	if statusFlag == "" {
		return "", fmt.Errorf("status flag cannot be empty")
	}

	// Convert to uppercase and look up in map
	upperStatus := strings.ToUpper(statusFlag)
	if calDAVStatus, ok := statusToCalDAV[upperStatus]; ok {
		return calDAVStatus, nil
	}

	return "", fmt.Errorf("invalid status: %s (valid: TODO/T, DONE/D, PROCESSING/P, CANCELLED/C)", statusFlag)
}

func (nB *NextcloudBackend) StatusToDisplayName(backendStatus string) string {
	// Convert CalDAV status to display name
	upperStatus := strings.ToUpper(backendStatus)
	if displayName, ok := calDAVToDisplay[upperStatus]; ok {
		return displayName
	}
	return backendStatus
}

func (nB *NextcloudBackend) GetPriorityColor(priority int) string {
	// Nextcloud priority color scheme
	if priority >= 1 && priority <= 4 {
		return "\033[31m" // Red (high priority)
	} else if priority == 5 {
		return "\033[33m" // Yellow (medium priority)
	} else if priority >= 6 && priority <= 9 {
		return "\033[34m" // Blue (low priority)
	}
	return "" // No color for 0 (undefined)
}

func (nB *NextcloudBackend) GetBackendDisplayName() string {
	username := nB.getUsername()
	host := ""
	if nB.Connector.URL != nil {
		host = nB.Connector.URL.Host
	}
	return fmt.Sprintf("[nextcloud:%s@%s]", username, host)
}

func (nB *NextcloudBackend) GetBackendType() string {
	return "nextcloud"
}

func (nB *NextcloudBackend) GetBackendContext() string {
	username := nB.getUsername()
	host := ""
	if nB.Connector.URL != nil {
		host = nB.Connector.URL.Host
	}
	return fmt.Sprintf("%s@%s", username, host)
}

func NewNextcloudBackend(connectorConfig backend.ConnectorConfig) (backend.TaskManager, error) {
	nB := &NextcloudBackend{
		Connector: connectorConfig,
	}

	if err := nB.BasicValidation(); err != nil {
		return nil, err
	}

	// SECURITY: Check if AllowHTTP is enabled and warn
	if nB.Connector.AllowHTTP && !nB.Connector.SuppressHTTPWarning {
		// Check if this will result in HTTP connections
		host := nB.Connector.URL.Host
		if strings.Contains(host, ":80") || strings.Contains(host, ":8080") || strings.Contains(host, ":8000") {
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "╔═══════════════════════════════════════════════════════════════════╗")
			fmt.Fprintln(os.Stderr, "║                     ⚠️  SECURITY WARNING  ⚠️                      ║")
			fmt.Fprintln(os.Stderr, "╠═══════════════════════════════════════════════════════════════════╣")
			fmt.Fprintln(os.Stderr, "║ HTTP connections are INSECURE and transmit data in PLAINTEXT     ║")
			fmt.Fprintln(os.Stderr, "║ including your username and password!                            ║")
			fmt.Fprintln(os.Stderr, "║                                                                   ║")
			fmt.Fprintln(os.Stderr, "║ Only use HTTP for local testing with trusted networks.           ║")
			fmt.Fprintln(os.Stderr, "║ For production, use HTTPS with valid certificates.               ║")
			fmt.Fprintln(os.Stderr, "╚═══════════════════════════════════════════════════════════════════╝")
			fmt.Fprintln(os.Stderr, "")
		}
	}

	// SECURITY: Warn if TLS verification is disabled
	if nB.Connector.InsecureSkipVerify && !nB.Connector.SuppressSSLWarning {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "╔═══════════════════════════════════════════════════════════════════╗")
		fmt.Fprintln(os.Stderr, "║                     ⚠️  SECURITY WARNING  ⚠️                      ║")
		fmt.Fprintln(os.Stderr, "╠═══════════════════════════════════════════════════════════════════╣")
		fmt.Fprintln(os.Stderr, "║ TLS certificate verification is DISABLED!                        ║")
		fmt.Fprintln(os.Stderr, "║ This makes you vulnerable to man-in-the-middle attacks.          ║")
		fmt.Fprintln(os.Stderr, "║                                                                   ║")
		fmt.Fprintln(os.Stderr, "║ Only use this for development with self-signed certificates.     ║")
		fmt.Fprintln(os.Stderr, "║ For production, use properly signed certificates or add your     ║")
		fmt.Fprintln(os.Stderr, "║ CA certificate to the system trust store.                        ║")
		fmt.Fprintln(os.Stderr, "╚═══════════════════════════════════════════════════════════════════╝")
		fmt.Fprintln(os.Stderr, "")
	}

	return nB, nil
}

// newNextcloudBackendFromBackendConfig creates a Nextcloud backend from BackendConfig
func newNextcloudBackendFromBackendConfig(bc backend.BackendConfig) (backend.TaskManager, error) {
	// Convert BackendConfig to ConnectorConfig
	u, err := url.Parse(bc.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL for nextcloud backend: %w", err)
	}

	connConfig := backend.ConnectorConfig{
		URL:                 u,
		InsecureSkipVerify:  bc.InsecureSkipVerify,
		SuppressSSLWarning:  bc.SuppressSSLWarning,
		AllowHTTP:           bc.AllowHTTP,
		SuppressHTTPWarning: bc.SuppressHTTPWarning,
	}

	return NewNextcloudBackend(connConfig)
}

func (nB *NextcloudBackend) BasicValidation() error {
	if nB.Connector.URL == nil {
		return fmt.Errorf("connector URL is nil")
	}

	if nB.Connector.URL.User == nil {
		return fmt.Errorf("no user credentials in URL")
	}
	return nil
}
