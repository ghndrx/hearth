package handlers

import (
	"encoding/base64"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// E2EEHandler handles E2EE key management endpoints
type E2EEHandler struct {
	service *services.E2EEServiceImpl
}

// NewE2EEHandler creates a new E2EE handler
func NewE2EEHandler(service *services.E2EEServiceImpl) *E2EEHandler {
	return &E2EEHandler{service: service}
}

// KeyUploadRequest is the API request format for uploading device keys
type KeyUploadRequest struct {
	DeviceID       string              `json:"device_id"`
	DeviceName     string              `json:"device_name,omitempty"`
	DeviceType     string              `json:"device_type,omitempty"`
	IdentityKey    string              `json:"identity_key"` // Base64 encoded
	RegistrationID int                 `json:"registration_id"`
	SignedPreKey   SignedPreKeyRequest `json:"signed_pre_key"`
	OneTimePreKeys []PreKeyRequest     `json:"one_time_pre_keys,omitempty"`
}

// SignedPreKeyRequest is the API format for signed prekey
type SignedPreKeyRequest struct {
	KeyID     int    `json:"key_id"`
	PublicKey string `json:"public_key"` // Base64 encoded
	Signature string `json:"signature"`  // Base64 encoded
}

// PreKeyRequest is the API format for one-time prekey
type PreKeyRequest struct {
	KeyID     int    `json:"key_id"`
	PublicKey string `json:"public_key"` // Base64 encoded
}

// PreKeyBundleResponse is the API response format for prekey bundle
type PreKeyBundleResponse struct {
	UserID         string  `json:"user_id"`
	DeviceID       string  `json:"device_id"`
	RegistrationID int     `json:"registration_id"`
	IdentityKey    string  `json:"identity_key"` // Base64 encoded
	SignedPreKeyID int     `json:"signed_pre_key_id"`
	SignedPreKey   string  `json:"signed_pre_key"`       // Base64 encoded
	SignedKeySign  string  `json:"signed_key_signature"` // Base64 encoded
	PreKeyID       *int    `json:"pre_key_id,omitempty"`
	PreKey         *string `json:"pre_key,omitempty"` // Base64 encoded
}

// UploadKeys handles POST /api/v1/keys/upload
// @Summary Upload device keys
// @Description Uploads identity key, signed prekey, and one-time prekeys for the current device
// @Tags E2EE
// @Accept json
// @Produce json
// @Param body body KeyUploadRequest true "Key upload request"
// @Success 201 {object} fiber.Map "Device registered successfully"
// @Failure 400 {object} fiber.Map "Invalid request or validation error"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /keys/upload [post]
func (h *E2EEHandler) UploadKeys(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req KeyUploadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
	}

	// Validate required fields
	if req.DeviceID == "" || len(req.DeviceID) > 64 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_device_id",
			"message": "Device ID is required and must be <= 64 characters",
		})
	}
	if req.IdentityKey == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_identity_key",
			"message": "Identity key is required",
		})
	}
	if req.RegistrationID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_registration_id",
			"message": "Valid registration ID is required",
		})
	}

	// Decode base64 keys
	identityKey, err := base64.StdEncoding.DecodeString(req.IdentityKey)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_identity_key",
			"message": "Identity key must be valid base64",
		})
	}

	signedPKKey, err := base64.StdEncoding.DecodeString(req.SignedPreKey.PublicKey)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_signed_prekey",
			"message": "Signed prekey public key must be valid base64",
		})
	}

	signedPKSig, err := base64.StdEncoding.DecodeString(req.SignedPreKey.Signature)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_signed_prekey_signature",
			"message": "Signed prekey signature must be valid base64",
		})
	}

	// Convert to model
	modelReq := &models.KeyUploadRequest{
		DeviceID:       req.DeviceID,
		DeviceName:     req.DeviceName,
		DeviceType:     req.DeviceType,
		IdentityKey:    identityKey,
		RegistrationID: req.RegistrationID,
		SignedPreKey: models.SignedPreKeyData{
			KeyID:     req.SignedPreKey.KeyID,
			PublicKey: signedPKKey,
			Signature: signedPKSig,
		},
	}

	// Decode one-time prekeys
	for _, pk := range req.OneTimePreKeys {
		pubKey, err := base64.StdEncoding.DecodeString(pk.PublicKey)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_prekey",
				"message": "Prekey public key must be valid base64",
			})
		}
		modelReq.OneTimePreKeys = append(modelReq.OneTimePreKeys, models.PreKeyData{
			KeyID:     pk.KeyID,
			PublicKey: pubKey,
		})
	}

	device, err := h.service.RegisterDevice(c.Context(), userID, modelReq)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidIdentityKey):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_identity_key",
				"message": "Identity key format is invalid",
			})
		case errors.Is(err, services.ErrInvalidSignedPreKey):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_signed_prekey",
				"message": "Signed prekey format is invalid",
			})
		case errors.Is(err, services.ErrInvalidPreKey):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_prekey",
				"message": err.Error(),
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "internal_error",
				"message": "Failed to register device keys",
			})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"device_id":        device.DeviceID,
		"device_type":      device.DeviceType,
		"registered_at":    device.CreatedAt,
		"prekeys_uploaded": len(req.OneTimePreKeys),
	})
}

