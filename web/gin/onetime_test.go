package webgin_test

import (
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/EduGoGroup/wapp-shared/web"
	webgin "github.com/EduGoGroup/wapp-shared/web/gin"
)

const (
	rutaPantalla      = "/pantalla"
	codigoDeUnSoloUso = "ACT-EMITIDO-UNICO"
)

// servidorDeSecreto monta el POST-Redirect-GET completo: el POST emite el secreto
// en la cookie efímera y redirige (303); el GET destino la CONSUME; y hay una
// tercera ruta, ajena, para comprobar el alcance del Path.
func servidorDeSecreto(t *testing.T) (*httptest.Server, web.OneTimeCookieOptions) {
	t.Helper()

	opts := web.OneTimeCookieOptions{
		Name:     "wapp_secreto",
		Path:     rutaPantalla,
		SameSite: "lax",
	}

	r := gin.New()
	r.POST("/emitir", func(c *gin.Context) {
		valor, err := web.EncodeCookiePayload(map[string]string{"c": codigoDeUnSoloUso})
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		webgin.SetOneTimeCookie(c, opts, valor)
		c.Redirect(http.StatusSeeOther, rutaPantalla)
	})
	r.GET(rutaPantalla, func(c *gin.Context) {
		crudo := webgin.TakeOneTimeCookie(c, opts)
		if crudo == "" {
			c.String(http.StatusOK, "SIN-SECRETO")
			return
		}
		var carga map[string]string
		if err := web.DecodeCookiePayload(crudo, &carga); err != nil {
			c.String(http.StatusOK, "ILEGIBLE")
			return
		}
		c.String(http.StatusOK, "SECRETO=%s", carga["c"])
	})
	r.GET("/otra-pantalla", func(c *gin.Context) {
		c.String(http.StatusOK, "COOKIES=%q", c.Request.Header.Get("Cookie"))
	})

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv, opts
}

