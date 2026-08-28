package webgin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/EduGoGroup/wapp-shared/web"
	webgin "github.com/EduGoGroup/wapp-shared/web/gin"
)

const cookieDePrueba = "wapp_test_csrf"

// conCookie adjunta una cookie a la PETICIÓN. Se escribe la cabecera a mano en
// vez de construir un http.Cookie porque los atributos (Secure, HttpOnly,
// SameSite) son cosa de la RESPUESTA: en la petición no existen, y un literal
// http.Cookie sin ellos solo consigue que el analizador de seguridad avise de
// algo que aquí no aplica.
func conCookie(req *http.Request, nombre, valor string) {
	req.Header.Add("Cookie", nombre+"="+valor)
}

func routerCSRF(opts web.CSRFOptions) *gin.Engine {
	r := nuevoEngine()
	r.Use(webgin.CSRF(opts))
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, webgin.CSRFTokenFromContext(c)) })
	r.POST("/", func(c *gin.Context) { c.String(http.StatusOK, "mutado") })
	return r
}

// TestCSRF_ValidaAntesDeSembrar es D-5, y el test discrimina de verdad entre las
// dos implementaciones que este módulo reconcilia: la que sembraba primero
// devolvía el 403 CON un Set-Cookie que el atacante puede provocar a voluntad.
// Aquí un POST sin cookie tiene que salir con 403 y SIN Set-Cookie.
func TestCSRF_ValidaAntesDeSembrar(t *testing.T) {
	t.Parallel()

	r := routerCSRF(web.CSRFOptions{CookieName: cookieDePrueba})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("un POST sin token debe dar 403, got %d", rec.Code)
	}
	if sc := rec.Header().Values("Set-Cookie"); len(sc) != 0 {
		t.Fatalf("el rechazo NO debe sembrar cookie (validar va antes de sembrar), got %v", sc)
	}
}

// TestCSRF_SiembraEnElGETYAceptaElPOST: el camino feliz completo.
func TestCSRF_SiembraEnElGETYAceptaElPOST(t *testing.T) {
	t.Parallel()

	r := routerCSRF(web.CSRFOptions{CookieName: cookieDePrueba, Secure: true})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET = %d", rec.Code)
	}
	token := rec.Body.String()
	if token == "" {
		t.Fatal("el GET debe sembrar el token en el contexto para el formulario")
	}
	setCookie := rec.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, cookieDePrueba+"=") {
		t.Fatalf("el GET debe fijar la cookie, got %q", setCookie)
	}
	for _, atributo := range []string{"HttpOnly", "SameSite=Lax", "Secure"} {
		if !strings.Contains(setCookie, atributo) {
			t.Errorf("la cookie CSRF debe llevar %s, got %q", atributo, setCookie)
		}
	}

	// POST por cabecera, con la cookie de vuelta: pasa.
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	conCookie(req, cookieDePrueba, token)
	req.Header.Set(web.CSRFHeaderName, token)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusOK {
		t.Fatalf("POST con token válido = %d, want 200", rec2.Code)
	}

	// El mismo POST con un token que no casa: 403.
	req3 := httptest.NewRequest(http.MethodPost, "/", nil)
	conCookie(req3, cookieDePrueba, token)
	req3.Header.Set(web.CSRFHeaderName, token+"x")
	rec3 := httptest.NewRecorder()
	r.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusForbidden {
		t.Fatalf("POST con token que no casa = %d, want 403", rec3.Code)
	}
}

// TestCSRF_NoRotaElTokenExistente: rotarlo en cada GET invalidaría los
// formularios abiertos en otras pestañas.
func TestCSRF_NoRotaElTokenExistente(t *testing.T) {
	t.Parallel()

	r := routerCSRF(web.CSRFOptions{CookieName: cookieDePrueba})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	conCookie(req, cookieDePrueba, "token-que-ya-tenia")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if got := rec.Body.String(); got != "token-que-ya-tenia" {
		t.Fatalf("el token del contexto = %q, want el que ya traía la cookie", got)
	}
	if sc := rec.Header().Values("Set-Cookie"); len(sc) != 0 {
		t.Fatalf("no debe re-sembrarse la cookie existente, got %v", sc)
	}
}

// TestCSRF_FallaCerradoSinEntropia: sin entropía no se puede fabricar el token,
// y sin token no se sirve el formulario.
func TestCSRF_FallaCerradoSinEntropia(t *testing.T) {
	t.Parallel()

	r := routerCSRF(web.CSRFOptions{CookieName: cookieDePrueba, Rand: sinEntropia{}})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("sin entropía el GET debe dar 500, got %d", rec.Code)
	}
	if sc := rec.Header().Values("Set-Cookie"); len(sc) != 0 {
		t.Errorf("no debe sembrarse ninguna cookie, got %v", sc)
	}
}
