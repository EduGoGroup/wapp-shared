package webgin

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/EduGoGroup/wapp-shared/web"
)

// CSRF implementa la defensa double-submit sobre los métodos que mutan estado.
//
// El ORDEN dentro del middleware importa: primero VALIDA y solo después siembra.
// Si sembrara antes, un 403 saldría acompañado de un Set-Cookie que el atacante
// puede provocar a voluntad, y el rechazo dejaría de ser un rechazo limpio.
//
// El token existente NO se rota en cada GET: rotarlo invalidaría los formularios
// que el usuario tenga abiertos en otras pestañas.
func CSRF(opts web.CSRFOptions) gin.HandlerFunc {
	opts = opts.WithDefaults()

	return func(c *gin.Context) {
		cookieToken, err := c.Cookie(opts.CookieName)
		if err != nil {
			cookieToken = ""
		}

		if web.IsUnsafeMethod(c.Request.Method) && !web.ValidateCSRF(cookieToken, csrfTokenFromRequest(c)) {
			// Ni el token de la cookie ni el presentado se loguean: uno de los
			// dos es la credencial que estamos defendiendo.
			slog.Warn("petición rechazada por CSRF",
				"method", c.Request.Method, "path", c.Request.URL.Path)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Petición no válida (token de seguridad ausente o incorrecto). Recarga la página e inténtalo de nuevo.",
			})
			return
		}

		token := cookieToken
		if token == "" {
			fresh, gerr := web.NewCSRFToken(opts.Rand)
			if gerr != nil {
				slog.Error("no se pudo generar el token CSRF", "error", gerr)
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			token = fresh
			http.SetCookie(c.Writer, web.CSRFCookie(opts, token))
		}
		c.Set(ContextCSRFToken, token)
		c.Next()
	}
}

// csrfTokenFromRequest extrae el token presentado: primero el campo del
// formulario, luego la cabecera.
func csrfTokenFromRequest(c *gin.Context) string {
	if v := c.PostForm(web.CSRFFieldName); v != "" {
		return v
	}
	return c.GetHeader(web.CSRFHeaderName)
}
