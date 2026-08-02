package jwt_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
	"time"

	"github.com/EduGoGroup/wapp-shared/auth/jwt"
	"github.com/EduGoGroup/wapp-shared/auth/rbac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newECKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return key
}

func TestJWTManagerES256_GenerateAndValidate(t *testing.T) {
	t.Parallel()
	key := newECKey(t)
	m, err := jwt.NewJWTManagerES256(key, testIssuer)
	require.NoError(t, err)

	grants := rbac.Grants{Allow: []string{"flows.*"}, Deny: []string{"crypto.rekey"}}
	token, expiresAt, err := m.GenerateToken("user-es", "tenant-es", []string{"operator"}, grants, time.Hour)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	assert.WithinDuration(t, time.Now().Add(time.Hour), expiresAt, 5*time.Second)

	claims, err := m.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-es", claims.UserID)
	assert.Equal(t, "tenant-es", claims.TenantID)
	assert.Equal(t, jwt.TokenUseAccess, claims.TokenUse)
	assert.Equal(t, testIssuer, claims.Issuer)
	assert.Equal(t, grants.Allow, claims.Grants.Allow)
}

func TestJWTManagerES256_NilKey(t *testing.T) {
	t.Parallel()
	_, err := jwt.NewJWTManagerES256(nil, testIssuer)
	assert.ErrorIs(t, err, jwt.ErrInvalidInput)
}

func TestJWTVerifierES256_ValidatesButCannotSign(t *testing.T) {
	t.Parallel()
	key := newECKey(t)
	signer, err := jwt.NewJWTManagerES256(key, testIssuer)
	require.NoError(t, err)

	verifier, err := jwt.NewJWTVerifierES256(&key.PublicKey, testIssuer)
	require.NoError(t, err)

	token, _, err := signer.GenerateToken("user-1", "tenant-1", nil, rbac.Grants{Allow: []string{"*"}}, time.Hour)
	require.NoError(t, err)

	claims, err := verifier.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)

	_, _, err = verifier.GenerateToken("user-1", "tenant-1", nil, rbac.Grants{}, time.Hour)
	assert.ErrorIs(t, err, jwt.ErrInvalidInput)
}

func TestJWTVerifierES256_NilKey(t *testing.T) {
	t.Parallel()
	_, err := jwt.NewJWTVerifierES256(nil, testIssuer)
	assert.ErrorIs(t, err, jwt.ErrInvalidInput)
}

func TestJWTVerifierES256_WrongPublicKeyRejected(t *testing.T) {
	t.Parallel()
	signer, err := jwt.NewJWTManagerES256(newECKey(t), testIssuer)
	require.NoError(t, err)

	otherKey := newECKey(t)
	verifier, err := jwt.NewJWTVerifierES256(&otherKey.PublicKey, testIssuer)
	require.NoError(t, err)

	token, _, err := signer.GenerateToken("user-1", "tenant-1", nil, rbac.Grants{}, time.Hour)
	require.NoError(t, err)

	_, err = verifier.ValidateToken(token)
	assert.ErrorIs(t, err, jwt.ErrInvalidToken)
}

func TestJWT_CrossAlgRejection(t *testing.T) {
	t.Parallel()

	hs := jwt.NewJWTManager(testSecret, testIssuer)
	es, err := jwt.NewJWTManagerES256(newECKey(t), testIssuer)
	require.NoError(t, err)

	hsToken, _, err := hs.GenerateToken("u", "t", nil, rbac.Grants{}, time.Hour)
	require.NoError(t, err)
	esToken, _, err := es.GenerateToken("u", "t", nil, rbac.Grants{}, time.Hour)
	require.NoError(t, err)

	_, err = es.ValidateToken(hsToken)
	assert.ErrorIs(t, err, jwt.ErrInvalidToken)

	_, err = hs.ValidateToken(esToken)
	assert.ErrorIs(t, err, jwt.ErrInvalidToken)
}
