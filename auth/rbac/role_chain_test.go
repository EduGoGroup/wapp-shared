package rbac_test

import (
	"errors"
	"testing"

	"github.com/EduGoGroup/wapp-shared/auth/rbac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parentMapFn(parents map[string]string) func(string) (string, bool, error) {
	return func(id string) (string, bool, error) {
		p, ok := parents[id]
		return p, ok, nil
	}
}

func TestResolveRoleChain_Inheritance(t *testing.T) {
	t.Parallel()
	parents := map[string]string{
		"operator": "viewer",
	}
	chain, err := rbac.ResolveRoleChain("operator", parentMapFn(parents))
	require.NoError(t, err)
	assert.Equal(t, []string{"operator", "viewer"}, chain)
}

func TestResolveRoleChain_Canonical(t *testing.T) {
	t.Parallel()
	chain, err := rbac.ResolveRoleChain("tenant_admin", parentMapFn(map[string]string{}))
	require.NoError(t, err)
	assert.Equal(t, []string{"tenant_admin"}, chain)
}

func TestResolveRoleChain_DepthN(t *testing.T) {
	t.Parallel()
	parents := map[string]string{
		"a": "b",
		"b": "c",
	}
	chain, err := rbac.ResolveRoleChain("a", parentMapFn(parents))
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, chain)
}

func TestResolveRoleChain_CycleGuard(t *testing.T) {
	t.Parallel()
	parents := map[string]string{
		"a": "b",
		"b": "a",
	}
	chain, err := rbac.ResolveRoleChain("a", parentMapFn(parents))
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, chain)
}

func TestResolveRoleChain_PropagatesError(t *testing.T) {
	t.Parallel()
	boom := errors.New("db caída")
	_, err := rbac.ResolveRoleChain("a", func(string) (string, bool, error) {
		return "", false, boom
	})
	assert.ErrorIs(t, err, boom)
}

func TestMergeGrantChain(t *testing.T) {
	t.Parallel()
	operator := rbac.Grants{Allow: []string{"flows.*", "messages.send"}, Deny: []string{"crypto.rekey"}}
	viewer := rbac.Grants{Allow: []string{"*.read"}}

	merged := rbac.MergeGrantChain([]rbac.Grants{operator, viewer})
	assert.ElementsMatch(t, []string{"flows.*", "messages.send", "*.read"}, merged.Allow)
	assert.ElementsMatch(t, []string{"crypto.rekey"}, merged.Deny)

	assert.True(t, rbac.EvaluateGrants(merged, "flows.create"))
	assert.True(t, rbac.EvaluateGrants(merged, "contacts.read"))
	assert.False(t, rbac.EvaluateGrants(merged, "crypto.rekey"))
}

func TestMergeGrantChain_Dedup(t *testing.T) {
	t.Parallel()
	a := rbac.Grants{Allow: []string{"flows.*", "media.*"}, Deny: []string{"crypto.rekey"}}
	b := rbac.Grants{Allow: []string{"flows.*"}, Deny: []string{"crypto.rekey"}}
	merged := rbac.MergeGrantChain([]rbac.Grants{a, b})
	assert.Equal(t, []string{"flows.*", "media.*"}, merged.Allow)
	assert.Equal(t, []string{"crypto.rekey"}, merged.Deny)
}
