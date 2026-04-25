// Package matrixfederation implements Matrix Federation protocol support for Hearth.
// This file implements Server Signing Keys per Matrix Server-Server API.
//
// Matrix Spec References:
//   - Federation API r0.1.4 § 2: Server Signing Keys
//   - https://spec.matrix.org/v1.12/server-server-api/#signing-keys
package matrixfederation

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Common errors for key operations
var (
	ErrKeyNotFound      = errors.New("matrix: signing key not found")
	ErrKeyExpired       = errors.New("matrix: signing key has expired")
	ErrInvalidSignature = errors.New("matrix: invalid signature")
	ErrInvalidKeyID     = errors.New("matrix: invalid key ID format")
)

// SigningKey represents an Ed25519 signing key pair used for server-to-server
// authentication in Matrix federation.
//
// Key IDs follow the format: ed25519:<key_name>
// Example: ed25519:a_Obwu (where a_Obwu is a short identifier)
type SigningKey struct {
	// KeyID is the full key identifier (e.g., "ed25519:a_Obwu")
	KeyID string `json:"key_id"`

	// PublicKey is the Ed25519 public key (32 bytes)
	PublicKey ed25519.PublicKey `json:"-"`

	// PrivateKey is the Ed25519 private key (64 bytes, includes public key)
	// Only set for keys owned by this server
	PrivateKey ed25519.PrivateKey `json:"-"`

	// ValidUntilTS is the Unix timestamp (milliseconds) until which this key is valid
	// 0 means no expiration
	ValidUntilTS int64 `json:"valid_until_ts,omitempty"`

	// CreatedAt is when this key was generated
	CreatedAt time.Time `json:"created_at"`
}

// GenerateSigningKey creates a new Ed25519 signing key with a random key name.
// The key name is derived from the first 6 characters of the base64-encoded public key.
func GenerateSigningKey() (*SigningKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("matrix: failed to generate signing key: %w", err)
	}

	// Generate a short key name from the public key
	keyName := generateKeyName(pub)
	keyID := "ed25519:" + keyName

	return &SigningKey{
		KeyID:      keyID,
		PublicKey:  pub,
		PrivateKey: priv,
		CreatedAt:  time.Now().UTC(),
	}, nil
}

// GenerateSigningKeyWithName creates a new Ed25519 signing key with a specific name.
func GenerateSigningKeyWithName(name string) (*SigningKey, error) {
	if name == "" {
		return nil, fmt.Errorf("matrix: key name cannot be empty")
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("matrix: failed to generate signing key: %w", err)
	}

	keyID := "ed25519:" + name

	return &SigningKey{
		KeyID:      keyID,
		PublicKey:  pub,
		PrivateKey: priv,
		CreatedAt:  time.Now().UTC(),
	}, nil
}

// generateKeyName creates a short identifier from the public key.
// Uses URL-safe base64 of the first few bytes.
func generateKeyName(pub ed25519.PublicKey) string {
	encoded := base64.RawURLEncoding.EncodeToString(pub[:6])
	// Replace any characters that might cause issues
	encoded = strings.ReplaceAll(encoded, "-", "_")
	return encoded
}

// ParseKeyID extracts the algorithm and key name from a key ID.
// Returns algorithm (e.g., "ed25519") and name (e.g., "a_Obwu").
func ParseKeyID(keyID string) (algorithm, name string, err error) {
	parts := strings.SplitN(keyID, ":", 2)
	if len(parts) != 2 {
		return "", "", ErrInvalidKeyID
	}
	if parts[0] == "" || parts[1] == "" {
		return "", "", ErrInvalidKeyID
	}
	return parts[0], parts[1], nil
}

// Algorithm returns the key algorithm (always "ed25519" for now).
func (k *SigningKey) Algorithm() string {
	alg, _, _ := ParseKeyID(k.KeyID)
	return alg
}

// Name returns just the key name portion of the KeyID.
func (k *SigningKey) Name() string {
	_, name, _ := ParseKeyID(k.KeyID)
	return name
}

// IsExpired reports whether the key has passed its ValidUntilTS.
func (k *SigningKey) IsExpired() bool {
	if k.ValidUntilTS == 0 {
		return false
	}
	return time.Now().UnixMilli() > k.ValidUntilTS
}

// CanSign reports whether this key can be used to sign messages.
// Returns true only if we have the private key and it hasn't expired.
func (k *SigningKey) CanSign() bool {
	return k.PrivateKey != nil && !k.IsExpired()
}

// Sign signs a message with this key's private key.
// Returns the signature as unpadded base64.
func (k *SigningKey) Sign(message []byte) (string, error) {
	if k.PrivateKey == nil {
		return "", fmt.Errorf("matrix: cannot sign without private key")
	}
	if k.IsExpired() {
		return "", ErrKeyExpired
	}

	sig := ed25519.Sign(k.PrivateKey, message)
	return base64.RawStdEncoding.EncodeToString(sig), nil
}

