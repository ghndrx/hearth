package auth

import (
	"crypto/rand"
	"fmt"
	"image/png"
	"strings"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// TOTPConfig holds configuration for TOTP generation
type TOTPConfig struct {
	Issuer      string
	AccountName string
	SecretSize  uint
}

// GenerateTOTPSecret generates a new TOTP secret
func GenerateTOTPSecret(config TOTPConfig) (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      config.Issuer,
		AccountName: config.AccountName,
		SecretSize:  config.SecretSize,
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate TOTP key: %w", err)
	}

	return key.Secret(), nil
}

// GenerateQRCode generates a QR code URL for the TOTP secret
func GenerateQRCode(secret, issuer, accountName string) (string, error) {
	key, err := otp.NewKeyFromURL(fmt.Sprintf(
		"otpauth://totp/%s:%s?secret=%s&issuer=%s",
		issuer, accountName, secret, issuer,
	))
	if err != nil {
		return "", fmt.Errorf("failed to create TOTP key: %w", err)
	}

	return key.String(), nil
}

// GenerateQRCodeImage generates a QR code image as PNG bytes
func GenerateQRCodeImage(secret, issuer, accountName string) ([]byte, error) {
	key, err := otp.NewKeyFromURL(fmt.Sprintf(
		"otpauth://totp/%s:%s?secret=%s&issuer=%s",
		issuer, accountName, secret, issuer,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to create TOTP key: %w", err)
	}

	img, err := key.Image(200, 200)
	if err != nil {
		return nil, fmt.Errorf("failed to generate QR code image: %w", err)
	}

	var buf []byte
	writer := &bytesWriter{buf: &buf}
	if err := png.Encode(writer, img); err != nil {
		return nil, fmt.Errorf("failed to encode QR code image: %w", err)
	}

	return buf, nil
}

// bytesWriter implements io.Writer for []byte
type bytesWriter struct {
	buf *[]byte
}

func (w *bytesWriter) Write(p []byte) (n int, err error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}

// VerifyTOTP verifies a TOTP code against a secret
func VerifyTOTP(code, secret string) bool {
	return totp.Validate(code, secret)
}

// GenerateBackupCodes generates backup codes for 2FA recovery
func GenerateBackupCodes(count int) ([]string, error) {
	codes := make([]string, count)

	for i := 0; i < count; i++ {
		code, err := generateRandomCode(8)
		if err != nil {
			return nil, fmt.Errorf("failed to generate backup code: %w", err)
		}
		codes[i] = code
	}

	return codes, nil
}

// generateRandomCode generates a random alphanumeric code
func generateRandomCode(length int) (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}

	return string(bytes), nil
}

// FormatBackupCode formats a backup code with dashes for readability
func FormatBackupCode(code string) string {
	if len(code) != 8 {
		return code
	}
	return fmt.Sprintf("%s-%s", code[:4], code[4:])
}

// CleanBackupCode removes formatting from a backup code
func CleanBackupCode(code string) string {
	return strings.ReplaceAll(strings.ToUpper(code), "-", "")
}

// HashBackupCodes hashes backup codes for secure storage
func HashBackupCodes(codes []string) ([]string, error) {
	hashedCodes := make([]string, len(codes))

	for i, code := range codes {
		// Use the same password hashing function for consistency
		hashedCode, err := HashPassword(CleanBackupCode(code))
		if err != nil {
			return nil, fmt.Errorf("failed to hash backup code: %w", err)
		}
		hashedCodes[i] = hashedCode
	}

	return hashedCodes, nil
}

// VerifyBackupCode verifies a backup code against its hash
func VerifyBackupCode(code, hashedCode string) error {
	return CheckPassword(CleanBackupCode(code), hashedCode)
}
