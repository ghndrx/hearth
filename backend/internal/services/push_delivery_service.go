package services

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// PushDeliveryService handles sending push notifications to Web Push endpoints
type PushDeliveryService struct {
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDSubject    string

	httpClient *http.Client
	subsRepo   PushSubscriptionRepository
	cache      SmartNotificationCache
	eventBus   EventBus
}

// PushSubscriptionRepository defines push subscription data access
type PushSubscriptionRepository interface {
	Create(ctx context.Context, sub *models.PushSubscription) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByEndpoint(ctx context.Context, userID uuid.UUID, endpoint string) error
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.PushSubscription, error)
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*models.PushSubscription, error)
}

// NewPushDeliveryService creates a new push delivery service
func NewPushDeliveryService(
	VAPIDPublicKey, VAPIDPrivateKey, VAPIDSubject string,
	subsRepo PushSubscriptionRepository,
	cache SmartNotificationCache,
	eventBus EventBus,
) *PushDeliveryService {
	return &PushDeliveryService{
		VAPIDPublicKey:  VAPIDPublicKey,
		VAPIDPrivateKey: VAPIDPrivateKey,
		VAPIDSubject:    VAPIDSubject,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		subsRepo: subsRepo,
		cache:    cache,
		eventBus: eventBus,
	}
}

// RegisterSubscription registers a push subscription for a user
func (s *PushDeliveryService) RegisterSubscription(ctx context.Context, userID uuid.UUID, req *models.CreatePushSubscriptionRequest) error {
	sub := &models.PushSubscription{
		ID:        uuid.New(),
		UserID:    userID,
		Endpoint:  req.Endpoint,
		P256dh:    req.P256dh,
		Auth:      req.Auth,
		UserAgent: req.UserAgent,
		CreatedAt: time.Now(),
	}

	// Check if subscription already exists
	existing, err := s.subsRepo.GetActiveByUserID(ctx, userID)
	if err == nil {
		for _, existingSub := range existing {
			if existingSub.Endpoint == req.Endpoint {
				return nil // already registered
			}
		}
	}

	return s.subsRepo.Create(ctx, sub)
}

// UnregisterSubscription removes a push subscription
func (s *PushDeliveryService) UnregisterSubscription(ctx context.Context, userID uuid.UUID, endpoint string) error {
	return s.subsRepo.DeleteByEndpoint(ctx, userID, endpoint)
}

// GetPreferences returns notification preferences for a user
func (s *PushDeliveryService) GetPreferences(ctx context.Context, userID uuid.UUID) (*models.NotificationPreferences, error) {
	key := fmt.Sprintf("notif_prefs:%s", userID.String())
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return models.DefaultNotificationPreferences(userID), nil
	}

	var prefs models.NotificationPreferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return models.DefaultNotificationPreferences(userID), nil
	}

	return &prefs, nil
}

// UpdatePreferences updates notification preferences for a user
func (s *PushDeliveryService) UpdatePreferences(ctx context.Context, userID uuid.UUID, req *models.UpdateNotificationPreferencesRequest) (*models.NotificationPreferences, error) {
	prefs, err := s.GetPreferences(ctx, userID)
	if err != nil {
		prefs = models.DefaultNotificationPreferences(userID)
	}

	if req.PushEnabled != nil {
		prefs.PushEnabled = *req.PushEnabled
	}
	if req.PushMentions != nil {
		prefs.PushMentions = *req.PushMentions
	}
	if req.PushDirectMessages != nil {
		prefs.PushDirectMessages = *req.PushDirectMessages
	}
	if req.PushReplies != nil {
		prefs.PushReplies = *req.PushReplies
	}
	if req.PushFriendRequests != nil {
		prefs.PushFriendRequests = *req.PushFriendRequests
	}
	if req.PushServerInvites != nil {
		prefs.PushServerInvites = *req.PushServerInvites
	}
	if req.SoundEnabled != nil {
		prefs.SoundEnabled = *req.SoundEnabled
	}
	if req.SoundMessage != nil {
		prefs.SoundMessage = *req.SoundMessage
	}
	if req.SoundMention != nil {
		prefs.SoundMention = *req.SoundMention
	}
	if req.DesktopEnabled != nil {
		prefs.DesktopEnabled = *req.DesktopEnabled
	}
	if req.DesktopPreviews != nil {
		prefs.DesktopPreviews = *req.DesktopPreviews
	}
	if req.DoNotDisturb != nil {
		prefs.DoNotDisturb = *req.DoNotDisturb
	}
	if req.DoNotDisturbUntil != nil {
		prefs.DoNotDisturbUntil = req.DoNotDisturbUntil
	}
	prefs.UpdatedAt = time.Now()

	key := fmt.Sprintf("notif_prefs:%s", userID.String())
	data, _ := json.Marshal(prefs)
	_ = s.cache.Set(ctx, key, data, 0)

	return prefs, nil
}

