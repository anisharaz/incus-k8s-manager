// Package auth issues and validates the JWTs used for session cookies.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/anisharaz/incus-k8s-manager/be/internal/models"
	"github.com/golang-jwt/jwt/v5"
)

// TTL is how long an issued token (and its cookie) is valid for.
const TTL = 24 * time.Hour

// Claims are embedded in every token so handlers/middleware can identify
// the caller and their role without a database lookup per request.
type Claims struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken issues a signed JWT for user, valid for TTL.
func GenerateToken(secret []byte, user models.User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(TTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ParseToken validates tokenString's signature and expiry and returns its claims.
func ParseToken(secret []byte, tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
