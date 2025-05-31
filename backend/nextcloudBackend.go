package backend

import (
    "crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type NextcloudBackend struct {
	Connector ConnectorConfig
}


func (nB *NextcloudBackend) GetTasks(listID string) ([]Task, error) {
    username := nB.Connector.URL.User.Username()
    password, _ := nB.Connector.URL.User.Password()
    baseURL := fmt.Sprintf("https://%s", nB.Connector.URL.Host)

    calendarURL := fmt.Sprintf("%s/remote.php/dav/calendars/%s/%s/", baseURL, username, listID)

    // Create client that ignores SSL certificates
    tr := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    }
    client := &http.Client{Transport: tr}

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
    if nB.Connector.URL == nil {
        return nil, fmt.Errorf("connector URL is nil")
    }

    if nB.Connector.URL.User == nil {
        return nil, fmt.Errorf("no user credentials in URL")
    }

    username := nB.Connector.URL.User.Username()
    password, _ := nB.Connector.URL.User.Password()

    if username == "" {
        return nil, fmt.Errorf("username is empty")
    }

    baseURL := fmt.Sprintf("https://%s", nB.Connector.URL.Host)
    calendarURL := fmt.Sprintf("%s/remote.php/dav/calendars/%s/", baseURL, username)

    // Create client that ignores SSL certificates
    tr := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    }
    client := &http.Client{Transport: tr}

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

    // Debug: print response
    fmt.Printf("Response status: %d\n", resp.StatusCode)
    // fmt.Printf("Response body: %s\n", string(respBody))

    return nB.parseTaskLists(string(respBody), calendarURL)
}

func NewNextcloudBackend(url *url.URL) (TaskManager, error) {
	nB := &NextcloudBackend{
		Connector: ConnectorConfig{URL: url},
	}
	return nB, nil
}
