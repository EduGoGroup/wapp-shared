package auth_test

import (
	"testing"
	"time"

	"github.com/EduGoGroup/wapp-shared/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	kidES = "es-2026"
	kidHS = "hs-legacy"
)

// TestMultiVerifier_SelectsByKid: un token con `kid` conocido se valida con la
// entrada correcta (ES256 y HS256 en el mismo verificador).
func TestMultiVerifier_SelectsByKid(t *testing.T) {
	t.Parallel()
	esKey := newECKey(t)
	esSigner, err := auth.NewJWTManagerES256(esKey, testIssuer)
	require.NoError(t, err)
	hsSigner := auth.NewJWTManager(testSecret, testIssuer)

	mv, err := auth.NewMultiVerifier(testIssuer, map[string]auth.VerifierKey{
		kidES: auth.ES256VerifierKey(&esKey.PublicKey),
		kidHS: auth.HS256VerifierKey(testSecret),
	}, auth.VerifierKey{})
	require.NoError(t, err)

	esToken, _, err := esSigner.WithKid(kidES).GenerateToken("user-es", "tenant-es", []string{"operator"}, auth.Grants{Allow: []string{"*"}}, time.Hour)
	require.NoError(t, err)
	hsToken, _, err := hsSigner.WithKid(kidHS).GenerateToken("user-hs", "tenant-hs", nil, auth.Grants{}, time.Hour)
	require.NoError(t, err)

	esClaims, err := mv.ValidateToken(esToken)
	require.NoError(t, err)
	assert.Equal(t, "user-es", esClaims.UserID)

	hsClaims, err := mv.ValidateToken(hsToken)
	require.NoError(t, err)
	assert.Equal(t, "user-hs", hsClaims.UserID)
}

// TestMultiVerifier_NoKidUsesDefault: un token SIN `kid` (legacy) se valida con
// la entrada default.
func TestMultiVerifier_NoKidUsesDefault(t *testing.T) {
	t.Parallel()
	esKey := newECKey(t)
	legacySigner := auth.NewJWTManager(testSecret, testIssuer) // emite sin kid

	mv, err := auth.NewMultiVerifier(testIssuer, map[string]auth.VerifierKey{
		kidES: auth.ES256VerifierKey(&esKey.PublicKey),
	}, auth.HS256VerifierKey(testSecret))
	require.NoError(t, err)

	token, _, err := legacySigner.GenerateToken("legacy-user", "tenant-x", nil, auth.Grants{}, time.Hour)
	require.NoError(t, err)

	claims, err := mv.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "legacy-user", claims.UserID)
}

