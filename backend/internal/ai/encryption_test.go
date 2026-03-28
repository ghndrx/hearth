package ai

import (
	"testing"
)

func TestAESEncryptionService(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name:    "16-byte key",
			key:     "1234567890123456",
			wantErr: false,
		},
		{
			name:    "24-byte key",
			key:     "123456789012345678901234",
			wantErr: false,
		},
		{
			name:    "32-byte key",
			key:     "12345678901234567890123456789012",
			wantErr: false,
		},
		{
			name:    "arbitrary length key (derived)",
			key:     "any-length-key",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := NewAESEncryptionService(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAESEncryptionService() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && enc == nil {
				t.Error("Expected encryption service, got nil")
			}
		})
	}
}

func TestAESEncryptDecrypt(t *testing.T) {
	enc, err := NewAESEncryptionService("test-encryption-key-32-bytes!!!")
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}

	testCases := []string{
		"simple text",
		"",
		`{"api_key": "sk-secret-key-123", "secret_key": "secret"}`,
		"Unicode: 你好世界 🎉",
		string(make([]byte, 10000)), // Long text
	}

	for _, plaintext := range testCases {
		encrypted, err := enc.Encrypt(plaintext)
		if err != nil {
			t.Errorf("Encrypt(%q) error: %v", plaintext, err)
			continue
		}

		if encrypted == plaintext && plaintext != "" {
			t.Errorf("Encrypted should differ from plaintext")
		}

		decrypted, err := enc.Decrypt(encrypted)
		if err != nil {
			t.Errorf("Decrypt() error: %v", err)
			continue
		}

		if decrypted != plaintext {
			t.Errorf("Decrypted = %q, want %q", decrypted, plaintext)
		}
	}
}

func TestAESDecryptInvalid(t *testing.T) {
	enc, _ := NewAESEncryptionService("test-key")

	tests := []struct {
		name       string
		ciphertext string
	}{
		{
			name:       "invalid base64",
			ciphertext: "not-valid-base64!!!",
		},
		{
			name:       "too short",
			ciphertext: "YWJj", // base64 of "abc"
		},
		{
			name:       "wrong key",
			ciphertext: "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkw", // Random valid base64
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := enc.Decrypt(tt.ciphertext)
			if err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

func TestNoOpEncryptionService(t *testing.T) {
	enc := NewNoOpEncryptionService()

	plaintext := "test data"

	encrypted, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error: %v", err)
	}

	decrypted, err := enc.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt() error: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestNoOpDecryptInvalidBase64(t *testing.T) {
	enc := NewNoOpEncryptionService()

	_, err := enc.Decrypt("not-valid-base64!!!")
	if err == nil {
		t.Error("Expected error for invalid base64")
	}
}

func TestDeriveKey(t *testing.T) {
	tests := []struct {
		name   string
		input  []byte
		length int
	}{
		{
			name:   "16 bytes",
			input:  []byte("test"),
			length: 16,
		},
		{
			name:   "32 bytes",
			input:  []byte("test"),
			length: 32,
		},
		{
			name:   "longer than 32",
			input:  []byte("test"),
			length: 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deriveKey(tt.input, tt.length)
			if len(result) != tt.length {
				t.Errorf("deriveKey() length = %d, want %d", len(result), tt.length)
			}
		})
	}
}

func TestEncryptionConsistency(t *testing.T) {
	enc, _ := NewAESEncryptionService("consistent-key")

	plaintext := "test data"

	// Encrypt twice
	enc1, _ := enc.Encrypt(plaintext)
	enc2, _ := enc.Encrypt(plaintext)

	// Should be different (due to random nonce)
	if enc1 == enc2 {
		t.Error("Two encryptions of same plaintext should differ (random nonce)")
	}

	// But both should decrypt to same plaintext
	dec1, _ := enc.Decrypt(enc1)
	dec2, _ := enc.Decrypt(enc2)

	if dec1 != dec2 {
		t.Error("Both should decrypt to same plaintext")
	}
}
