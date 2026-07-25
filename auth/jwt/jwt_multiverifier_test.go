package jwt_test

import (
	"testing"
	"time"

	"github.com/EduGoGroup/wapp-shared/auth/jwt"
	"github.com/EduGoGroup/wapp-shared/auth/rbac"
	jwt5 "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	kidES = "es-2026"
	kidHS = "hs-legacy"
)

func TestMultiVerifier_SelectsByKid(t *testing.T) {
	t.Parallel()
	esKey := newECKey(t)
	esSigner, err := jwt.NewJWTManagerES256(esKey, testIssuer)
	require.NoError(t, err)
	hsSigner := jwt.NewJWTManager(testSecret, testIssuer)

	mv, err := jwt.NewMultiVerifier(testIssuer, map[string]jwt.VerifierKey{
		kidES: jwt.ES256VerifierKey(&esKey.PublicKey),
		kidHS: jwt.HS256VerifierKey(testSecret),
	}, jwt.VerifierKey{})
	require.NoError(t, err)

	esToken, _, err := esSigner.WithKid(kidES).GenerateToken("user-es", "tenant-es", []string{"operator"}, rbac.Grants{Allow: []string{"*"}}, time.Hour)
	require.NoError(t, err)
	hsToken, _, err := hsSigner.WithKid(kidHS).GenerateToken("user-hs", "tenant-hs", nil, rbac.Grants{}, time.Hour)
	require.NoError(t, err)

	esClaims, err := mv.ValidateToken(esToken)
	require.NoError(t, err)
	assert.Equal(t, "user-es", esClaims.UserID)

	hsClaims, err := mv.ValidateToken(hsToken)
	require.NoError(t, err)
	assert.Equal(t, "user-hs", hsClaims.UserID)
}

func TestMultiVerifier_NoKidUsesDefault(t *testing.T) {
	t.Parallel()
	esKey := newECKey(t)
	legacySigner := jwt.NewJWTManager(testSecret, testIssuer)

	mv, err := jwt.NewMultiVerifier(testIssuer, map[string]jwt.VerifierKey{
		kidES: jwt.ES256VerifierKey(&esKey.PublicKey),
	}, jwt.HS256VerifierKey(testSecret))
	require.NoError(t, err)

	token, _, err := legacySigner.GenerateToken("legacy-user", "tenant-x", nil, rbac.Grants{}, time.Hour)
	require.NoError(t, err)

	claims, err := mv.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "legacy-user", claims.UserID)
}

func TestMultiVerifier_NoKidNoDefaultRejected(t *testing.T) {
	t.Parallel()
	legacySigner := jwt.NewJWTManager(testSecret, testIssuer)
	mv, err := jwt.NewMultiVerifier(testIssuer, map[string]jwt.VerifierKey{
		kidHS: jwt.HS256VerifierKey(testSecret),
	}, jwt.VerifierKey{})
	require.NoError(t, err)

	token, _, err := legacySigner.GenerateToken("u", "t", nil, rbac.Grants{}, time.Hour)
	require.NoError(t, err)

	_, err = mv.ValidateToken(token)
	assert.ErrorIs(t, err, jwt.ErrInvalidToken)
}

func TestMultiVerifier_UnknownKidRejected(t *testing.T) {
	t.Parallel()
	signer := jwt.NewJWTManager(testSecret, testIssuer)
	mv, err := jwt.NewMultiVerifier(testIssuer, map[string]jwt.VerifierKey{
		kidHS: jwt.HS256VerifierKey(testSecret),
	}, jwt.VerifierKey{})
	require.NoError(t, err)

	token, _, err := signer.WithKid("otro-kid").GenerateToken("u", "t", nil, rbac.Grants{}, time.Hour)
	require.NoError(t, err)

	_, err = mv.ValidateToken(token)
	assert.ErrorIs(t, err, jwt.ErrInvalidToken)
}

