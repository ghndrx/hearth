package matrixfederation

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestGenerateSigningKey(t *testing.T) {
	key, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("GenerateSigningKey() error = %v", err)
	}

	// Verify key ID format
	alg, name, err := ParseKeyID(key.KeyID)
	if err != nil {
		t.Errorf("ParseKeyID(%q) error = %v", key.KeyID, err)
	}
	if alg != "ed25519" {
		t.Errorf("Algorithm() = %q, want ed25519", alg)
	}
	if name == "" {
		t.Error("Key name is empty")
	}

	// Verify key lengths
	if len(key.PublicKey) != ed25519.PublicKeySize {
		t.Errorf("PublicKey length = %d, want %d", len(key.PublicKey), ed25519.PublicKeySize)
	}
	if len(key.PrivateKey) != ed25519.PrivateKeySize {
		t.Errorf("PrivateKey length = %d, want %d", len(key.PrivateKey), ed25519.PrivateKeySize)
	}

	// Verify CanSign
	if !key.CanSign() {
		t.Error("CanSign() = false, want true")
	}
}

func TestGenerateSigningKeyWithName(t *testing.T) {
	key, err := GenerateSigningKeyWithName("test_key")
	if err != nil {
		t.Fatalf("GenerateSigningKeyWithName() error = %v", err)
	}

	if key.KeyID != "ed25519:test_key" {
		t.Errorf("KeyID = %q, want ed25519:test_key", key.KeyID)
	}
	if key.Name() != "test_key" {
		t.Errorf("Name() = %q, want test_key", key.Name())
	}
}

func TestGenerateSigningKeyWithName_Empty(t *testing.T) {
	_, err := GenerateSigningKeyWithName("")
	if err == nil {
		t.Error("GenerateSigningKeyWithName(\"\") expected error, got nil")
	}
}

func TestParseKeyID(t *testing.T) {
	tests := []struct {
		name     string
		keyID    string
		wantAlg  string
		wantName string
		wantErr  bool
	}{
		{
			name:     "valid ed25519 key",
			keyID:    "ed25519:a_Obwu",
			wantAlg:  "ed25519",
			wantName: "a_Obwu",
		},
		{
			name:     "valid with underscore",
			keyID:    "ed25519:key_name_123",
			wantAlg:  "ed25519",
			wantName: "key_name_123",
		},
		{
			name:    "missing colon",
			keyID:   "ed25519",
			wantErr: true,
		},
		{
			name:    "empty algorithm",
			keyID:   ":keyname",
			wantErr: true,
		},
		{
			name:    "empty name",
			keyID:   "ed25519:",
			wantErr: true,
		},
		{
			name:    "empty string",
			keyID:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alg, name, err := ParseKeyID(tt.keyID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseKeyID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if alg != tt.wantAlg {
					t.Errorf("algorithm = %q, want %q", alg, tt.wantAlg)
				}
				if name != tt.wantName {
					t.Errorf("name = %q, want %q", name, tt.wantName)
				}
			}
		})
	}
}

func TestSigningKey_SignAndVerify(t *testing.T) {
	key, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("GenerateSigningKey() error = %v", err)
	}

	message := []byte("Hello, Matrix!")

	// Sign the message
	sig, err := key.Sign(message)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	// Verify the signature
	if err := key.Verify(message, sig); err != nil {
		t.Errorf("Verify() error = %v", err)
	}

	// Verify with wrong message should fail
	if err := key.Verify([]byte("wrong message"), sig); err == nil {
		t.Error("Verify() with wrong message should fail")
	}

	// Verify with wrong signature should fail
	if err := key.Verify(message, "wrongsignature"); err == nil {
		t.Error("Verify() with wrong signature should fail")
	}
}

func TestSigningKey_IsExpired(t *testing.T) {
	key, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("GenerateSigningKey() error = %v", err)
	}

	// No expiration
	if key.IsExpired() {
		t.Error("Key with no ValidUntilTS should not be expired")
	}

	// Set to future
	key.ValidUntilTS = time.Now().Add(time.Hour).UnixMilli()
	if key.IsExpired() {
		t.Error("Key with future ValidUntilTS should not be expired")
	}

	// Set to past
	key.ValidUntilTS = time.Now().Add(-time.Hour).UnixMilli()
	if !key.IsExpired() {
		t.Error("Key with past ValidUntilTS should be expired")
	}
}

