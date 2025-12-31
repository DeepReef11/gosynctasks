package nextcloud

import (
	"net/url"
	"os"
	"testing"

	"gosynctasks/backend"
	"gosynctasks/internal/credentials"
)

// TestBackendInitialization_WithHostAndUsername tests that a backend can be created
// with just host and username (credentials to come from keyring/env)
func TestBackendInitialization_WithHostAndUsername(t *testing.T) {
	backendConfig := backend.BackendConfig{
		Name:     "test-backend",
		Type:     "nextcloud",
		Enabled:  true,
		Host:     "localhost:8080",
		Username: "testuser",
		// No URL - credentials should come from keyring/env
	}

	backend, err := newNextcloudBackendFromBackendConfig(backendConfig)
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	nb, ok := backend.(*NextcloudBackend)
	if !ok {
		t.Fatal("Backend is not a NextcloudBackend")
	}

	// Verify backend fields are set correctly
	if nb.BackendName != "test-backend" {
		t.Errorf("BackendName = %q, want %q", nb.BackendName, "test-backend")
	}

	if nb.ConfigHost != "localhost:8080" {
		t.Errorf("ConfigHost = %q, want %q", nb.ConfigHost, "localhost:8080")
	}

	if nb.ConfigUsername != "testuser" {
		t.Errorf("ConfigUsername = %q, want %q", nb.ConfigUsername, "testuser")
	}

	// Verify URL was constructed without credentials
	if nb.Connector.URL == nil {
		t.Fatal("Connector.URL is nil")
	}

	if nb.Connector.URL.Host != "localhost:8080" {
		t.Errorf("URL.Host = %q, want %q", nb.Connector.URL.Host, "localhost:8080")
	}

	if nb.Connector.URL.User != nil {
		t.Errorf("URL.User should be nil, got %v", nb.Connector.URL.User)
	}
}

// TestBackendInitialization_WithURL tests backward compatibility with URL format
func TestBackendInitialization_WithURL(t *testing.T) {
	backendConfig := backend.BackendConfig{
		Name:    "test-backend",
		Type:    "nextcloud",
		Enabled: true,
		URL:     "nextcloud://user:pass@localhost:8080",
	}

	backend, err := newNextcloudBackendFromBackendConfig(backendConfig)
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	nb, ok := backend.(*NextcloudBackend)
	if !ok {
		t.Fatal("Backend is not a NextcloudBackend")
	}

	// Verify URL credentials are parsed
	if nb.Connector.URL.User == nil {
		t.Fatal("URL.User should not be nil for URL with credentials")
	}

	username := nb.Connector.URL.User.Username()
	if username != "user" {
		t.Errorf("Username = %q, want %q", username, "user")
	}

	password, _ := nb.Connector.URL.User.Password()
	if password != "pass" {
		t.Errorf("Password = %q, want %q", password, "pass")
	}
}

// TestCredentialResolution_EnvironmentVariables tests that credentials
// can be resolved from environment variables
func TestCredentialResolution_EnvironmentVariables(t *testing.T) {
	// Set up environment variables
	os.Setenv("GOSYNCTASKS_ENVTEST_USERNAME", "envuser")
	os.Setenv("GOSYNCTASKS_ENVTEST_PASSWORD", "envpass")
	os.Setenv("GOSYNCTASKS_ENVTEST_HOST", "env.example.com")
	defer func() {
		os.Unsetenv("GOSYNCTASKS_ENVTEST_USERNAME")
		os.Unsetenv("GOSYNCTASKS_ENVTEST_PASSWORD")
		os.Unsetenv("GOSYNCTASKS_ENVTEST_HOST")
	}()

	backendConfig := backend.BackendConfig{
		Name:     "envtest",
		Type:     "nextcloud",
		Enabled:  true,
		Username: "configuser", // Should be overridden by env
		Host:     "config.example.com",
	}

	backend, err := newNextcloudBackendFromBackendConfig(backendConfig)
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	nb, ok := backend.(*NextcloudBackend)
	if !ok {
		t.Fatal("Backend is not a NextcloudBackend")
	}

	// Get credentials - should come from environment
	username := nb.getUsername()
	password := nb.getPassword()

	if username != "envuser" {
		t.Errorf("Username = %q, want %q (from environment)", username, "envuser")
	}

	if password != "envpass" {
		t.Errorf("Password = %q, want %q (from environment)", password, "envpass")
	}
}

