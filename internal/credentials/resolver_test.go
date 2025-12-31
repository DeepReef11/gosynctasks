package credentials

import (
	"net/url"
	"os"
	"testing"
)

func TestResolver_Resolve_EnvironmentVariables(t *testing.T) {
	// Set up test environment
	os.Setenv("GOSYNCTASKS_TESTENV_USERNAME", "envuser")
	os.Setenv("GOSYNCTASKS_TESTENV_PASSWORD", "envpass")
	os.Setenv("GOSYNCTASKS_TESTENV_HOST", "env.example.com")
	defer func() {
		os.Unsetenv("GOSYNCTASKS_TESTENV_USERNAME")
		os.Unsetenv("GOSYNCTASKS_TESTENV_PASSWORD")
		os.Unsetenv("GOSYNCTASKS_TESTENV_HOST")
	}()

	resolver := NewResolver()
	creds, err := resolver.Resolve("testenv", "", "", nil)

	if err != nil {
		t.Fatalf("Resolve() error = %v, want nil", err)
	}

	if creds.Username != "envuser" {
		t.Errorf("Username = %q, want %q", creds.Username, "envuser")
	}

	if creds.Password != "envpass" {
		t.Errorf("Password = %q, want %q", creds.Password, "envpass")
	}

	if creds.Host != "env.example.com" {
		t.Errorf("Host = %q, want %q", creds.Host, "env.example.com")
	}

	if creds.Source != SourceEnv {
		t.Errorf("Source = %q, want %q", creds.Source, SourceEnv)
	}
}

func TestResolver_Resolve_URLCredentials(t *testing.T) {
	testURL, _ := url.Parse("nextcloud://urluser:urlpass@url.example.com")

	resolver := NewResolver()
	creds, err := resolver.Resolve("testurl", "", "", testURL)

	if err != nil {
		t.Fatalf("Resolve() error = %v, want nil", err)
	}

	if creds.Username != "urluser" {
		t.Errorf("Username = %q, want %q", creds.Username, "urluser")
	}

	if creds.Password != "urlpass" {
		t.Errorf("Password = %q, want %q", creds.Password, "urlpass")
	}

	if creds.Host != "url.example.com" {
		t.Errorf("Host = %q, want %q", creds.Host, "url.example.com")
	}

	if creds.Source != SourceURL {
		t.Errorf("Source = %q, want %q", creds.Source, SourceURL)
	}
}

func TestResolver_Resolve_Priority_EnvOverURL(t *testing.T) {
	// Set up environment variables
	os.Setenv("GOSYNCTASKS_PRIORITY_USERNAME", "envuser")
	os.Setenv("GOSYNCTASKS_PRIORITY_PASSWORD", "envpass")
	defer func() {
		os.Unsetenv("GOSYNCTASKS_PRIORITY_USERNAME")
		os.Unsetenv("GOSYNCTASKS_PRIORITY_PASSWORD")
	}()

	// Also provide URL credentials
	testURL, _ := url.Parse("nextcloud://urluser:urlpass@url.example.com")

	resolver := NewResolver()
	creds, err := resolver.Resolve("priority", "", "", testURL)

	if err != nil {
		t.Fatalf("Resolve() error = %v, want nil", err)
	}

	// Environment variables should take priority over URL
	if creds.Username != "envuser" {
		t.Errorf("Username = %q, want %q (env should override URL)", creds.Username, "envuser")
	}

	if creds.Password != "envpass" {
		t.Errorf("Password = %q, want %q (env should override URL)", creds.Password, "envpass")
	}

	if creds.Source != SourceEnv {
		t.Errorf("Source = %q, want %q", creds.Source, SourceEnv)
	}
}

func TestResolver_Resolve_NoCredentials(t *testing.T) {
	resolver := NewResolver()
	_, err := resolver.Resolve("nonexistent", "", "", nil)

	if err == nil {
		t.Error("Resolve() error = nil, want error when no credentials found")
	}
}

func TestResolver_Resolve_EmptyBackendName(t *testing.T) {
	resolver := NewResolver()
	_, err := resolver.Resolve("", "", "", nil)

	if err == nil {
		t.Error("Resolve() error = nil, want error when backend name is empty")
	}
}

