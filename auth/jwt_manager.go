package auth

import (
	"crypto/ecdsa"
	stdErrors "errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// clockLeeway es la tolerancia de reloj aplicada a `nbf`/`iat`/`exp` para
// evitar rechazos por skew de pocos segundos entre instancias.
const clockLeeway = 30 * time.Second

// JWTManager emite y valida JWT de usuario. Soporta dos algoritmos de firma que
// coexisten (ADR-0019):
//
//   - HS256 (simétrico): [NewJWTManager]. Firma y valida con el mismo secreto.
//   - ES256 (asimétrico, ECDSA P-256): [NewJWTManagerES256] firma con la clave
//     privada y valida con la pública; [NewJWTVerifierES256] valida con SOLO la
//     pública (para validadores que no emiten, p. ej. el BFF).
//
// Un manager valida EXCLUSIVAMENTE su propio algoritmo (guard anti
// alg-confusion): un token HS256 verificado por un manager ES256 —o viceversa—
// se rechaza con ErrInvalidToken.
type JWTManager struct {
	issuer       string
	method       jwt.SigningMethod
	signKey      any // clave para firmar: []byte (HS256) o *ecdsa.PrivateKey (ES256); nil = solo validación.
	verifyKey    any // clave para validar: []byte (HS256) o *ecdsa.PublicKey (ES256).
	validMethods []string
}

// NewJWTManager crea un JWTManager HS256 con el secreto y el issuer esperado.
// Firma y valida con el mismo secreto (simétrico).
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

// NewJWTManagerES256 crea un JWTManager que FIRMA y valida con ES256 (ECDSA
// P-256). Usa la clave privada para emitir y su pública para validar. Devuelve
// ErrInvalidInput si privateKey es nil.
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

// NewJWTVerifierES256 crea un JWTManager de SOLO validación con la clave pública
// ES256. Es la construcción para validadores que no emiten (mínimo privilegio,
// ADR-0019): al no tener clave privada no pueden forjar tokens y [GenerateToken]
// devuelve ErrInvalidInput. Devuelve ErrInvalidInput si publicKey es nil.
func NewJWTVerifierES256(publicKey *ecdsa.PublicKey, issuer string) (*JWTManager, error) {
	if publicKey == nil {
		return nil, fmt.Errorf("%w: publicKey no puede ser nil", ErrInvalidInput)
	}
	return &JWTManager{
		issuer:       issuer,
		method:       jwt.SigningMethodES256,
		signKey:      nil, // verificador puro: no puede firmar.
		verifyKey:    publicKey,
		validMethods: []string{jwt.SigningMethodES256.Alg()},
	}, nil
}

// GenerateToken firma un access token de usuario con el algoritmo del manager.
//
// Parámetros:
//   - userID: identificador del usuario (requerido).
//   - tenantID: tenant al que pertenece la sesión (requerido).
//   - roles: roles asignados (snapshot informativo).
//   - grants: grants efectivos ya resueltos (allow/deny) que el middleware
//     evalúa por request.
//   - ttl: duración hasta la expiración (mínimo 1 minuto).
//
// Devuelve el token firmado y su instante de expiración. Si el manager es un
// verificador puro (sin clave de firma) devuelve ErrInvalidInput.
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
	signedToken, err := token.SignedString(m.signKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("no se pudo firmar el token JWT: %w", err)
	}

	return signedToken, expiresAt, nil
}

// ValidateToken parsea y valida un JWT: firma (con el algoritmo del manager),
// issuer y expiración (`exp` obligatorio). Devuelve los claims o
// ErrTokenExpired/ErrInvalidToken.
//
// El issuer lo verifica el propio parser vía [jwt.WithIssuer] (única fuente de
// verdad); un issuer inesperado se propaga como ErrInvalidToken. El algoritmo se
// restringe a [jwt.WithValidMethods] y, como defensa en profundidad, la keyfunc
// exige que el `alg` del token coincida con el del manager (anti alg-confusion).
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
