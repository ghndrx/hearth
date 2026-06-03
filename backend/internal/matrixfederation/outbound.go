// Package matrixfederation implements Matrix Federation protocol support for Hearth.
// This file implements the outbound federation client for Server-Server API.
//
// Matrix Spec References:
//   - Federation API r0.1.4 § 12: Transactions
//   - Federation API r0.1.4 § 11: Room joining
//   - https://spec.matrix.org/v1.12/server-server-api/
package matrixfederation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"hearth/internal/matrix"
)

// FederationClient handles outbound federation requests to remote Matrix homeservers.
// It implements the Matrix Server-Server API client.
type FederationClient struct {
	httpClient    *http.Client
	keyStore      *KeyStore
	homeserverCfg *matrix.HomeserverConfig
	userAgent     string
	rateLimiter  *FederationRateLimiter
}

// FederationClientOptions configures the federation client.
type FederationClientOptions struct {
	// HTTPClient is a custom HTTP client (optional).
	HTTPClient *http.Client

	// UserAgent is the User-Agent string for requests.
	UserAgent string

	// RequestTimeout is the timeout for federation requests.
	RequestTimeout time.Duration

	// MaxRequestsPerWindow is the max requests per remote server per window.
	// Default: 100 requests per 10 seconds.
	MaxRequestsPerWindow int

	// RateLimitWindow is the time window for rate limiting.
	RateLimitWindow time.Duration
}

// NewFederationClient creates a new outbound federation client.
func NewFederationClient(keyStore *KeyStore, homeserverCfg *matrix.HomeserverConfig, opts *FederationClientOptions) *FederationClient {
	if opts == nil {
		opts = &FederationClientOptions{}
	}

	timeout := opts.RequestTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: timeout,
		}
	}

	userAgent := opts.UserAgent
	if userAgent == "" {
		userAgent = "Hearth Federation Client"
	}

	return &FederationClient{
		httpClient:    httpClient,
		keyStore:      keyStore,
		homeserverCfg: homeserverCfg,
		userAgent:     userAgent,
		rateLimiter:  NewFederationRateLimiter(opts.MaxRequestsPerWindow, opts.RateLimitWindow),
	}
}

// ResolveServerName performs server discovery per Matrix spec § 3.1.
// It resolves a server name to a target URL for federation requests.
//
// Steps:
// 1. Check for an explicit port in the server name
// 2. Check .well-known/matrix/server
// 3. Use HTTPS on port 8448.
func (c *FederationClient) ResolveServerName(ctx context.Context, serverName string) (string, error) {
	// Step 1: Check for explicit port.
	if strings.Contains(serverName, ":") {
		// Has explicit port - use it directly.
		return "https://" + serverName, nil
	}

	// Step 2: Try .well-known discovery.
	wellKnownURL := "https://" + serverName + "/.well-known/matrix/server"
	wellKnownReq, err := http.NewRequestWithContext(ctx, http.MethodGet, wellKnownURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create well-known request: %w", err)
	}

	wellKnownResp, err := c.httpClient.Do(wellKnownReq)
	if err == nil && wellKnownResp.StatusCode == http.StatusOK {
		defer wellKnownResp.Body.Close()

		var wellKnown ServerWellKnown
		if err := json.NewDecoder(wellKnownResp.Body).Decode(&wellKnown); err == nil && wellKnown.ServerName != "" {
			return "https://" + wellKnown.ServerName, nil
		}
	} else if wellKnownResp != nil {
		wellKnownResp.Body.Close()
	}

	// Step 3: Fallback to HTTPS on port 8448.
	return "https://" + serverName + ":8448", nil
}

