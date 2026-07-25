package jwt

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"
)

const refreshTokenBytes = 32

type RefreshToken struct {
	Token     string
	TokenHash string
	ExpiresAt time.Time
}

func GenerateRefreshToken(ttl time.Duration) (*RefreshToken, error) {
	buf := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return nil, fmt.Errorf("no se pudo generar entropía para el refresh token: %w", err)
	}

	token := base64.URLEncoding.EncodeToString(buf)

	return &RefreshToken{
		Token:     token,
		TokenHash: HashToken(token),
		ExpiresAt: time.Now().Add(ttl),
	}, nil
}

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
