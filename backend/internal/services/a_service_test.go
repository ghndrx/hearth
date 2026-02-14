package services

import (
	"database/sql"
	"errors"
	"testing"
)

// sqlmockRow simulates the interface needed for sqlmock or SQL DB.
type sqlmockRow struct {
	result interface{}
	err    error
}

func (m *sqlmockRow) Scan(dest ...interface{}) error {
	// This is a simplified mock for illustration.
	// A real test would use github.com/DATA-DOG/go-sqlmock
	if m.err != nil {
		return m.err
	}
	return errors.New("mock: scan not implemented in simplified example")
}

// mockDB provides a mock sql.DB interface.
type mockDB struct {
	shouldError bool
	noRows      bool
}

func (m *mockDB) QueryRow(query string, args ...interface{}) *sqlmockRow {
	if m.noRows {
		return &sqlmockRow{err: sql.ErrNoRows}
	}
	if m.shouldError {
		return &sqlmockRow{err: errors.New("database connection failed")}
	}
	return &sqlmockRow{result: nil}
}

func TestGetUserByID(t *testing.T) {
	tests := []struct {
		name          string
		dbMock        *mockDB
		userID        int
		wantErr       bool
		expectedError error
	}{
		{
			name: "Happy Path: User Found",
			dbMock: &mockDB{
				shouldError: false,
				noRows:      false,
			},
			userID:   1,
			wantErr:  false,
		},
		{
			name: "Sad Path: User Not Found",
			dbMock: &mockDB{
				shouldError: false,
				noRows:      true,
			},
			userID:       999,
			wantErr:      true,
			expectedError: ErrUserNotFound,
		},
		{
			name: "Sad Path: Database Error",
			dbMock: &mockDB{
				shouldError: true,
				noRows:      false,
			},
			userID:       1,
			wantErr:      true,
			expectedError: errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize the service with our injected mock DB
			service := &UserServiceImpl{db: tt.dbMock}

			user, err := service.GetUserByID(tt.userID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != tt.expectedError {
				t.Errorf("GetUserByID() error = %v, want error %v", err, tt.expectedError)
			}

			// Scenario: User Found
			if !tt.wantErr && user.ID != tt.userID {
				t.Errorf("Expected ID %d, got ID %d", tt.userID, user.ID)
			}
			if !tt.wantErr {
				t.Logf("Successfully retrieved user: %+v", user)
			}
		})
	}
}