func TestSigningKey_ExpiredCannotSign(t *testing.T) {
	key, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("GenerateSigningKey() error = %v", err)
	}

	// Expire the key
	key.ValidUntilTS = time.Now().Add(-time.Hour).UnixMilli()

	if key.CanSign() {
		t.Error("Expired key should not be able to sign")
	}

	_, err = key.Sign([]byte("test"))
	if err != ErrKeyExpired {
		t.Errorf("Sign() error = %v, want ErrKeyExpired", err)
	}
}

func TestSigningKey_PublicKeyBase64(t *testing.T) {
	key, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("GenerateSigningKey() error = %v", err)
	}

	b64 := key.PublicKeyBase64()
	if b64 == "" {
		t.Error("PublicKeyBase64() returned empty string")
	}

	// Should be decodable
	decoded, err := base64.RawStdEncoding.DecodeString(b64)
	if err != nil {
		t.Errorf("Failed to decode base64: %v", err)
	}
	if len(decoded) != ed25519.PublicKeySize {
		t.Errorf("Decoded length = %d, want %d", len(decoded), ed25519.PublicKeySize)
	}
}

func TestKeyStore_AddAndGetKey(t *testing.T) {
	ks := NewKeyStore("hearth.example.com")

	key1, _ := GenerateSigningKeyWithName("key1")
	key2, _ := GenerateSigningKeyWithName("key2")

	// Add first key as primary
	ks.AddKey(key1, true)

	// Add second key, not primary
	ks.AddKey(key2, false)

	// Get primary
	primary, err := ks.GetPrimaryKey()
	if err != nil {
		t.Fatalf("GetPrimaryKey() error = %v", err)
	}
	if primary.KeyID != key1.KeyID {
		t.Errorf("Primary key = %q, want %q", primary.KeyID, key1.KeyID)
	}

	// Get by ID
	got, err := ks.GetKey(key2.KeyID)
	if err != nil {
		t.Fatalf("GetKey() error = %v", err)
	}
	if got.KeyID != key2.KeyID {
		t.Errorf("GetKey() = %q, want %q", got.KeyID, key2.KeyID)
	}
}

func TestKeyStore_RetireKey(t *testing.T) {
	ks := NewKeyStore("hearth.example.com")

	key1, _ := GenerateSigningKeyWithName("key1")
	key2, _ := GenerateSigningKeyWithName("key2")

	ks.AddKey(key1, true)
	ks.AddKey(key2, false)

	// Retire key1 (the primary)
	err := ks.RetireKey(key1.KeyID)
	if err != nil {
		t.Fatalf("RetireKey() error = %v", err)
	}

	// key2 should now be primary
	primary, err := ks.GetPrimaryKey()
	if err != nil {
		t.Fatalf("GetPrimaryKey() error = %v", err)
	}
	if primary.KeyID != key2.KeyID {
		t.Errorf("Primary after retire = %q, want %q", primary.KeyID, key2.KeyID)
	}

	// key1 should still be retrievable
	got, err := ks.GetKey(key1.KeyID)
	if err != nil {
		t.Fatalf("GetKey() retired key error = %v", err)
	}
	if got.ValidUntilTS == 0 {
		t.Error("Retired key should have ValidUntilTS set")
	}
}

func TestKeyStore_RetireNonexistent(t *testing.T) {
	ks := NewKeyStore("hearth.example.com")

	err := ks.RetireKey("ed25519:nonexistent")
	if err != ErrKeyNotFound {
		t.Errorf("RetireKey() error = %v, want ErrKeyNotFound", err)
	}
}

