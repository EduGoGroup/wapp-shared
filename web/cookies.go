package web

import (
	"net/http"
	"strings"
	"time"
)

// DefaultSessionMaxAge es la vida de la cookie de sesión cuando el upstream no
// dice cuándo expira (o dice algo que no se puede leer).
const DefaultSessionMaxAge = time.Hour

// SessionCookieOptions es la política de la cookie de SESIÓN. El nombre es
// parámetro por la misma razón que el de la cookie CSRF: dos consolas del
// ecosistema no pueden compartir cookie.
type SessionCookieOptions struct {
	// Name es el nombre de la cookie; vacío cae a DefaultSessionCookieName.
	Name string
	// Secure marca la cookie como Secure (solo TLS).
	Secure bool
	// SameSite es el modo en texto ("lax", "strict", "none"); ver SameSiteMode.
	SameSite string
}

// DefaultSessionCookieName es el nombre que se usa si no se indica otro.
// Deliberadamente genérico: cada consola debe poner el suyo.
const DefaultSessionCookieName = "wapp_session"

// WithDefaults devuelve una copia con los huecos rellenos.
func (o SessionCookieOptions) WithDefaults() SessionCookieOptions {
	if o.Name == "" {
		o.Name = DefaultSessionCookieName
	}
	return o
}

// Mode traduce el SameSite configurado al valor de net/http.
func (o SessionCookieOptions) Mode() http.SameSite {
	return SameSiteMode(o.SameSite, o.Secure)
}

// SameSiteMode traduce el modo SameSite escrito en configuración al de net/http.
//
// "none" SIN Secure degrada a Lax a propósito: un SameSite=None sin Secure lo
// rechazan los navegadores modernos, y quedarse sin ninguna protección SameSite
// es peor que quedarse con Lax. Cualquier valor desconocido cae también a Lax.
func SameSiteMode(mode string, secure bool) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		if secure {
			return http.SameSiteNoneMode
		}
		return http.SameSiteLaxMode
	default:
		return http.SameSiteLaxMode
	}
}

// SessionMaxAge calcula la vida de la cookie, en segundos, a partir del
// expires_at en RFC3339 que devuelve el upstream. Si no se puede leer, o si ya
// pasó, cae a DefaultSessionMaxAge: una cookie con MaxAge negativo se borraría
// sola y dejaría al usuario en un bucle de login.
func SessionMaxAge(expiresAt string) int {
	t, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return int(DefaultSessionMaxAge.Seconds())
	}
	secs := int(time.Until(t).Seconds())
	if secs <= 0 {
		return int(DefaultSessionMaxAge.Seconds())
	}
	return secs
}

// SessionCookie arma la cookie de sesión. Es HttpOnly SIEMPRE: el valor lo lee
// el servidor, nunca el JavaScript de la página. maxAge en segundos; negativo la
// borra (ver ClearSessionCookie en el adaptador).
func SessionCookie(opts SessionCookieOptions, value string, maxAge int) *http.Cookie {
	opts = opts.WithDefaults()
	// #nosec G124 -- HttpOnly va fijo; Secure y SameSite son política de
	// despliegue de cada consola (ver SessionCookieOptions), no del módulo.
	return &http.Cookie{
		Name:     opts.Name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		Secure:   opts.Secure,
		HttpOnly: true,
		SameSite: opts.Mode(),
	}
}
