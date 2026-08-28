package webgin

import (
	"github.com/gin-gonic/gin"

	"github.com/EduGoGroup/wapp-shared/web"
)

// SetTrustedProxies configura la allowlist de proxies de confianza del engine a
// partir del CSV de configuración.
//
// Con la lista vacía Gin no confía en NINGÚN proxy y resuelve ClientIP() desde
// la IP de la conexión, ignorando el X-Forwarded-For. Eso es lo que blinda el
// rate-limit por IP del login contra la suplantación de esa cabecera.
//
// El error se devuelve en vez de tragarse: una allowlist malformada en el
// arranque debe impedir arrancar (fail-closed), porque con ella el ClientIP() ya
// no es de fiar y el rate-limit deja de defender lo que dice defender.
func SetTrustedProxies(engine *gin.Engine, csv string) error {
	return engine.SetTrustedProxies(web.ParseTrustedProxies(csv))
}
