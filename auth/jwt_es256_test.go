package auth_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
	"time"

	"github.com/EduGoGroup/wapp-shared/auth"
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
	m, err := auth.NewJWTManagerES256(key, testIssuer)
	require.NoError(t, err)

	grants := auth.Grants{Allow: []string{"flows.*"}, Deny: []string{"crypto.rekey"}}
	token, expiresAt, err := m.GenerateToken("user-es", "tenant-es", []string{"operator"}, grants, time.Hour)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	assert.WithinDuration(t, time.Now().Add(time.Hour), expiresAt, 5*time.Second)

	claims, err := m.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-es", claims.UserID)
	assert.Equal(t, "tenant-es", claims.TenantID)
	assert.Equal(t, auth.TokenUseAccess, claims.TokenUse)
	assert.Equal(t, testIssuer, claims.Issuer)
	assert.Equal(t, grants.Allow, claims.Grants.Allow)
}

func TestJWTManagerES256_NilKey(t *testing.T) {
	t.Parallel()
	_, err := auth.NewJWTManagerES256(nil, testIssuer)
	assert.ErrorIs(t, err, auth.ErrInvalidInput)
}

func TestJWTVerifierES256_ValidatesButCannotSign(t *testing.T) {
	t.Parallel()
	key := newECKey(t)
	signer, err := auth.NewJWTManagerES256(key, testIssuer)
	require.NoError(t, err)

	// El validador solo tiene la clave pública.
	verifier, err := auth.NewJWTVerifierES256(&key.PublicKey, testIssuer)
	require.NoError(t, err)

	token, _, err := signer.GenerateToken("user-1", "tenant-1", nil, auth.Grants{Allow: []string{"*"}}, time.Hour)
	require.NoError(t, err)

	// Valida un token emitido por el firmante.
	claims, err := verifier.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)

	// Pero NO puede firmar (no tiene clave privada).
	_, _, err = verifier.GenerateToken("user-1", "tenant-1", nil, auth.Grants{}, time.Hour)
	assert.ErrorIs(t, err, auth.ErrInvalidInput)
}

func TestJWTVerifierES256_NilKey(t *testing.T) {
	t.Parallel()
	_, err := auth.NewJWTVerifierES256(nil, testIssuer)
	assert.ErrorIs(t, err, auth.ErrInvalidInput)
}

func TestJWTVerifierES256_WrongPublicKeyRejected(t *testing.T) {
	t.Parallel()
	signer, err := auth.NewJWTManagerES256(newECKey(t), testIssuer)
	require.NoError(t, err)

	// Verificador con una pública DISTINTA a la que firmó.
	otherKey := newECKey(t)
	verifier, err := auth.NewJWTVerifierES256(&otherKey.PublicKey, testIssuer)
	require.NoError(t, err)

	token, _, err := signer.GenerateToken("user-1", "tenant-1", nil, auth.Grants{}, time.Hour)
	require.NoError(t, err)

	_, err = verifier.ValidateToken(token)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

// TestJWT_CrossAlgRejection cubre el guard anti alg-confusion en ambos sentidos:
// un token firmado con un algoritmo se rechaza limpio al validarlo con un
// manager del otro algoritmo.
func TestJWT_CrossAlgRejection(t *testing.T) {
	t.Parallel()

	hs := auth.NewJWTManager(testSecret, testIssuer)
	es, err := auth.NewJWTManagerES256(newECKey(t), testIssuer)
	require.NoError(t, err)

	hsToken, _, err := hs.GenerateToken("u", "t", nil, auth.Grants{}, time.Hour)
	require.NoError(t, err)
	esToken, _, err := es.GenerateToken("u", "t", nil, auth.Grants{}, time.Hour)
	require.NoError(t, err)

	// HS256 validado como ES256 → rechazo.
	_, err = es.ValidateToken(hsToken)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)

	// ES256 validado como HS256 → rechazo.
	_, err = hs.ValidateToken(esToken)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}
