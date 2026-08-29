package web

import (
	"net/http"
	"time"
)

// DefaultOneTimeCookieName es el nombre que se usa si no se indica otro.
// Deliberadamente genérico: cada consola debe poner el suyo, por la misma razón
// que la cookie de sesión y la de CSRF (ver SessionCookieOptions).
const DefaultOneTimeCookieName = "wapp_once"

// DefaultOneTimeCookieMaxAge es la vida por defecto de la cookie efímera: lo que
// tarda el navegador en seguir el redirect que la puso, con holgura para una red
// lenta. No es una sesión de trabajo: si el GET no llega en este plazo, el valor
// se pierde y hay que repetir la acción a propósito.
const DefaultOneTimeCookieMaxAge = 60 * time.Second

// OneTimeCookieOptions es la política de una cookie EFÍMERA DE UN SOLO USO: la
// que transporta un dato del POST al GET siguiente en un POST-Redirect-GET.
//
// # Por qué existe, y por qué no es un flash
//
// Una pantalla que emite un secreto (un código de enrolamiento, una contraseña
// generada, un token de invitación) tiene que mostrarlo UNA vez tras el redirect.
// Las dos vías que ya hay en este módulo no sirven:
//
//   - FlashCatalog viaja por query string y solo transporta CÓDIGOS de catálogo.
//     Meter ahí el secreto lo pondría en la URL, y una URL acaba en el log de
//     acceso del proxy, en la cabecera Referer de cualquier recurso externo que
//     la página cargue, y en el historial del navegador.
//   - SessionData viaja en la cookie de sesión, que dura horas y va en TODAS las
//     peticiones del sitio. Un secreto de un solo uso no tiene nada que hacer ahí.
//
// # Por qué el valor NO se cifra ni se firma
//
// Es deliberado, y no es una simplificación que haya que “arreglar” después:
//
//   - El destinatario del secreto es EXACTAMENTE quien tiene esta cookie. No hay
//     nadie más a quien proteger de leerla: el navegador que la recibe es el del
//     operador que acaba de pedir el secreto, y el secreto se le va a pintar en
//     la pantalla dos milisegundos después de todas formas.
//   - Cifrar protegería el valor frente al propio cliente, que es justo a quien
//     va dirigido. Firmar detectaría que el cliente lo manipuló, y manipularlo
//     solo le sirve para mostrarse a sí mismo un secreto falso.
//   - Lo que esta cookie compra —lo ÚNICO que compra— es que el secreto no pase
//     por la URL. Eso lo consigue el transporte, no la criptografía.
//
// Añadir una llave aquí sería añadir una llave que custodiar, rotar y desplegar
// en cada consola sin cerrar ninguna fuga. La regla, entonces: por esta cookie
// viaja lo que el usuario ya va a ver en pantalla, nunca material que el servidor
// deba seguir custodiando (una DEK, una clave privada, el refresh token de otro).
//
// # El HttpOnly sí importa
//
// Va SIEMPRE y no es parámetro: aunque el valor se pinte en la página, dejarlo
// legible desde JavaScript amplía la superficie gratis (un XSS lo leería de la
// cookie sin necesidad de raspar el DOM, y con el Path acotado la cookie ni
// siquiera llega a las demás pantallas). Secure y SameSite sí siguen la config,
// igual que en SessionCookieOptions: son política de despliegue de cada consola.
type OneTimeCookieOptions struct {
	// Name es el nombre de la cookie; vacío cae a DefaultOneTimeCookieName.
	Name string
	// Path acota la cookie a la PANTALLA destino del redirect (por ejemplo
	// "/tenants/abc/enrollment-code"), no al sitio entero: fuera de esa ruta el
	// navegador no la envía, así que el secreto no viaja en peticiones que no
	// tienen nada que ver con él.
	//
	// Vacío NO se rellena a "/" a propósito: eso lo ensancharía al sitio entero
	// justo cuando el llamante se olvidó de acotarlo. Vacío significa lo que
	// significa en HTTP —el navegador lo deduce del directorio de la respuesta
	// que la puso—, que ya es más estrecho que la raíz.
	Path string
	// MaxAge es la vida de la cookie; <= 0 cae a DefaultOneTimeCookieMaxAge.
	// Es un TOPE DE SEGURIDAD, no el mecanismo: quien la retira de verdad es el
	// GET que la consume (ver ClearOneTimeCookie).
	MaxAge time.Duration
	// Secure marca la cookie como Secure (solo TLS).
	Secure bool
	// SameSite es el modo en texto ("lax", "strict", "none"); ver SameSiteMode.
	// Lax basta y sobra: el GET que la consume es una navegación de primer nivel
	// del MISMO sitio, disparada por un 303 que emitió este servidor.
	SameSite string
}

// WithDefaults devuelve una copia con los huecos rellenos. Path NO se rellena:
// ver el comentario del campo.
func (o OneTimeCookieOptions) WithDefaults() OneTimeCookieOptions {
	if o.Name == "" {
		o.Name = DefaultOneTimeCookieName
	}
	if o.MaxAge <= 0 {
		o.MaxAge = DefaultOneTimeCookieMaxAge
	}
	return o
}

// Mode traduce el SameSite configurado al valor de net/http. Comparte la
// traducción con la cookie de sesión y la de CSRF: una sola tabla, un solo
// degradado seguro para "none" sin Secure.
func (o OneTimeCookieOptions) Mode() http.SameSite {
	return SameSiteMode(o.SameSite, o.Secure)
}

// MaxAgeSeconds es la vida de la cookie en segundos, como la quiere http.Cookie.
func (o OneTimeCookieOptions) MaxAgeSeconds() int {
	return int(o.WithDefaults().MaxAge.Seconds())
}

// OneTimeCookie arma la cookie efímera que se pone en la respuesta del redirect
// (303). value debe ser seguro para una cookie: base64 URL-safe, que es
// exactamente lo que devuelve EncodeCookiePayload.
func OneTimeCookie(opts OneTimeCookieOptions, value string) *http.Cookie {
	opts = opts.WithDefaults()
	// #nosec G124 -- HttpOnly va fijo (ver el doc de OneTimeCookieOptions);
	// Secure y SameSite son política de despliegue de cada consola.
	return &http.Cookie{
		Name:     opts.Name,
		Value:    value,
		Path:     opts.Path,
		MaxAge:   opts.MaxAgeSeconds(),
		Secure:   opts.Secure,
		HttpOnly: true,
		SameSite: opts.Mode(),
	}
}

// ClearOneTimeCookie arma el borrado de la cookie efímera: MaxAge negativo, el
// mismo gesto que ClearSessionCookie.
//
// El Name y el Path tienen que ser LOS MISMOS con los que se puso: el navegador
// identifica una cookie por la terna (dominio, ruta, nombre), y un borrado con
// otro Path crea una cookie distinta en vez de retirar la que hay. Por eso esta
// función recibe las MISMAS opciones y no un par de cadenas sueltas.
func ClearOneTimeCookie(opts OneTimeCookieOptions) *http.Cookie {
	opts = opts.WithDefaults()
	// #nosec G124 -- mismo razonamiento que OneTimeCookie.
	return &http.Cookie{
		Name:     opts.Name,
		Value:    "",
		Path:     opts.Path,
		MaxAge:   -1,
		Secure:   opts.Secure,
		HttpOnly: true,
		SameSite: opts.Mode(),
	}
}
