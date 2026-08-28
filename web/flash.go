package web

// FlashCatalog traduce CÓDIGOS estables —los que un handler pone en
// `?error=`/`?success=` al redirigir— a mensajes fijos para el usuario.
//
// Es seguridad, no comodidad: el texto que se pinta sale SIEMPRE de esta tabla y
// NUNCA del query string ni del upstream. Un código desconocido (por ejemplo,
// texto arbitrario inyectado en la URL por un tercero) cae al mensaje genérico;
// el detalle real del fallo va al log, identificado por el mismo código.
//
// Los códigos son de cada consola —dependen de sus pantallas—, así que el módulo
// aporta el mecanismo y la invariante, no el vocabulario.
type FlashCatalog struct {
	messages map[string]string
	fallback string
}

// DefaultFlashFallback es el mensaje genérico cuando el código no se reconoce.
const DefaultFlashFallback = "Ocurrió un error inesperado."

// NewFlashCatalog construye el catálogo. fallback vacío cae a
// DefaultFlashFallback. El mapa se copia: el catálogo no cambia bajo los pies de
// quien lo consulta.
func NewFlashCatalog(fallback string, messages map[string]string) *FlashCatalog {
	if fallback == "" {
		fallback = DefaultFlashFallback
	}
	copied := make(map[string]string, len(messages))
	for k, v := range messages {
		copied[k] = v
	}
	return &FlashCatalog{messages: copied, fallback: fallback}
}

// Message traduce un código a su mensaje. El código vacío (no hay flash que
// pintar) devuelve cadena vacía; un código desconocido devuelve el mensaje
// genérico, jamás el texto recibido.
func (c *FlashCatalog) Message(code string) string {
	if code == "" {
		return ""
	}
	if msg, ok := c.messages[code]; ok {
		return msg
	}
	return c.fallback
}

// Known dice si el código está en el catálogo. Útil para que un test de la
// consola verifique que todo código que sus handlers emiten tiene traducción.
func (c *FlashCatalog) Known(code string) bool {
	_, ok := c.messages[code]
	return ok
}
