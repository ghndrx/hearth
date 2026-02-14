package services

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
)

// MockAuthRepository is a mock for authRepository
type MockAuthRepository struct {
	ctrl     *gomock.Controller
	recorder *MockAuthRepositoryMockRecorder
}

type MockAuthRepositoryMockRecorder struct {
	mock *MockAuthRepository
}

func NewMockAuthRepository(ctrl *gomock.Controller) *MockAuthRepository {
	mock := &MockAuthRepository{ctrl: ctrl}
	mock.recorder = &MockAuthRepositoryMockRecorder{mock}
	return mock
}

func (m *MockAuthRepository) EXPECT() *MockAuthRepositoryMockRecorder {
	return m.recorder
}

// CreateUser mocks base method
func (m *MockAuthRepository) CreateUser(ctx context.Context, user *models.User) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateUser", ctx, user)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockAuthRepositoryMockRecorder) CreateUser(ctx, user interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateUser", reflect.TypeOf((*MockAuthRepository)(nil).CreateUser), ctx, user)
}

// GetUserByEmail mocks base method
func (m *MockAuthRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserByEmail", ctx, email)
	ret0, _ := ret[0].(*models.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockAuthRepositoryMockRecorder) GetUserByEmail(ctx, email interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserByEmail", reflect.TypeOf((*MockAuthRepository)(nil).GetUserByEmail), ctx, email)
}

func TestAuthService_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockAuthRepository(ctrl)
	service := NewAuthService(mockRepo)

	ctx := context.Background()
	email := "test@example.com"
	username := "testuser"
	password := "password123"

	t.Run("Success", func(t *testing.T) {
		// Expect check for existing user returns not found
		mockRepo.EXPECT().GetUserByEmail(ctx, email).Return(nil, ErrUserNotFound)
		
		// Expect create user to be called
		mockRepo.EXPECT().CreateUser(ctx, gomock.Any()).Return(nil).Do(func(_ context.Context, user *models.User) {
			assert.Equal(t, email, user.Email)
			assert.Equal(t, username, user.Username)
			assert.NotEmpty(t, user.Password) // Password should be hashed
			assert.NotEqual(t, password, user.Password)
		})

		user, err := service.Register(ctx, email, username, password)
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, email, user.Email)
	})

	t.Run("User Exists", func(t *testing.T) {
		existingUser := &models.User{Email: email}
		mockRepo.EXPECT().GetUserByEmail(ctx, email).Return(existingUser, nil)

		user, err := service.Register(ctx, email, username, password)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.True(t, errors.Is(err, ErrUserExists))
	})
}

func TestAuthService_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockAuthRepository(ctrl)
	service := NewAuthService(mockRepo)

	ctx := context.Background()
	email := "test@example.com"
	password := "password123"

	// Hash the password manually to match what the mock repo should return
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)

	t.Run("Success", func(t *testing.T) {
		user := &models.User{
			ID:       uuid.New(),
			Email:    email,
			Username: "testuser",
			Password: string(hashedPassword),
		}

		mockRepo.EXPECT().GetUserByEmail(ctx, email).Return(user, nil)

		token, returnedUser, err := service.Login(ctx, email, password)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.NotNil(t, returnedUser)
		assert.Equal(t, user.ID, returnedUser.ID)
	})

	t.Run("User Not Found", func(t *testing.T) {
		mockRepo.EXPECT().GetUserByEmail(ctx, email).Return(nil, ErrUserNotFound)

		token, user, err := service.Login(ctx, email, password)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Empty(t, token)
		assert.True(t, errors.Is(err, ErrInvalidCredentials))
	})

	t.Run("Invalid Password", func(t *testing.T) {
		user := &models.User{
			ID:       uuid.New(),
			Email:    email,
			Password: string(hashedPassword),
		}

		mockRepo.EXPECT().GetUserByEmail(ctx, email).Return(user, nil)

		token, returnedUser, err := service.Login(ctx, email, "wrongpassword")
		assert.Error(t, err)
		assert.Nil(t, returnedUser)
		assert.Empty(t, token)
		assert.True(t, errors.Is(err, ErrInvalidCredentials))
	})
}