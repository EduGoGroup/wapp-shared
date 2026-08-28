package webgin

import (
	"sync"

	"github.com/gin-gonic/gin"
)

// modeOnce asegura que gin.SetMode corra una sola vez por proceso.
var modeOnce sync.Once

// SetReleaseMode pone Gin en modo release. Es idempotente y seguro para
// concurrencia.
//
// Va bajo un sync.Once y no en un func init(): gin.SetMode escribe variables
// GLOBALES del paquete gin sin mutex propio, y aunque en producción el router se
// monta una sola vez y de forma síncrona, los tests SÍ montan routers en
// paralelo. Como el valor que se fija es siempre la misma constante, ejecutarlo
// una única vez es válido en los dos mundos y elimina la carrera de raíz. En un
// init() además se ejecutaría por el mero hecho de importar el paquete, que es
// justo lo que un módulo compartido no debe hacerle al proceso de nadie.
func SetReleaseMode() {
	modeOnce.Do(func() { gin.SetMode(gin.ReleaseMode) })
}
