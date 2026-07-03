package auth

import (
	stdErrors "errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// clockLeeway es la tolerancia de reloj aplicada a `nbf`/`iat`/`exp` para
// evitar rechazos por skew de pocos segundos entre instancias.
const clockLeeway = 30 * time.Second

// JWTManager emite y valida JWT de usuario firmados con HS256, usando un
// secreto e issuer fijos.
type JWTManager struct {
	issuer    string
	secretKey []byte
}

// NewJWTManager crea un JWTManager con el secreto HS256 y el issuer esperado.
func NewJWTManager(secretKey, issuer string) *JWTManager {
	return &JWTManager{
		secretKey: []byte(secretKey),
		issuer:    issuer,
	}
}

// GenerateToken firma un access token de usuario.
//
// Parámetros:
//   - userID: identificador del usuario (requerido).
//   - tenantID: tenant al que pertenece la sesión (requerido).
//   - roles: roles asignados (snapshot informativo).
//   - grants: grants efectivos ya resueltos (allow/deny) que el middleware
//     evalúa por request.
//   - ttl: duración hasta la expiración (mínimo 1 minuto).
//
// Devuelve el token firmado y su instante de expiración.
func (m *JWTManager) GenerateToken(
	userID, tenantID string,
	roles []string,
	grants Grants,
	ttl time.Duration,
) (string, time.Time, error) {
	if userID == "" {
		return "", time.Time{}, fmt.Errorf("%w: userID no puede estar vacío", ErrInvalidInput)
	}
	if tenantID == "" {
		return "", time.Time{}, fmt.Errorf("%w: tenantID no puede estar vacío", ErrInvalidInput)
	}
	if ttl < time.Minute {
		return "", time.Time{}, fmt.Errorf("%w: ttl debe ser mayor a 1 minuto", ErrInvalidInput)
	}

	now := time.Now()
	expiresAt := now.Add(ttl)

	claims := Claims{
		UserID:   userID,
		TenantID: tenantID,
		Roles:    roles,
		Grants:   grants,
		TokenUse: TokenUseAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("no se pudo firmar el token JWT: %w", err)
	}

	return signedToken, expiresAt, nil
}

// ValidateToken parsea y valida un JWT: firma HS256, issuer y expiración
// (`exp` obligatorio). Devuelve los claims o ErrTokenExpired/ErrInvalidToken.
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	parser := jwt.NewParser(
		jwt.WithIssuer(m.issuer),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(clockLeeway),
	)

	token, err := parser.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: método de firma inesperado", ErrInvalidToken)
		}
		return m.secretKey, nil
	})
	if err != nil {
		if stdErrors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	if claims.Issuer != m.issuer {
		return nil, fmt.Errorf("%w: issuer inesperado", ErrInvalidToken)
	}

	return claims, nil
}
