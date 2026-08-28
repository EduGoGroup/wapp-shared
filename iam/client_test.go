package iam

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestNewClientRechazaOpcionesQueNoPuedenFuncionar: el `system` dejó de ser una constante del
// binario para ser un campo, así que lo que antes no podía estar mal ahora sí. Un `system` vacío no
// falla al construir sino en el login, con un 403 del System Gate que parece un problema de permisos
// del usuario; una URL sin esquema falla dentro de la primera llamada. Las dos se cazan aquí.
func TestNewClientRechazaOpcionesQueNoPuedenFuncionar(t *testing.T) {
	t.Parallel()
	base := Options{System: "wapp.bff", IdentityBaseURL: "http://id", PlatformBaseURL: "http://pf"}
	casos := map[string]func(o *Options){
		"system vacío":               func(o *Options) { o.System = "" },
		"system solo espacios":       func(o *Options) { o.System = "   " },
		"identity vacía":             func(o *Options) { o.IdentityBaseURL = "" },
		"plataforma vacía":           func(o *Options) { o.PlatformBaseURL = "" },
		"identity sin esquema":       func(o *Options) { o.IdentityBaseURL = "127.0.0.1:8200" },
		"plataforma sin esquema":     func(o *Options) { o.PlatformBaseURL = "localhost/api" },
		"esquema que no es http":     func(o *Options) { o.IdentityBaseURL = "ftp://id" },
		"esquema http pero sin host": func(o *Options) { o.PlatformBaseURL = "http://" },
	}
	for name, romper := range casos {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			opts := base
			romper(&opts)
			c, err := NewClient(opts)
			if !errors.Is(err, ErrInvalidOptions) {
				t.Fatalf("NewClient con %s = (%v, %v), want ErrInvalidOptions", name, c, err)
			}
			if c != nil {
				t.Error("un NewClient que falla no debe devolver cliente")
			}
		})
	}
}

// TestNewClientNormalizaLaBarraFinal: una URL base con `/` al final dejaría `//api/v1/auth/login` en
// el cable, y hay servidores que responden 404 a eso.
func TestNewClientNormalizaLaBarraFinal(t *testing.T) {
	t.Parallel()
	srv, captured := newStub(t, http.StatusOK, okIdentityBody("idt", "rt"))

	c := newTestClient(t, "wapp.bff", srv.URL+"/", srv.URL+"/")
	if _, err := c.IdentityLogin(context.Background(), "a@b.com", "x"); err != nil {
		t.Fatalf("el login no debía fallar: %v", err)
	}
	if path, _, _ := captured.last(); path != "/api/v1/auth/login" {
		t.Errorf("ruta = %q, want /api/v1/auth/login (sin barra duplicada)", path)
	}
}

