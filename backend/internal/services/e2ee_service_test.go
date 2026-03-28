package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
)

// --- Mock E2EERepository ---

type mockE2EERepository struct {
	mock.Mock
}

func (m *mockE2EERepository) RegisterDevice(ctx context.Context, userID uuid.UUID, req *models.KeyUploadRequest) (*models.DeviceKey, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DeviceKey), args.Error(1)
}

func (m *mockE2EERepository) GetUserDevices(ctx context.Context, userID uuid.UUID) ([]*models.E2EEDeviceInfo, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.E2EEDeviceInfo), args.Error(1)
}

func (m *mockE2EERepository) DeleteDevice(ctx context.Context, userID uuid.UUID, deviceID string) error {
	args := m.Called(ctx, userID, deviceID)
	return args.Error(0)
}

func (m *mockE2EERepository) UpdateLastSeen(ctx context.Context, userID uuid.UUID, deviceID string) error {
	args := m.Called(ctx, userID, deviceID)
	return args.Error(0)
}

func (m *mockE2EERepository) GetPreKeyBundle(ctx context.Context, targetUserID uuid.UUID, targetDeviceID string, requestingUserID uuid.UUID) (*models.E2EEPreKeyBundle, error) {
	args := m.Called(ctx, targetUserID, targetDeviceID, requestingUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.E2EEPreKeyBundle), args.Error(1)
}

func (m *mockE2EERepository) GetPreKeyCount(ctx context.Context, userID uuid.UUID, deviceID string) (*models.PreKeyCount, error) {
	args := m.Called(ctx, userID, deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PreKeyCount), args.Error(1)
}

func (m *mockE2EERepository) UploadPreKeys(ctx context.Context, userID uuid.UUID, deviceID string, prekeys []models.PreKeyData) error {
	args := m.Called(ctx, userID, deviceID, prekeys)
	return args.Error(0)
}

// --- Test Helpers ---

// Create a valid test key upload request
func testKeyUploadRequest() *models.KeyUploadRequest {
	return &models.KeyUploadRequest{
		DeviceID:       "test-device-001",
		DeviceName:     "Test Device",
		DeviceType:     "desktop",
		IdentityKey:    make([]byte, 32), // 32 bytes for Curve25519
		RegistrationID: 12345,
		SignedPreKey: models.SignedPreKeyData{
			KeyID:     1,
			PublicKey: make([]byte, 32),
			Signature: make([]byte, 64), // 64 bytes for signature
		},
		OneTimePreKeys: []models.PreKeyData{
			{KeyID: 1, PublicKey: make([]byte, 32)},
			{KeyID: 2, PublicKey: make([]byte, 32)},
		},
	}
}

// --- NewE2EEServiceImpl ---

func TestNewE2EEServiceImpl(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)

	require.NotNil(t, svc)
	assert.Equal(t, repo, svc.repo)
}

func TestNewE2EEServiceImpl_NilRepo(t *testing.T) {
	svc := NewE2EEServiceImpl(nil)
	require.NotNil(t, svc)
	assert.Nil(t, svc.repo)
}

// --- RegisterDevice ---

func TestE2EEService_RegisterDevice_Success(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	userID := uuid.New()
	req := testKeyUploadRequest()

	expectedDevice := &models.DeviceKey{
		ID:             uuid.New(),
		UserID:         userID,
		DeviceID:       req.DeviceID,
		IdentityKey:    req.IdentityKey,
		RegistrationID: req.RegistrationID,
		DeviceType:     req.DeviceType,
		LastSeen:       time.Now(),
		CreatedAt:      time.Now(),
	}

	repo.On("RegisterDevice", ctx, userID, req).Return(expectedDevice, nil)

	result, err := svc.RegisterDevice(ctx, userID, req)

	require.NoError(t, err)
	assert.Equal(t, expectedDevice, result)
	repo.AssertExpectations(t)
}

func TestE2EEService_RegisterDevice_InvalidIdentityKey(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	tests := []struct {
		name    string
		keySize int
	}{
		{"too_short", 31},
		{"too_long", 66},
		{"empty", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testKeyUploadRequest()
			req.IdentityKey = make([]byte, tt.keySize)

			result, err := svc.RegisterDevice(ctx, uuid.New(), req)

			assert.Nil(t, result)
			assert.ErrorIs(t, err, ErrInvalidIdentityKey)
		})
	}
}