func TestKeyStore_GetServerKeyResponse(t *testing.T) {
	ks := NewKeyStore("hearth.example.com")

	key1, _ := GenerateSigningKeyWithName("primary")
	key2, _ := GenerateSigningKeyWithName("secondary")

	ks.AddKey(key1, true)
	ks.AddKey(key2, false)

	// Retire key2
	ks.RetireKey(key2.KeyID)

	resp, err := ks.GetServerKeyResponse(24 * time.Hour)
	if err != nil {
		t.Fatalf("GetServerKeyResponse() error = %v", err)
	}

	// Check server name
	if resp.ServerName != "hearth.example.com" {
		t.Errorf("ServerName = %q, want hearth.example.com", resp.ServerName)
	}

	// Check verify_keys contains primary
	if _, ok := resp.VerifyKeys[key1.KeyID]; !ok {
		t.Errorf("VerifyKeys missing %s", key1.KeyID)
	}

	// Check old_verify_keys contains retired key
	if _, ok := resp.OldVerifyKeys[key2.KeyID]; !ok {
		t.Errorf("OldVerifyKeys missing %s", key2.KeyID)
	}

	// Check valid_until_ts is in the future
	if resp.ValidUntilTS <= time.Now().UnixMilli() {
		t.Error("ValidUntilTS should be in the future")
	}

	// Check signatures
	if resp.Signatures == nil {
		t.Error("Signatures should not be nil")
	}
	if _, ok := resp.Signatures["hearth.example.com"]; !ok {
		t.Error("Signatures missing server entry")
	}
	if _, ok := resp.Signatures["hearth.example.com"][key1.KeyID]; !ok {
		t.Error("Signatures missing key entry")
	}
}

func TestKeyStore_SignJSON(t *testing.T) {
	ks := NewKeyStore("hearth.example.com")

	key, _ := GenerateSigningKeyWithName("sign_test")
	ks.AddKey(key, true)

	obj := map[string]any{
		"type":    "m.room.message",
		"content": "Hello!",
	}

	err := ks.SignJSON(obj)
	if err != nil {
		t.Fatalf("SignJSON() error = %v", err)
	}

	// Check signatures field was added
	sigs, ok := obj["signatures"].(map[string]map[string]string)
	if !ok {
		t.Fatal("signatures field not found or wrong type")
	}

	serverSigs, ok := sigs["hearth.example.com"]
	if !ok {
		t.Error("Server signatures not found")
	}

	if _, ok := serverSigs[key.KeyID]; !ok {
		t.Errorf("Key signature not found for %s", key.KeyID)
	}
}

func TestKeyStore_VerifyJSON(t *testing.T) {
	ks := NewKeyStore("hearth.example.com")

	key, _ := GenerateSigningKeyWithName("verify_test")
	ks.AddKey(key, true)

	obj := map[string]any{
		"type":    "m.room.message",
		"content": "Hello!",
	}

	// Sign it
	err := ks.SignJSON(obj)
	if err != nil {
		t.Fatalf("SignJSON() error = %v", err)
	}

	// Verify it - need to re-decode to simulate JSON roundtrip
	objJSON, _ := json.Marshal(obj)
	var roundtripped map[string]any
	json.Unmarshal(objJSON, &roundtripped)

	err = ks.VerifyJSON(roundtripped, "hearth.example.com", key.KeyID)
	if err != nil {
		t.Errorf("VerifyJSON() error = %v", err)
	}
}

func TestKeyStore_GetAllCurrentKeys(t *testing.T) {
	ks := NewKeyStore("hearth.example.com")

	key1, _ := GenerateSigningKeyWithName("key1")
	key2, _ := GenerateSigningKeyWithName("key2")
	key3, _ := GenerateSigningKeyWithName("key3")

	ks.AddKey(key1, true)
	ks.AddKey(key2, false)
	ks.AddKey(key3, false)

	// Retire one
	ks.RetireKey(key2.KeyID)

	keys := ks.GetAllCurrentKeys()
	if len(keys) != 2 {
		t.Errorf("GetAllCurrentKeys() returned %d keys, want 2", len(keys))
	}

	// Should not include retired key
	for _, k := range keys {
		if k.KeyID == key2.KeyID {
			t.Error("Retired key should not be in current keys")
		}
	}
}

func TestKeyStore_EmptyStore(t *testing.T) {
	ks := NewKeyStore("hearth.example.com")

	// GetPrimaryKey on empty store
	_, err := ks.GetPrimaryKey()
	if err != ErrKeyNotFound {
		t.Errorf("GetPrimaryKey() on empty store = %v, want ErrKeyNotFound", err)
	}

	// GetKey on empty store
	_, err = ks.GetKey("ed25519:nonexistent")
	if err != ErrKeyNotFound {
		t.Errorf("GetKey() on empty store = %v, want ErrKeyNotFound", err)
	}
}
