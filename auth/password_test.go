package auth_test

import (
	"strings"
	"testing"

	"github.com/EduGoGroup/wapp-shared/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashAndVerifyPassword(t *testing.T) {
	t.Parallel()
	hash, err := auth.HashPassword("s3cr3t-p4ss")
	require.NoError(t, err)
	require.NotEmpty(t, hash)
	assert.NotEqual(t, "s3cr3t-p4ss", hash, "el hash no puede ser el password en claro")

	assert.NoError(t, auth.VerifyPassword(hash, "s3cr3t-p4ss"))
	assert.Error(t, auth.VerifyPassword(hash, "password-incorrecto"))
}

func TestHashPassword_Salted(t *testing.T) {
	t.Parallel()
	h1, err := auth.HashPassword("misma-clave")
	require.NoError(t, err)
	h2, err := auth.HashPassword("misma-clave")
	require.NoError(t, err)
	assert.NotEqual(t, h1, h2, "el salt aleatorio produce hashes distintos")
}

func TestHashPassword_TooLong(t *testing.T) {
	t.Parallel()
	_, err := auth.HashPassword(strings.Repeat("a", 73))
	assert.ErrorIs(t, err, auth.ErrInvalidInput)
}
