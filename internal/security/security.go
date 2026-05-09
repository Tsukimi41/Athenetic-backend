package security

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// TokenClaims represents JWT claims used by the API.
type TokenClaims struct {
	UserID    string `json:"uid"`
	TokenType string `json:"typ"`
	jwt.RegisteredClaims
}

func HashPassword(password string) (string, error) {
	// bcrypt cost 12 is a reasonable default for 2026 hardware
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func GenerateToken(userID uuid.UUID, tokenType string, ttl time.Duration) (string, time.Time, string, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(ttl)
	tokenID := uuid.NewString()

	claims := TokenClaims{
		UserID:    userID.String(),
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(JWTSecret())
	if err != nil {
		return "", time.Time{}, "", err
	}

	return signed, expiresAt, tokenID, nil
}

func ParseToken(tokenStr string, expectedType string) (*TokenClaims, error) {
	parsed, err := jwt.ParseWithClaims(tokenStr, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return JWTSecret(), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := parsed.Claims.(*TokenClaims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.TokenType != expectedType {
		return nil, errors.New("invalid token type")
	}

	return claims, nil
}

func AccessTokenTTL() time.Duration {
	return getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute)
}

func RefreshTokenTTL() time.Duration {
	return getEnvDuration("JWT_REFRESH_TTL", 720*time.Hour)
}

func JWTSecret() []byte {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		secret = "dev-insecure-secret-change-me"
	}
	return []byte(secret)
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}