func TestMultiVerifier_CrossAlgRejected(t *testing.T) {
	t.Parallel()
	esKey := newECKey(t)
	esSigner, err := jwt.NewJWTManagerES256(esKey, testIssuer)
	require.NoError(t, err)
	hsSigner := jwt.NewJWTManager(testSecret, testIssuer)

	mv, err := jwt.NewMultiVerifier(testIssuer, map[string]jwt.VerifierKey{
		kidES: jwt.ES256VerifierKey(&esKey.PublicKey),
		kidHS: jwt.HS256VerifierKey(testSecret),
	}, jwt.VerifierKey{})
	require.NoError(t, err)

	hsWithESKid, _, err := hsSigner.WithKid(kidES).GenerateToken("u", "t", nil, rbac.Grants{}, time.Hour)
	require.NoError(t, err)
	_, err = mv.ValidateToken(hsWithESKid)
	assert.ErrorIs(t, err, jwt.ErrInvalidToken)

	esWithHSKid, _, err := esSigner.WithKid(kidHS).GenerateToken("u", "t", nil, rbac.Grants{}, time.Hour)
	require.NoError(t, err)
	_, err = mv.ValidateToken(esWithHSKid)
	assert.ErrorIs(t, err, jwt.ErrInvalidToken)
}

func TestMultiVerifier_Expired(t *testing.T) {
	t.Parallel()
	mv, err := jwt.NewMultiVerifier(testIssuer, map[string]jwt.VerifierKey{
		kidHS: jwt.HS256VerifierKey(testSecret),
	}, jwt.VerifierKey{})
	require.NoError(t, err)

	past := time.Now().Add(-2 * time.Hour)
	claims := jwt.Claims{
		UserID:   "u",
		TenantID: "t",
		TokenUse: jwt.TokenUseAccess,
		RegisteredClaims: jwt5.RegisteredClaims{
			Issuer:    testIssuer,
			Subject:   "u",
			IssuedAt:  jwt5.NewNumericDate(past),
			NotBefore: jwt5.NewNumericDate(past),
			ExpiresAt: jwt5.NewNumericDate(past.Add(time.Minute)),
		},
	}
	tok := jwt5.NewWithClaims(jwt5.SigningMethodHS256, claims)
	tok.Header["kid"] = kidHS
	signed, err := tok.SignedString([]byte(testSecret))
	require.NoError(t, err)

	_, err = mv.ValidateToken(signed)
	assert.ErrorIs(t, err, jwt.ErrTokenExpired)
}

func TestMultiVerifier_MalformedRejected(t *testing.T) {
	t.Parallel()
	mv, err := jwt.NewMultiVerifier(testIssuer, map[string]jwt.VerifierKey{
		kidHS: jwt.HS256VerifierKey(testSecret),
	}, jwt.VerifierKey{})
	require.NoError(t, err)

	_, err = mv.ValidateToken("no-es-un-jwt")
	assert.ErrorIs(t, err, jwt.ErrInvalidToken)
}

func TestNewMultiVerifier_InvalidInput(t *testing.T) {
	t.Parallel()

	_, err := jwt.NewMultiVerifier(testIssuer, nil, jwt.VerifierKey{})
	assert.ErrorIs(t, err, jwt.ErrInvalidInput)

	_, err = jwt.NewMultiVerifier(testIssuer, map[string]jwt.VerifierKey{
		kidES: jwt.ES256VerifierKey(nil),
	}, jwt.VerifierKey{})
	assert.ErrorIs(t, err, jwt.ErrInvalidInput)

	_, err = jwt.NewMultiVerifier(testIssuer, map[string]jwt.VerifierKey{
		kidHS: jwt.HS256VerifierKey(""),
	}, jwt.VerifierKey{})
	assert.ErrorIs(t, err, jwt.ErrInvalidInput)

	_, err = jwt.NewMultiVerifier(testIssuer, map[string]jwt.VerifierKey{
		"": jwt.HS256VerifierKey(testSecret),
	}, jwt.VerifierKey{})
	assert.ErrorIs(t, err, jwt.ErrInvalidInput)

	_, err = jwt.NewMultiVerifier(testIssuer, map[string]jwt.VerifierKey{
		kidHS: jwt.HS256VerifierKey(testSecret),
	}, jwt.ES256VerifierKey(nil))
	assert.ErrorIs(t, err, jwt.ErrInvalidInput)
}