// GetUserDevices handles GET /api/v1/keys/:userId/devices
// @Summary Get user devices
// @Description Returns all devices for a user that have E2EE keys
// @Tags E2EE
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} fiber.Map "List of devices"
// @Failure 400 {object} fiber.Map "Invalid user ID format"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /keys/{userId}/devices [get]
func (h *E2EEHandler) GetUserDevices(c *fiber.Ctx) error {
	userIDStr := c.Params("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_user_id",
			"message": "Invalid user ID format",
		})
	}

	devices, err := h.service.GetUserDevices(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to get devices",
		})
	}

	return c.JSON(fiber.Map{
		"user_id": userIDStr,
		"devices": devices,
	})
}

// GetPreKeyBundle handles GET /api/v1/keys/:userId/devices/:deviceId/bundle
// @Summary Get prekey bundle
// @Description Returns the prekey bundle needed to establish an E2EE session
// @Tags E2EE
// @Produce json
// @Param userId path string true "User ID"
// @Param deviceId path string true "Device ID"
// @Success 200 {object} PreKeyBundleResponse "Prekey bundle"
// @Failure 400 {object} fiber.Map "Invalid user ID or device ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Device not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /keys/{userId}/devices/{deviceId}/bundle [get]
func (h *E2EEHandler) GetPreKeyBundle(c *fiber.Ctx) error {
	requestingUserID := c.Locals("userID").(uuid.UUID)

	targetUserIDStr := c.Params("userId")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_user_id",
			"message": "Invalid user ID format",
		})
	}

	deviceID := c.Params("deviceId")
	if deviceID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_device_id",
			"message": "Device ID is required",
		})
	}

	bundle, err := h.service.GetPreKeyBundle(c.Context(), targetUserID, deviceID, requestingUserID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrDeviceNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "device_not_found",
				"message": "Device not found or has no keys",
			})
		case errors.Is(err, services.ErrSelfKeyRequest):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "self_key_request",
				"message": "Cannot request your own prekey bundle",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "internal_error",
				"message": "Failed to get prekey bundle",
			})
		}
	}

	// Convert to API response with base64 encoding
	response := PreKeyBundleResponse{
		UserID:         bundle.UserID.String(),
		DeviceID:       bundle.DeviceID,
		RegistrationID: bundle.RegistrationID,
		IdentityKey:    base64.StdEncoding.EncodeToString(bundle.IdentityKey),
		SignedPreKeyID: bundle.SignedPreKeyID,
		SignedPreKey:   base64.StdEncoding.EncodeToString(bundle.SignedPreKey),
		SignedKeySign:  base64.StdEncoding.EncodeToString(bundle.SignedKeySign),
	}

	if bundle.PreKeyID != nil && len(bundle.PreKey) > 0 {
		response.PreKeyID = bundle.PreKeyID
		preKeyB64 := base64.StdEncoding.EncodeToString(bundle.PreKey)
		response.PreKey = &preKeyB64
	}

	return c.JSON(response)
}

