package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"
)

// refreshTokenBytes es el tamaño en bytes de entropía del refresh token opaco.
const refreshTokenBytes = 32

// RefreshToken es un refresh token OPACO: `Token` se entrega al cliente y
// `TokenHash` (SHA256 de `Token`) es lo único que se persiste, de modo que un
// leak de la BD no revela tokens usables.
type RefreshToken struct {
	Token     string    // texto plano, se retorna al cliente
	TokenHash string    // SHA256 hex del token, se guarda en BD
	ExpiresAt time.Time // instante de expiración
}

// GenerateRefreshToken genera un refresh token opaco criptográficamente seguro
// (32 bytes de crypto/rand, base64 URL-safe) junto con su hash SHA256 y su
// expiración.
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

// HashToken devuelve el SHA256 (hex) de un token opaco. Sirve para comparar un
// token recibido contra el `token_hash` persistido.
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
