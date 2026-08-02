package jwt_test

import (
	"testing"
	"time"

	"github.com/EduGoGroup/wapp-shared/auth/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	svcSecret   = "service-secret-hs256"
	svcIssuer   = "wapp-iam"
	svcAudience = "wapp-public-api"
)

func TestServiceJWTManager_GenerateAndValidate(t *testing.T) {
	t.Parallel()
	m := jwt.NewServiceJWTManager(svcSecret, svcIssuer, svcAudience)

	scopes := []string{"messages.send", "flows.read"}
	token, expiresAt, err := m.GenerateServiceToken("client-abc", "tenant-1", scopes, time.Hour)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(time.Hour), expiresAt, 5*time.Second)

	claims, err := m.ValidateServiceToken(token)
	require.NoError(t, err)
	assert.Equal(t, "client-abc", claims.ClientID)
	assert.Equal(t, "tenant-1", claims.TenantID)
	assert.Equal(t, jwt.TokenUseService, claims.TokenUse)
	assert.Equal(t, "service:client-abc", claims.Subject)
	assert.Equal(t, scopes, claims.Scopes)
}

func TestServiceJWTManager_Validation(t *testing.T) {
	t.Parallel()
	m := jwt.NewServiceJWTManager(svcSecret, svcIssuer, svcAudience)

	_, _, err := m.GenerateServiceToken("", "tenant-1", nil, time.Hour)
	assert.ErrorIs(t, err, jwt.ErrInvalidInput)

	_, _, err = m.GenerateServiceToken("client-abc", "", nil, time.Hour)
	assert.ErrorIs(t, err, jwt.ErrInvalidInput)

	_, _, err = m.GenerateServiceToken("client-abc", "tenant-1", nil, time.Second)
	assert.ErrorIs(t, err, jwt.ErrInvalidInput)
}

func TestServiceJWTManager_WrongAudience(t *testing.T) {
	t.Parallel()
	signer := jwt.NewServiceJWTManager(svcSecret, svcIssuer, "aud-a")
	verifier := jwt.NewServiceJWTManager(svcSecret, svcIssuer, "aud-b")

	token, _, err := signer.GenerateServiceToken("client-abc", "tenant-1", []string{"messages.send"}, time.Hour)
	require.NoError(t, err)

	_, err = verifier.ValidateServiceToken(token)
	assert.ErrorIs(t, err, jwt.ErrInvalidToken)
}

func TestServiceToken_NotAcceptedAsUserToken(t *testing.T) {
	t.Parallel()
	svc := jwt.NewServiceJWTManager(svcSecret, svcIssuer, svcAudience)
	usr := jwt.NewJWTManager("otro-secret-de-usuarios", svcIssuer)

	token, _, err := svc.GenerateServiceToken("client-abc", "tenant-1", []string{"messages.send"}, time.Hour)
	require.NoError(t, err)

	_, err = usr.ValidateToken(token)
	assert.Error(t, err)
}
