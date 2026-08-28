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
