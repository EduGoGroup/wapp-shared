package web

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// SecurityOptions es lo MÍNIMO que necesitan las cabeceras de seguridad. No
// recibe la config del repo consumidor a propósito: aquí solo entra lo que esta
// capa usa.
type SecurityOptions struct {
	// HSTS emite Strict-Transport-Security. Solo tiene sentido tras TLS:
	// mandarlo sobre http:// en local no aporta nada y ensucia el navegador del
	// desarrollador.
	HSTS bool
	// Rand es la fuente de entropía del nonce. nil significa crypto/rand.Reader;
	// los tests inyectan una que falla para verificar el fail-closed.
	Rand io.Reader
}

// BuildCSP arma la Content-Security-Policy con el nonce de la petición.
//
// Todo es mismo-origen ('self'); el 'nonce-...' autoriza solo los bloques inline
// propios y JAMÁS se emite '*' ni una CDN de terceros. No hay 'unsafe-inline':
// un estilo o script inyectado (que no lleva el nonce de ESTA respuesta) no
// ejecuta.
//
// La política es la UNIÓN de las dos consolas que existían antes del módulo:
// `font-src 'self'` y `object-src 'none'` venían del BFF; `base-uri 'none'`
// —más estricto que el `'self'` del BFF— venía de la consola de plataforma.
func BuildCSP(nonce string) string {
	return strings.Join(CSPDirectives(nonce), "; ")
}

// CSPDirectives devuelve las directivas de la CSP por separado, en el orden en
// que se sirven. Útil para tests y para quien quiera inspeccionar la política
// sin parsear la cadena.
func CSPDirectives(nonce string) []string {
	return []string{
		"default-src 'self'",
		fmt.Sprintf("script-src 'self' 'nonce-%s'", nonce),
		fmt.Sprintf("style-src 'self' 'nonce-%s'", nonce),
		"font-src 'self'",
		"img-src 'self' data:", // los iconos/ilustraciones SVG inline viajan como data:.
		"connect-src 'self'",
		"base-uri 'none'",
		"form-action 'self'",
		"frame-ancestors 'none'", // refuerza X-Frame-Options: DENY.
		"object-src 'none'",
	}
}

// ApplySecurityHeaders escribe en h las cabeceras de seguridad de una web
// pública, con la CSP ya armada para el nonce dado.
//
// Permissions-Policy apaga las APIs potentes del navegador que ninguna consola
// de wApp usa. HSTS solo se emite si opts.HSTS; `preload` se omite a propósito
// (es una decisión de despliegue, no de la aplicación).
func ApplySecurityHeaders(h http.Header, nonce string, opts SecurityOptions) {
	h.Set("Content-Security-Policy", BuildCSP(nonce))
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

	if opts.HSTS {
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	}
}