// TestMultiVerifier_NoKidNoDefaultRejected: sin default, un token sin `kid` se
// rechaza.
func TestMultiVerifier_NoKidNoDefaultRejected(t *testing.T) {
	t.Parallel()
	legacySigner := auth.NewJWTManager(testSecret, testIssuer)
	mv, err := auth.NewMultiVerifier(testIssuer, map[string]auth.VerifierKey{
		kidHS: auth.HS256VerifierKey(testSecret),
	}, auth.VerifierKey{})
	require.NoError(t, err)

	token, _, err := legacySigner.GenerateToken("u", "t", nil, auth.Grants{}, time.Hour)
	require.NoError(t, err)

	_, err = mv.ValidateToken(token)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

// TestMultiVerifier_UnknownKidRejected: un `kid` no registrado se rechaza.
func TestMultiVerifier_UnknownKidRejected(t *testing.T) {
	t.Parallel()
	signer := auth.NewJWTManager(testSecret, testIssuer)
	mv, err := auth.NewMultiVerifier(testIssuer, map[string]auth.VerifierKey{
		kidHS: auth.HS256VerifierKey(testSecret),
	}, auth.VerifierKey{})
	require.NoError(t, err)

	token, _, err := signer.WithKid("otro-kid").GenerateToken("u", "t", nil, auth.Grants{}, time.Hour)
	require.NoError(t, err)

	_, err = mv.ValidateToken(token)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

// TestMultiVerifier_CrossAlgRejected cubre el guard anti alg-confusion por
// entrada en ambos sentidos: un token cuyo `alg` no coincide con el de la
// entrada seleccionada por su `kid` se rechaza.
func TestMultiVerifier_CrossAlgRejected(t *testing.T) {
	t.Parallel()
	esKey := newECKey(t)
	esSigner, err := auth.NewJWTManagerES256(esKey, testIssuer)
	require.NoError(t, err)
	hsSigner := auth.NewJWTManager(testSecret, testIssuer)

	mv, err := auth.NewMultiVerifier(testIssuer, map[string]auth.VerifierKey{
		kidES: auth.ES256VerifierKey(&esKey.PublicKey),
		kidHS: auth.HS256VerifierKey(testSecret),
	}, auth.VerifierKey{})
	require.NoError(t, err)

	// Token HS256 estampado con el kid de la entrada ES256 → rechazo.
	hsWithESKid, _, err := hsSigner.WithKid(kidES).GenerateToken("u", "t", nil, auth.Grants{}, time.Hour)
	require.NoError(t, err)
	_, err = mv.ValidateToken(hsWithESKid)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)

	// Token ES256 estampado con el kid de la entrada HS256 → rechazo.
	esWithHSKid, _, err := esSigner.WithKid(kidHS).GenerateToken("u", "t", nil, auth.Grants{}, time.Hour)
	require.NoError(t, err)
	_, err = mv.ValidateToken(esWithHSKid)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

// TestMultiVerifier_Expired: un token bien firmado pero vencido devuelve
// ErrTokenExpired (misma semántica que JWTManager.ValidateToken). Forjamos el
// token con `exp` en el pasado directamente con jwt/v5 porque la API pública
// impone ttl>=1min y no permite un vencimiento determinista.
func TestMultiVerifier_Expired(t *testing.T) {
	t.Parallel()
	mv, err := auth.NewMultiVerifier(testIssuer, map[string]auth.VerifierKey{
		kidHS: auth.HS256VerifierKey(testSecret),
	}, auth.VerifierKey{})
	require.NoError(t, err)

	past := time.Now().Add(-2 * time.Hour)
	claims := auth.Claims{
		UserID:   "u",
		TenantID: "t",
		TokenUse: auth.TokenUseAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    testIssuer,
			Subject:   "u",
			IssuedAt:  jwt.NewNumericDate(past),
			NotBefore: jwt.NewNumericDate(past),
			ExpiresAt: jwt.NewNumericDate(past.Add(time.Minute)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tok.Header["kid"] = kidHS
	signed, err := tok.SignedString([]byte(testSecret))
	require.NoError(t, err)

	_, err = mv.ValidateToken(signed)
	assert.ErrorIs(t, err, auth.ErrTokenExpired)
}

// TestMultiVerifier_MalformedRejected: un token basura se rechaza limpio.
func TestMultiVerifier_MalformedRejected(t *testing.T) {
	t.Parallel()
	mv, err := auth.NewMultiVerifier(testIssuer, map[string]auth.VerifierKey{
		kidHS: auth.HS256VerifierKey(testSecret),
	}, auth.VerifierKey{})
	require.NoError(t, err)

	_, err = mv.ValidateToken("no-es-un-jwt")
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

// TestNewMultiVerifier_InvalidInput cubre los rechazos de construcción.
func TestNewMultiVerifier_InvalidInput(t *testing.T) {
	t.Parallel()

	// Sin entradas ni default.
	_, err := auth.NewMultiVerifier(testIssuer, nil, auth.VerifierKey{})
	assert.ErrorIs(t, err, auth.ErrInvalidInput)

	// Entrada ES256 con clave nil.
	_, err = auth.NewMultiVerifier(testIssuer, map[string]auth.VerifierKey{
		kidES: auth.ES256VerifierKey(nil),
	}, auth.VerifierKey{})
	assert.ErrorIs(t, err, auth.ErrInvalidInput)

	// Entrada HS256 con secreto vacío.
	_, err = auth.NewMultiVerifier(testIssuer, map[string]auth.VerifierKey{
		kidHS: auth.HS256VerifierKey(""),
	}, auth.VerifierKey{})
	assert.ErrorIs(t, err, auth.ErrInvalidInput)

	// kid vacío.
	_, err = auth.NewMultiVerifier(testIssuer, map[string]auth.VerifierKey{
		"": auth.HS256VerifierKey(testSecret),
	}, auth.VerifierKey{})
	assert.ErrorIs(t, err, auth.ErrInvalidInput)

	// Default inválido.
	_, err = auth.NewMultiVerifier(testIssuer, map[string]auth.VerifierKey{
		kidHS: auth.HS256VerifierKey(testSecret),
	}, auth.ES256VerifierKey(nil))
	assert.ErrorIs(t, err, auth.ErrInvalidInput)
}
