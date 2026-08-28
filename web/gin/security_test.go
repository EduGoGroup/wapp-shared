package webgin_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/EduGoGroup/wapp-shared/web"
	webgin "github.com/EduGoGroup/wapp-shared/web/gin"
)

// sinEntropia agota la fuente de aleatoriedad a propósito.
type sinEntropia struct{}

func (sinEntropia) Read([]byte) (int, error) { return 0, errors.New("sin entropía") }

// nuevoEngine monta un engine desnudo para los tests de middleware.
func nuevoEngine() *gin.Engine {
	webgin.SetReleaseMode()
	return gin.New()
}

// TestSecurityHeaders_FallaCerradoSinEntropia es el requisito duro: si no se
// puede generar el nonce, NO se sirve la página. Un 200 con inline y sin nonce
// sería servir sin defensa anti-XSS justo cuando el sistema está peor.
func TestSecurityHeaders_FallaCerradoSinEntropia(t *testing.T) {
	t.Parallel()

	r := nuevoEngine()
	r.Use(webgin.SecurityHeaders(web.SecurityOptions{HSTS: true, Rand: sinEntropia{}}))
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "no debería llegar aquí") })

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("sin nonce hay que fallar cerrado con 500, got %d", rec.Code)
	}
	if csp := rec.Header().Get("Content-Security-Policy"); csp != "" {
		t.Errorf("no debe emitirse CSP cuando falla el nonce, got %q", csp)
	}
	if rec.Body.String() != "" {
		t.Errorf("no debe servirse cuerpo alguno, got %q", rec.Body.String())
	}
}

// TestSecurityHeaders_SiembraNonceYPolitica.
func TestSecurityHeaders_SiembraNonceYPolitica(t *testing.T) {
	t.Parallel()

	r := nuevoEngine()
	r.Use(webgin.SecurityHeaders(web.SecurityOptions{HSTS: true}))
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, webgin.NonceFromContext(c)) })

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	nonce := rec.Body.String()
	if nonce == "" {
		t.Fatal("el middleware debe sembrar el nonce en el contexto")
	}
	csp := rec.Header().Get("Content-Security-Policy")
	for _, quiere := range []string{"'nonce-" + nonce + "'", "base-uri 'none'", "object-src 'none'", "font-src 'self'"} {
		if !strings.Contains(csp, quiere) {
			t.Errorf("la CSP servida no contiene %q; got %q", quiere, csp)
		}
	}
	if rec.Header().Get("Strict-Transport-Security") == "" {
		t.Error("con HSTS activado debe emitirse la cabecera")
	}
}

// TestSecurityHeaders_NonceDistintoPorPeticion: un nonce reutilizado no sirve de
// nada (un atacante que lo leyera una vez podría inyectar en la siguiente).
func TestSecurityHeaders_NonceDistintoPorPeticion(t *testing.T) {
	t.Parallel()

	r := nuevoEngine()
	r.Use(webgin.SecurityHeaders(web.SecurityOptions{}))
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, webgin.NonceFromContext(c)) })

	pide := func() string {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		return rec.Body.String()
	}
	primero, segundo := pide(), pide()
	if primero == "" || primero == segundo {
		t.Fatalf("el nonce debe ser único por petición: %q vs %q", primero, segundo)
	}
}