// SendPushNotification sends a push notification to all of a user's active subscriptions
func (s *PushDeliveryService) SendPushNotification(ctx context.Context, userID uuid.UUID, payload *models.PushPayload, prefs *models.NotificationPreferences) error {
	if !prefs.PushEnabled {
		return nil
	}

	// Check DND
	if prefs.DoNotDisturb {
		if prefs.DoNotDisturbUntil != nil && time.Now().After(*prefs.DoNotDisturbUntil) {
			// DND expired
		} else if prefs.DoNotDisturbUntil == nil {
			// Indefinite DND
			return nil
		}
	}

	subs, err := s.subsRepo.GetActiveByUserID(ctx, userID)
	if err != nil || len(subs) == 0 {
		return fmt.Errorf("no active push subscriptions")
	}

	var lastErr error
	for _, sub := range subs {
		if err := s.sendToSubscription(ctx, sub, payload); err != nil {
			lastErr = err
			if strings.Contains(err.Error(), "410") || strings.Contains(err.Error(), "400") {
				_ = s.subsRepo.Delete(ctx, sub.ID)
			}
		}
	}

	return lastErr
}

// sendToSubscription sends a push notification to a single subscription
func (s *PushDeliveryService) sendToSubscription(ctx context.Context, sub *models.PushSubscription, payload *models.PushPayload) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	encrypted, err := s.encryptPayload(sub, payloadBytes)
	if err != nil {
		return fmt.Errorf("failed to encrypt payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.Endpoint, bytes.NewReader(encrypted))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("TTL", "0")
	req.Header.Set("Urgency", "normal")
	req.Header.Set("Authorization", fmt.Sprintf("WebPush %s", s.VAPIDPublicKey))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send push: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusGone {
		return fmt.Errorf("subscription expired (410)")
	}
	if resp.StatusCode == http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bad request: %s (400)", string(body))
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("push service returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// encryptPayload encrypts a payload for Web Push using ECIES-AES-GCM
func (s *PushDeliveryService) encryptPayload(sub *models.PushSubscription, plaintext []byte) ([]byte, error) {
	// Decode subscription keys
	p256dh, err := base64.URLEncoding.DecodeString(sub.P256dh)
	if err != nil {
		return nil, fmt.Errorf("failed to decode p256dh: %w", err)
	}

	auth, err := base64.URLEncoding.DecodeString(sub.Auth)
	if err != nil {
		return nil, fmt.Errorf("failed to decode auth: %w", err)
	}

	// Generate ephemeral ECDH key pair (P-256)
	ephemeralPrivKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ephemeral key: %w", err)
	}
	ephemeralX, ephemeralY := ephemeralPrivKey.PublicKey.X, ephemeralPrivKey.PublicKey.Y

	// Parse peer public key (uncompressed format 0x04 || X || Y)
	peerX := new(big.Int).SetBytes(p256dh[1:33])
	peerY := new(big.Int).SetBytes(p256dh[33:65])

	// Derive shared point
	sharedX, _ := elliptic.P256().ScalarMult(peerX, peerY, ephemeralX.Bytes())

	// Shared secret is SHA-256 of the X coordinate
	sharedSecret := sha256.Sum256(sharedX.Bytes())

	// HKDF-like key derivation with auth as salt
	// Content Encryption Key (CEK) = HMAC-SHA256(auth, sharedSecret)[0:16]
	authHMAC := hmacSHA256(auth, sharedSecret[:])
	cek := authHMAC[:16]

	// Generate random 12-byte nonce
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// AES-128-GCM encryption
	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Build Web Push message format:
	// [0x04 || ephemeralX (32 bytes) || ephemeralY (32 bytes)] (65 bytes)
	// [nonce (12 bytes)]
	// [ciphertext]
	result := make([]byte, 0, 65+12+len(ciphertext))
	result = append(result, 0x04)
	result = append(result, elliptic.Marshal(elliptic.P256(), ephemeralX, ephemeralY)...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

// hmacSHA256 computes HMAC-SHA256
func hmacSHA256(key, msg []byte) [32]byte {
	// Use crypto/hkdf-like approach with crypto/hmac
	// Simplified implementation
	var kpad [64]byte
	var opad [64]byte

	copy(kpad[:], key)
	if len(key) > 64 {
		h := sha256.Sum256(key)
		copy(kpad[:], h[:])
	}

	for i := range kpad {
		opad[i] = kpad[i] ^ 0x5c
		kpad[i] ^= 0x36
	}

	h1 := sha256.Sum256(append(kpad[:], msg...))
	return sha256.Sum256(append(opad[:], h1[:]...))
}
