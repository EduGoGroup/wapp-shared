package webgin_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/EduGoGroup/wapp-shared/web"
	webgin "github.com/EduGoGroup/wapp-shared/web/gin"
)

// TestRateLimit_429ConRetryAfter: al agotar la ráfaga se corta con 429 y se dice
// cuándo volver.
func TestRateLimit_429ConRetryAfter(t *testing.T) {
	t.Parallel()

	limiter := web.NewKeyedRateLimiter(web.RateLimiterOptions{RPS: 0.0001, Burst: 2})
	r := nuevoEngine()
	r.Use(webgin.RateLimit(limiter))
	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	pide := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		req.RemoteAddr = "203.0.113.7:12345"
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	if got := pide().Code; got != http.StatusOK {
		t.Fatalf("1ª petición = %d, want 200", got)
	}
	if got := pide().Code; got != http.StatusOK {
		t.Fatalf("2ª petición = %d, want 200", got)
	}
	tercera := pide()
	if tercera.Code != http.StatusTooManyRequests {
		t.Fatalf("3ª petición = %d, want 429", tercera.Code)
	}
	if tercera.Header().Get("Retry-After") == "" {
		t.Error("el 429 debe incluir Retry-After")
	}
}

// TestRateLimitKey_UsaElUserIDCuandoHaySesion fija D-2 desde el lado del
// adaptador: la clave de un usuario lleva prefijo y no se confunde con una IP.
func TestRateLimitKey_UsaElUserIDCuandoHaySesion(t *testing.T) {
	t.Parallel()

	r := nuevoEngine()
	r.GET("/con", func(c *gin.Context) {
		c.Set(webgin.ContextUserID, "u-42")
		c.String(http.StatusOK, webgin.RateLimitKey(c))
	})
	r.GET("/sin", func(c *gin.Context) { c.String(http.StatusOK, webgin.RateLimitKey(c)) })

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/con", nil))
	if got := rec.Body.String(); got != "u:u-42" {
		t.Errorf("con sesión la clave = %q, want u:u-42", got)
	}

	req := httptest.NewRequest(http.MethodGet, "/sin", nil)
	req.RemoteAddr = "198.51.100.9:1111"
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req)
	if got := rec2.Body.String(); !strings.HasPrefix(got, "ip:") {
		t.Errorf("sin sesión la clave = %q, want prefijo ip:", got)
	}
}

