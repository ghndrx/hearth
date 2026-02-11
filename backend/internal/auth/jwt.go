package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

// Claims represents JWT claims
type Claims struct {
	jwt.RegisteredClaims
	UserID   uuid.UUID `json:"uid"`
	Username string    `json:"usr"`
	Type     string    `json:"typ"` // "access" or "refresh"
}

// JWTService handles JWT operations
type JWTService struct {
	secretKey     []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
	issuer        string
}

// NewJWTService creates a new JWT service
func NewJWTService(secretKey string, accessExpiry, refreshExpiry time.Duration) *JWTService {
	return &JWTService{
		secretKey:     []byte(secretKey),
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
		issuer:        "hearth",
	}
}

// GenerateAccessToken creates an access token
func (s *JWTService) GenerateAccessToken(userID uuid.UUID, username string) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessExpiry)),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
		UserID:   userID,
		Username: username,
		Type:     "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// GenerateRefreshToken creates a refresh token
func (s *JWTService) GenerateRefreshToken(userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshExpiry)),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
		UserID: userID,
		Type:   "refresh",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// GenerateTokenPair creates both access and refresh tokens
func (s *JWTService) GenerateTokenPair(userID uuid.UUID, username string) (accessToken, refreshToken string, err error) {
	accessToken, err = s.GenerateAccessToken(userID, username)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = s.GenerateRefreshToken(userID)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// ValidateToken validates a token and returns claims
func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidateAccessToken validates an access token
func (s *JWTService) ValidateAccessToken(tokenString string) (*Claims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.Type != "access" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token
func (s *JWTService) ValidateRefreshToken(tokenString string) (*Claims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.Type != "refresh" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GetExpirySeconds returns access token expiry in seconds
func (s *JWTService) GetExpirySeconds() int {
	return int(s.accessExpiry.Seconds())
}
