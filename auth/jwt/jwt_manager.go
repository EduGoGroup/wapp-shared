package jwt

import (
	"crypto/ecdsa"
	stdErrors "errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const clockLeeway = 30 * time.Second

type JWTManager struct {
	issuer       string
	method       jwt.SigningMethod
	signKey      any
	verifyKey    any
	validMethods []string
	kid          string
}

func NewJWTManager(secretKey, issuer string) *JWTManager {
	key := []byte(secretKey)
	return &JWTManager{
		issuer:       issuer,
		method:       jwt.SigningMethodHS256,
		signKey:      key,
		verifyKey:    key,
		validMethods: []string{jwt.SigningMethodHS256.Alg()},
	}
}

func NewJWTManagerES256(privateKey *ecdsa.PrivateKey, issuer string) (*JWTManager, error) {
	if privateKey == nil {
		return nil, fmt.Errorf("%w: privateKey no puede ser nil", ErrInvalidInput)
	}
	return &JWTManager{
		issuer:       issuer,
		method:       jwt.SigningMethodES256,
		signKey:      privateKey,
		verifyKey:    &privateKey.PublicKey,
		validMethods: []string{jwt.SigningMethodES256.Alg()},
	}, nil
}

func NewJWTVerifierES256(publicKey *ecdsa.PublicKey, issuer string) (*JWTManager, error) {
	if publicKey == nil {
		return nil, fmt.Errorf("%w: publicKey no puede ser nil", ErrInvalidInput)
	}
	return &JWTManager{
		issuer:       issuer,
		method:       jwt.SigningMethodES256,
		signKey:      nil,
		verifyKey:    publicKey,
		validMethods: []string{jwt.SigningMethodES256.Alg()},
	}, nil
}

func (m *JWTManager) WithKid(kid string) *JWTManager {
	clone := *m
	clone.kid = kid
	return &clone
}

func (m *JWTManager) GenerateToken(
	userID, tenantID string,
	roles []string,
	grants Grants,
	ttl time.Duration,
) (string, time.Time, error) {
	if m.signKey == nil {
		return "", time.Time{}, fmt.Errorf("%w: este manager es solo de validación (sin clave de firma)", ErrInvalidInput)
	}
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

	token := jwt.NewWithClaims(m.method, claims)
	if m.kid != "" {
		token.Header["kid"] = m.kid
	}
	signedToken, err := token.SignedString(m.signKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("no se pudo firmar el token JWT: %w", err)
	}

	return signedToken, expiresAt, nil
}

func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	parser := jwt.NewParser(
		jwt.WithIssuer(m.issuer),
		jwt.WithValidMethods(m.validMethods),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(clockLeeway),
	)

	token, err := parser.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != m.method.Alg() {
			return nil, fmt.Errorf("%w: método de firma inesperado", ErrInvalidToken)
		}
		return m.verifyKey, nil
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

	return claims, nil
}
