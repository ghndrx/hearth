package services

import (
	"testing"
)

// mockUserRepository implements userCredentialRepository for testing.
type mockUserRepository struct {
	findUserByIDFunc    func(id string) (*User, error)
	checkEmailExistsFunc func(email string) (bool, error)
	saveUserFunc        func(user *User) error
}

func (m *mockUserRepository) FindUserByID(id string) (*User, error) {
	if m.findUserByIDFunc != nil {
		return m.findUserByIDFunc(id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockUserRepository) CheckEmailExists(email string) (bool, error) {
	if m.checkEmailExistsFunc != nil {
		return m.checkEmailExistsFunc(email)
	}
	return false, nil
}

func (m *mockUserRepository) SaveUser(user *User) error {
	if m.saveUserFunc != nil {
		return m.saveUserFunc(user)
	}
	return nil
}

// TestGetUserByID_Success tests successful retrieval
func TestGetUserByID_Success(t *testing.T) {
	repo := &mockUserRepository{}
	expectedUser := &User{ID: "U_1", Username: "Alice", Email: "alice@example.com"}
	
	repo.findUserByIDFunc = func(id string) (*User, error) {
		if id != "U_1" {
			t.Fatalf("Expected ID U_1, got %s", id)
		}
		return expectedUser, nil
	}

	service := NewUserService(repo)
	user, err := service.GetUserByID("U_1")

	if err != nil {
		t.Fatalf("GetUserByID returned error: %v", err)
	}

	if user.ID != expectedUser.ID || user.Username != expectedUser.Username {
		t.Errorf("Returned user mismatch. Got %+v, want %+v", user, expectedUser)
	}
}

// TestGetUserByID_InvalidID tests error case for empty ID
func TestGetUserByID_InvalidID(t *testing.T) {
	repo := &mockUserRepository{}
	service := NewUserService(repo)

	_, err := service.GetUserByID("")
	if err == nil {
		t.Error("Expected error for empty ID, got nil")
	}
}

// TestCreateUser_Success tests happy path creation
func TestCreateUser_Success(t *testing.T) {
	mockedRepo := &mockUserRepository{}
	
	// Setup mock expectations
	emailExists := false
	mockedRepo.checkEmailExistsFunc = func(email string) (bool, error) {
		return emailExists, nil
	}
	mockedRepo.saveUserFunc = func(user *User) error {
		if user.ID == "" || user.Username == "" {
			t.Error("SaveUser called with invalid user data")
		}
		return nil
	}

	service := NewUserService(mockedRepo)
	
	username := "NewUser"
	email := "new@example.com"

	user, id, err := service.CreateUser(username, email)

	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	expectedID := "U_" + username

	// Assertions
	if id != expectedID {
		t.Errorf("Expected ID %s, got %s", expectedID, id)
	}
	if user.ID != id {
		t.Errorf("User.ID %s does not match ID %s", user.ID, id)
	}
	if user.Username != username {
		t.Errorf("Username mismatch")
	}
}

// TestCreateUser_EmailExists tests the validation logic for duplicate emails
func TestCreateUser_EmailExists(t *testing.T) {
	mockedRepo := &mockUserRepository{}

	// Setup mock to return true for email exists
	mockedRepo.checkEmailExistsFunc = func(email string) (bool, error) {
		return true, nil
	}

	service := NewUserService(mockedRepo)

	_, _, err := service.CreateUser("Bob", "bob@example.com")

	if err == nil {
		t.Error("Expected error for existing email, got nil")
	}

	if err.Error() != "email already registered" {
		t.Errorf("Expected error message 'email already registered', got: %v", err)
	}
}

// TestCreateUser_MissingFields tests validation logic for missing inputs
func TestCreateUser_MissingFields(t *testing.T) {
	repo := &mockUserRepository{}
	service := NewUserService(repo)

	// Test empty username
	_, _, err := service.CreateUser("", "test@test.com")
	if err == nil {
		t.Error("Expected error for empty username, got nil")
	}

	// Test empty email
	_, _, err = service.CreateUser("TestUser", "")
	if err == nil {
		t.Error("Expected error for empty email, got nil")
	}
}