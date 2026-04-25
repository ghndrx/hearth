package matrixfederation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/matrix"
)

func TestNewFederationClient(t *testing.T) {
	ks := NewKeyStore("hearth.example.com")
	cfg := &matrix.HomeserverConfig{
		ServerName: "hearth.example.com",
		BaseURL:    "https://hearth.example.com",
	}

	client := NewFederationClient(ks, cfg, nil)
	assert.NotNil(t, client)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, "Hearth Federation Client", client.userAgent)
}

func TestNewFederationClient_WithOptions(t *testing.T) {
	ks := NewKeyStore("hearth.example.com")
	cfg := &matrix.HomeserverConfig{
		ServerName: "hearth.example.com",
		BaseURL:    "https://hearth.example.com",
	}

	customClient := &http.Client{Timeout: 10 * time.Second}
	opts := &FederationClientOptions{
		HTTPClient:     customClient,
		UserAgent:      "CustomAgent",
		RequestTimeout: 10 * time.Second,
	}

	client := NewFederationClient(ks, cfg, opts)
	assert.NotNil(t, client)
	assert.Equal(t, customClient, client.httpClient)
	assert.Equal(t, "CustomAgent", client.userAgent)
}

func TestFederationClient_ResolveServerName_WellKnown(t *testing.T) {
	// Create a mock server that responds to .well-known.
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/matrix/server" {
			resp := ServerWellKnown{ServerName: "federation.example.com:8448"}
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	ks := NewKeyStore("hearth.example.com")
	cfg := &matrix.HomeserverConfig{
		ServerName: "hearth.example.com",
		BaseURL:    "https://hearth.example.com",
	}

	// Use a custom HTTP client that routes "mockserver" to the test server
	transport := &mockTransport{server: mockServer}
	customClient := &http.Client{Transport: transport, Timeout: 10 * time.Second}
	opts := &FederationClientOptions{HTTPClient: customClient}

	client := NewFederationClient(ks, cfg, opts)

	resolved, err := client.ResolveServerName(context.Background(), "mockserver")
	require.NoError(t, err)
	assert.Equal(t, "https://federation.example.com:8448", resolved)
}

func TestFederationClient_ResolveServerName_Fallback(t *testing.T) {
	// Create a mock server that returns 404 for .well-known.
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	ks := NewKeyStore("hearth.example.com")
	cfg := &matrix.HomeserverConfig{
		ServerName: "hearth.example.com",
		BaseURL:    "https://hearth.example.com",
	}

	transport := &mockTransport{server: mockServer}
	customClient := &http.Client{Transport: transport, Timeout: 10 * time.Second}
	opts := &FederationClientOptions{HTTPClient: customClient}

	client := NewFederationClient(ks, cfg, opts)

	resolved, err := client.ResolveServerName(context.Background(), "mockserver")
	require.NoError(t, err)
	// Should fallback to port 8448.
	assert.Equal(t, "https://mockserver:8448", resolved)
}

func TestFederationClient_GetServerKeys(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_matrix/key/v2/server" {
			resp := ServerKeyResponse{
				ServerName: "remote.example.com",
				VerifyKeys: map[string]VerifyKey{
					"ed25519:test": {Key: "dGVzdGtleQ=="},
				},
				ValidUntilTS: time.Now().Add(24 * time.Hour).UnixMilli(),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	client, _ := newMockFederationClient(mockServer)

	result, err := client.GetServerKeys(context.Background(), "mockserver")
	require.NoError(t, err)
	assert.Equal(t, "remote.example.com", result.ServerName)
	assert.Contains(t, result.VerifyKeys, "ed25519:test")
}

func TestFederationClient_QueryDirectory(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_matrix/federation/v1/query/directory" {
			resp := RoomDirectoryResponse{
				RoomID:  "!abc123:remote.example.com",
				Servers: []string{"remote.example.com"},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	client, _ := newMockFederationClient(mockServer)

	result, err := client.QueryDirectory(context.Background(), "mockserver", "#general:remote.example.com")
	require.NoError(t, err)
	assert.Equal(t, "!abc123:remote.example.com", result.RoomID)
}

func TestFederationClient_QueryDirectory_NotFound(t *testing.T) {
	// Create a mock server that returns 404.
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errcode":"M_NOT_FOUND"}`))
	}))
	defer mockServer.Close()

	ks := NewKeyStore("hearth.example.com")
	cfg := &matrix.HomeserverConfig{
		ServerName: "hearth.example.com",
		BaseURL:    "https://hearth.example.com",
	}

	client := NewFederationClient(ks, cfg, nil)

	serverURL, _ := url.Parse(mockServer.URL)
	serverName := serverURL.Host

	_, err := client.QueryDirectory(context.Background(), serverName, "#nonexistent:remote.example.com")
	assert.Error(t, err)
}

func TestFederationClient_QueryProfile(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_matrix/federation/v1/query/profile" {
			resp := UserProfile{
				UserID:      "@alice:remote.example.com",
				DisplayName: strPtr("Alice"),
				AvatarURL:   strPtr("mxc://remote.example.com/abc123"),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	client, _ := newMockFederationClient(mockServer)

	result, err := client.QueryProfile(context.Background(), "mockserver", "@alice:remote.example.com", "")
	require.NoError(t, err)
	assert.Equal(t, "@alice:remote.example.com", result.UserID)
}

func TestFederationClient_GetEvent(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_matrix/federation/v1/event/$abc123" {
			resp := map[string]interface{}{
				"event_id": "$abc123",
				"type":     "m.room.message",
				"content": map[string]interface{}{
					"body": "Hello",
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	client, _ := newMockFederationClient(mockServer)

	result, err := client.GetEvent(context.Background(), "mockserver", "$abc123")
	require.NoError(t, err)
	assert.Equal(t, "$abc123", result["event_id"])
}

func TestFederationClient_Backfill(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_matrix/federation/v1/backfill/!room:remote.example.com" {
			resp := struct {
				PDUs []json.RawMessage `json:"pdus"`
			}{
				PDUs: []json.RawMessage{
					json.RawMessage(`{"event_id":"$1","type":"m.room.message"}`),
					json.RawMessage(`{"event_id":"$2","type":"m.room.message"}`),
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	client, _ := newMockFederationClient(mockServer)

	result, err := client.Backfill(context.Background(), "mockserver", "!room:remote.example.com", 10, []string{"$start"})
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func strPtr(s string) *string {
	return &s
}

// mockTransport is a test HTTP transport that routes requests for "mockserver"
// to a local httptest.Server, avoiding port-in-hostname issues.
type mockTransport struct {
	server *httptest.Server
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == "mockserver" || req.URL.Host == "mockserver:8448" {
		// Rewrite URL to point to the test server
		serverURL, _ := url.Parse(t.server.URL)
		req.URL.Scheme = serverURL.Scheme
		req.URL.Host = serverURL.Host
		req.Host = serverURL.Host
	}
	return http.DefaultTransport.RoundTrip(req)
}

func newMockFederationClient(mockServer *httptest.Server) (*FederationClient, *matrix.HomeserverConfig) {
	ks := NewKeyStore("hearth.example.com")
	cfg := &matrix.HomeserverConfig{
		ServerName: "hearth.example.com",
		BaseURL:    "https://hearth.example.com",
	}
	transport := &mockTransport{server: mockServer}
	customClient := &http.Client{Transport: transport, Timeout: 10 * time.Second}
	opts := &FederationClientOptions{HTTPClient: customClient}
	return NewFederationClient(ks, cfg, opts), cfg
}
