package ai

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

var (
	ErrInvalidKey        = errors.New("invalid encryption key")
	ErrDecryptionFailed  = errors.New("decryption failed")
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
)

// AESEncryptionService implements EncryptionService using AES-GCM
type AESEncryptionService struct {
	key []byte
}

// NewAESEncryptionService creates a new AES encryption service
// The key must be 16, 24, or 32 bytes for AES-128, AES-192, or AES-256
func NewAESEncryptionService(key string) (*AESEncryptionService, error) {
	keyBytes := []byte(key)

	switch len(keyBytes) {
	case 16, 24, 32:
		// Valid key length
	default:
		// Derive a 32-byte key using simple hash
		// In production, use a proper key derivation function like PBKDF2 or Argon2
		keyBytes = deriveKey(keyBytes, 32)
	}

	return &AESEncryptionService{key: keyBytes}, nil
}

// Encrypt encrypts plaintext using AES-GCM
func (s *AESEncryptionService) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext using AES-GCM
func (s *AESEncryptionService) Decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", ErrInvalidCiphertext
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", ErrInvalidCiphertext
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}

// deriveKey derives a key of the specified length using SHA256
// In production, use a proper KDF like PBKDF2 or Argon2
func deriveKey(input []byte, length int) []byte {
	hash := sha256.Sum256(input)
	if length <= 32 {
		return hash[:length]
	}
	// For longer keys, repeat the hash (not recommended for production)
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = hash[i%32]
	}
	return result
}

// NoOpEncryptionService is a no-op encryption service for testing/development
type NoOpEncryptionService struct{}

// NewNoOpEncryptionService creates a no-op encryption service
func NewNoOpEncryptionService() *NoOpEncryptionService {
	return &NoOpEncryptionService{}
}

// Encrypt returns the plaintext base64 encoded (not actually encrypted)
func (s *NoOpEncryptionService) Encrypt(plaintext string) (string, error) {
	return base64.StdEncoding.EncodeToString([]byte(plaintext)), nil
}

// Decrypt returns the base64 decoded plaintext
func (s *NoOpEncryptionService) Decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