// TestCredentialResolution_URLFallback tests that URL credentials
// are used when keyring/env are not available
func TestCredentialResolution_URLFallback(t *testing.T) {
	backendConfig := backend.BackendConfig{
		Name:     "urltest",
		Type:     "nextcloud",
		Enabled:  true,
		URL:      "nextcloud://urluser:urlpass@localhost:8080",
		Username: "configuser", // Should be ignored when URL has credentials
	}

	backend, err := newNextcloudBackendFromBackendConfig(backendConfig)
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	nb, ok := backend.(*NextcloudBackend)
	if !ok {
		t.Fatal("Backend is not a NextcloudBackend")
	}

	// Get credentials - should come from URL
	username := nb.getUsername()
	password := nb.getPassword()

	if username != "urluser" {
		t.Errorf("Username = %q, want %q (from URL)", username, "urluser")
	}

	if password != "urlpass" {
		t.Errorf("Password = %q, want %q (from URL)", password, "urlpass")
	}
}

// TestCredentialResolution_Priority tests the priority order:
// Environment Variables > URL
func TestCredentialResolution_Priority(t *testing.T) {
	// Set up environment variables
	os.Setenv("GOSYNCTASKS_PRIORITYTEST_USERNAME", "envuser")
	os.Setenv("GOSYNCTASKS_PRIORITYTEST_PASSWORD", "envpass")
	defer func() {
		os.Unsetenv("GOSYNCTASKS_PRIORITYTEST_USERNAME")
		os.Unsetenv("GOSYNCTASKS_PRIORITYTEST_PASSWORD")
	}()

	backendConfig := backend.BackendConfig{
		Name:    "prioritytest",
		Type:    "nextcloud",
		Enabled: true,
		URL:     "nextcloud://urluser:urlpass@localhost:8080",
	}

	backend, err := newNextcloudBackendFromBackendConfig(backendConfig)
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	nb, ok := backend.(*NextcloudBackend)
	if !ok {
		t.Fatal("Backend is not a NextcloudBackend")
	}

	// Get credentials - environment should take priority over URL
	username := nb.getUsername()
	password := nb.getPassword()

	if username != "envuser" {
		t.Errorf("Username = %q, want %q (env should override URL)", username, "envuser")
	}

	if password != "envpass" {
		t.Errorf("Password = %q, want %q (env should override URL)", password, "envpass")
	}
}

// TestBasicValidation_WithBackendName tests that validation passes
// when BackendName is set (even if URL.User is nil)
func TestBasicValidation_WithBackendName(t *testing.T) {
	backendConfig := backend.BackendConfig{
		Name:     "validtest",
		Type:     "nextcloud",
		Enabled:  true,
		Host:     "localhost:8080",
		Username: "testuser",
	}

	backend, err := newNextcloudBackendFromBackendConfig(backendConfig)
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	nb, ok := backend.(*NextcloudBackend)
	if !ok {
		t.Fatal("Backend is not a NextcloudBackend")
	}

	// BasicValidation should pass because BackendName is set
	err = nb.BasicValidation()
	if err != nil {
		t.Errorf("BasicValidation failed: %v (should pass when BackendName is set)", err)
	}
}

// TestBasicValidation_WithoutBackendNameAndURLUser tests that validation
// fails when neither BackendName nor URL.User are available
func TestBasicValidation_WithoutBackendNameAndURLUser(t *testing.T) {
	nb := &NextcloudBackend{
		Connector: backend.ConnectorConfig{
			URL: mustParseURL("nextcloud://localhost:8080"), // No user
		},
		BackendName: "", // Not set
	}

	err := nb.BasicValidation()
	if err == nil {
		t.Error("BasicValidation should fail when BackendName is empty and URL.User is nil")
	}
}

