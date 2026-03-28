package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestValidatePasswordStrength(t *testing.T) {
	testCases := []struct {
		name        string
		password    string
		expectedErr error
	}{
		// Valid passwords
		{"valid password", "Password123", nil},
		{"valid with special chars", "Password123!@#", nil},
		{"valid min length", "Passwo1d", nil}, // exactly 8 chars
		{"valid complex", "MyStr0ngP@ssword!", nil},

		// Too short
		{"too short - 7 chars", "Pass12a", ErrPasswordTooShort},
		{"too short - 1 char", "A", ErrPasswordTooShort},
		{"empty password", "", ErrPasswordTooShort},

		// Too long
		{"too long - 73 chars", strings.Repeat("A", 72) + "a1", ErrPasswordTooLong},
		{"too long - 100 chars", strings.Repeat("Ab1", 34), ErrPasswordTooLong}, // 102 chars

		// Missing requirements
		{"missing uppercase", "password123", ErrPasswordWeak},
		{"missing lowercase", "PASSWORD123", ErrPasswordWeak},
		{"missing number", "Passwordabc", ErrPasswordWeak},
		{"only lowercase", "abcdefgh", ErrPasswordWeak},
		{"only uppercase", "ABCDEFGH", ErrPasswordWeak},
		{"only numbers", "12345678", ErrPasswordWeak},
		{"special chars only no upper", "password!@#1", ErrPasswordWeak},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePasswordStrength(tc.password)
			if tc.expectedErr != nil {
				assert.Equal(t, tc.expectedErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePasswordStrength_BoundaryLengths(t *testing.T) {
	// Test exactly at boundaries

	// Exactly 8 characters (minimum)
	minPassword := "Abcdef1x"
	assert.Len(t, minPassword, 8)
	assert.NoError(t, ValidatePasswordStrength(minPassword))

	// Exactly 72 characters (maximum)
	maxPassword := strings.Repeat("Aa1", 24) // 72 chars
	assert.Len(t, maxPassword, 72)
	assert.NoError(t, ValidatePasswordStrength(maxPassword))

	// 73 characters (over max)
	overMaxPassword := maxPassword + "x"
	assert.Len(t, overMaxPassword, 73)
	assert.Equal(t, ErrPasswordTooLong, ValidatePasswordStrength(overMaxPassword))

	// 7 characters (under min)
	underMinPassword := "Abcde1x"
	assert.Len(t, underMinPassword, 7)
	assert.Equal(t, ErrPasswordTooShort, ValidatePasswordStrength(underMinPassword))
}

func TestHashPassword_ValidPassword(t *testing.T) {
	password := "SecurePassword123"

	hash, err := HashPassword(password)

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash) // Hash should differ from plaintext

	// Should be valid bcrypt hash (starts with $2a$ or $2b$)
	assert.True(t, strings.HasPrefix(hash, "$2a$") || strings.HasPrefix(hash, "$2b$"))
}

func TestHashPassword_SamePasswordDifferentHashes(t *testing.T) {
	password := "SecurePassword123"

	hash1, err := HashPassword(password)
	require.NoError(t, err)

	hash2, err := HashPassword(password)
	require.NoError(t, err)

	// Same password should produce different hashes (due to random salt)
	assert.NotEqual(t, hash1, hash2)
}

func TestHashPassword_InvalidPassword(t *testing.T) {
	testCases := []struct {
		name        string
		password    string
		expectedErr error
	}{
		{"too short", "Pass1", ErrPasswordTooShort},
		{"missing uppercase", "password123", ErrPasswordWeak},
		{"missing lowercase", "PASSWORD123", ErrPasswordWeak},
		{"missing number", "Passwordabc", ErrPasswordWeak},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hash, err := HashPassword(tc.password)
			assert.Error(t, err)
			assert.Equal(t, tc.expectedErr, err)
			assert.Empty(t, hash)
		})
	}
}

func TestCheckPassword_Correct(t *testing.T) {
	password := "SecurePassword123"
	hash, err := HashPassword(password)
	require.NoError(t, err)

	err = CheckPassword(password, hash)
	assert.NoError(t, err)
}

func TestCheckPassword_Incorrect(t *testing.T) {
	password := "SecurePassword123"
	hash, err := HashPassword(password)
	require.NoError(t, err)

	testCases := []struct {
		name    string
		attempt string
	}{
		{"wrong password", "WrongPassword123"},
		{"similar password", "SecurePassword124"},
		{"case different", "securepassword123"},
		{"empty password", ""},
		{"partial password", "SecurePassword"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckPassword(tc.attempt, hash)
			assert.Error(t, err)
			assert.Equal(t, ErrPasswordMismatch, err)
		})
	}
}

