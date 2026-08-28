package webgin

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/EduGoGroup/wapp-shared/web"
)

// SecurityHeaders emite las cabeceras de seguridad y siembra un nonce CSP por
// petición, que el renderizador inyecta en los bloques inline de la plantilla.
//
// FALLA CERRADO: si no hay entropía para el nonce, responde 500 y NO emite CSP
// ni sirve la página. Servir con inline sin nonce sería servir sin defensa.
func SecurityHeaders(opts web.SecurityOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		nonce, err := web.Nonce(opts.Rand)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.Set(ContextCSPNonce, nonce)
		web.ApplySecurityHeaders(c.Writer.Header(), nonce, opts)
		c.Next()
	}
}
