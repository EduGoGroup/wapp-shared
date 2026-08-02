package jwt_test

import (
	"testing"
	"time"

	"github.com/EduGoGroup/wapp-shared/auth/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateRefreshToken(t *testing.T) {
	t.Parallel()
	rt, err := jwt.GenerateRefreshToken(24 * time.Hour)
	require.NoError(t, err)
	require.NotNil(t, rt)

	assert.NotEmpty(t, rt.Token)
	assert.NotEmpty(t, rt.TokenHash)
	assert.NotEqual(t, rt.Token, rt.TokenHash)
	assert.WithinDuration(t, time.Now().Add(24*time.Hour), rt.ExpiresAt, 5*time.Second)

	assert.Equal(t, rt.TokenHash, jwt.HashToken(rt.Token))
}

func TestGenerateRefreshToken_Unique(t *testing.T) {
	t.Parallel()
	a, err := jwt.GenerateRefreshToken(time.Hour)
	require.NoError(t, err)
	b, err := jwt.GenerateRefreshToken(time.Hour)
	require.NoError(t, err)
	assert.NotEqual(t, a.Token, b.Token)
	assert.NotEqual(t, a.TokenHash, b.TokenHash)
}

func TestHashToken_Stable(t *testing.T) {
	t.Parallel()
	assert.Equal(t, jwt.HashToken("abc"), jwt.HashToken("abc"))
	assert.NotEqual(t, jwt.HashToken("abc"), jwt.HashToken("abd"))
	assert.Len(t, jwt.HashToken("abc"), 64)
}
