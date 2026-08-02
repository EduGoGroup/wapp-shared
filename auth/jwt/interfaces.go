package jwt

import (
	"time"
)

// TokenGenerator define el contrato para la generación de tokens JWT.
type TokenGenerator interface {
	GenerateToken(userID, tenantID string, roles []string, grants Grants, ttl time.Duration) (string, time.Time, error)
}

// TokenVerifier define el contrato para la validación de tokens JWT.
type TokenVerifier interface {
	ValidateToken(tokenString string) (*Claims, error)
}

// TokenManager combina las capacidades de generación y verificación de tokens.
type TokenManager interface {
	TokenGenerator
	TokenVerifier
}
