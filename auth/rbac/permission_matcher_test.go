package rbac_test

import (
	"testing"

	"github.com/EduGoGroup/wapp-shared/auth/rbac"
	"github.com/stretchr/testify/assert"
)

func TestPermissionMatches(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		pattern string
		request string
		want    bool
	}{
		{"wildcard total", "*", "flows.create", true},
		{"exacto", "messages.send", "messages.send", true},
		{"exacto no coincide", "messages.send", "messages.read", false},
		{"subarbol raiz", "flows.*", "flows", true},
		{"subarbol hijo", "flows.*", "flows.create", true},
		{"subarbol nieto", "flows.*", "flows.node.read", true},
		{"subarbol no aplica", "flows.*", "messages.send", false},
		{"suffix read", "*.read", "contacts.read", true},
		{"suffix read multi", "*.read", "flows.node.read", true},
		{"suffix no coincide", "*.read", "contacts.write", false},
		{"suffix no matchea solo suffix", "*.read", "read", false},
		{"prefix.*.suffix ok", "flows.*.read", "flows.node.read", true},
		{"prefix.*.suffix sin intermedio", "flows.*.read", "flows.read", false},
		{"prefix.*.suffix prefijo malo", "flows.*.read", "messages.node.read", false},
		{"no coincide vacio", "flows.create", "", false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, rbac.PermissionMatches(tc.pattern, tc.request))
		})
	}
}

func TestEvaluateGrants_DefaultDeny(t *testing.T) {
	t.Parallel()
	g := rbac.Grants{}
	assert.False(t, rbac.EvaluateGrants(g, "messages.send"))

	g2 := rbac.Grants{Allow: []string{"flows.*"}}
	assert.False(t, rbac.EvaluateGrants(g2, "messages.send"))
}

func TestEvaluateGrants_DenyPrecedeAllow(t *testing.T) {
	t.Parallel()
	g := rbac.Grants{
		Allow: []string{"*"},
		Deny:  []string{"crypto.rekey"},
	}
	assert.True(t, rbac.EvaluateGrants(g, "messages.send"))
	assert.False(t, rbac.EvaluateGrants(g, "crypto.rekey"))

	g2 := rbac.Grants{
		Allow: []string{"messages.send"},
		Deny:  []string{"messages.*"},
	}
	assert.False(t, rbac.EvaluateGrants(g2, "messages.send"))
}

func TestEvaluateGrants_CanonicalRoles(t *testing.T) {
	t.Parallel()

	tenantAdmin := rbac.Grants{Allow: []string{"*"}}
	operator := rbac.Grants{Allow: []string{
		"flows.*", "messages.send", "media.*", "contacts.read", "integrations.read",
	}}
	viewer := rbac.Grants{Allow: []string{"*.read"}}

	for _, perm := range []string{"flows.create", "messages.send", "crypto.rekey", "leases.revoke"} {
		assert.True(t, rbac.EvaluateGrants(tenantAdmin, perm))
	}

	assert.True(t, rbac.EvaluateGrants(operator, "flows.create"))
	assert.True(t, rbac.EvaluateGrants(operator, "flows.start"))
	assert.True(t, rbac.EvaluateGrants(operator, "messages.send"))
	assert.True(t, rbac.EvaluateGrants(operator, "media.upload"))
	assert.True(t, rbac.EvaluateGrants(operator, "contacts.read"))
	assert.True(t, rbac.EvaluateGrants(operator, "integrations.read"))
	assert.False(t, rbac.EvaluateGrants(operator, "crypto.rekey"))
	assert.False(t, rbac.EvaluateGrants(operator, "integrations.manage"))
	assert.False(t, rbac.EvaluateGrants(operator, "messages.read"))

	assert.True(t, rbac.EvaluateGrants(viewer, "contacts.read"))
	assert.True(t, rbac.EvaluateGrants(viewer, "flows.read"))
	assert.False(t, rbac.EvaluateGrants(viewer, "messages.send"))
	assert.False(t, rbac.EvaluateGrants(viewer, "flows.create"))
}