func TestE2EEService_RegisterDevice_InvalidSignedPreKey(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	tests := []struct {
		name          string
		publicKeySize int
		signatureSize int
	}{
		{"public_key_too_short", 31, 64},
		{"public_key_too_long", 66, 64},
		{"signature_too_short", 32, 63},
		{"signature_empty", 32, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testKeyUploadRequest()
			req.SignedPreKey.PublicKey = make([]byte, tt.publicKeySize)
			req.SignedPreKey.Signature = make([]byte, tt.signatureSize)

			result, err := svc.RegisterDevice(ctx, uuid.New(), req)

			assert.Nil(t, result)
			assert.ErrorIs(t, err, ErrInvalidSignedPreKey)
		})
	}
}

func TestE2EEService_RegisterDevice_InvalidOneTimePreKeys(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	req := testKeyUploadRequest()
	req.OneTimePreKeys = []models.PreKeyData{
		{KeyID: 1, PublicKey: make([]byte, 31)}, // Invalid: too short
	}

	result, err := svc.RegisterDevice(ctx, uuid.New(), req)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrInvalidPreKey)
	assert.Contains(t, err.Error(), "index 0")
}

func TestE2EEService_RegisterDevice_DefaultDeviceType(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	userID := uuid.New()
	req := testKeyUploadRequest()
	req.DeviceType = "" // Empty device type

	expectedDevice := &models.DeviceKey{ID: uuid.New()}

	// Capture the modified request
	var capturedReq *models.KeyUploadRequest
	repo.On("RegisterDevice", ctx, userID, mock.AnythingOfType("*models.KeyUploadRequest")).
		Run(func(args mock.Arguments) {
			capturedReq = args.Get(2).(*models.KeyUploadRequest)
		}).
		Return(expectedDevice, nil)

	result, err := svc.RegisterDevice(ctx, userID, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.E2EEDeviceTypeUnknown, capturedReq.DeviceType)
	repo.AssertExpectations(t)
}

// --- GetUserDevices ---

func TestE2EEService_GetUserDevices_Success(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	userID := uuid.New()
	expectedDevices := []*models.E2EEDeviceInfo{
		{
			DeviceID:         "device-1",
			DeviceType:       "desktop",
			HasPreKeys:       true,
			RemainingPreKeys: 50,
			LastSeen:         time.Now(),
			CreatedAt:        time.Now(),
		},
		{
			DeviceID:         "device-2",
			DeviceType:       "mobile_ios",
			HasPreKeys:       true,
			RemainingPreKeys: 30,
			LastSeen:         time.Now(),
			CreatedAt:        time.Now(),
		},
	}

	repo.On("GetUserDevices", ctx, userID).Return(expectedDevices, nil)

	result, err := svc.GetUserDevices(ctx, userID)

	require.NoError(t, err)
	assert.Equal(t, expectedDevices, result)
	repo.AssertExpectations(t)
}

func TestE2EEService_GetUserDevices_NoDevices(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	userID := uuid.New()
	repo.On("GetUserDevices", ctx, userID).Return([]*models.E2EEDeviceInfo{}, nil)

	result, err := svc.GetUserDevices(ctx, userID)

	require.NoError(t, err)
	assert.Empty(t, result)
	repo.AssertExpectations(t)
}

// --- GetPreKeyBundle ---

func TestE2EEService_GetPreKeyBundle_Success(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	targetUserID := uuid.New()
	requestingUserID := uuid.New()
	deviceID := "target-device"

	preKeyID := 5
	expectedBundle := &models.E2EEPreKeyBundle{
		UserID:         targetUserID,
		DeviceID:       deviceID,
		RegistrationID: 12345,
		IdentityKey:    make([]byte, 32),
		SignedPreKeyID: 1,
		SignedPreKey:   make([]byte, 32),
		SignedKeySign:  make([]byte, 64),
		PreKeyID:       &preKeyID,
		PreKey:         make([]byte, 32),
	}

	repo.On("GetPreKeyBundle", ctx, targetUserID, deviceID, requestingUserID).
		Return(expectedBundle, nil)

	result, err := svc.GetPreKeyBundle(ctx, targetUserID, deviceID, requestingUserID)

	require.NoError(t, err)
	assert.Equal(t, expectedBundle, result)
	repo.AssertExpectations(t)
}

