package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"kingsway/backend/internal/domain"
)

var ErrInvalidToken = errors.New("invalid token")
var ErrWeakJWTSecret = errors.New("weak jwt secret")

const (
	DefaultJWTSecret             = "dev-only-change-me"
	minProductionJWTSecretLength = 32
)

type Claims struct {
	UserID       string      `json:"uid"`
	BranchID     string      `json:"bid,omitempty"`
	Role         domain.Role `json:"role"`
	TokenVersion int         `json:"tv"`
	jwt.RegisteredClaims
}

func SignToken(secret string, claims Claims) (string, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "", ErrWeakJWTSecret
	}
	if claims.ExpiresAt == nil {
		claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(12 * time.Hour))
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseToken(secret, raw string) (*Claims, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, ErrInvalidToken
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}

		return []byte(secret), nil
	})
	if err != nil || !token.Valid || !claims.Role.IsValid() || claims.TokenVersion <= 0 {
		return nil, ErrInvalidToken
	}

	if claims.Role.RequiresBranchScope() && claims.BranchID == "" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func ValidateJWTSecret(appEnv, secret string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return ErrWeakJWTSecret
	}
	if isLocalLikeEnv(appEnv) {
		return nil
	}
	if isWeakJWTSecret(secret) {
		return ErrWeakJWTSecret
	}

	return nil
}

func isLocalLikeEnv(appEnv string) bool {
	switch strings.ToLower(strings.TrimSpace(appEnv)) {
	case "", "local", "dev", "development", "test", "testing":
		return true
	default:
		return false
	}
}

func isWeakJWTSecret(secret string) bool {
	normalized := strings.ToLower(strings.TrimSpace(secret))
	switch normalized {
	case DefaultJWTSecret, "secret", "jwt-secret", "change-me", "changeme", "password":
		return true
	default:
		return len(secret) < minProductionJWTSecretLength
	}
}
