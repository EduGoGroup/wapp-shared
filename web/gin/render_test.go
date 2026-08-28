package webgin_test

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/EduGoGroup/wapp-shared/web"
	webgin "github.com/EduGoGroup/wapp-shared/web/gin"
)

// TestRenderer_SiembraNonceYTokenEnTodaPagina es el motivo de que el
// renderizador exista: la consola que no lo tenía repetía el nonce a mano en una
// decena de sitios, y esa es exactamente la repetición que un día se olvida en
// una pantalla nueva.
func TestRenderer_SiembraNonceYTokenEnTodaPagina(t *testing.T) {
	t.Parallel()

	tmpl := template.Must(template.New("base.html").Parse(
		`nonce={{ .Nonce }} csrf={{ .CSRFToken }} path={{ .CurrentPath }} auth={{ .IsAuthenticated }} content={{ .ContentTemplate }} propio={{ .Propio }}`))

	r := nuevoEngine()
	r.SetHTMLTemplate(tmpl)
	r.Use(webgin.SecurityHeaders(web.SecurityOptions{}))
	r.Use(webgin.CSRF(web.CSRFOptions{CookieName: "wapp_test_csrf"}))
	render := webgin.NewRenderer("")
	r.GET("/pagina", func(c *gin.Context) {
		c.Set(webgin.ContextAccessToken, "a.b.c")
		render.HTML(c, http.StatusOK, "pages/algo.html", gin.H{"Propio": "dato"})
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/pagina", nil))

	cuerpo := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, cuerpo = %q", rec.Code, cuerpo)
	}
	for _, quiere := range []string{"path=/pagina", "auth=true", "content=pages/algo.html", "propio=dato"} {
		if !strings.Contains(cuerpo, quiere) {
			t.Errorf("el render no incluye %q; got %q", quiere, cuerpo)
		}
	}
	if strings.Contains(cuerpo, "nonce= ") || strings.Contains(cuerpo, "csrf= ") {
		t.Fatalf("el nonce y el token deben llegar rellenos: %q", cuerpo)
	}
	// El nonce del cuerpo tiene que ser EL MISMO que el de la cabecera CSP, o el
	// navegador bloquea el bloque inline.
	nonce := entre(cuerpo, "nonce=", " csrf=")
	if nonce == "" || !strings.Contains(rec.Header().Get("Content-Security-Policy"), "'nonce-"+nonce+"'") {
		t.Fatalf("el nonce de la página (%q) no coincide con el de la CSP (%q)", nonce, rec.Header().Get("Content-Security-Policy"))
	}
}

// TestRenderer_DataNilEsValido.
func TestRenderer_DataNilEsValido(t *testing.T) {
	t.Parallel()

	r := nuevoEngine()
	r.SetHTMLTemplate(template.Must(template.New("maestro.html").Parse(`ok {{ .ContentTemplate }}`)))
	render := webgin.NewRenderer("maestro.html")
	r.GET("/", func(c *gin.Context) { render.HTML(c, http.StatusOK, "x.html", nil) })

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "ok x.html") {
		t.Fatalf("status = %d, cuerpo = %q", rec.Code, rec.Body.String())
	}
}

// entre devuelve lo que hay entre dos marcas (cadena vacía si no están).
func entre(s, desde, hasta string) string {
	i := strings.Index(s, desde)
	if i < 0 {
		return ""
	}
	resto := s[i+len(desde):]
	j := strings.Index(resto, hasta)
	if j < 0 {
		return ""
	}
	return resto[:j]
}