func TestResolver_Resolve_HostPriority(t *testing.T) {
	// Set up environment with host
	os.Setenv("GOSYNCTASKS_HOSTTEST_USERNAME", "user")
	os.Setenv("GOSYNCTASKS_HOSTTEST_PASSWORD", "pass")
	os.Setenv("GOSYNCTASKS_HOSTTEST_HOST", "env.example.com")
	defer func() {
		os.Unsetenv("GOSYNCTASKS_HOSTTEST_USERNAME")
		os.Unsetenv("GOSYNCTASKS_HOSTTEST_PASSWORD")
		os.Unsetenv("GOSYNCTASKS_HOSTTEST_HOST")
	}()

	testURL, _ := url.Parse("nextcloud://url.example.com")

	resolver := NewResolver()

	tests := []struct {
		name         string
		providedHost string
		expectedHost string
		description  string
	}{
		{
			name:         "provided host takes priority",
			providedHost: "provided.example.com",
			expectedHost: "provided.example.com",
			description:  "when host is explicitly provided",
		},
		{
			name:         "env host used when not provided",
			providedHost: "",
			expectedHost: "env.example.com",
			description:  "when host not provided, use from env",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds, err := resolver.Resolve("hosttest", "", tt.providedHost, testURL)
			if err != nil {
				t.Fatalf("Resolve() error = %v, want nil", err)
			}

			if creds.Host != tt.expectedHost {
				t.Errorf("Host = %q, want %q (%s)", creds.Host, tt.expectedHost, tt.description)
			}
		})
	}
}

func TestResolveWithConfig(t *testing.T) {
	// Set up environment
	os.Setenv("GOSYNCTASKS_CONFIGTEST_USERNAME", "envuser")
	os.Setenv("GOSYNCTASKS_CONFIGTEST_PASSWORD", "envpass")
	defer func() {
		os.Unsetenv("GOSYNCTASKS_CONFIGTEST_USERNAME")
		os.Unsetenv("GOSYNCTASKS_CONFIGTEST_PASSWORD")
	}()

	resolver := NewResolver()

	tests := []struct {
		name           string
		backendName    string
		configUsername string
		configHost     string
		configURL      string
		wantUsername   string
		wantPassword   string
		wantHost       string
		wantSource     Source
		wantErr        bool
	}{
		{
			name:           "env variables",
			backendName:    "configtest",
			configUsername: "",
			configHost:     "",
			configURL:      "",
			wantUsername:   "envuser",
			wantPassword:   "envpass",
			wantHost:       "",
			wantSource:     SourceEnv,
			wantErr:        false,
		},
		{
			name:           "URL with credentials",
			backendName:    "urltest",
			configUsername: "",
			configHost:     "",
			configURL:      "nextcloud://urluser:urlpass@url.example.com",
			wantUsername:   "urluser",
			wantPassword:   "urlpass",
			wantHost:       "url.example.com",
			wantSource:     SourceURL,
			wantErr:        false,
		},
		{
			name:           "invalid URL",
			backendName:    "invalid",
			configUsername: "",
			configHost:     "",
			configURL:      "://invalid",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds, err := resolver.ResolveWithConfig(tt.backendName, tt.configUsername, tt.configHost, tt.configURL)

			if tt.wantErr {
				if err == nil {
					t.Error("ResolveWithConfig() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Fatalf("ResolveWithConfig() error = %v, want nil", err)
			}

			if creds.Username != tt.wantUsername {
				t.Errorf("Username = %q, want %q", creds.Username, tt.wantUsername)
			}

			if creds.Password != tt.wantPassword {
				t.Errorf("Password = %q, want %q", creds.Password, tt.wantPassword)
			}

			if tt.wantHost != "" && creds.Host != tt.wantHost {
				t.Errorf("Host = %q, want %q", creds.Host, tt.wantHost)
			}

			if creds.Source != tt.wantSource {
				t.Errorf("Source = %q, want %q", creds.Source, tt.wantSource)
			}
		})
	}
}
