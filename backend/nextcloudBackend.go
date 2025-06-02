package backend

import (
	"crypto/tls"
	"time"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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

func (nB *NextcloudBackend) GetTasks(listID string) ([]Task, error) {

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

	body := `<?xml version="1.0" encoding="utf-8" ?>
<c:calendar-query xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:prop>
    <d:getetag />
    <c:calendar-data />
  </d:prop>
  <c:filter>
    <c:comp-filter name="VCALENDAR">
      <c:comp-filter name="VTODO" />
    </c:comp-filter>
  </c:filter>
</c:calendar-query>`

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