// navegador es un cliente con TARRO DE COOKIES real y sin seguir redirects, para
// poder mirar el 303 por dentro. El tarro es lo que hace honesto el test: honra
// el MaxAge negativo igual que un navegador, así que "el GET la borra" no se
// afirma leyendo una cabecera, se DEMUESTRA en el estado del cliente.
func navegador(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	return &http.Client{
		Jar:           jar,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func parsear(t *testing.T, crudo string) *url.URL {
	t.Helper()
	u, err := url.Parse(crudo)
	if err != nil {
		t.Fatalf("url.Parse(%q): %v", crudo, err)
	}
	return u
}

// comprobarCookieDelRedirect saca la cookie efímera del 303 y fija sus dos propiedades: HttpOnly
// siempre, y acotada a la pantalla destino.
func comprobarCookieDelRedirect(t *testing.T, resp *http.Response, nombre string) {
	t.Helper()

	var puesta *http.Cookie
	for _, ck := range resp.Cookies() {
		if ck.Name == nombre {
			puesta = ck
		}
	}
	if puesta == nil {
		t.Fatal("el 303 no trae la cookie efímera: el valor no llegaría al GET")
	}
	if !puesta.HttpOnly {
		t.Error("la cookie efímera debe ser HttpOnly")
	}
	if puesta.Path != rutaPantalla {
		t.Errorf("Path = %q, want %q: sin acotar, el valor viaja en todas las peticiones del sitio", puesta.Path, rutaPantalla)
	}
}

func cuerpo(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	var sb strings.Builder
	buf := make([]byte, 512)
	for {
		n, err := resp.Body.Read(buf)
		sb.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return sb.String()
}

// TestOneTimeCookie_ElGETLaConsumeYElF5NoLaVe es el criterio entero del helper,
// con un tarro de cookies de verdad:
//
//	POST -> 303 + Set-Cookie (HttpOnly, acotada a la pantalla)
//	GET  -> ve el secreto Y emite el borrado
//	GET  -> (el F5) ya no lo ve
func TestOneTimeCookie_ElGETLaConsumeYElF5NoLaVe(t *testing.T) {
	t.Parallel()

	srv, opts := servidorDeSecreto(t)
	cliente := navegador(t)

	resp, err := cliente.PostForm(srv.URL+"/emitir", url.Values{})
	if err != nil {
		t.Fatalf("POST /emitir: %v", err)
	}
	_ = cuerpo(t, resp)

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST /emitir = %d, want 303 (el PRG entero depende de esto)", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != rutaPantalla {
		t.Fatalf("Location = %q, want %q", loc, rutaPantalla)
	}

	comprobarCookieDelRedirect(t, resp, opts.Name)

	// 🔴 El secreto NO puede haber pasado por la URL: ni en el Location ni en
	// ninguna parte de la petición. Es la única fuga que esta cookie cierra.
	if strings.Contains(resp.Header.Get("Location"), codigoDeUnSoloUso) {
		t.Fatal("el secreto viaja en la URL del redirect: acabaría en el log de acceso y en el Referer")
	}

	// 1.er GET: el navegador manda la cookie y la pantalla muestra el secreto.
	primero, err := cliente.Get(srv.URL + rutaPantalla)
	if err != nil {
		t.Fatalf("GET %s: %v", rutaPantalla, err)
	}
	if got := cuerpo(t, primero); got != "SECRETO="+codigoDeUnSoloUso {
		t.Fatalf("primer GET = %q, want %q", got, "SECRETO="+codigoDeUnSoloUso)
	}

	// El tarro ya no la tiene: el GET emitió el borrado en la MISMA respuesta.
	u := parsear(t, srv.URL+rutaPantalla)
	for _, ck := range cliente.Jar.Cookies(u) {
		if ck.Name == opts.Name {
			t.Fatalf("la cookie %q sigue en el navegador tras consumirla", opts.Name)
		}
	}

	// 2.º GET: el F5. Ya no hay secreto que mostrar, y no hay POST que reenviar.
	segundo, err := cliente.Get(srv.URL + rutaPantalla)
	if err != nil {
		t.Fatalf("segundo GET %s: %v", rutaPantalla, err)
	}
	got := cuerpo(t, segundo)
	if got != "SIN-SECRETO" {
		t.Fatalf("segundo GET = %q: el secreto debe ser ILEGIBLE en una recarga", got)
	}
	if strings.Contains(got, codigoDeUnSoloUso) {
		t.Fatal("el secreto reaparece en la recarga")
	}
}

// TestOneTimeCookie_NoViajaFueraDeSuPantalla comprueba el Path acotado: con el
// secreto aún sin consumir, una petición a otra ruta del MISMO sitio no lo lleva.
func TestOneTimeCookie_NoViajaFueraDeSuPantalla(t *testing.T) {
	t.Parallel()

	srv, opts := servidorDeSecreto(t)
	cliente := navegador(t)

	resp, err := cliente.PostForm(srv.URL+"/emitir", url.Values{})
	if err != nil {
		t.Fatalf("POST /emitir: %v", err)
	}
	_ = cuerpo(t, resp)

	// Control POSITIVO antes del assert negativo: sin esto, un test que nunca
	// llegara a poner la cookie pasaría igual y no estaría vigilando nada.
	pantalla := parsear(t, srv.URL+rutaPantalla)
	var enSuPantalla bool
	for _, ck := range cliente.Jar.Cookies(pantalla) {
		if ck.Name == opts.Name {
			enSuPantalla = true
		}
	}
	if !enSuPantalla {
		t.Fatal("la cookie ni siquiera llegó a su propia pantalla: el assert de abajo sería vacuo")
	}

	otra, err := cliente.Get(srv.URL + "/otra-pantalla")
	if err != nil {
		t.Fatalf("GET /otra-pantalla: %v", err)
	}
	if got := cuerpo(t, otra); strings.Contains(got, "wapp_secreto") {
		t.Fatalf("la cookie del secreto viajó a otra pantalla: %s", got)
	}
}

// TestTakeOneTimeCookie_BorraAunqueNoHubieraNada: el borrado no cuelga de una
// rama. Si solo se emitiera cuando hay valor, cualquier salida temprana del
// handler dejaría el secreto vivo en el navegador hasta el MaxAge.
func TestTakeOneTimeCookie_BorraAunqueNoHubieraNada(t *testing.T) {
	t.Parallel()

	r := gin.New()
	opts := web.OneTimeCookieOptions{Name: "wapp_secreto", Path: rutaPantalla}
	r.GET(rutaPantalla, func(c *gin.Context) {
		if v := webgin.TakeOneTimeCookie(c, opts); v != "" {
			t.Errorf("sin cookie en la petición, TakeOneTimeCookie devolvió %q", v)
		}
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, rutaPantalla, nil))

	var borrado *http.Cookie
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == opts.Name {
			borrado = ck
		}
	}
	if borrado == nil {
		t.Fatal("no se emitió el borrado cuando no venía cookie: el gesto tiene una rama que se puede olvidar")
	}
	if borrado.MaxAge >= 0 {
		t.Fatalf("MaxAge del borrado = %d, debe ser negativo", borrado.MaxAge)
	}
}