// TestRequestDeadline_AcotaLaCadena.
func TestRequestDeadline_AcotaLaCadena(t *testing.T) {
	t.Parallel()

	r := nuevoEngine()
	r.Use(webgin.RequestDeadline(50 * time.Millisecond))
	r.GET("/", func(c *gin.Context) {
		plazo, ok := c.Request.Context().Deadline()
		if !ok {
			c.String(http.StatusInternalServerError, "sin deadline")
			return
		}
		if time.Until(plazo) > 50*time.Millisecond {
			c.String(http.StatusInternalServerError, "deadline demasiado largo")
			return
		}
		c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, cuerpo = %q", rec.Code, rec.Body.String())
	}
}

// TestRequestDeadline_ConCeroEsTransparente.
func TestRequestDeadline_ConCeroEsTransparente(t *testing.T) {
	t.Parallel()

	r := nuevoEngine()
	r.Use(webgin.RequestDeadline(0))
	r.GET("/", func(c *gin.Context) {
		if _, ok := c.Request.Context().Deadline(); ok {
			c.String(http.StatusInternalServerError, "no debía haber deadline")
			return
		}
		c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, cuerpo = %q", rec.Code, rec.Body.String())
	}
}

// TestRequestDeadline_HeredaLaCancelacionDelCliente: el deadline no sustituye al
// contexto de la petición, deriva de él.
func TestRequestDeadline_HeredaLaCancelacionDelCliente(t *testing.T) {
	t.Parallel()

	r := nuevoEngine()
	r.Use(webgin.RequestDeadline(time.Hour))
	r.GET("/", func(c *gin.Context) {
		select {
		case <-c.Request.Context().Done():
			c.String(http.StatusOK, "cancelado")
		case <-time.After(time.Second):
			c.String(http.StatusInternalServerError, "no heredó la cancelación")
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	cancel()
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Body.String() != "cancelado" {
		t.Fatalf("cuerpo = %q", rec.Body.String())
	}
}

// TestBodyLimit_SoloLasRutasDeclaradas y el 413 por Content-Length.
func TestBodyLimit_SoloLasRutasDeclaradas(t *testing.T) {
	t.Parallel()

	r := nuevoEngine()
	r.Use(webgin.BodyLimit(10, "/import"))
	r.POST("/import", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.POST("/otra", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	grande := strings.Repeat("x", 100)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/import", strings.NewReader(grande)))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("la ruta acotada debía dar 413, got %d", rec.Code)
	}

	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, httptest.NewRequest(http.MethodPost, "/otra", strings.NewReader(grande)))
	if rec2.Code != http.StatusOK {
		t.Fatalf("una ruta no declarada no se acota, got %d", rec2.Code)
	}

	rec3 := httptest.NewRecorder()
	r.ServeHTTP(rec3, httptest.NewRequest(http.MethodPost, "/import", strings.NewReader("corto")))
	if rec3.Code != http.StatusOK {
		t.Fatalf("un cuerpo dentro del tope debe pasar, got %d", rec3.Code)
	}
}

// TestSlogLogger_NoRompeLaCadena.
func TestSlogLogger_NoRompeLaCadena(t *testing.T) {
	t.Parallel()

	r := nuevoEngine()
	r.Use(webgin.SlogLogger())
	r.GET("/", func(c *gin.Context) { c.String(http.StatusTeapot, "té") })

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusTeapot {
		t.Fatalf("status = %d", rec.Code)
	}
}

// TestSetTrustedProxies_SinListaIgnoraElXForwardedFor es lo que hace que la
// clave del rate-limit no se pueda falsear con una cabecera.
func TestSetTrustedProxies_SinListaIgnoraElXForwardedFor(t *testing.T) {
	t.Parallel()

	r := nuevoEngine()
	if err := webgin.SetTrustedProxies(r, ""); err != nil {
		t.Fatalf("SetTrustedProxies: %v", err)
	}
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, c.ClientIP()) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.7:1234"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if got := rec.Body.String(); got != "203.0.113.7" {
		t.Fatalf("ClientIP = %q: sin proxies de confianza debe ignorarse X-Forwarded-For", got)
	}

	if err := webgin.SetTrustedProxies(r, "esto-no-es-una-ip"); err == nil {
		t.Error("una allowlist malformada debe devolver error (fail-closed en el arranque)")
	}
}

// TestCookiesDeSesion_IdaYVuelta.
func TestCookiesDeSesion_IdaYVuelta(t *testing.T) {
	t.Parallel()

	opts := web.SessionCookieOptions{Name: "wapp_test_session", Secure: true, SameSite: "lax"}

	r := nuevoEngine()
	r.GET("/set", func(c *gin.Context) {
		webgin.SetSessionCookie(c, opts, "valor-de-sesion", 3600)
		c.Status(http.StatusOK)
	})
	r.GET("/leer", func(c *gin.Context) { c.String(http.StatusOK, webgin.SessionCookieValue(c, opts)) })
	r.GET("/borrar", func(c *gin.Context) {
		webgin.ClearSessionCookie(c, opts)
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/set", nil))
	sc := rec.Header().Get("Set-Cookie")
	if !strings.Contains(sc, "wapp_test_session=valor-de-sesion") || !strings.Contains(sc, "HttpOnly") {
		t.Fatalf("Set-Cookie = %q", sc)
	}

	req := httptest.NewRequest(http.MethodGet, "/leer", nil)
	conCookie(req, opts.Name, "valor-de-sesion")
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req)
	if got := rec2.Body.String(); got != "valor-de-sesion" {
		t.Fatalf("lectura = %q", got)
	}

	rec3 := httptest.NewRecorder()
	r.ServeHTTP(rec3, httptest.NewRequest(http.MethodGet, "/borrar", nil))
	if got := rec3.Header().Get("Set-Cookie"); !strings.Contains(got, "Max-Age=0") {
		t.Fatalf("el borrado debe caducar la cookie, got %q", got)
	}

	vacio := httptest.NewRecorder()
	r.ServeHTTP(vacio, httptest.NewRequest(http.MethodGet, "/leer", nil))
	if got := vacio.Body.String(); got != "" {
		t.Fatalf("sin cookie la lectura debe ser vacía, got %q", got)
	}
}