func TestE2EEService_GetPreKeyBundle_SelfRequest(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	userID := uuid.New()
	deviceID := "device-1"

	result, err := svc.GetPreKeyBundle(ctx, userID, deviceID, userID)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrSelfKeyRequest)
	repo.AssertNotCalled(t, "GetPreKeyBundle")
}

func TestE2EEService_GetPreKeyBundle_DeviceNotFound(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	targetUserID := uuid.New()
	requestingUserID := uuid.New()
	deviceID := "nonexistent"

	repo.On("GetPreKeyBundle", ctx, targetUserID, deviceID, requestingUserID).
		Return(nil, ErrDeviceNotFound)

	result, err := svc.GetPreKeyBundle(ctx, targetUserID, deviceID, requestingUserID)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrDeviceNotFound)
	repo.AssertExpectations(t)
}

// --- GetPreKeyBundlesForUser ---

func TestE2EEService_GetPreKeyBundlesForUser_Success(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	targetUserID := uuid.New()
	requestingUserID := uuid.New()

	devices := []*models.E2EEDeviceInfo{
		{DeviceID: "device-1", HasPreKeys: true},
		{DeviceID: "device-2", HasPreKeys: true},
	}

	preKeyID := 5
	bundle1 := &models.E2EEPreKeyBundle{DeviceID: "device-1", PreKeyID: &preKeyID}
	bundle2 := &models.E2EEPreKeyBundle{DeviceID: "device-2", PreKeyID: &preKeyID}

	repo.On("GetUserDevices", ctx, targetUserID).Return(devices, nil)
	repo.On("GetPreKeyBundle", ctx, targetUserID, "device-1", requestingUserID).Return(bundle1, nil)
	repo.On("GetPreKeyBundle", ctx, targetUserID, "device-2", requestingUserID).Return(bundle2, nil)

	result, err := svc.GetPreKeyBundlesForUser(ctx, targetUserID, requestingUserID)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Contains(t, result, bundle1)
	assert.Contains(t, result, bundle2)
	repo.AssertExpectations(t)
}

func TestE2EEService_GetPreKeyBundlesForUser_SkipsDevicesWithoutPreKeys(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	targetUserID := uuid.New()
	requestingUserID := uuid.New()

	devices := []*models.E2EEDeviceInfo{
		{DeviceID: "device-1", HasPreKeys: true},
		{DeviceID: "device-2", HasPreKeys: false}, // No prekeys
	}

	preKeyID := 5
	bundle1 := &models.E2EEPreKeyBundle{DeviceID: "device-1", PreKeyID: &preKeyID}

	repo.On("GetUserDevices", ctx, targetUserID).Return(devices, nil)
	repo.On("GetPreKeyBundle", ctx, targetUserID, "device-1", requestingUserID).Return(bundle1, nil)

	result, err := svc.GetPreKeyBundlesForUser(ctx, targetUserID, requestingUserID)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, bundle1, result[0])
	// device-2 should not have been queried
	repo.AssertNotCalled(t, "GetPreKeyBundle", ctx, targetUserID, "device-2", requestingUserID)
	repo.AssertExpectations(t)
}

func TestE2EEService_GetPreKeyBundlesForUser_SelfRequest(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	userID := uuid.New()

	result, err := svc.GetPreKeyBundlesForUser(ctx, userID, userID)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrSelfKeyRequest)
	repo.AssertNotCalled(t, "GetUserDevices")
}

func TestE2EEService_GetPreKeyBundlesForUser_ContinuesOnIndividualErrors(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	targetUserID := uuid.New()
	requestingUserID := uuid.New()

	devices := []*models.E2EEDeviceInfo{
		{DeviceID: "device-1", HasPreKeys: true},
		{DeviceID: "device-2", HasPreKeys: true},
	}

	preKeyID := 5
	bundle2 := &models.E2EEPreKeyBundle{DeviceID: "device-2", PreKeyID: &preKeyID}

	repo.On("GetUserDevices", ctx, targetUserID).Return(devices, nil)
	repo.On("GetPreKeyBundle", ctx, targetUserID, "device-1", requestingUserID).
		Return(nil, errors.New("device error"))
	repo.On("GetPreKeyBundle", ctx, targetUserID, "device-2", requestingUserID).
		Return(bundle2, nil)

	result, err := svc.GetPreKeyBundlesForUser(ctx, targetUserID, requestingUserID)

	require.NoError(t, err)
	assert.Len(t, result, 1) // Only device-2's bundle
	assert.Equal(t, bundle2, result[0])
	repo.AssertExpectations(t)
}

