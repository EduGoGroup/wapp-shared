package webgin

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/EduGoGroup/wapp-shared/web"
)

// RateLimit limita las peticiones por clave: el user_id de la sesión si lo hay
// (la clave más específica gana), si no la IP del cliente.
//
// El limitador se recibe YA CONSTRUIDO y no se crea aquí: quien lo crea es el
// dueño de su ciclo de vida y quien decide si llama a Close al apagar. Este
// middleware no tiene nada que cerrar ni que filtrar — web.KeyedRateLimiter purga
// en perezoso dentro de Allow y no arranca ninguna goroutine.
//
// Al exceder el límite responde 429 con Retry-After. La clave NO se loguea:
// puede ser un user_id, y el diagnóstico se apaña con método y ruta.
func RateLimit(limiter *web.KeyedRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !limiter.Allow(RateLimitKey(c)) {
			slog.Warn("petición rechazada por rate-limit",
				"method", c.Request.Method, "path", c.Request.URL.Path)
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Demasiadas solicitudes. Espera un momento e inténtalo de nuevo.",
			})
			return
		}
		c.Next()
	}
}

// RateLimitKey es la clave de limitación de esta petición.
func RateLimitKey(c *gin.Context) string {
	return web.RateKey(UserIDFromContext(c), c.ClientIP())
}
