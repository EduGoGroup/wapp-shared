package web

import (
	"net/http"
	"strings"
)

// Valores por defecto de la política CORS. Los métodos incluyen OPTIONS porque
// el propio middleware contesta el preflight; las cabeceras admiten el
// Authorization y el X-CSRF-Token que una consola SSR con API auxiliar necesita.
const (
	DefaultCORSMethods = "GET, POST, OPTIONS"
	DefaultCORSHeaders = "Content-Type, Authorization, X-CSRF-Token"
	DefaultCORSMaxAge  = "600"
)

// CORSOptions describe la política CORS. Los orígenes se dan ya como lista (ver
// ParseOrigins si vienen de un CSV de configuración).
type CORSOptions struct {
	// AllowedOrigins es la allowlist EXACTA. Un "*" que se cuele por config se
	// descarta: estas consolas nunca abren wildcard.
	AllowedOrigins []string
	// Methods, Headers y MaxAge caen a los valores por defecto si están vacíos.
	Methods string
	Headers string
	MaxAge  string
}

// CORSPolicy es la política CORS ya resuelta: fail-closed y sin wildcard.
//
// Solo refleja un Origin presente en la allowlist; nunca emite "*" ni hace eco
// de un origen no listado. Con la allowlist vacía no se emite ninguna cabecera
// CORS (postura same-origin: el navegador bloquea el cross-origin por defecto).
// En las consolas de wApp, que son de mismo origen, el CORS es defensa en
// profundidad.
type CORSPolicy struct {
	allowed map[string]bool
	methods string
	headers string
	maxAge  string
}

// NewCORSPolicy construye la política a partir de las opciones, aplicando los
// valores por defecto y descartando orígenes vacíos y el wildcard.
func NewCORSPolicy(opts CORSOptions) *CORSPolicy {
	p := &CORSPolicy{
		allowed: make(map[string]bool, len(opts.AllowedOrigins)),
		methods: firstNonEmpty(opts.Methods, DefaultCORSMethods),
		headers: firstNonEmpty(opts.Headers, DefaultCORSHeaders),
		maxAge:  firstNonEmpty(opts.MaxAge, DefaultCORSMaxAge),
	}
	for _, o := range opts.AllowedOrigins {
		o = strings.TrimSpace(o)
		if o == "" || o == "*" {
			continue
		}
		p.allowed[o] = true
	}
	return p
}

// Allowed dice si el origen está en la allowlist. Un origen vacío nunca lo está.
func (p *CORSPolicy) Allowed(origin string) bool {
	return origin != "" && p.allowed[origin]
}

// Apply escribe las cabeceras CORS en h si el origen está permitido, y devuelve
// si las escribió. El eco es del origen EXACTO, jamás "*".
//
// El `Vary: Origin` se añade siempre (esté o no permitido el origen): la
// respuesta depende de la cabecera Origin en los dos casos, y sin Vary una caché
// intermedia podría servirle a un origen la respuesta calculada para otro.
func (p *CORSPolicy) Apply(h http.Header, origin string) bool {
	AppendVary(h, "Origin")
	if !p.Allowed(origin) {
		return false
	}
	h.Set("Access-Control-Allow-Origin", origin)
	h.Set("Access-Control-Allow-Credentials", "true")
	h.Set("Access-Control-Allow-Methods", p.methods)
	h.Set("Access-Control-Allow-Headers", p.headers)
	h.Set("Access-Control-Max-Age", p.maxAge)
	return true
}

// ParseOrigins convierte un CSV de orígenes en lista, descartando los vacíos y,
// por seguridad, cualquier "*" que se haya colado por configuración.
func ParseOrigins(csv string) []string {
	parts := strings.Split(csv, ",")
	origins := make([]string, 0, len(parts))
	for _, raw := range parts {
		o := strings.TrimSpace(raw)
		if o == "" || o == "*" {
			continue
		}
		origins = append(origins, o)
	}
	return origins
}

// AppendVary agrega un valor a la cabecera Vary sin pisar lo que ya hubiera
// (cacheo correcto del CORS).
func AppendVary(h http.Header, value string) {
	existing := h.Get("Vary")
	switch {
	case existing == "":
		h.Set("Vary", value)
	case strings.Contains(existing, value):
		// ya está.
	default:
		h.Set("Vary", existing+", "+value)
	}
}

// firstNonEmpty devuelve el primer valor no vacío.
func firstNonEmpty(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
