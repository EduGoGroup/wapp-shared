package web

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

// failingReader agota la entropía a propósito.
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("sin entropía") }

// TestBuildCSP_UnionDeLasDosConsolas fija la política que este módulo sirve, que
// es la UNIÓN de las dos que reconcilia: `font-src 'self'` y `object-src 'none'`
// venían del BFF, y `base-uri 'none'` —más estricto que el `'self'` del BFF— de
// la consola de plataforma. Si alguien "simplifica" la política, esto cae.
func TestBuildCSP_UnionDeLasDosConsolas(t *testing.T) {
	t.Parallel()

	csp := BuildCSP("N0NC3")

	for _, quiere := range []string{
		"default-src 'self'",
		"script-src 'self' 'nonce-N0NC3'",
		"style-src 'self' 'nonce-N0NC3'",
		"font-src 'self'",
		"img-src 'self' data:",
		"connect-src 'self'",
		"base-uri 'none'",
		"form-action 'self'",
		"frame-ancestors 'none'",
		"object-src 'none'",
	} {
		if !strings.Contains(csp, quiere) {
			t.Errorf("la CSP no contiene %q; got %q", quiere, csp)
		}
	}
	if strings.Contains(csp, "base-uri 'self'") {
		t.Errorf("base-uri debe ser 'none' (ganó la consola de plataforma), got %q", csp)
	}
	if strings.Contains(csp, "'unsafe-inline'") {
		t.Errorf("la CSP NO debe usar 'unsafe-inline' (para eso está el nonce), got %q", csp)
	}
	if strings.Contains(csp, "*") {
		t.Errorf("la CSP no admite comodines, got %q", csp)
	}
}

// TestNonce_AlfabetoURLSafe verifica que el nonce no trae caracteres que
// html/template escaparía dentro del atributo `nonce` ('+', '/', '='): si los
// trajera, el nonce del atributo dejaría de coincidir con el de la cabecera y el
// navegador bloquearía el bloque inline propio.
func TestNonce_AlfabetoURLSafe(t *testing.T) {
	t.Parallel()

	for i := 0; i < 200; i++ {
		n, err := Nonce(nil)
		if err != nil {
			t.Fatalf("Nonce(nil) devolvió error: %v", err)
		}
		if strings.ContainsAny(n, "+/=") {
			t.Fatalf("el nonce debe ser base64 URL-safe sin padding, got %q", n)
		}
	}
}

// TestNonceYToken_FallanCerradoSinEntropia: los dos comparten la MISMA fuente a
// propósito (un solo punto de fallo), así que los dos tienen que fallar.
func TestNonceYToken_FallanCerradoSinEntropia(t *testing.T) {
	t.Parallel()

	if _, err := Nonce(failingReader{}); err == nil {
		t.Error("Nonce debe devolver error si no hay entropía")
	}
	if _, err := NewCSRFToken(failingReader{}); err == nil {
		t.Error("NewCSRFToken debe devolver error si no hay entropía")
	}
}

func TestApplySecurityHeaders_HSTSSoloSiSePide(t *testing.T) {
	t.Parallel()

	sin := http.Header{}
	ApplySecurityHeaders(sin, "n", SecurityOptions{HSTS: false})
	if got := sin.Get("Strict-Transport-Security"); got != "" {
		t.Errorf("sin HSTS no debe emitirse la cabecera, got %q", got)
	}
	if got := sin.Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q, want DENY", got)
	}
	if got := sin.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := sin.Get("Referrer-Policy"); got != "no-referrer" {
		t.Errorf("Referrer-Policy = %q, want no-referrer", got)
	}

	con := http.Header{}
	ApplySecurityHeaders(con, "n", SecurityOptions{HSTS: true})
	if got := con.Get("Strict-Transport-Security"); got != "max-age=31536000; includeSubDomains" {
		t.Errorf("HSTS = %q", got)
	}
}
