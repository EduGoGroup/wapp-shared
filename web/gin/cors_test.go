package webgin_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/EduGoGroup/wapp-shared/web"
	webgin "github.com/EduGoGroup/wapp-shared/web/gin"
)

func routerCORS(origenes ...string) *gin.Engine {
	r := nuevoEngine()
	r.Use(webgin.CORS(web.CORSOptions{AllowedOrigins: origenes}))
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	return r
}

func pideConOrigen(r *gin.Engine, metodo, origen string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(metodo, "/", nil)
	if origen != "" {
		req.Header.Set("Origin", origen)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// TestCORS_PreflightDeOrigenNoPermitidoDa403 es lo que se rescató de la consola
// de plataforma: el preflight de un origen que no está invitado se corta con
// 403, no con el 204 de cortesía.
func TestCORS_PreflightDeOrigenNoPermitidoDa403(t *testing.T) {
	t.Parallel()

	r := routerCORS("https://ok.example")

	rec := pideConOrigen(r, http.MethodOptions, "https://evil.example")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("preflight de origen no permitido = %d, want 403", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("jamás se hace eco de un origen no listado, got %q", got)
	}

	ok := pideConOrigen(r, http.MethodOptions, "https://ok.example")
	if ok.Code != http.StatusNoContent {
		t.Fatalf("preflight permitido = %d, want 204", ok.Code)
	}
	if got := ok.Header().Get("Access-Control-Allow-Origin"); got != "https://ok.example" {
		t.Errorf("ACAO = %q", got)
	}

	sinOrigen := pideConOrigen(r, http.MethodOptions, "")
	if sinOrigen.Code != http.StatusNoContent {
		t.Fatalf("preflight sin Origin = %d, want 204", sinOrigen.Code)
	}
}

// TestCORS_ElWildcardEnConfigNoAbreNada: si alguien pone "*" en la variable de
// entorno, no se convierte en una allowlist que lo permita todo.
func TestCORS_ElWildcardEnConfigNoAbreNada(t *testing.T) {
	t.Parallel()

	r := routerCORS("*")

	rec := pideConOrigen(r, http.MethodGet, "https://cualquiera.example")
	if rec.Code != http.StatusOK {
		t.Fatalf("un GET normal no se corta, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf(`con "*" en config no debe emitirse ACAO, got %q`, got)
	}

	pre := pideConOrigen(r, http.MethodOptions, "https://cualquiera.example")
	if pre.Code != http.StatusForbidden {
		t.Fatalf(`con "*" descartado el preflight debe dar 403, got %d`, pre.Code)
	}
}