// TestNewClientAplicaElTimeoutConfigurado fija el hallazgo de la consola (CODE-REVIEW-2026-08-15 #4):
// el timeout estaba clavado a 15s sin importar la configuración, así que la variable de entorno que
// lo gobernaba no tenía ningún efecto sobre login/refresh/logout.
func TestNewClientAplicaElTimeoutConfigurado(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(Options{
		System: "wapp.bff", IdentityBaseURL: srv.URL, PlatformBaseURL: srv.URL,
		Timeout: 20 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	inicio := time.Now()
	if err := c.Logout(context.Background(), "rt-1"); err == nil {
		t.Fatal("con timeout de 20ms contra un servidor de 300ms el logout debía fallar")
	}
	if transcurrido := time.Since(inicio); transcurrido > 250*time.Millisecond {
		t.Fatalf("tardó %v: el timeout configurado no se aplicó (habría cortado mucho antes)", transcurrido)
	}
}

// TestNewClientTimeoutCeroCaeAlDefault: un timeout cero en el http.Client significa «sin límite», no
// «el que venga», y dejaría al cliente colgado ante un upstream que no responde nunca.
func TestNewClientTimeoutCeroCaeAlDefault(t *testing.T) {
	t.Parallel()
	for _, timeout := range []time.Duration{0, -time.Second} {
		c, err := NewClient(Options{
			System: "wapp.bff", IdentityBaseURL: "http://id", PlatformBaseURL: "http://pf", Timeout: timeout,
		})
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		if c.httpClient.Timeout != DefaultTimeout {
			t.Errorf("Timeout con %v = %v, want el default %v", timeout, c.httpClient.Timeout, DefaultTimeout)
		}
	}
}

// TestNewClientUsaElHTTPClientInyectado: quien necesite TLS propio, un proxy o instrumentación trae
// su http.Client, y entonces el plazo es el suyo. Construir uno nuevo por debajo tiraría esa
// configuración a la basura sin avisar.
func TestNewClientUsaElHTTPClientInyectado(t *testing.T) {
	t.Parallel()
	propio := &http.Client{Timeout: 3 * time.Second}
	c, err := NewClient(Options{
		System: "wapp.bff", IdentityBaseURL: "http://id", PlatformBaseURL: "http://pf",
		Timeout: time.Minute, HTTPClient: propio,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.httpClient != propio {
		t.Fatal("el http.Client inyectado debía usarse tal cual")
	}
}

// TestTodasLasOperacionesRespetanElContexto: el ctx llega hasta la construcción de la petición, así
// que una cancelación corta la llamada de verdad y no espera al timeout del http.Client. Con el
// contexto ya cancelado ninguna petición debe llegar a salir.
func TestTodasLasOperacionesRespetanElContexto(t *testing.T) {
	t.Parallel()
	srv, captured := newStub(t, http.StatusOK, okIdentityBody("idt", "rt"))
	c := newTestClient(t, "wapp.bff", srv.URL, srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	operaciones := map[string]func() error{
		"IdentityLogin":   func() error { _, err := c.IdentityLogin(ctx, "a@b.com", "x"); return err },
		"IdentityRefresh": func() error { _, err := c.IdentityRefresh(ctx, "rt"); return err },
		"Exchange":        func() error { _, err := c.Exchange(ctx, "idt"); return err },
		"Login":           func() error { _, err := c.Login(ctx, "a@b.com", "x"); return err },
		"Refresh":         func() error { _, err := c.Refresh(ctx, "rt"); return err },
		"Logout":          func() error { return c.Logout(ctx, "rt") },
	}
	for name, op := range operaciones {
		t.Run(name, func(t *testing.T) {
			err := op()
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("%s con el contexto cancelado = %v, want que envuelva context.Canceled", name, err)
			}
		})
	}
	if captured.count() != 0 {
		t.Errorf("con el contexto cancelado no debía salir ninguna petición, salieron %d", captured.count())
	}
}

// TestCuerpoIlegibleEsErrorDeDecodificacion: un 200 que no es JSON no puede pasar por sesión válida.
func TestCuerpoIlegibleEsErrorDeDecodificacion(t *testing.T) {
	t.Parallel()
	srv, _ := newStub(t, http.StatusOK, `{"identity_token": roto`)
	c := newTestClient(t, "wapp.bff", srv.URL, srv.URL)

	if _, err := c.IdentityLogin(context.Background(), "a@b.com", "x"); err == nil {
		t.Fatal("un cuerpo ilegible debía devolver error")
	}
	if _, err := c.Exchange(context.Background(), "idt"); err == nil {
		t.Fatal("un canje con cuerpo ilegible debía devolver error")
	}
}

// TestNingunErrorFiltraLosSecretos recorre TODOS los caminos de error del módulo con credenciales
// marcadas y comprueba que ninguna acaba en el texto del error.
//
// El upstream de este test es hostil a propósito: devuelve DE VUELTA el cuerpo que se le mandó, que
// es el peor caso realista —un emisor que hace eco de la petición en su mensaje de error—. Si algún
// camino incluyera el cuerpo de la respuesta en el error, la contraseña acabaría en el log del
// llamante. Por eso APIError guarda solo la operación y el status.
func TestNingunErrorFiltraLosSecretos(t *testing.T) {
	t.Parallel()
	const (
		email    = "victima@ejemplo.com"
		password = "contrasena-super-secreta-42"
		refresh  = "rt-super-secreto-42"
		identity = "idt-super-secreto-42"
	)
	secretos := []string{password, refresh, identity}

	// eco responde el status pedido repitiendo el cuerpo recibido.
	eco := func(status int) *httptest.Server {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, readErr := io.ReadAll(r.Body)
			if readErr != nil {
				t.Errorf("el eco no pudo leer el cuerpo de la peticion: %v", readErr)
			}
			w.WriteHeader(status)
			if _, writeErr := w.Write(raw); writeErr != nil {
				t.Errorf("el eco no pudo escribir la respuesta: %v", writeErr)
			}
		}))
		t.Cleanup(srv.Close)
		return srv
	}

	caidoURL := func() string {
		srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		url := srv.URL
		srv.Close()
		return url
	}()

	ctxCancelado, cancel := context.WithCancel(context.Background())
	cancel()

	casos := map[string]func() error{}
	for name, status := range map[string]int{"401": 401, "403": 403, "409": 409, "500": 500, "503": 503} {
		srv := eco(status)
		c := newTestClient(t, "wapp.bff", srv.URL, srv.URL)
		casos["login "+name] = func() error { _, err := c.Login(context.Background(), email, password); return err }
		casos["refresh "+name] = func() error { _, err := c.Refresh(context.Background(), refresh); return err }
		casos["logout "+name] = func() error { return c.Logout(context.Background(), refresh) }
		casos["exchange "+name] = func() error { _, err := c.Exchange(context.Background(), identity); return err }
	}
	roto := eco(http.StatusOK) // 200 con eco: el cuerpo NO es la respuesta esperada
	cRoto := newTestClient(t, "wapp.bff", roto.URL, roto.URL)
	casos["login con respuesta que no es la esperada"] = func() error {
		_, err := cRoto.Login(context.Background(), email, password)
		return err
	}
	cCaido := newTestClient(t, "wapp.bff", caidoURL, caidoURL)
	casos["upstream caído"] = func() error { _, err := cCaido.Login(context.Background(), email, password); return err }
	casos["contexto cancelado"] = func() error { _, err := cRoto.Login(ctxCancelado, email, password); return err }

	for name, op := range casos {
		t.Run(name, func(t *testing.T) {
			err := op()
			if err == nil {
				t.Fatalf("el caso %q debía producir un error", name)
			}
			for _, texto := range []string{err.Error(), fmt.Sprintf("%+v", err)} {
				for _, secreto := range secretos {
					if strings.Contains(texto, secreto) {
						t.Errorf("el error de %q filtró un secreto: %q", name, texto)
					}
				}
			}
		})
	}
}
