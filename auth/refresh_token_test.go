package auth_test

import (
	"testing"
	"time"

	"github.com/EduGoGroup/wapp-shared/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateRefreshToken(t *testing.T) {
	t.Parallel()
	rt, err := auth.GenerateRefreshToken(24 * time.Hour)
	require.NoError(t, err)
	require.NotNil(t, rt)

	assert.NotEmpty(t, rt.Token)
	assert.NotEmpty(t, rt.TokenHash)
	assert.NotEqual(t, rt.Token, rt.TokenHash, "se persiste el hash, no el token")
	assert.WithinDuration(t, time.Now().Add(24*time.Hour), rt.ExpiresAt, 5*time.Second)

	// El hash es determinístico y coincide con HashToken.
	assert.Equal(t, rt.TokenHash, auth.HashToken(rt.Token))
}

func TestGenerateRefreshToken_Unique(t *testing.T) {
	t.Parallel()
	a, err := auth.GenerateRefreshToken(time.Hour)
	require.NoError(t, err)
	b, err := auth.GenerateRefreshToken(time.Hour)
	require.NoError(t, err)
	assert.NotEqual(t, a.Token, b.Token, "cada refresh token es único")
	assert.NotEqual(t, a.TokenHash, b.TokenHash)
}

func TestHashToken_Stable(t *testing.T) {
	t.Parallel()
	assert.Equal(t, auth.HashToken("abc"), auth.HashToken("abc"))
	assert.NotEqual(t, auth.HashToken("abc"), auth.HashToken("abd"))
	// SHA256 hex = 64 chars.
	assert.Len(t, auth.HashToken("abc"), 64)
}
