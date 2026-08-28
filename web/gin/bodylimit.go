package webgin

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/EduGoGroup/wapp-shared/web"
)

// BodyLimit acota el cuerpo de las rutas indicadas (las que aceptan archivos).
//
// 🔴 EL ORDEN NO ES NEGOCIABLE: se registra ANTES del CSRF. El CSRF lee el
// formulario para comparar el token y con eso consume el cuerpo entero —a
// memoria hasta MaxMultipartMemory y a disco lo que sobre—, así que un tope
// montado después llegaría cuando el daño ya está hecho.
//
// La defensa es doble porque un solo control no basta: el Content-Length se mira
// primero para poder decir qué pasó (un navegador siempre lo manda en una
// subida), y el MaxBytesReader cubre el caso en que no venga (cuerpo troceado),
// aunque ahí el corte se note como un formulario ilegible.
func BodyLimit(limit int64, paths ...string) gin.HandlerFunc {
	guard := web.NewBodyLimit(limit, paths...)

	return func(c *gin.Context) {
		if !guard.Guards(c.Request.Method, c.Request.URL.Path) {
			c.Next()
			return
		}
		if c.Request.ContentLength > guard.Limit() {
			slog.Warn("petición rechazada por tamaño",
				"path", c.Request.URL.Path, "bytes", c.Request.ContentLength, "limite", guard.Limit())
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "El archivo es demasiado grande para esta pantalla. Sube un documento más pequeño.",
			})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, guard.Limit())
		c.Next()
	}
}
