package webgin

import (
	"github.com/gin-gonic/gin"
)

// DefaultLayout es la plantilla maestra que se ejecuta si no se indica otra.
const DefaultLayout = "base.html"

// Claves que Renderer añade al data de TODA página. Que las ponga el
// renderizador y no cada handler es justo el punto: la consola que no lo tenía
// repetía el nonce a mano en una decena de sitios, y ese es exactamente el tipo
// de repetición que un día se olvida en una pantalla nueva y la deja sin CSP
// utilizable.
const (
	// TemplateKeyNonce es el nonce CSP de la petición.
	TemplateKeyNonce = "Nonce"
	// TemplateKeyCSRFToken es el token que va en el <input hidden> del formulario.
	TemplateKeyCSRFToken = "CSRFToken"
	// TemplateKeyCurrentPath permite a la navegación resaltar la sección activa.
	TemplateKeyCurrentPath = "CurrentPath"
	// TemplateKeyIsAuthenticated dice si hay sesión.
	TemplateKeyIsAuthenticated = "IsAuthenticated"
	// TemplateKeyContentTemplate es el fragmento que el layout maestro ejecuta.
	TemplateKeyContentTemplate = "ContentTemplate"
)

// Renderer pinta páginas sobre un layout maestro, sembrando en cada una el
// nonce, el token CSRF y el estado de sesión.
type Renderer struct {
	layout string
}

// NewRenderer crea el renderizador. layout vacío cae a DefaultLayout.
func NewRenderer(layout string) *Renderer {
	if layout == "" {
		layout = DefaultLayout
	}
	return &Renderer{layout: layout}
}

// HTML renderiza contentTemplate dentro del layout maestro con el status dado.
// data nil es válido (página sin datos propios).
func (r *Renderer) HTML(c *gin.Context, status int, contentTemplate string, data gin.H) {
	if data == nil {
		data = gin.H{}
	}
	data[TemplateKeyCurrentPath] = c.Request.URL.Path
	data[TemplateKeyIsAuthenticated] = IsAuthenticated(c)
	data[TemplateKeyContentTemplate] = contentTemplate
	data[TemplateKeyNonce] = NonceFromContext(c)
	data[TemplateKeyCSRFToken] = CSRFTokenFromContext(c)
	c.HTML(status, r.layout, data)
}
