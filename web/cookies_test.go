package web

import (
	"net/http"
	"testing"
	"time"
)

func TestSameSiteMode(t *testing.T) {
	t.Parallel()

	casos := []struct {
		modo     string
		secure   bool
		esperado http.SameSite
	}{
		{"strict", true, http.SameSiteStrictMode},
		{"  STRICT ", true, http.SameSiteStrictMode},
		{"lax", false, http.SameSiteLaxMode},
		{"none", true, http.SameSiteNoneMode},
		{"none", false, http.SameSiteLaxMode}, // None sin Secure lo rechaza el navegador: degradar a Lax.
		{"loquesea", false, http.SameSiteLaxMode},
		{"", false, http.SameSiteLaxMode},
	}
	for _, c := range casos {
		if got := SameSiteMode(c.modo, c.secure); got != c.esperado {
			t.Errorf("SameSiteMode(%q, %v) = %v, want %v", c.modo, c.secure, got, c.esperado)
		}
	}
}

func TestSessionMaxAge(t *testing.T) {
	t.Parallel()

	porDefecto := int(DefaultSessionMaxAge.Seconds())

	if got := SessionMaxAge("no es una fecha"); got != porDefecto {
		t.Errorf("una fecha ilegible debe caer al valor por defecto, got %d", got)
	}
	if got := SessionMaxAge(time.Now().Add(-time.Hour).Format(time.RFC3339)); got != porDefecto {
		t.Errorf("una fecha ya pasada debe caer al valor por defecto (no a un MaxAge negativo que borre la cookie), got %d", got)
	}
	got := SessionMaxAge(time.Now().Add(2 * time.Hour).Format(time.RFC3339))
	if got < 7100 || got > 7200 {
		t.Errorf("SessionMaxAge = %d, want ~7200", got)
	}
}

func TestSessionCookie_HttpOnlySiempreYNombreParametro(t *testing.T) {
	t.Parallel()

	ck := SessionCookie(SessionCookieOptions{Name: "wapp_guardian_session", Secure: true, SameSite: "strict"}, "v", 3600)
	if ck.Name != "wapp_guardian_session" {
		t.Errorf("nombre = %q, debe ser parámetro", ck.Name)
	}
	if !ck.HttpOnly {
		t.Error("la cookie de sesión debe ser HttpOnly")
	}
	if ck.SameSite != http.SameSiteStrictMode {
		t.Errorf("SameSite = %v, want Strict", ck.SameSite)
	}
	if got := (SessionCookieOptions{}).WithDefaults().Name; got != DefaultSessionCookieName {
		t.Errorf("sin nombre cae al de por defecto, got %q", got)
	}
}
