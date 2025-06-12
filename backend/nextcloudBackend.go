package backend

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type NextcloudBackend struct {
	Connector ConnectorConfig
	username  string
	password  string
	baseURL   string
	client    *http.Client
}

func (nB *NextcloudBackend) getClient() *http.Client {
	if nB.client == nil {
		nB.client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
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
			nB.baseURL = fmt.Sprintf("https://%s", nB.Connector.URL.Host) //TODO: Test for uncomplete and complete url like nextloud.com/remote.php/... and just nextcloud.com
		}
	}
	return nB.baseURL
}

func (nB *NextcloudBackend) buildCalendarQuery(filter *TaskFilter) string {
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

			standarizedStatuses := StatusStringTranslateToStandardStatus(filter.Statuses)
			fmt.Println(standarizedStatuses)

			for _, status := range *standarizedStatuses {
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
func (nB *NextcloudBackend) GetTasks(listID string, taskFilter *TaskFilter) ([]Task, error) {

	if nB.Connector.URL.User == nil {
		return nil, fmt.Errorf("no user credentials in URL")
	}
	username := nB.getUsername()
	password := nB.getPassword()
	baseURL := nB.getBaseURL()

	calendarURL := fmt.Sprintf("%s/remote.php/dav/calendars/%s/%s/", baseURL, username, listID)

	client := nB.getClient()

	req, err := http.NewRequest("REPORT", calendarURL, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Depth", "1")

	body := nB.buildCalendarQuery(taskFilter)
	req.Body = io.NopCloser(strings.NewReader(body))

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return nB.parseVTODOs(string(respBody))
}

func (nB *NextcloudBackend) GetTaskLists() ([]TaskList, error) {
	username := nB.getUsername()
	password := nB.getPassword()
	baseURL := nB.getBaseURL()
	calendarURL := fmt.Sprintf("%s/remote.php/dav/calendars/%s/", baseURL, username)

	client := nB.getClient()
	req, err := http.NewRequest("PROPFIND", calendarURL, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Depth", "1")

	body := `<?xml version="1.0" encoding="utf-8" ?>
<d:propfind xmlns:d="DAV:" xmlns:cs="http://calendarserver.org/ns/" xmlns:c="urn:ietf:params:xml:ns:caldav" xmlns:ic="http://apple.com/ns/ical/">
  <d:prop>
    <d:displayname />
    <cs:getctag />
    <c:supported-calendar-component-set />
    <ic:calendar-color />
  </d:prop>
</d:propfind>`

	req.Body = io.NopCloser(strings.NewReader(body))

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	fmt.Printf("Response status: %d\n", resp.StatusCode)
	// fmt.Printf("Response body: %s\n", string(respBody))

	return nB.parseTaskLists(string(respBody), calendarURL)
}


func (nB *NextcloudBackend) AddTask(listID string, task Task) (error) {

	username := nB.getUsername()
	password := nB.getPassword()
	baseURL := nB.getBaseURL()
	if task.UID == "" {
		task.UID = fmt.Sprintf("task-%d", time.Now().Unix())
	}

	// Set created time if not provided
	if task.Created.IsZero() {
		task.Created = time.Now()
	}

	// Default status
	if task.Status == "" {
		task.Status = "NEEDS-ACTION"
	}

	// Build the iCalendar content
	icalContent := nB.buildICalContent(task)

	// Construct the URL

	url := fmt.Sprintf("%s/remote.php/dav/calendars/%s/%s/%s.ics",
		baseURL, username, listID, task.UID)

	// Create HTTP request
	req, err := http.NewRequest("PUT", url, bytes.NewBufferString(icalContent))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "text/calendar; charset=utf-8")
	req.SetBasicAuth(username, password)

	// Send request
	client := nB.getClient()
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	fmt.Println("OKOK")
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("request failed with status: %d %s", resp.StatusCode, resp.Status)
	}

	return nil

}

func (nb *NextcloudBackend) buildICalContent(task Task) string {
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
	
	icalContent.WriteString("END:VTODO\r\n")
	icalContent.WriteString("END:VCALENDAR\r\n")

	return icalContent.String()
}

func NewNextcloudBackend(url *url.URL) (TaskManager, error) {
	nB := &NextcloudBackend{
		Connector: ConnectorConfig{URL: url},
	}

	if err := nB.BasicValidation(); err != nil {
		return nil, err
	}
	return nB, nil
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