// TestGetUsername_Caching tests that username is cached after first resolution
func TestGetUsername_Caching(t *testing.T) {
	// Set up environment variables
	os.Setenv("GOSYNCTASKS_CACHETEST_USERNAME", "envuser")
	os.Setenv("GOSYNCTASKS_CACHETEST_PASSWORD", "envpass")
	defer func() {
		os.Unsetenv("GOSYNCTASKS_CACHETEST_USERNAME")
		os.Unsetenv("GOSYNCTASKS_CACHETEST_PASSWORD")
	}()

	backendConfig := backend.BackendConfig{
		Name:    "cachetest",
		Type:    "nextcloud",
		Enabled: true,
		Host:    "localhost:8080",
	}

	backend, err := newNextcloudBackendFromBackendConfig(backendConfig)
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	nb, ok := backend.(*NextcloudBackend)
	if !ok {
		t.Fatal("Backend is not a NextcloudBackend")
	}

	// First call - should resolve from environment
	username1 := nb.getUsername()
	if username1 != "envuser" {
		t.Errorf("First call: Username = %q, want %q", username1, "envuser")
	}

	// Change environment variable
	os.Setenv("GOSYNCTASKS_CACHETEST_USERNAME", "newuser")

	// Second call - should return cached value, not new env value
	username2 := nb.getUsername()
	if username2 != "envuser" {
		t.Errorf("Second call: Username = %q, want %q (should be cached)", username2, "envuser")
	}
}

// TestMultipleBackends_SeparateCredentials tests that different backend names
// can have different credentials
func TestMultipleBackends_SeparateCredentials(t *testing.T) {
	// Set up environment for prod backend
	os.Setenv("GOSYNCTASKS_NEXTCLOUD_PROD_USERNAME", "produser")
	os.Setenv("GOSYNCTASKS_NEXTCLOUD_PROD_PASSWORD", "prodpass")

	// Set up environment for test backend
	os.Setenv("GOSYNCTASKS_NEXTCLOUD_TEST_USERNAME", "testuser")
	os.Setenv("GOSYNCTASKS_NEXTCLOUD_TEST_PASSWORD", "testpass")

	defer func() {
		os.Unsetenv("GOSYNCTASKS_NEXTCLOUD_PROD_USERNAME")
		os.Unsetenv("GOSYNCTASKS_NEXTCLOUD_PROD_PASSWORD")
		os.Unsetenv("GOSYNCTASKS_NEXTCLOUD_TEST_USERNAME")
		os.Unsetenv("GOSYNCTASKS_NEXTCLOUD_TEST_PASSWORD")
	}()

	// Create prod backend
	prodConfig := backend.BackendConfig{
		Name:    "nextcloud-prod",
		Type:    "nextcloud",
		Enabled: true,
		Host:    "nextcloud.example.com",
	}

	prodBackend, err := newNextcloudBackendFromBackendConfig(prodConfig)
	if err != nil {
		t.Fatalf("Failed to create prod backend: %v", err)
	}

	// Create test backend
	testConfig := backend.BackendConfig{
		Name:    "nextcloud-test",
		Type:    "nextcloud",
		Enabled: true,
		Host:    "localhost:8080",
	}

	testBackend, err := newNextcloudBackendFromBackendConfig(testConfig)
	if err != nil {
		t.Fatalf("Failed to create test backend: %v", err)
	}

	// Verify prod credentials
	prodNB := prodBackend.(*NextcloudBackend)
	if prodNB.getUsername() != "produser" {
		t.Errorf("Prod username = %q, want %q", prodNB.getUsername(), "produser")
	}
	if prodNB.getPassword() != "prodpass" {
		t.Errorf("Prod password = %q, want %q", prodNB.getPassword(), "prodpass")
	}

	// Verify test credentials
	testNB := testBackend.(*NextcloudBackend)
	if testNB.getUsername() != "testuser" {
		t.Errorf("Test username = %q, want %q", testNB.getUsername(), "testuser")
	}
	if testNB.getPassword() != "testpass" {
		t.Errorf("Test password = %q, want %q", testNB.getPassword(), "testpass")
	}
}

// Helper function to parse URL without error handling
func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return u
}