// GetAllPreKeyBundles handles GET /api/v1/keys/:userId/bundles
// @Summary Get all prekey bundles
// @Description Returns prekey bundles for all of a user's devices
// @Tags E2EE
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} fiber.Map "List of prekey bundles"
// @Failure 400 {object} fiber.Map "Invalid user ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /keys/{userId}/bundles [get]
func (h *E2EEHandler) GetAllPreKeyBundles(c *fiber.Ctx) error {
	requestingUserID := c.Locals("userID").(uuid.UUID)

	targetUserIDStr := c.Params("userId")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_user_id",
			"message": "Invalid user ID format",
		})
	}

	bundles, err := h.service.GetPreKeyBundlesForUser(c.Context(), targetUserID, requestingUserID)
	if err != nil {
		if errors.Is(err, services.ErrSelfKeyRequest) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "self_key_request",
				"message": "Cannot request your own prekey bundles",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to get prekey bundles",
		})
	}

	// Convert to API response
	var responses []PreKeyBundleResponse
	for _, bundle := range bundles {
		resp := PreKeyBundleResponse{
			UserID:         bundle.UserID.String(),
			DeviceID:       bundle.DeviceID,
			RegistrationID: bundle.RegistrationID,
			IdentityKey:    base64.StdEncoding.EncodeToString(bundle.IdentityKey),
			SignedPreKeyID: bundle.SignedPreKeyID,
			SignedPreKey:   base64.StdEncoding.EncodeToString(bundle.SignedPreKey),
			SignedKeySign:  base64.StdEncoding.EncodeToString(bundle.SignedKeySign),
		}
		if bundle.PreKeyID != nil && len(bundle.PreKey) > 0 {
			resp.PreKeyID = bundle.PreKeyID
			preKeyB64 := base64.StdEncoding.EncodeToString(bundle.PreKey)
			resp.PreKey = &preKeyB64
		}
		responses = append(responses, resp)
	}

	return c.JSON(fiber.Map{
		"user_id": targetUserIDStr,
		"bundles": responses,
	})
}

// GetMyDevices handles GET /api/v1/keys/devices
// Returns the current user's devices
func (h *E2EEHandler) GetMyDevices(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	devices, err := h.service.GetUserDevices(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to get devices",
		})
	}

	return c.JSON(fiber.Map{
		"devices": devices,
	})
}

// GetPreKeyCount handles GET /api/v1/keys/devices/:deviceId/count
// Returns the count of available prekeys for the current user's device
func (h *E2EEHandler) GetPreKeyCount(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	deviceID := c.Params("deviceId")

	count, err := h.service.GetPreKeyCount(c.Context(), userID, deviceID)
	if err != nil {
		if errors.Is(err, services.ErrDeviceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "device_not_found",
				"message": "Device not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to get prekey count",
		})
	}

	return c.JSON(count)
}

