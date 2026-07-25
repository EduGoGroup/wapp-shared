package jwt

import (
	stdErrors "errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const TokenUseService = "service"

type ServiceClaims struct {
	TokenUse string   `json:"token_use"`
	ClientID string   `json:"client_id"`
	TenantID string   `json:"tenant_id"`
	Scopes   []string `json:"scopes"`
	jwt.RegisteredClaims
}

func (c *ServiceClaims) HasScope(scope string) bool {
	for _, s := range c.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

type ServiceJWTManager struct {
	issuer    string
	audience  string
	secretKey []byte
}

func NewServiceJWTManager(secretKey, issuer, audience string) *ServiceJWTManager {
	return &ServiceJWTManager{
		secretKey: []byte(secretKey),
		issuer:    issuer,
		audience:  audience,
	}
}

func (m *ServiceJWTManager) GenerateServiceToken(
	clientID, tenantID string,
	scopes []string,
	ttl time.Duration,
) (string, time.Time, error) {
	if clientID == "" {
		return "", time.Time{}, fmt.Errorf("%w: clientID no puede estar vacío", ErrInvalidInput)
	}
	if tenantID == "" {
		return "", time.Time{}, fmt.Errorf("%w: tenantID no puede estar vacío", ErrInvalidInput)
	}
	if ttl < time.Minute {
		return "", time.Time{}, fmt.Errorf("%w: ttl debe ser mayor a 1 minuto", ErrInvalidInput)
	}

	now := time.Now()
	expiresAt := now.Add(ttl)

	claims := ServiceClaims{
		TokenUse: TokenUseService,
		ClientID: clientID,
		TenantID: tenantID,
		Scopes:   scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    m.issuer,
			Subject:   "service:" + clientID,
			Audience:  jwt.ClaimStrings{m.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("no se pudo firmar el service JWT: %w", err)
	}

	return signedToken, expiresAt, nil
}

func (m *ServiceJWTManager) ValidateServiceToken(tokenString string) (*ServiceClaims, error) {
	parser := jwt.NewParser(
		jwt.WithIssuer(m.issuer),
		jwt.WithAudience(m.audience),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(clockLeeway),
	)

	token, err := parser.ParseWithClaims(tokenString, &ServiceClaims{}, func(token *jwt.Token) (any, error) {
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

	claims, ok := token.Claims.(*ServiceClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	if claims.Issuer != m.issuer {
		return nil, fmt.Errorf("%w: issuer inesperado", ErrInvalidToken)
	}
	if claims.TokenUse != TokenUseService {
		return nil, fmt.Errorf("%w: se esperaba un service token", ErrInvalidToken)
	}

	return claims, nil
}
