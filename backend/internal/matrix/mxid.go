// Package matrix provides Matrix Protocol utilities for Hearth federation.
//
// Matrix Specification References:
//   - Matrix Client-Server API r0.6.1: https://matrix.org/docs/spec/client_server/r0.6.1
//   - Matrix Federation API r0.1.0: https://matrix.org/docs/spec/server_server/r0.1.0
//   - Matrix Client-Server API r0.6.1 § 8.2: MXID format
package matrix

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// Common errors
var (
	ErrInvalidMXID       = errors.New("matrix: invalid MXID format")
	ErrInvalidLocalpart  = errors.New("matrix: invalid localpart")
	ErrInvalidServerName = errors.New("matrix: invalid server name")
	ErrMissingColon      = errors.New("matrix: MXID must contain a colon after the localpart")
	ErrEmptyLocalpart    = errors.New("matrix: localpart cannot be empty")
	ErrTooManyColons     = errors.New("matrix: MXID may only contain one colon after the localpart")
)

// validMXID matches a full MXID: @localpart:server_name
var validMXID = regexp.MustCompile(`^@([^@:]+):(.+)$`)

// MXID represents a fully-qualified Matrix User ID.
//
// Format: @localpart:server_name
// Example: @alice:matrix.org
//
// Per Matrix spec § 8.2, an MXID encodes a localpart (username) and a server name
// (the homeserver that owns the account).
type MXID struct {
	Localpart  string // The username portion (before the colon)
	ServerName string // The homeserver domain (after the colon)
}

// ParseMXID parses a string into an MXID.
//
// Returns ErrInvalidMXID if the string does not conform to the Matrix spec format.
// The localpart is returned as-is (case-sensitive per spec).
//
// Examples:
//   - "@alice:matrix.org" → MXID{Localpart: "alice", ServerName: "matrix.org"}
//   - "@user:hearth.example.com" → MXID{Localpart: "user", ServerName: "hearth.example.com"}
func ParseMXID(raw string) (MXID, error) {
	if raw == "" {
		return MXID{}, ErrInvalidMXID
	}
	if !strings.HasPrefix(raw, "@") {
		return MXID{}, ErrInvalidMXID
	}

	matches := validMXID.FindStringSubmatch(raw)
	if matches == nil {
		return MXID{}, ErrInvalidMXID
	}

	localpart := matches[1]
	serverName := matches[2]

	if localpart == "" {
		return MXID{}, ErrEmptyLocalpart
	}

	if err := ValidateLocalpart(localpart); err != nil {
		return MXID{}, fmt.Errorf("%w: %v", ErrInvalidMXID, err)
	}
	if err := ValidateServerName(serverName); err != nil {
		return MXID{}, fmt.Errorf("%w: %v", ErrInvalidMXID, err)
	}

	return MXID{
		Localpart:  localpart,
		ServerName: serverName,
	}, nil
}

// MustParseMXID parses a string into an MXID and panics on error.
// Use only when you are certain the input is valid.
func MustParseMXID(raw string) MXID {
	mxid, err := ParseMXID(raw)
	if err != nil {
		panic("matrix: MustParseMXID called with invalid input: " + raw)
	}
	return mxid
}

// String returns the canonical string representation of the MXID.
// This always returns the fully-qualified form: @localpart:server_name
func (m MXID) String() string {
	return "@" + m.Localpart + ":" + m.ServerName
}

// LocalID returns the local part of the MXID (same as Localpart).
func (m MXID) LocalID() string {
	return m.Localpart
}

// Domain returns the server name portion.
func (m MXID) Domain() string {
	return m.ServerName
}

// IsValid reports whether this MXID is well-formed.
func (m MXID) IsValid() bool {
	return m.Localpart != "" &&
		ValidateLocalpart(m.Localpart) == nil &&
		ValidateServerName(m.ServerName) == nil
}

// Equal reports whether two MXIDs represent the same user on the same server.
func (m MXID) Equal(other MXID) bool {
	return m.Localpart == other.Localpart && m.ServerName == other.ServerName
}

// IsSameServer reports whether two MXIDs belong to the same homeserver.
func (m MXID) IsSameServer(other MXID) bool {
	return m.ServerName == other.ServerName
}

// ValidateLocalpart checks whether a localpart string is valid per Matrix spec.
// A valid localpart may not contain @ or :.
func ValidateLocalpart(localpart string) error {
	if localpart == "" {
		return ErrEmptyLocalpart
	}
	// Per spec: localpart must not contain @ or :
	if strings.ContainsAny(localpart, "@:") {
		return ErrInvalidLocalpart
	}
	return nil
}

// ValidateServerName checks whether a server name string is valid per Matrix spec.
// A valid server name is a DNS host or IP literal with an optional port.
// Note: This does NOT perform DNS resolution; it only validates format.
func ValidateServerName(serverName string) error {
	if serverName == "" {
		return ErrInvalidServerName
	}

	// Check for illegal characters
	if strings.ContainsAny(serverName, "/@#") {
		return ErrInvalidServerName
	}

	// Handle IPv6 literals: [::1] or [::1]:port
	if strings.HasPrefix(serverName, "[") {
		end := strings.LastIndex(serverName, "]")
		if end == -1 {
			return ErrInvalidServerName
		}
		// Valid IPv6 literal: must have ] at the end
		host := serverName[1:end] // the part inside [...]
		if host == "" {
			return ErrInvalidServerName
		}
		// Validate it's a valid IPv6 (contains only hex digits, :, ., or letters)
		for _, c := range host {
			if !isHexDigit(byte(c)) && c != ':' && c != '.' {
				return ErrInvalidServerName
			}
		}
		// Check port if present
		if end < len(serverName)-1 {
			if serverName[end+1] != ':' {
				return ErrInvalidServerName
			}
			portPart := serverName[end+2:]
			if !isValidPort(portPart) {
				return ErrInvalidServerName
			}
		}
		return nil
	}

	// Handle regular DNS name or IPv4:port
	colonIdx := strings.LastIndex(serverName, ":")
	if colonIdx != -1 {
		// Has a port
		host := serverName[:colonIdx]
		portPart := serverName[colonIdx+1:]
		if !isValidPort(portPart) {
			return ErrInvalidServerName
		}
		if host == "" {
			return ErrInvalidServerName
		}
		// Validate host doesn't contain illegal chars
		if strings.ContainsAny(host, "/@#[]:") {
			return ErrInvalidServerName
		}
	} else {
		// No port, just validate the host
		if strings.ContainsAny(serverName, "/@#[]:") {
			return ErrInvalidServerName
		}
	}

	return nil
}

// isHexDigit returns true if c is a valid character in an IPv6 address component.
func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// isValidPort checks that a port string is a valid port number (1-65535).
func isValidPort(port string) bool {
	if port == "" {
		return false
	}
	if len(port) > 5 {
		return false
	}
	var p int
	for _, c := range port {
		if c < '0' || c > '9' {
			return false
		}
		p = p*10 + int(c-'0')
	}
	return p >= 1 && p <= 65535
}

// ParseMatrixURI parses a Matrix URI (e.g., matrix:roomid?server=example.com).
func ParseMatrixURI(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	if u.Scheme != "matrix" {
		return "", fmt.Errorf("matrix: not a matrix URI scheme")
	}
	// Matrix URIs use opaque format: matrix:path
	// For user IDs: matrix:@alice:example.com → opaque = "@alice:example.com"
	// For room IDs: matrix:r101:example.com → opaque = "r101:example.com"
	if u.Opaque != "" {
		return u.Opaque, nil
	}
	return u.Path, nil
}