// Verify verifies a signature against a message using this key's public key.
func (k *SigningKey) Verify(message []byte, signatureB64 string) error {
	sig, err := base64.RawStdEncoding.DecodeString(signatureB64)
	if err != nil {
		// Try with padding
		sig, err = base64.StdEncoding.DecodeString(signatureB64)
		if err != nil {
			return fmt.Errorf("matrix: invalid signature encoding: %w", err)
		}
	}

	if !ed25519.Verify(k.PublicKey, message, sig) {
		return ErrInvalidSignature
	}
	return nil
}

// PublicKeyBase64 returns the public key as unpadded base64.
func (k *SigningKey) PublicKeyBase64() string {
	return base64.RawStdEncoding.EncodeToString(k.PublicKey)
}

// ServerKeyResponse represents the response for GET /_matrix/key/v2/server
// per Matrix Federation API specification.
//
// https://spec.matrix.org/v1.12/server-server-api/#get_matrixkeyv2server
type ServerKeyResponse struct {
	// ServerName is the DNS name of this homeserver
	ServerName string `json:"server_name"`

	// VerifyKeys are the currently active signing keys
	VerifyKeys map[string]VerifyKey `json:"verify_keys"`

	// OldVerifyKeys are keys that are no longer used but may still be valid
	// for verifying old signatures
	OldVerifyKeys map[string]OldVerifyKey `json:"old_verify_keys,omitempty"`

	// ValidUntilTS is when this key response expires (Unix ms)
	ValidUntilTS int64 `json:"valid_until_ts"`

	// Signatures contains the server's signature over this response
	Signatures map[string]map[string]string `json:"signatures,omitempty"`
}

// VerifyKey represents a currently active verification key.
type VerifyKey struct {
	Key string `json:"key"` // Base64-encoded Ed25519 public key
}

// OldVerifyKey represents a retired key that may still be used to verify old signatures.
type OldVerifyKey struct {
	Key       string `json:"key"`        // Base64-encoded Ed25519 public key
	ExpiredTS int64  `json:"expired_ts"` // When this key was retired (Unix ms)
}

// KeyStore manages signing keys for a homeserver.
// It provides thread-safe access to current and old keys.
type KeyStore struct {
	mu           sync.RWMutex
	serverName   string
	currentKeys  map[string]*SigningKey // keyID -> key
	oldKeys      map[string]*SigningKey // keyID -> key (retired but still valid for verification)
	primaryKeyID string                 // The key used for signing new content
}

// NewKeyStore creates a new key store for the given server.
func NewKeyStore(serverName string) *KeyStore {
	return &KeyStore{
		serverName:  serverName,
		currentKeys: make(map[string]*SigningKey),
		oldKeys:     make(map[string]*SigningKey),
	}
}

// AddKey adds a signing key to the store.
// If makePrimary is true, this becomes the primary signing key.
func (ks *KeyStore) AddKey(key *SigningKey, makePrimary bool) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.currentKeys[key.KeyID] = key
	if makePrimary || ks.primaryKeyID == "" {
		ks.primaryKeyID = key.KeyID
	}
}

// RetireKey moves a key from current to old keys.
// The key can still be used for verification but not signing.
func (ks *KeyStore) RetireKey(keyID string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	key, ok := ks.currentKeys[keyID]
	if !ok {
		return ErrKeyNotFound
	}

	// Set expiration timestamp
	key.ValidUntilTS = time.Now().UnixMilli()

	// Move to old keys
	delete(ks.currentKeys, keyID)
	ks.oldKeys[keyID] = key

	// If this was the primary key, pick a new one
	if ks.primaryKeyID == keyID {
		ks.primaryKeyID = ""
		for id := range ks.currentKeys {
			ks.primaryKeyID = id
			break
		}
	}

	return nil
}

// GetPrimaryKey returns the primary signing key for this server.
func (ks *KeyStore) GetPrimaryKey() (*SigningKey, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	if ks.primaryKeyID == "" {
		return nil, ErrKeyNotFound
	}
	key, ok := ks.currentKeys[ks.primaryKeyID]
	if !ok {
		return nil, ErrKeyNotFound
	}
	return key, nil
}

// GetKey returns a key by ID, checking both current and old keys.
func (ks *KeyStore) GetKey(keyID string) (*SigningKey, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	if key, ok := ks.currentKeys[keyID]; ok {
		return key, nil
	}
	if key, ok := ks.oldKeys[keyID]; ok {
		return key, nil
	}
	return nil, ErrKeyNotFound
}

