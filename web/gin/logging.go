package webgin

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// SlogLogger manda cada petición HTTP a slog. Las respuestas con error suben a
// WARN y llevan además la IP: una consola que ejecuta acciones sensibles
// necesita la traza mínima de quién, desde dónde y con qué resultado.
//
// La ruta se captura ANTES de c.Next() a propósito: un handler puede reescribir
// c.Request.URL, y lo que interesa es lo que pidió el cliente.
func SlogLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		if status >= 400 {
			slog.Warn("petición web con error",
				"status", status, "method", c.Request.Method, "path", path,
				"latency", latency, "ip", c.ClientIP())
			return
		}
		slog.Info("petición web completada",
			"status", status, "method", c.Request.Method, "path", path, "latency", latency)
	}
}
