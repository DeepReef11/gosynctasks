package credentials

import (
	"fmt"
	"net/url"
)

// Source indicates where credentials were found
type Source string

const (
	SourceKeyring Source = "keyring"
	SourceEnv     Source = "env"
	SourceURL     Source = "url"
	SourceNone    Source = "none"
)

// Credentials represents resolved authentication credentials
type Credentials struct {
	Username string
	Password string
	Host     string
	Source   Source
}

// Resolver handles credential resolution from multiple sources with priority order
type Resolver struct {
	// Priority order: Keyring > Environment Variables > Config URL
}

// NewResolver creates a new credential resolver
func NewResolver() *Resolver {
	return &Resolver{}
}

// Resolve attempts to find credentials using the priority order:
// 1. Keyring (if username is provided)
// 2. Environment variables
// 3. URL credentials (backward compatible)
//
// Parameters:
//   - backendName: Name of the backend (e.g., "nextcloud")
//   - username: Optional username hint (used for keyring lookup)
//   - configURL: Optional URL from config (may contain credentials)
//
// Returns credentials with Source indicating where they were found
func (r *Resolver) Resolve(backendName, username string, host string, configURL *url.URL) (*Credentials, error) {
	if backendName == "" {
		return nil, fmt.Errorf("backend name is required for credential resolution")
	}

	creds := &Credentials{
		Username: username,
		Host:     host,
		Source:   SourceNone,
	}

	// Priority 1: Try keyring if username is known
	if username != "" && IsAvailable() {
		password, err := Get(backendName, username)
		if err == nil {
			creds.Password = password
			creds.Source = SourceKeyring

			// Get host from env if not provided
			if creds.Host == "" {
				if envHost := GetHost(backendName); envHost != "" {
					creds.Host = envHost
				} else if configURL != nil {
					creds.Host = configURL.Host
				}
			}

			return creds, nil
		}
		// If error is not "not found", it's a keyring access issue
		// Log but continue to next source
	}

	// Priority 2: Try environment variables
	envUsername := GetUsername(backendName)
	envPassword := GetPassword(backendName)
	if envUsername != "" && envPassword != "" {
		creds.Username = envUsername
		creds.Password = envPassword
		creds.Source = SourceEnv

		// Get host from env if not provided
		if creds.Host == "" {
			if envHost := GetHost(backendName); envHost != "" {
				creds.Host = envHost
			} else if configURL != nil {
				creds.Host = configURL.Host
			}
		}

		return creds, nil
	}

	// Priority 3: Try URL credentials (backward compatible)
	if configURL != nil && configURL.User != nil {
		urlUsername := configURL.User.Username()
		urlPassword, _ := configURL.User.Password()

		if urlUsername != "" && urlPassword != "" {
			creds.Username = urlUsername
			creds.Password = urlPassword
			creds.Host = configURL.Host
			creds.Source = SourceURL
			return creds, nil
		}
	}

	// No credentials found
	return nil, fmt.Errorf("no credentials found for backend %q (tried: keyring, environment variables, config URL)", backendName)
}

// ResolveWithConfig is a convenience method that accepts config values
// and constructs the URL internally if needed
func (r *Resolver) ResolveWithConfig(backendName, configUsername, configHost, configURL string) (*Credentials, error) {
	var parsedURL *url.URL
	var err error

	if configURL != "" {
		parsedURL, err = url.Parse(configURL)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %w", err)
		}
	}

	// Use username from config if provided
	username := configUsername

	// Use host from config if provided
	host := configHost

	return r.Resolve(backendName, username, host, parsedURL)
}
