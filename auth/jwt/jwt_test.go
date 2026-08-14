package jwt_test

import (
	"errors"
	"testing"
	"time"

	"github.com/EduGoGroup/identity-shared/auth/rbac"
	"github.com/EduGoGroup/wapp-shared/auth/jwt"
	jwt5 "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testSecret = "super-secret-para-tests-hs256"
	testIssuer = "wapp-iam"
)

func TestJWTManager_GenerateAndValidate(t *testing.T) {
	t.Parallel()
	m := jwt.NewJWTManager(testSecret, testIssuer)

	grants := rbac.Grants{Allow: []string{"flows.*", "messages.send"}, Deny: []string{"crypto.rekey"}}
	token, expiresAt, err := m.GenerateToken("user-1", "tenant-1", []string{"operator"}, grants, time.Hour)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	assert.WithinDuration(t, time.Now().Add(time.Hour), expiresAt, 5*time.Second)

	claims, err := m.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Equal(t, "tenant-1", claims.TenantID)
	assert.Equal(t, []string{"operator"}, claims.Roles)
	assert.Equal(t, jwt.TokenUseAccess, claims.TokenUse)
	assert.Equal(t, grants.Allow, claims.Grants.Allow)
	assert.Equal(t, grants.Deny, claims.Grants.Deny)
	assert.Equal(t, testIssuer, claims.Issuer)
	assert.Equal(t, "user-1", claims.Subject)
}

// TestJWTManager_GenerateTenantlessToken cubre el emisor SIN tenant (wApp Plan
// 056 · D-056.12): una identidad acreditada que todavía no pertenece a ninguna
// empresa. El token es válido y no autoriza nada.
func TestJWTManager_GenerateTenantlessToken(t *testing.T) {
	t.Parallel()
	m := jwt.NewJWTManager(testSecret, testIssuer)

	token, expiresAt, err := m.GenerateTenantlessToken("user-1", time.Hour)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	assert.WithinDuration(t, time.Now().Add(time.Hour), expiresAt, 5*time.Second)

	claims, err := m.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Empty(t, claims.TenantID, "un token sin empresa no lleva tenant_id")
	// Vacías pero NO nil: el wire format no cambia de forma por no haber tenant.
	assert.Equal(t, []string{}, claims.Roles)
	assert.Equal(t, []string{}, claims.Grants.Allow)
	assert.Equal(t, []string{}, claims.Grants.Deny)
	// Lo demás es idéntico a GenerateToken: misma función privada arma los claims.
	assert.Equal(t, jwt.TokenUseAccess, claims.TokenUse)
	assert.Equal(t, testIssuer, claims.Issuer)
	assert.Equal(t, "user-1", claims.Subject)
	assert.NotEmpty(t, claims.ID)
	assert.False(t, rbac.EvaluateGrants(claims.Grants, "flows.read"),
		"sin grants el default DENY tiene que cerrarlo todo")
}

func TestJWTManager_GenerateTenantlessTokenValidation(t *testing.T) {
	t.Parallel()
	m := jwt.NewJWTManager(testSecret, testIssuer)

	_, _, err := m.GenerateTenantlessToken("", time.Hour)
	assert.ErrorIs(t, err, jwt.ErrInvalidInput)

	_, _, err = m.GenerateTenantlessToken("user-1", time.Second)
	assert.ErrorIs(t, err, jwt.ErrInvalidInput)
}

func TestJWTManager_GenerateValidation(t *testing.T) {
	t.Parallel()
	m := jwt.NewJWTManager(testSecret, testIssuer)

	_, _, err := m.GenerateToken("", "tenant-1", nil, rbac.Grants{}, time.Hour)
	assert.ErrorIs(t, err, jwt.ErrInvalidInput)

	_, _, err = m.GenerateToken("user-1", "", nil, rbac.Grants{}, time.Hour)
	assert.ErrorIs(t, err, jwt.ErrInvalidInput)

	_, _, err = m.GenerateToken("user-1", "tenant-1", nil, rbac.Grants{}, time.Second)
	assert.ErrorIs(t, err, jwt.ErrInvalidInput)
}

func TestJWTManager_Expired(t *testing.T) {
	t.Parallel()
	m := jwt.NewJWTManager(testSecret, testIssuer)

	past := time.Now().Add(-2 * time.Hour)
	claims := jwt.Claims{
		UserID:   "user-1",
		TenantID: "tenant-1",
		TokenUse: jwt.TokenUseAccess,
		RegisteredClaims: jwt5.RegisteredClaims{
			Issuer:    testIssuer,
			Subject:   "user-1",
			IssuedAt:  jwt5.NewNumericDate(past),
			ExpiresAt: jwt5.NewNumericDate(past.Add(time.Minute)),
		},
	}
	signed, err := jwt5.NewWithClaims(jwt5.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	require.NoError(t, err)

	_, err = m.ValidateToken(signed)
	assert.ErrorIs(t, err, jwt.ErrTokenExpired)
}

func TestJWTManager_RejectsNoneAlg(t *testing.T) {
	t.Parallel()
	m := jwt.NewJWTManager(testSecret, testIssuer)

	claims := jwt.Claims{
		UserID:   "user-1",
		TenantID: "tenant-1",
		RegisteredClaims: jwt5.RegisteredClaims{
			Issuer:    testIssuer,
			ExpiresAt: jwt5.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	signed, err := jwt5.NewWithClaims(jwt5.SigningMethodNone, claims).SignedString(jwt5.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = m.ValidateToken(signed)
	assert.ErrorIs(t, err, jwt.ErrInvalidToken)
}

func TestJWTManager_WrongSecret(t *testing.T) {
	t.Parallel()
	good := jwt.NewJWTManager(testSecret, testIssuer)
	bad := jwt.NewJWTManager("otro-secreto-distinto", testIssuer)

	token, _, err := good.GenerateToken("user-1", "tenant-1", nil, rbac.Grants{Allow: []string{"*"}}, time.Hour)
	require.NoError(t, err)

	_, err = bad.ValidateToken(token)
	assert.ErrorIs(t, err, jwt.ErrInvalidToken)
}

func TestJWTManager_WrongIssuer(t *testing.T) {
	t.Parallel()
	issuerA := jwt.NewJWTManager(testSecret, "issuer-a")
	issuerB := jwt.NewJWTManager(testSecret, "issuer-b")

	token, _, err := issuerA.GenerateToken("user-1", "tenant-1", nil, rbac.Grants{Allow: []string{"*"}}, time.Hour)
	require.NoError(t, err)

	_, err = issuerB.ValidateToken(token)
	assert.ErrorIs(t, err, jwt.ErrInvalidToken)
}

func TestJWTManager_Garbage(t *testing.T) {
	t.Parallel()
	m := jwt.NewJWTManager(testSecret, testIssuer)
	_, err := m.ValidateToken("no-es-un-jwt")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, jwt.ErrInvalidToken) || errors.Is(err, jwt.ErrTokenExpired))
}