// SendTransaction sends a transaction (batch of PDUs and EDUs) to a remote server.
//
// Matrix spec § 12: PUT /_matrix/federation/v1/send/{txnId}.
func (c *FederationClient) SendTransaction(ctx context.Context, serverName string, txnID string, pdus []map[string]interface{}, edus []map[string]interface{}) error {
	// Check rate limit before attempting request
	if c.rateLimiter != nil && !c.rateLimiter.Allow(serverName) {
		return fmt.Errorf("rate limited for server %s", serverName)
	}

	targetURL, err := c.ResolveServerName(ctx, serverName)
	if err != nil {
		return fmt.Errorf("failed to resolve server %s: %w", serverName, err)
	}

	body := map[string]interface{}{
		"pdus": pdus,
		"edus": edus,
	}

	// Sign the request.
	if err := c.keyStore.SignJSON(body); err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	u := targetURL + "/_matrix/federation/v1/send/" + url.PathEscape(txnID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	// Encode body.
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to encode body: %w", err)
	}
	req.Body = io.NopCloser(bytes.NewReader(bodyJSON))
	req.ContentLength = int64(len(bodyJSON))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("transaction failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetServerKeys fetches the signing keys from a remote server.
//
// Matrix spec § 2: GET /_matrix/key/v2/server.
func (c *FederationClient) GetServerKeys(ctx context.Context, serverName string) (*ServerKeyResponse, error) {
	targetURL, err := c.ResolveServerName(ctx, serverName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server %s: %w", serverName, err)
	}

	u := targetURL + "/_matrix/key/v2/server"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch server keys: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server keys request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result ServerKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode server keys response: %w", err)
	}

	return &result, nil
}

// QueryProfile fetches a user profile from a remote server.
//
// Matrix spec § 11.2: GET /_matrix/federation/v1/query/profile.
func (c *FederationClient) QueryProfile(ctx context.Context, serverName string, userID string, field string) (*UserProfile, error) {
	targetURL, err := c.ResolveServerName(ctx, serverName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server %s: %w", serverName, err)
	}

	query := url.Values{}
	query.Set("user_id", userID)
	if field != "" {
		query.Set("field", field)
	}

	u := targetURL + "/_matrix/federation/v1/query/profile?" + query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrUserNotFound
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("profile query failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result UserProfile
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode profile response: %w", err)
	}

	return &result, nil
}

// MakeJoin initiates a join request for a room on a remote server.
//
// Matrix spec § 11.1.1: GET /_matrix/federation/v1/make_join/{roomId}/{userId}.
func (c *FederationClient) MakeJoin(ctx context.Context, serverName string, roomID string, userID string) (*RoomStateResponse, error) {
	targetURL, err := c.ResolveServerName(ctx, serverName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server %s: %w", serverName, err)
	}

	u := fmt.Sprintf("%s/_matrix/federation/v1/make_join/%s/%s",
		targetURL,
		url.PathEscape(roomID),
		url.PathEscape(userID),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make join: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("make_join failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result RoomStateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode make_join response: %w", err)
	}

	return &result, nil
}

// SendJoin sends a join event to a remote server.
//
// Matrix spec § 11.1.2: PUT /_matrix/federation/v1/send_join/{roomId}/{eventId}.
func (c *FederationClient) SendJoin(ctx context.Context, serverName string, roomID string, eventID string, event map[string]interface{}) (*RoomStateResponse, error) {
	targetURL, err := c.ResolveServerName(ctx, serverName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server %s: %w", serverName, err)
	}

	// Sign the event.
	if err := c.keyStore.SignJSON(event); err != nil {
		return nil, fmt.Errorf("failed to sign join event: %w", err)
	}

	u := fmt.Sprintf("%s/_matrix/federation/v1/send_join/%s/%s",
		targetURL,
		url.PathEscape(roomID),
		url.PathEscape(eventID),
	)

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to encode event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, bytes.NewReader(eventJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send join: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("send_join failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result RoomStateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode send_join response: %w", err)
	}

	return &result, nil
}

// QueryDirectory resolves a room alias on a remote server.
//
// Matrix spec § 10.5.2: GET /_matrix/federation/v1/query/directory.
func (c *FederationClient) QueryDirectory(ctx context.Context, serverName string, roomAlias string) (*RoomDirectoryResponse, error) {
	targetURL, err := c.ResolveServerName(ctx, serverName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server %s: %w", serverName, err)
	}

	query := url.Values{}
	query.Set("room_alias", roomAlias)

	u := targetURL + "/_matrix/federation/v1/query/directory?" + query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query directory: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrAliasNotFound
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("directory query failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result RoomDirectoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode directory response: %w", err)
	}

	return &result, nil
}

// Backfill fetches older events in a room from a remote server.
//
// Matrix spec § 13: GET /_matrix/federation/v1/backfill/{roomId}.
func (c *FederationClient) Backfill(ctx context.Context, serverName string, roomID string, limit int, eventIDs []string) ([]map[string]interface{}, error) {
	targetURL, err := c.ResolveServerName(ctx, serverName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server %s: %w", serverName, err)
	}

	query := url.Values{}
	query.Set("v", "1") // room version.
	query.Set("limit", fmt.Sprintf("%d", limit))
	for _, id := range eventIDs {
		query.Add("event_id", id)
	}

	u := fmt.Sprintf("%s/_matrix/federation/v1/backfill/%s?%s",
		targetURL,
		url.PathEscape(roomID),
		query.Encode(),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to backfill: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("backfill failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		PDUs []json.RawMessage `json:"pdus"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode backfill response: %w", err)
	}

	events := make([]map[string]interface{}, len(result.PDUs))
	for i, pdu := range result.PDUs {
		var event map[string]interface{}
		if err := json.Unmarshal(pdu, &event); err != nil {
			return nil, fmt.Errorf("failed to decode PDU %d: %w", i, err)
		}
		events[i] = event
	}

	return events, nil
}

// GetEvent fetches a specific event from a remote server.
//
// Matrix spec § 12.1: GET /_matrix/federation/v1/event/{eventId}.
func (c *FederationClient) GetEvent(ctx context.Context, serverName string, eventID string) (map[string]interface{}, error) {
	targetURL, err := c.ResolveServerName(ctx, serverName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server %s: %w", serverName, err)
	}

	u := targetURL + "/_matrix/federation/v1/event/" + url.PathEscape(eventID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("event not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get event failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode event response: %w", err)
	}

	return result, nil
}

// InviteUser sends an invite to a user on a remote server.
//
// Matrix spec § 11.2: PUT /_matrix/federation/v1/invite/{roomId}/{eventId}.
func (c *FederationClient) InviteUser(ctx context.Context, serverName string, roomID string, eventID string, event map[string]interface{}) error {
	targetURL, err := c.ResolveServerName(ctx, serverName)
	if err != nil {
		return fmt.Errorf("failed to resolve server %s: %w", serverName, err)
	}

	// Sign the event.
	if err := c.keyStore.SignJSON(event); err != nil {
		return fmt.Errorf("failed to sign invite event: %w", err)
	}

	u := fmt.Sprintf("%s/_matrix/federation/v2/invite/%s/%s",
		targetURL,
		url.PathEscape(roomID),
		url.PathEscape(eventID),
	)

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to encode event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, bytes.NewReader(eventJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send invite: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("invite failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
