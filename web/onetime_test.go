package web

import (
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestOneTimeCookie_HttpOnlySiempre(t *testing.T) {
	t.Parallel()

	// HttpOnly no es parámetro: sale true con CUALQUIER combinación de opciones.
	casos := []OneTimeCookieOptions{
		{},
		{Name: "x", Path: "/p", MaxAge: time.Second, Secure: true, SameSite: "strict"},
		{Name: "x", Path: "/p", Secure: false, SameSite: "none"},
	}
	for _, opts := range casos {
		if ck := OneTimeCookie(opts, "v"); !ck.HttpOnly {
			t.Errorf("OneTimeCookie(%+v) debe ser HttpOnly", opts)
		}
		if ck := ClearOneTimeCookie(opts); !ck.HttpOnly {
			t.Errorf("ClearOneTimeCookie(%+v) debe ser HttpOnly", opts)
		}
	}
}

func TestOneTimeCookie_SecureYSameSiteSiguenLaConfig(t *testing.T) {
	t.Parallel()

	ck := OneTimeCookie(OneTimeCookieOptions{Name: "n", Path: "/p", Secure: true, SameSite: "strict"}, "v")
	if !ck.Secure {
		t.Error("Secure debe seguir la config")
	}
	if ck.SameSite != http.SameSiteStrictMode {
		t.Errorf("SameSite = %v, want Strict", ck.SameSite)
	}

	// Misma tabla que la cookie de sesión: "none" sin Secure degrada a Lax en vez
	// de servir un SameSite que el navegador va a rechazar entero.
	ck = OneTimeCookie(OneTimeCookieOptions{Name: "n", Path: "/p", Secure: false, SameSite: "none"}, "v")
	if ck.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax (none sin Secure degrada)", ck.SameSite)
	}
}

// TestOneTimeCookie_PathVacioNoSeEnsanchaARaiz fija la decisión escrita en el
// campo Path: rellenarlo a "/" convertiría un olvido del llamante en una cookie
// con secreto enviada en TODAS las peticiones del sitio. Vacío se deja vacío, que
// en HTTP significa "el directorio de esta respuesta" y ya es más estrecho.
func TestOneTimeCookie_PathVacioNoSeEnsanchaARaiz(t *testing.T) {
	t.Parallel()

	if got := (OneTimeCookieOptions{}).WithDefaults().Path; got != "" {
		t.Fatalf("WithDefaults() inventó Path = %q; un Path por defecto ensancha el alcance del secreto", got)
	}
	if got := OneTimeCookie(OneTimeCookieOptions{}, "v").Path; got != "" {
		t.Fatalf("OneTimeCookie con Path vacío emitió Path = %q, want \"\"", got)
	}
	if got := OneTimeCookie(OneTimeCookieOptions{Path: "/tenants/t-1/enrollment-code"}, "v").Path; got != "/tenants/t-1/enrollment-code" {
		t.Fatalf("Path = %q, debe conservarse tal cual", got)
	}
}

// TestClearOneTimeCookie_MismaTernaQueLaQuePuso: el navegador identifica una
// cookie por (dominio, ruta, nombre). Un borrado con otro Path CREA otra cookie
// en vez de retirar la que hay, y el secreto se quedaría vivo hasta el MaxAge.
func TestClearOneTimeCookie_MismaTernaQueLaQuePuso(t *testing.T) {
	t.Parallel()

	opts := OneTimeCookieOptions{Name: "wapp_secreto", Path: "/pantalla", MaxAge: 30 * time.Second}
	puesta := OneTimeCookie(opts, "valor")
	borrado := ClearOneTimeCookie(opts)

	if borrado.Name != puesta.Name {
		t.Errorf("Name del borrado = %q, want %q", borrado.Name, puesta.Name)
	}
	if borrado.Path != puesta.Path {
		t.Errorf("Path del borrado = %q, want %q", borrado.Path, puesta.Path)
	}
	if borrado.MaxAge >= 0 {
		t.Errorf("MaxAge del borrado = %d, debe ser negativo para retirar la cookie", borrado.MaxAge)
	}
	if borrado.Value != "" {
		t.Errorf("el borrado no debe reenviar el valor: %q", borrado.Value)
	}
}

func TestOneTimeCookie_MaxAgePorDefectoYCorto(t *testing.T) {
	t.Parallel()

	if got := OneTimeCookie(OneTimeCookieOptions{Path: "/p"}, "v").MaxAge; got != int(DefaultOneTimeCookieMaxAge.Seconds()) {
		t.Fatalf("MaxAge por defecto = %d, want %d", got, int(DefaultOneTimeCookieMaxAge.Seconds()))
	}
	// El tope es de "lo que tarda un redirect", no de una sesión de trabajo: si
	// alguien lo sube a horas, esto lo cuenta como cambio consciente.
	if DefaultOneTimeCookieMaxAge > 5*time.Minute {
		t.Fatalf("DefaultOneTimeCookieMaxAge = %v: una cookie efímera con secreto no puede vivir tanto", DefaultOneTimeCookieMaxAge)
	}
	if got := OneTimeCookie(OneTimeCookieOptions{Path: "/p", MaxAge: 5 * time.Second}, "v").MaxAge; got != 5 {
		t.Fatalf("MaxAge = %d, want 5", got)
	}
}

// TestEncodeCookiePayload_AlfabetoSeguroEnCookie: los adaptadores de framework
// (gin.Context.Cookie) aplican url.QueryUnescape al leer, así que un '+' del
// base64 estándar volvería como espacio y el valor llegaría corrupto.
func TestEncodeCookiePayload_AlfabetoSeguroEnCookie(t *testing.T) {
	t.Parallel()

	type carga struct {
		Codigo string `json:"c"`
		Vence  string `json:"e"`
	}
	original := carga{Codigo: "ACT-12345-67890/+=?", Vence: "2026-08-29T00:00:00Z"}

	valor, err := EncodeCookiePayload(original)
	if err != nil {
		t.Fatalf("EncodeCookiePayload: %v", err)
	}
	if strings.ContainsAny(valor, "+/=%") {
		t.Fatalf("el valor %q lleva caracteres que el unescape del framework destroza", valor)
	}

	var vuelta carga
	if err := DecodeCookiePayload(valor, &vuelta); err != nil {
		t.Fatalf("DecodeCookiePayload: %v", err)
	}
	if vuelta != original {
		t.Fatalf("round-trip = %+v, want %+v", vuelta, original)
	}
}

func TestDecodeCookiePayload_ValorIlegible(t *testing.T) {
	t.Parallel()

	var destino struct {
		C string `json:"c"`
	}
	if err := DecodeCookiePayload("no es base64 ###", &destino); err == nil {
		t.Fatal("un valor ilegible debe devolver error, no silencio")
	}
	// base64 válido con contenido que no es el JSON esperado: también falla.
	basura := base64.RawURLEncoding.EncodeToString([]byte("esto no es json"))
	if err := DecodeCookiePayload(basura, &destino); err == nil {
		t.Fatal("un JSON inválido debe devolver error")
	}
}
