package auth_test

import (
	"testing"

	"github.com/EduGoGroup/wapp-shared/auth"
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
			assert.Equal(t, tc.want, auth.PermissionMatches(tc.pattern, tc.request))
		})
	}
}

func TestEvaluateGrants_DefaultDeny(t *testing.T) {
	t.Parallel()
	// Sin allow → deny por defecto.
	g := auth.Grants{}
	assert.False(t, auth.EvaluateGrants(g, "messages.send"))

	g2 := auth.Grants{Allow: []string{"flows.*"}}
	assert.False(t, auth.EvaluateGrants(g2, "messages.send"))
}

func TestEvaluateGrants_DenyPrecedeAllow(t *testing.T) {
	t.Parallel()
	// Allow amplio pero deny específico gana.
	g := auth.Grants{
		Allow: []string{"*"},
		Deny:  []string{"crypto.rekey"},
	}
	assert.True(t, auth.EvaluateGrants(g, "messages.send"), "cubierto por allow *")
	assert.False(t, auth.EvaluateGrants(g, "crypto.rekey"), "deny específico precede al allow amplio")

	// Deny por wildcard gana sobre allow exacto.
	g2 := auth.Grants{
		Allow: []string{"messages.send"},
		Deny:  []string{"messages.*"},
	}
	assert.False(t, auth.EvaluateGrants(g2, "messages.send"))
}

// TestEvaluateGrants_CanonicalRoles verifica los roles canónicos de design.md §5.
func TestEvaluateGrants_CanonicalRoles(t *testing.T) {
	t.Parallel()

	tenantAdmin := auth.Grants{Allow: []string{"*"}}
	operator := auth.Grants{Allow: []string{
		"flows.*", "messages.send", "media.*", "contacts.read", "integrations.read",
	}}
	viewer := auth.Grants{Allow: []string{"*.read"}}

	// tenant_admin puede todo.
	for _, perm := range []string{"flows.create", "messages.send", "crypto.rekey", "leases.revoke"} {
		assert.True(t, auth.EvaluateGrants(tenantAdmin, perm), "tenant_admin debe permitir %s", perm)
	}

	// operator: puede lo suyo, no crypto.rekey ni integrations.manage.
	assert.True(t, auth.EvaluateGrants(operator, "flows.create"))
	assert.True(t, auth.EvaluateGrants(operator, "flows.start"))
	assert.True(t, auth.EvaluateGrants(operator, "messages.send"))
	assert.True(t, auth.EvaluateGrants(operator, "media.upload"))
	assert.True(t, auth.EvaluateGrants(operator, "contacts.read"))
	assert.True(t, auth.EvaluateGrants(operator, "integrations.read"))
	assert.False(t, auth.EvaluateGrants(operator, "crypto.rekey"))
	assert.False(t, auth.EvaluateGrants(operator, "integrations.manage"))
	assert.False(t, auth.EvaluateGrants(operator, "messages.read"), "operator solo tiene messages.send")

	// viewer: solo *.read.
	assert.True(t, auth.EvaluateGrants(viewer, "contacts.read"))
	assert.True(t, auth.EvaluateGrants(viewer, "flows.read"))
	assert.False(t, auth.EvaluateGrants(viewer, "messages.send"))
	assert.False(t, auth.EvaluateGrants(viewer, "flows.create"))
}
