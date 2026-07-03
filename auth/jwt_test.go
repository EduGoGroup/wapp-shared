package auth_test

import (
	"errors"
	"testing"
	"time"

	"github.com/EduGoGroup/wapp-shared/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testSecret = "super-secret-para-tests-hs256"
	testIssuer = "wapp-iam"
)

func TestJWTManager_GenerateAndValidate(t *testing.T) {
	t.Parallel()
	m := auth.NewJWTManager(testSecret, testIssuer)

	grants := auth.Grants{Allow: []string{"flows.*", "messages.send"}, Deny: []string{"crypto.rekey"}}
	token, expiresAt, err := m.GenerateToken("user-1", "tenant-1", []string{"operator"}, grants, time.Hour)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	assert.WithinDuration(t, time.Now().Add(time.Hour), expiresAt, 5*time.Second)

	claims, err := m.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Equal(t, "tenant-1", claims.TenantID)
	assert.Equal(t, []string{"operator"}, claims.Roles)
	assert.Equal(t, auth.TokenUseAccess, claims.TokenUse)
	assert.Equal(t, grants.Allow, claims.Grants.Allow)
	assert.Equal(t, grants.Deny, claims.Grants.Deny)
	assert.Equal(t, testIssuer, claims.Issuer)
	assert.Equal(t, "user-1", claims.Subject)
}

func TestJWTManager_GenerateValidation(t *testing.T) {
	t.Parallel()
	m := auth.NewJWTManager(testSecret, testIssuer)

	_, _, err := m.GenerateToken("", "tenant-1", nil, auth.Grants{}, time.Hour)
	assert.ErrorIs(t, err, auth.ErrInvalidInput)

	_, _, err = m.GenerateToken("user-1", "", nil, auth.Grants{}, time.Hour)
	assert.ErrorIs(t, err, auth.ErrInvalidInput)

	_, _, err = m.GenerateToken("user-1", "tenant-1", nil, auth.Grants{}, time.Second)
	assert.ErrorIs(t, err, auth.ErrInvalidInput)
}

func TestJWTManager_Expired(t *testing.T) {
	t.Parallel()
	m := auth.NewJWTManager(testSecret, testIssuer)

	// Forjamos a mano un token ya vencido (exp pasado, más allá del leeway de
	// 30s), firmado con el mismo secreto e issuer que el manager.
	past := time.Now().Add(-2 * time.Hour)
	claims := auth.Claims{
		UserID:   "user-1",
		TenantID: "tenant-1",
		TokenUse: auth.TokenUseAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    testIssuer,
			Subject:   "user-1",
			IssuedAt:  jwt.NewNumericDate(past),
			ExpiresAt: jwt.NewNumericDate(past.Add(time.Minute)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	require.NoError(t, err)

	_, err = m.ValidateToken(signed)
	assert.ErrorIs(t, err, auth.ErrTokenExpired)
}

func TestJWTManager_RejectsNoneAlg(t *testing.T) {
	t.Parallel()
	m := auth.NewJWTManager(testSecret, testIssuer)

	// Token firmado con alg "none" debe rechazarse (no HS256).
	claims := auth.Claims{
		UserID:   "user-1",
		TenantID: "tenant-1",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    testIssuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = m.ValidateToken(signed)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestJWTManager_WrongSecret(t *testing.T) {
	t.Parallel()
	issuer := testIssuer
	good := auth.NewJWTManager(testSecret, issuer)
	bad := auth.NewJWTManager("otro-secreto-distinto", issuer)

	token, _, err := good.GenerateToken("user-1", "tenant-1", nil, auth.Grants{Allow: []string{"*"}}, time.Hour)
	require.NoError(t, err)

	_, err = bad.ValidateToken(token)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestJWTManager_WrongIssuer(t *testing.T) {
	t.Parallel()
	issuerA := auth.NewJWTManager(testSecret, "issuer-a")
	issuerB := auth.NewJWTManager(testSecret, "issuer-b")

	token, _, err := issuerA.GenerateToken("user-1", "tenant-1", nil, auth.Grants{Allow: []string{"*"}}, time.Hour)
	require.NoError(t, err)

	_, err = issuerB.ValidateToken(token)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestJWTManager_Garbage(t *testing.T) {
	t.Parallel()
	m := auth.NewJWTManager(testSecret, testIssuer)
	_, err := m.ValidateToken("no-es-un-jwt")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, auth.ErrInvalidToken) || errors.Is(err, auth.ErrTokenExpired))
}