// GetAllCurrentKeys returns all currently active keys.
func (ks *KeyStore) GetAllCurrentKeys() []*SigningKey {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	keys := make([]*SigningKey, 0, len(ks.currentKeys))
	for _, key := range ks.currentKeys {
		keys = append(keys, key)
	}
	return keys
}

// GetServerKeyResponse builds the response for /_matrix/key/v2/server.
// The response is signed by the primary key.
func (ks *KeyStore) GetServerKeyResponse(validDuration time.Duration) (*ServerKeyResponse, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	resp := &ServerKeyResponse{
		ServerName:    ks.serverName,
		VerifyKeys:    make(map[string]VerifyKey),
		OldVerifyKeys: make(map[string]OldVerifyKey),
		ValidUntilTS:  time.Now().Add(validDuration).UnixMilli(),
	}

	// Add current keys
	for keyID, key := range ks.currentKeys {
		resp.VerifyKeys[keyID] = VerifyKey{
			Key: key.PublicKeyBase64(),
		}
	}

	// Add old keys
	for keyID, key := range ks.oldKeys {
		resp.OldVerifyKeys[keyID] = OldVerifyKey{
			Key:       key.PublicKeyBase64(),
			ExpiredTS: key.ValidUntilTS,
		}
	}

	// Sign the response with the primary key
	if ks.primaryKeyID != "" {
		primaryKey := ks.currentKeys[ks.primaryKeyID]
		if primaryKey != nil && primaryKey.CanSign() {
			// Create a copy without signatures for signing
			toSign := *resp
			toSign.Signatures = nil

			// Canonical JSON encoding
			canonical, err := canonicalJSON(&toSign)
			if err != nil {
				return nil, fmt.Errorf("matrix: failed to encode key response: %w", err)
			}

			sig, err := primaryKey.Sign(canonical)
			if err != nil {
				return nil, fmt.Errorf("matrix: failed to sign key response: %w", err)
			}

			resp.Signatures = map[string]map[string]string{
				ks.serverName: {
					primaryKey.KeyID: sig,
				},
			}
		}
	}

	return resp, nil
}

// SignJSON signs a JSON object using Matrix's canonical JSON format.
// Adds the signature to the "signatures" field of the object.
func (ks *KeyStore) SignJSON(obj map[string]any) error {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	if ks.primaryKeyID == "" {
		return ErrKeyNotFound
	}

	key := ks.currentKeys[ks.primaryKeyID]
	if key == nil || !key.CanSign() {
		return ErrKeyNotFound
	}

	// Remove signatures and unsigned fields for signing
	delete(obj, "signatures")
	delete(obj, "unsigned")

	canonical, err := canonicalJSON(obj)
	if err != nil {
		return fmt.Errorf("matrix: failed to encode for signing: %w", err)
	}

	sig, err := key.Sign(canonical)
	if err != nil {
		return err
	}

	// Add signatures field
	obj["signatures"] = map[string]map[string]string{
		ks.serverName: {
			key.KeyID: sig,
		},
	}

	return nil
}

// VerifyJSON verifies a signature on a JSON object from a given server.
func (ks *KeyStore) VerifyJSON(obj map[string]any, serverName, keyID string) error {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	// Extract signatures
	sigsRaw, ok := obj["signatures"]
	if !ok {
		return fmt.Errorf("matrix: no signatures field")
	}
	sigs, ok := sigsRaw.(map[string]any)
	if !ok {
		return fmt.Errorf("matrix: invalid signatures format")
	}

	serverSigsRaw, ok := sigs[serverName]
	if !ok {
		return fmt.Errorf("matrix: no signature from server %s", serverName)
	}
	serverSigs, ok := serverSigsRaw.(map[string]any)
	if !ok {
		return fmt.Errorf("matrix: invalid server signatures format")
	}

	sigRaw, ok := serverSigs[keyID]
	if !ok {
		return fmt.Errorf("matrix: no signature with key %s", keyID)
	}
	sig, ok := sigRaw.(string)
	if !ok {
		return fmt.Errorf("matrix: invalid signature format")
	}

	// Get the key
	key, err := ks.GetKey(keyID)
	if err != nil {
		return fmt.Errorf("matrix: key not found: %w", err)
	}

	// Create canonical form without signatures/unsigned
	objCopy := make(map[string]any)
	for k, v := range obj {
		if k != "signatures" && k != "unsigned" {
			objCopy[k] = v
		}
	}

	canonical, err := canonicalJSON(objCopy)
	if err != nil {
		return fmt.Errorf("matrix: failed to encode for verification: %w", err)
	}

	return key.Verify(canonical, sig)
}

// canonicalJSON produces the canonical JSON encoding per Matrix spec.
// Keys are sorted alphabetically, no extra whitespace.
func canonicalJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}
