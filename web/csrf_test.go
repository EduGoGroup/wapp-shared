package web

import (
	"net/http"
	"testing"
	"time"
)

func TestValidateCSRF(t *testing.T) {
	t.Parallel()

	casos := []struct {
		nombre   string
		cookie   string
		enviado  string
		esperado bool
	}{
		{"coinciden", "tok", "tok", true},
		{"no coinciden", "tok", "otro", false},
		{"sin cookie", "", "tok", false},
		{"sin enviado", "tok", "", false},
		{"los dos vacíos", "", "", false},
	}
	for _, c := range casos {
		if got := ValidateCSRF(c.cookie, c.enviado); got != c.esperado {
			t.Errorf("%s: ValidateCSRF(%q, %q) = %v", c.nombre, c.cookie, c.enviado, got)
		}
	}
}

func TestIsUnsafeMethod(t *testing.T) {
	t.Parallel()

	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		if !IsUnsafeMethod(m) {
			t.Errorf("%s muta estado y debe exigir CSRF", m)
		}
	}
	for _, m := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		if IsUnsafeMethod(m) {
			t.Errorf("%s es seguro y no debe exigir CSRF", m)
		}
	}
}

// TestCSRFCookie_HttpOnlyYLaxSiempre: la cookie CSRF no sigue a la config
// SameSite de la cookie de SESIÓN. Aunque la sesión se configure como None, el
// fail-safe del CSRF no se degrada.
func TestCSRFCookie_HttpOnlyYLaxSiempre(t *testing.T) {
	t.Parallel()

	ck := CSRFCookie(CSRFOptions{CookieName: "mi_csrf", Secure: true}, "tok")
	if ck.Name != "mi_csrf" {
		t.Errorf("el nombre de la cookie debe ser PARÁMETRO, got %q", ck.Name)
	}
	if !ck.HttpOnly {
		t.Error("la cookie CSRF debe ser HttpOnly")
	}
	if ck.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax siempre", ck.SameSite)
	}
	if !ck.Secure {
		t.Error("Secure debía propagarse")
	}
	if ck.MaxAge != int(DefaultCSRFMaxAge.Seconds()) {
		t.Errorf("MaxAge = %d, want %d (12 h por defecto)", ck.MaxAge, int(DefaultCSRFMaxAge.Seconds()))
	}

	corta := CSRFCookie(CSRFOptions{CookieName: "x", MaxAge: 90 * time.Second}, "tok")
	if corta.MaxAge != 90 {
		t.Errorf("MaxAge = %d, want 90 (configurable)", corta.MaxAge)
	}
}

// TestCSRFOptions_NombreDeCookieEsParametro fija D-6: dos consolas del
// ecosistema no pueden compartir la cookie, así que el nombre no puede ser una
// constante del paquete.
func TestCSRFOptions_NombreDeCookieEsParametro(t *testing.T) {
	t.Parallel()

	a := CSRFOptions{CookieName: "wapp_guardian_csrf"}.WithDefaults()
	b := CSRFOptions{CookieName: "wapp_platform_csrf"}.WithDefaults()
	if a.CookieName == b.CookieName {
		t.Fatal("cada consola debe poder poner su propio nombre de cookie")
	}
	if got := (CSRFOptions{}).WithDefaults().CookieName; got != DefaultCSRFCookieName {
		t.Errorf("sin nombre debe caer al de por defecto, got %q", got)
	}
}
