package webgin

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestDeadline instala en c.Request un contexto con deadline acotado.
//
// Como los handlers relayan al upstream con c.Request.Context(), ese deadline
// acota de una sola vez TODA la cadena —incluida la secuencia intento → refresh
// → reintento—, de modo que un upstream lento no cuelgue el handler más allá del
// presupuesto: al vencer, las llamadas devuelven context.DeadlineExceeded y el
// handler cae a su modo degradado dentro del WriteTimeout del servidor.
//
// El deadline HEREDA del contexto de la petición, así que sigue cancelándose
// también si el cliente se desconecta o el servidor se apaga. Con d <= 0 el
// middleware es transparente (sin tope).
func RequestDeadline(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if d <= 0 {
			c.Next()
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
