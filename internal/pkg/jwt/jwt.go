package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ErrInvalidToken is returned when a token fails validation.
var ErrInvalidToken = errors.New("invalid or expired token")

// Claims is the JWT payload for an authenticated user.
type Claims struct {
	UserID   uuid.UUID `json:"uid"`
	Username string    `json:"username"`
	jwt.RegisteredClaims
}

// Manager issues and verifies HS256 access tokens.
type Manager struct {
	secret []byte
	expiry time.Duration
}

func NewManager(secret string, expiry time.Duration) *Manager {
	return &Manager{secret: []byte(secret), expiry: expiry}
}

// Generate returns a signed access token and its expiry time.
func (m *Manager) Generate(userID uuid.UUID, username string) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(m.expiry)
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, exp, nil
}

// Parse validates a token string and returns its claims.
func (m *Manager) Parse(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
