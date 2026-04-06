package matrix

import (
	"fmt"
	"net/url"
	"strings"
)

// HomeserverConfig holds the configuration for the local homeserver instance.
// It represents how this Hearth instance appears to other Matrix servers
// and clients.
type HomeserverConfig struct {
	// ServerName is the canonical homeserver name (e.g., "hearth.example.com").
	// This is the hostname clients use to reach this server.
	ServerName string

	// BaseURL is the public-facing base URL for the Client-Server API.
	// Example: "https://hearth.example.com"
	BaseURL string

	// FederationURL is the public-facing URL for the Server-Server (federation) API.
	// Defaults to BaseURL if not set.
	// Example: "https://hearth.example.com"
	FederationURL string

	// DefaultIdentityURL is the URL of the default identity server.
	// If empty, no identity server is used.
	DefaultIdentityURL string

	// Version is the Matrix protocol version supported by this homeserver.
	Version string

	// Name is a human-readable name for this homeserver.
	Name string

	// Port is the port where the federation listener runs (if applicable).
	Port int
}

// DefaultHomeserverConfig returns a default-initialized config.
func DefaultHomeserverConfig() *HomeserverConfig {
	return &HomeserverConfig{
		Version: "1.12.0",
	}
}

// Validate checks that the homeserver config has all required fields set.
func (c *HomeserverConfig) Validate() error {
	if c.ServerName == "" {
		return fmt.Errorf("matrix: homeserver ServerName is required")
	}
	if c.BaseURL == "" {
		return fmt.Errorf("matrix: homeserver BaseURL is required")
	}
	if u, err := url.Parse(c.BaseURL); err != nil || (u.Scheme == "" && u.Host == "") {
		return fmt.Errorf("matrix: invalid BaseURL: %w", err)
	}
	if c.FederationURL != "" {
		if u, err := url.Parse(c.FederationURL); err != nil || (u.Scheme == "" && u.Host == "") {
			return fmt.Errorf("matrix: invalid FederationURL: %w", err)
		}
	}
	if c.DefaultIdentityURL != "" {
		if u, err := url.Parse(c.DefaultIdentityURL); err != nil || (u.Scheme == "" && u.Host == "") {
			return fmt.Errorf("matrix: invalid DefaultIdentityURL: %w", err)
		}
	}
	return nil
}

// GetFederationURL returns the federation URL, falling back to BaseURL.
func (c *HomeserverConfig) GetFederationURL() string {
	if c.FederationURL != "" {
		return c.FederationURL
	}
	return c.BaseURL
}

// GetClientURL returns the base URL for client-server API endpoints.
func (c *HomeserverConfig) GetClientURL() string {
	return c.BaseURL
}

// MakeMXID creates a fully-qualified MXID for a local user on this homeserver.
func (c *HomeserverConfig) MakeMXID(localpart string) MXID {
	return MXID{
		Localpart:  localpart,
		ServerName: c.ServerName,
	}
}

// IsLocalServer reports whether the given server name refers to this homeserver.
func (c *HomeserverConfig) IsLocalServer(serverName string) bool {
	return strings.EqualFold(c.ServerName, serverName)
}

// IsLocalMXID reports whether the given MXID belongs to this homeserver.
func (c *HomeserverConfig) IsLocalMXID(mxid MXID) bool {
	return c.IsLocalServer(mxid.ServerName)
}

// WellKnownURI returns the .well-known URI for this homeserver's client API.
// Returns the well-known URL: {BaseURL}/.well-known/matrix/client
func (c *HomeserverConfig) WellKnownURI() string {
	return strings.TrimSuffix(c.BaseURL, "/") + "/.well-known/matrix/client"
}