// UploadPreKeys handles POST /api/v1/keys/devices/:deviceId/prekeys
// Uploads additional one-time prekeys
func (h *E2EEHandler) UploadPreKeys(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	deviceID := c.Params("deviceId")

	var req struct {
		PreKeys []PreKeyRequest `json:"pre_keys"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
	}

	if len(req.PreKeys) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_prekeys",
			"message": "At least one prekey is required",
		})
	}

	if len(req.PreKeys) > models.MaxOneTimePreKeysPerUpload {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "too_many_prekeys",
			"message": "Maximum 100 prekeys per upload",
		})
	}

	var prekeys []models.PreKeyData
	for _, pk := range req.PreKeys {
		pubKey, err := base64.StdEncoding.DecodeString(pk.PublicKey)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_prekey",
				"message": "Prekey public key must be valid base64",
			})
		}
		prekeys = append(prekeys, models.PreKeyData{
			KeyID:     pk.KeyID,
			PublicKey: pubKey,
		})
	}

	err := h.service.UploadPreKeys(c.Context(), userID, deviceID, prekeys)
	if err != nil {
		if errors.Is(err, services.ErrDeviceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "device_not_found",
				"message": "Device not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to upload prekeys",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"uploaded": len(prekeys),
	})
}

// DeleteDevice handles DELETE /api/v1/keys/devices/:deviceId
// Removes a device and all its keys
func (h *E2EEHandler) DeleteDevice(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	deviceID := c.Params("deviceId")

	err := h.service.DeleteDevice(c.Context(), userID, deviceID)
	if err != nil {
		if errors.Is(err, services.ErrDeviceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "device_not_found",
				"message": "Device not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to delete device",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetCapabilities handles GET /api/v1/keys/:userId/capabilities
// Returns E2EE capabilities for a user
func (h *E2EEHandler) GetCapabilities(c *fiber.Ctx) error {
	userIDStr := c.Params("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_user_id",
			"message": "Invalid user ID format",
		})
	}

	caps, err := h.service.GetE2EECapabilities(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to get capabilities",
		})
	}

	return c.JSON(caps)
}

// ClaimKeysRequest is the API request format for claiming prekeys
type ClaimKeysRequest struct {
	OneTimeKeys map[string]map[string]string `json:"one_time_keys"` // userId -> deviceId -> algorithm
}

// ClaimKeysResponse is the API response format for claimed keys
type ClaimKeysResponse struct {
	OneTimeKeys map[string]map[string]PreKeyBundleResponse `json:"one_time_keys"`      // userId -> deviceId -> bundle
	Failures    map[string][]string                        `json:"failures,omitempty"` // userId -> []deviceId
}

// ClaimKeys handles POST /api/v1/keys/claim
// @Summary Claim one-time prekeys
// @Description Claims one-time prekeys for multiple users/devices at once. This is used when initiating E2EE sessions with multiple recipients.
// @Tags E2EE
// @Accept json
// @Produce json
// @Param body body ClaimKeysRequest true "Keys to claim"
// @Success 200 {object} ClaimKeysResponse "Claimed keys"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /keys/claim [post]
func (h *E2EEHandler) ClaimKeys(c *fiber.Ctx) error {
	requestingUserID := c.Locals("userID").(uuid.UUID)

	var req ClaimKeysRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
	}

	if len(req.OneTimeKeys) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_keys",
			"message": "At least one user/device pair is required",
		})
	}

	response := ClaimKeysResponse{
		OneTimeKeys: make(map[string]map[string]PreKeyBundleResponse),
		Failures:    make(map[string][]string),
	}

	for userIDStr, devices := range req.OneTimeKeys {
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			// Skip invalid user IDs
			continue
		}

		// Don't allow claiming own keys
		if userID == requestingUserID {
			continue
		}

		response.OneTimeKeys[userIDStr] = make(map[string]PreKeyBundleResponse)

		for deviceID := range devices {
			bundle, err := h.service.GetPreKeyBundle(c.Context(), userID, deviceID, requestingUserID)
			if err != nil {
				// Track failures
				response.Failures[userIDStr] = append(response.Failures[userIDStr], deviceID)
				continue
			}

			// Convert to API response with base64 encoding
			bundleResp := PreKeyBundleResponse{
				UserID:         bundle.UserID.String(),
				DeviceID:       bundle.DeviceID,
				RegistrationID: bundle.RegistrationID,
				IdentityKey:    base64.StdEncoding.EncodeToString(bundle.IdentityKey),
				SignedPreKeyID: bundle.SignedPreKeyID,
				SignedPreKey:   base64.StdEncoding.EncodeToString(bundle.SignedPreKey),
				SignedKeySign:  base64.StdEncoding.EncodeToString(bundle.SignedKeySign),
			}

			if bundle.PreKeyID != nil && len(bundle.PreKey) > 0 {
				bundleResp.PreKeyID = bundle.PreKeyID
				preKeyB64 := base64.StdEncoding.EncodeToString(bundle.PreKey)
				bundleResp.PreKey = &preKeyB64
			}

			response.OneTimeKeys[userIDStr][deviceID] = bundleResp
		}
	}

	// Remove empty failures
	for userID, devices := range response.Failures {
		if len(devices) == 0 {
			delete(response.Failures, userID)
		}
	}

	return c.JSON(response)
}
