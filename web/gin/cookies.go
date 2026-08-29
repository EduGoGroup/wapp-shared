package webgin

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/EduGoGroup/wapp-shared/web"
)

// SetSessionCookie escribe la cookie de sesión (HttpOnly) con la vida indicada
// en segundos.
func SetSessionCookie(c *gin.Context, opts web.SessionCookieOptions, value string, maxAgeSeconds int) {
	http.SetCookie(c.Writer, web.SessionCookie(opts, value, maxAgeSeconds))
}

// ClearSessionCookie borra la cookie de sesión.
func ClearSessionCookie(c *gin.Context, opts web.SessionCookieOptions) {
	SetSessionCookie(c, opts, "", -1)
}

// SessionCookieValue lee el valor crudo de la cookie de sesión (cadena vacía si
// no viene).
func SessionCookieValue(c *gin.Context, opts web.SessionCookieOptions) string {
	v, err := c.Cookie(opts.WithDefaults().Name)
	if err != nil {
		return ""
	}
	return v
}

// SetOneTimeCookie escribe la cookie efímera de un solo uso. Va en la respuesta
// del REDIRECT (303), no en la de la pantalla: es lo que hace que el secreto
// llegue al GET siguiente sin pasar por la URL.
func SetOneTimeCookie(c *gin.Context, opts web.OneTimeCookieOptions, value string) {
	http.SetCookie(c.Writer, web.OneTimeCookie(opts, value))
}

// TakeOneTimeCookie lee la cookie efímera y la BORRA en la misma respuesta.
// Devuelve "" si no venía.
//
// Leer y borrar es UN SOLO GESTO a propósito: si fueran dos llamadas, cualquier
// camino de salida temprano del handler (un error al releer datos, un redirect)
// se dejaría la cookie puesta y el secreto seguiría vivo en el navegador hasta
// que venciera el MaxAge. Por eso el borrado se emite incluso cuando no había
// nada que leer: no hay ninguna rama en la que se pueda olvidar.
//
// Consecuencia buscada: un F5 sobre la pantalla ya no encuentra la cookie —y,
// como se llegó por un GET, tampoco hay ningún POST que el navegador reenvíe—,
// así que la pantalla no se puede repetir ni el secreto se puede reemitir sin
// que el usuario lo pida otra vez.
func TakeOneTimeCookie(c *gin.Context, opts web.OneTimeCookieOptions) string {
	opts = opts.WithDefaults()
	value, err := c.Cookie(opts.Name)
	http.SetCookie(c.Writer, web.ClearOneTimeCookie(opts))
	if err != nil {
		return ""
	}
	return value
}
