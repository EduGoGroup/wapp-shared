package password_test

import (
	"strings"
	"testing"

	"github.com/EduGoGroup/wapp-shared/auth/password"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashAndVerifyPassword(t *testing.T) {
	t.Parallel()
	hash, err := password.HashPassword("s3cr3t-p4ss")
	require.NoError(t, err)
	require.NotEmpty(t, hash)
	assert.NotEqual(t, "s3cr3t-p4ss", hash)

	assert.NoError(t, password.VerifyPassword(hash, "s3cr3t-p4ss"))
	assert.Error(t, password.VerifyPassword(hash, "password-incorrecto"))
}

func TestHasherInterface(t *testing.T) {
	t.Parallel()
	var h password.Hasher = password.NewHasher()
	hash, err := h.HashPassword("mi-clave")
	require.NoError(t, err)
	assert.True(t, h.CheckPasswordHash("mi-clave", hash))
	assert.False(t, h.CheckPasswordHash("clave-falsa", hash))
}

func TestHashPassword_Salted(t *testing.T) {
	t.Parallel()
	h1, err := password.HashPassword("misma-clave")
	require.NoError(t, err)
	h2, err := password.HashPassword("misma-clave")
	require.NoError(t, err)
	assert.NotEqual(t, h1, h2)
}

func TestHashPassword_TooLong(t *testing.T) {
	t.Parallel()
	_, err := password.HashPassword(strings.Repeat("a", 73))
	assert.ErrorIs(t, err, password.ErrInvalidInput)
}
