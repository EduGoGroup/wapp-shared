package webgin

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/EduGoGroup/wapp-shared/web"
)

// CORS aplica la política CORS fail-closed: solo refleja un Origin de la
// allowlist, nunca "*". Un preflight de origen NO permitido se corta con 403 en
// vez de con el 204 de cortesía: la respuesta debe decir que ese origen no está
// invitado, no dejarle deducirlo por la ausencia de cabeceras.
func CORS(opts web.CORSOptions) gin.HandlerFunc {
	policy := web.NewCORSPolicy(opts)

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := policy.Apply(c.Writer.Header(), origin)

		if c.Request.Method == http.MethodOptions {
			if origin != "" && !allowed {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