// TestCredentialResolution_NoCredentials tests error handling when no credentials found
func TestCredentialResolution_NoCredentials(t *testing.T) {
	backendConfig := backend.BackendConfig{
		Name:    "nocreds",
		Type:    "nextcloud",
		Enabled: true,
		Host:    "localhost:8080",
		// No username, no URL, no env vars
	}

	backend, err := newNextcloudBackendFromBackendConfig(backendConfig)
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	nb, ok := backend.(*NextcloudBackend)
	if !ok {
		t.Fatal("Backend is not a NextcloudBackend")
	}

	// getUsername should return empty string when no credentials found
	username := nb.getUsername()
	if username != "" {
		t.Errorf("Username should be empty when no credentials found, got %q", username)
	}

	// getPassword should return empty string when no credentials found
	password := nb.getPassword()
	if password != "" {
		t.Errorf("Password should be empty when no credentials found, got %q", password)
	}
}

// TestBackendName_Normalization tests that backend names with hyphens
// are properly normalized for environment variables
func TestBackendName_Normalization(t *testing.T) {
	// Environment variable for backend with hyphen
	os.Setenv("GOSYNCTASKS_NEXTCLOUD_TEST_USERNAME", "testuser")
	os.Setenv("GOSYNCTASKS_NEXTCLOUD_TEST_PASSWORD", "testpass")
	defer func() {
		os.Unsetenv("GOSYNCTASKS_NEXTCLOUD_TEST_USERNAME")
		os.Unsetenv("GOSYNCTASKS_NEXTCLOUD_TEST_PASSWORD")
	}()

	backendConfig := backend.BackendConfig{
		Name:    "nextcloud-test", // Has hyphen
		Type:    "nextcloud",
		Enabled: true,
		Host:    "localhost:8080",
	}

	backend, err := newNextcloudBackendFromBackendConfig(backendConfig)
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	nb, ok := backend.(*NextcloudBackend)
	if !ok {
		t.Fatal("Backend is not a NextcloudBackend")
	}

	// Should resolve credentials despite hyphen in backend name
	username := nb.getUsername()
	password := nb.getPassword()

	if username != "testuser" {
		t.Errorf("Username = %q, want %q (hyphen should be normalized to underscore)", username, "testuser")
	}

	if password != "testpass" {
		t.Errorf("Password = %q, want %q", password, "testpass")
	}
}

// TestCredentialSource_Tracking tests that we can determine where credentials came from
func TestCredentialSource_Tracking(t *testing.T) {
	tests := []struct {
		name           string
		setupEnv       func()
		cleanupEnv     func()
		config         backend.BackendConfig
		expectedSource credentials.Source
	}{
		{
			name: "URL credentials",
			setupEnv: func() {
				// No env vars
			},
			cleanupEnv: func() {},
			config: backend.BackendConfig{
				Name:    "urlsource",
				Type:    "nextcloud",
				Enabled: true,
				URL:     "nextcloud://user:pass@localhost:8080",
			},
			expectedSource: credentials.SourceURL,
		},
		{
			name: "Environment credentials",
			setupEnv: func() {
				os.Setenv("GOSYNCTASKS_ENVSOURCE_USERNAME", "envuser")
				os.Setenv("GOSYNCTASKS_ENVSOURCE_PASSWORD", "envpass")
			},
			cleanupEnv: func() {
				os.Unsetenv("GOSYNCTASKS_ENVSOURCE_USERNAME")
				os.Unsetenv("GOSYNCTASKS_ENVSOURCE_PASSWORD")
			},
			config: backend.BackendConfig{
				Name:    "envsource",
				Type:    "nextcloud",
				Enabled: true,
				Host:    "localhost:8080",
			},
			expectedSource: credentials.SourceEnv,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanupEnv()

			backend, err := newNextcloudBackendFromBackendConfig(tt.config)
			if err != nil {
				t.Fatalf("Failed to create backend: %v", err)
			}

			nb := backend.(*NextcloudBackend)
			resolver := credentials.NewResolver()
			creds, err := resolver.Resolve(nb.BackendName, nb.ConfigUsername, nb.ConfigHost, nb.Connector.URL)

			if err != nil {
				t.Fatalf("Failed to resolve credentials: %v", err)
			}

			if creds.Source != tt.expectedSource {
				t.Errorf("Source = %q, want %q", creds.Source, tt.expectedSource)
			}
		})
	}
}