// --- UpdateLastSeen ---

func TestE2EEService_UpdateLastSeen_Success(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	userID := uuid.New()
	deviceID := "device-1"

	repo.On("UpdateLastSeen", ctx, userID, deviceID).Return(nil)

	err := svc.UpdateLastSeen(ctx, userID, deviceID)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestE2EEService_UpdateLastSeen_DeviceNotFound(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	userID := uuid.New()
	deviceID := "nonexistent"

	repo.On("UpdateLastSeen", ctx, userID, deviceID).Return(ErrDeviceNotFound)

	err := svc.UpdateLastSeen(ctx, userID, deviceID)

	assert.ErrorIs(t, err, ErrDeviceNotFound)
	repo.AssertExpectations(t)
}

// --- DeleteDevice ---

func TestE2EEService_DeleteDevice_Success(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	userID := uuid.New()
	deviceID := "device-1"

	repo.On("DeleteDevice", ctx, userID, deviceID).Return(nil)

	err := svc.DeleteDevice(ctx, userID, deviceID)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestE2EEService_DeleteDevice_DeviceNotFound(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	userID := uuid.New()
	deviceID := "nonexistent"

	repo.On("DeleteDevice", ctx, userID, deviceID).Return(ErrDeviceNotFound)

	err := svc.DeleteDevice(ctx, userID, deviceID)

	assert.ErrorIs(t, err, ErrDeviceNotFound)
	repo.AssertExpectations(t)
}

// --- GetPreKeyCount ---

func TestE2EEService_GetPreKeyCount_Success(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	userID := uuid.New()
	deviceID := "device-1"

	expectedCount := &models.PreKeyCount{
		DeviceID:           deviceID,
		SignedPreKeys:      1,
		OneTimePreKeys:     50,
		MinRecommended:     25,
		NeedsReplenishment: false,
	}

	repo.On("GetPreKeyCount", ctx, userID, deviceID).Return(expectedCount, nil)

	result, err := svc.GetPreKeyCount(ctx, userID, deviceID)

	require.NoError(t, err)
	assert.Equal(t, expectedCount, result)
	repo.AssertExpectations(t)
}

func TestE2EEService_GetPreKeyCount_DeviceNotFound(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	userID := uuid.New()
	deviceID := "nonexistent"

	repo.On("GetPreKeyCount", ctx, userID, deviceID).Return(nil, ErrDeviceNotFound)

	result, err := svc.GetPreKeyCount(ctx, userID, deviceID)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrDeviceNotFound)
	repo.AssertExpectations(t)
}

// --- UploadPreKeys ---

func TestE2EEService_UploadPreKeys_Success(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	userID := uuid.New()
	deviceID := "device-1"

	prekeys := []models.PreKeyData{
		{KeyID: 10, PublicKey: make([]byte, 32)},
		{KeyID: 11, PublicKey: make([]byte, 32)},
	}

	repo.On("UploadPreKeys", ctx, userID, deviceID, prekeys).Return(nil)

	err := svc.UploadPreKeys(ctx, userID, deviceID, prekeys)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestE2EEService_UploadPreKeys_InvalidPreKeys(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	userID := uuid.New()
	deviceID := "device-1"

	prekeys := []models.PreKeyData{
		{KeyID: 10, PublicKey: make([]byte, 31)}, // Invalid: too short
	}

	err := svc.UploadPreKeys(ctx, userID, deviceID, prekeys)

	assert.ErrorIs(t, err, ErrInvalidPreKey)
	assert.Contains(t, err.Error(), "index 0")
	repo.AssertNotCalled(t, "UploadPreKeys")
}

func TestE2EEService_UploadPreKeys_DeviceNotFound(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	userID := uuid.New()
	deviceID := "nonexistent"

	prekeys := []models.PreKeyData{
		{KeyID: 10, PublicKey: make([]byte, 32)},
	}

	repo.On("UploadPreKeys", ctx, userID, deviceID, prekeys).Return(ErrDeviceNotFound)

	err := svc.UploadPreKeys(ctx, userID, deviceID, prekeys)

	assert.ErrorIs(t, err, ErrDeviceNotFound)
	repo.AssertExpectations(t)
}

// --- CheckAndNotifyLowPreKeys ---

func TestE2EEService_CheckAndNotifyLowPreKeys_ReplenishmentNeeded(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	userID := uuid.New()
	deviceID := "device-1"

	count := &models.PreKeyCount{
		DeviceID:           deviceID,
		OneTimePreKeys:     5,
		MinRecommended:     25,
		NeedsReplenishment: true,
	}

	repo.On("GetPreKeyCount", ctx, userID, deviceID).Return(count, nil)

	needsReplenishment, err := svc.CheckAndNotifyLowPreKeys(ctx, userID, deviceID)

	require.NoError(t, err)
	assert.True(t, needsReplenishment)
	repo.AssertExpectations(t)
}

func TestE2EEService_CheckAndNotifyLowPreKeys_NoReplenishmentNeeded(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	userID := uuid.New()
	deviceID := "device-1"

	count := &models.PreKeyCount{
		DeviceID:           deviceID,
		OneTimePreKeys:     50,
		MinRecommended:     25,
		NeedsReplenishment: false,
	}

	repo.On("GetPreKeyCount", ctx, userID, deviceID).Return(count, nil)

	needsReplenishment, err := svc.CheckAndNotifyLowPreKeys(ctx, userID, deviceID)

	require.NoError(t, err)
	assert.False(t, needsReplenishment)
	repo.AssertExpectations(t)
}

// --- GetE2EECapabilities ---

func TestE2EEService_GetE2EECapabilities_Supported(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	userID := uuid.New()

	devices := []*models.E2EEDeviceInfo{
		{DeviceID: "device-1", HasPreKeys: true},
		{DeviceID: "device-2", HasPreKeys: false},
	}

	repo.On("GetUserDevices", ctx, userID).Return(devices, nil)

	result, err := svc.GetE2EECapabilities(ctx, userID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.SupportsE2EE)
	assert.Equal(t, 1, result.ProtocolVersion)
	repo.AssertExpectations(t)
}

func TestE2EEService_GetE2EECapabilities_NotSupported(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	userID := uuid.New()

	devices := []*models.E2EEDeviceInfo{
		{DeviceID: "device-1", HasPreKeys: false},
		{DeviceID: "device-2", HasPreKeys: false},
	}

	repo.On("GetUserDevices", ctx, userID).Return(devices, nil)

	result, err := svc.GetE2EECapabilities(ctx, userID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.SupportsE2EE)
	assert.Equal(t, 1, result.ProtocolVersion)
	repo.AssertExpectations(t)
}

func TestE2EEService_GetE2EECapabilities_NoDevices(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	userID := uuid.New()

	repo.On("GetUserDevices", ctx, userID).Return([]*models.E2EEDeviceInfo{}, nil)

	result, err := svc.GetE2EECapabilities(ctx, userID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.SupportsE2EE)
	assert.Equal(t, 1, result.ProtocolVersion)
	repo.AssertExpectations(t)
}

// --- Edge Cases ---

func TestE2EEService_KeyValidation_BoundaryValues(t *testing.T) {
	repo := new(mockE2EERepository)
	svc := NewE2EEServiceImpl(repo)
	ctx := context.Background()

	userID := uuid.New()

	tests := []struct {
		name          string
		identitySize  int
		publicSize    int
		signatureSize int
		shouldFail    bool
	}{
		{"minimum_valid", 32, 32, 64, false},
		{"maximum_valid", 65, 65, 128, false},
		{"identity_too_small", 31, 32, 64, true},
		{"public_too_small", 32, 31, 64, true},
		{"signature_too_small", 32, 32, 63, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testKeyUploadRequest()
			req.IdentityKey = make([]byte, tt.identitySize)
			req.SignedPreKey.PublicKey = make([]byte, tt.publicSize)
			req.SignedPreKey.Signature = make([]byte, tt.signatureSize)

			if !tt.shouldFail {
				repo.On("RegisterDevice", ctx, userID, req).Return(&models.DeviceKey{}, nil).Once()
			}

			_, err := svc.RegisterDevice(ctx, userID, req)

			if tt.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
