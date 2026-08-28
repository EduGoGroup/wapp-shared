package web

import (
	"crypto/subtle"
	"io"
	"net/http"
	"time"
)

// Nombres del patrón double-submit: el mismo valor viaja en una cookie propia y,
// en cada formulario mutante, en un campo oculto (o en la cabecera). El servidor
// compara ambos en los métodos que mutan estado.
const (
	// CSRFFieldName es el nombre del <input hidden> que incrustan las plantillas.
	CSRFFieldName = "csrf_token"
	// CSRFHeaderName es la alternativa por cabecera (el primario es el campo).
	CSRFHeaderName = "X-CSRF-Token"
)

// DefaultCSRFMaxAge es la vida por defecto de la cookie CSRF (12 h). Cubre de
// sobra una sesión de trabajo sin rotar el token en cada GET: rotarlo
// invalidaría los formularios ya abiertos en otras pestañas.
const DefaultCSRFMaxAge = 12 * time.Hour

// CSRFOptions es lo que necesita la defensa CSRF. El nombre de la cookie es
// PARÁMETRO y no una constante del paquete: si no, las consolas del ecosistema
// —que se sirven en hosts distintos pero comparten este código— acabarían
// pisándose la cookie entre ellas.
type CSRFOptions struct {
	// CookieName es obligatorio en la práctica; vacío cae a DefaultCSRFCookieName.
	CookieName string
	// MaxAge es la vida de la cookie; 0 cae a DefaultCSRFMaxAge.
	MaxAge time.Duration
	// Secure marca la cookie como Secure (solo TLS). Fuera de local va a true.
	Secure bool
	// Rand es la fuente de entropía del token. nil significa crypto/rand.Reader.
	Rand io.Reader
}

// DefaultCSRFCookieName es el nombre de cookie que se usa si no se indica otro.
// Deliberadamente genérico: cada consola debe poner el suyo.
const DefaultCSRFCookieName = "wapp_csrf"

// WithDefaults devuelve una copia con los huecos rellenos.
func (o CSRFOptions) WithDefaults() CSRFOptions {
	if o.CookieName == "" {
		o.CookieName = DefaultCSRFCookieName
	}
	if o.MaxAge <= 0 {
		o.MaxAge = DefaultCSRFMaxAge
	}
	return o
}

// MaxAgeSeconds es la vida de la cookie en segundos, como la quieren
// http.Cookie y los helpers de los frameworks.
func (o CSRFOptions) MaxAgeSeconds() int {
	return int(o.WithDefaults().MaxAge.Seconds())
}

// IsUnsafeMethod dice si el método HTTP muta estado y por tanto exige validación
// CSRF. GET, HEAD y OPTIONS son seguros (idempotentes, sin efectos).
func IsUnsafeMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

// ValidateCSRF compara el token de la cookie con el presentado, en tiempo
// constante. Cualquiera de los dos vacío es un rechazo: un atacante no puede
// leer la cookie (SOP) ni conoce el token, y con SameSite=Lax la cookie ni
// siquiera se envía en un POST cross-site.
func ValidateCSRF(cookieToken, submitted string) bool {
	if cookieToken == "" || submitted == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(cookieToken), []byte(submitted)) == 1
}

// CSRFCookie arma la cookie del token: HttpOnly (el JS nunca la lee; el token lo
// incrusta el servidor al renderizar) y SameSite=Lax SIEMPRE, con independencia
// de cómo esté configurada la cookie de SESIÓN. El fail-safe CSRF no se degrada
// a None aunque la sesión sí se configure así.
func CSRFCookie(opts CSRFOptions, token string) *http.Cookie {
	opts = opts.WithDefaults()
	// #nosec G124 -- HttpOnly y SameSite=Lax van fijos aquí; Secure NO puede ser
	// un literal true: en local la consola se sirve por http y una cookie Secure
	// no viajaría, así que es decisión de despliegue (opts.Secure).
	return &http.Cookie{
		Name:     opts.CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   opts.MaxAgeSeconds(),
		Secure:   opts.Secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}
