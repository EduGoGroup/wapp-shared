package auth_test

import (
	"errors"
	"testing"

	"github.com/EduGoGroup/wapp-shared/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parentMapFn construye un parentOf a partir de un mapa hijo→padre.
func parentMapFn(parents map[string]string) func(string) (string, bool, error) {
	return func(id string) (string, bool, error) {
		p, ok := parents[id]
		return p, ok, nil
	}
}

func TestResolveRoleChain_Inheritance(t *testing.T) {
	t.Parallel()
	// operator hereda de viewer; viewer es canónico (sin padre).
	parents := map[string]string{
		"operator": "viewer",
	}
	chain, err := auth.ResolveRoleChain("operator", parentMapFn(parents))
	require.NoError(t, err)
	assert.Equal(t, []string{"operator", "viewer"}, chain, "propio primero, ancestros después")
}

func TestResolveRoleChain_Canonical(t *testing.T) {
	t.Parallel()
	chain, err := auth.ResolveRoleChain("tenant_admin", parentMapFn(map[string]string{}))
	require.NoError(t, err)
	assert.Equal(t, []string{"tenant_admin"}, chain)
}

func TestResolveRoleChain_DepthN(t *testing.T) {
	t.Parallel()
	parents := map[string]string{
		"a": "b",
		"b": "c",
	}
	chain, err := auth.ResolveRoleChain("a", parentMapFn(parents))
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, chain)
}

func TestResolveRoleChain_CycleGuard(t *testing.T) {
	t.Parallel()
	// Ciclo A→B→A: corta sin error, sin bucle infinito.
	parents := map[string]string{
		"a": "b",
		"b": "a",
	}
	chain, err := auth.ResolveRoleChain("a", parentMapFn(parents))
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, chain)
}

func TestResolveRoleChain_PropagatesError(t *testing.T) {
	t.Parallel()
	boom := errors.New("db caída")
	_, err := auth.ResolveRoleChain("a", func(string) (string, bool, error) {
		return "", false, boom
	})
	assert.ErrorIs(t, err, boom)
}

func TestMergeGrantChain(t *testing.T) {
	t.Parallel()
	// operator (propio) hereda de viewer (ancestro): se aplana el set efectivo.
	operator := auth.Grants{Allow: []string{"flows.*", "messages.send"}, Deny: []string{"crypto.rekey"}}
	viewer := auth.Grants{Allow: []string{"*.read"}}

	merged := auth.MergeGrantChain([]auth.Grants{operator, viewer})
	assert.ElementsMatch(t, []string{"flows.*", "messages.send", "*.read"}, merged.Allow)
	assert.ElementsMatch(t, []string{"crypto.rekey"}, merged.Deny)

	// El set efectivo permite lo del propio + lo heredado; el deny sigue ganando.
	assert.True(t, auth.EvaluateGrants(merged, "flows.create"))
	assert.True(t, auth.EvaluateGrants(merged, "contacts.read"))
	assert.False(t, auth.EvaluateGrants(merged, "crypto.rekey"))
}

func TestMergeGrantChain_Dedup(t *testing.T) {
	t.Parallel()
	a := auth.Grants{Allow: []string{"flows.*", "media.*"}, Deny: []string{"crypto.rekey"}}
	b := auth.Grants{Allow: []string{"flows.*"}, Deny: []string{"crypto.rekey"}}
	merged := auth.MergeGrantChain([]auth.Grants{a, b})
	assert.Equal(t, []string{"flows.*", "media.*"}, merged.Allow, "sin duplicados, orden preservado")
	assert.Equal(t, []string{"crypto.rekey"}, merged.Deny)
}