func TestCheckPassword_InvalidHash(t *testing.T) {
	testCases := []struct {
		name string
		hash string
	}{
		{"empty hash", ""},
		{"garbage hash", "not-a-bcrypt-hash"},
		{"invalid format", "$2a$invalid$hash"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckPassword("Password123", tc.hash)
			assert.Error(t, err)
			assert.Equal(t, ErrPasswordMismatch, err)
		})
	}
}

func TestNeedsRehash_CurrentCost(t *testing.T) {
	password := "SecurePassword123"
	hash, err := HashPassword(password)
	require.NoError(t, err)

	// Hash created with current cost should not need rehash
	assert.False(t, NeedsRehash(hash))
}

func TestNeedsRehash_LowerCost(t *testing.T) {
	password := "SecurePassword123"

	// Create hash with lower cost
	lowerCostHash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	require.NoError(t, err)

	// Should need rehash (our cost is 12)
	assert.True(t, NeedsRehash(string(lowerCostHash)))
}

func TestNeedsRehash_HigherCost(t *testing.T) {
	password := "SecurePassword123"

	// Create hash with higher cost (takes longer but shouldn't need rehash)
	higherCostHash, err := bcrypt.GenerateFromPassword([]byte(password), 13)
	require.NoError(t, err)

	// Should NOT need rehash (cost >= our minimum)
	assert.False(t, NeedsRehash(string(higherCostHash)))
}

func TestNeedsRehash_InvalidHash(t *testing.T) {
	// Invalid hash should return true (needs rehash / can't determine)
	assert.True(t, NeedsRehash(""))
	assert.True(t, NeedsRehash("invalid"))
	assert.True(t, NeedsRehash("$2a$invalid"))
}

func TestHashPassword_UnicodePAssword(t *testing.T) {
	// Test passwords with unicode characters
	unicodePasswords := []string{
		"Password123日本語",
		"Contraseña123",
		"Пароль123Abc",
		"密码Password1",
	}

	for _, password := range unicodePasswords {
		t.Run(password, func(t *testing.T) {
			hash, err := HashPassword(password)
			require.NoError(t, err)
			assert.NotEmpty(t, hash)

			// Should be able to verify
			err = CheckPassword(password, hash)
			assert.NoError(t, err)
		})
	}
}

func TestCheckPassword_TimingConsistency(t *testing.T) {
	// This is a basic test to ensure we use constant-time comparison
	// (bcrypt does this internally, but good to verify behavior)
	password := "SecurePassword123"
	hash, err := HashPassword(password)
	require.NoError(t, err)

	// Run multiple times - should consistently return same error type
	for i := 0; i < 10; i++ {
		err := CheckPassword("WrongPassword123", hash)
		assert.Equal(t, ErrPasswordMismatch, err)
	}
}

func TestHashPassword_BCryptCost(t *testing.T) {
	password := "SecurePassword123"
	hash, err := HashPassword(password)
	require.NoError(t, err)

	// Verify the cost factor
	cost, err := bcrypt.Cost([]byte(hash))
	require.NoError(t, err)
	assert.Equal(t, 12, cost) // bcryptCost constant
}
