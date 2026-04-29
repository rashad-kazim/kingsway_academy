package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"kingsway/backend/internal/domain"
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	UserID   string      `json:"uid"`
	BranchID string      `json:"bid,omitempty"`
	Role     domain.Role `json:"role"`
	jwt.RegisteredClaims
}

func SignToken(secret string, claims Claims) (string, error) {
	if claims.ExpiresAt == nil {
		claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(12 * time.Hour))
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseToken(secret, raw string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}

		return []byte(secret), nil
	})
	if err != nil || !token.Valid || !claims.Role.IsValid() {
		return nil, ErrInvalidToken
	}

	if claims.Role.RequiresBranchScope() && claims.BranchID == "" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
