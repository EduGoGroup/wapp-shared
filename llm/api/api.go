// Package api implementa el puerto llm.LLMProvider contra una API de modelo
// externa, consumida SIEMPRE desde el cloud con las credenciales del tenant
// (ADR-0030 §Decisión 3: la clave del tenant jamás viaja al Edge).
//
// Trae dos proveedores: anthropic, completo, y gemini, un stub que compila y
// falla con un error nombrado para que el día que se implemente no haya que
// tocar el cableado. Cualquier otro valor —«local» incluido— falla al construir,
// no al llamar: si la configuración de un tenant es imposible, se sabe en el
// arranque y no a mitad de un pipeline.
//
// La selección es un switch de arranque sobre Config.Provider y NADA más: no hay
// enrutador por tarea ni registro de vías (D-044.21). Quien recibe el
// llm.LLMProvider no sabe cuál le tocó, y así debe seguir.
//
// La configuración se INYECTA entera: este paquete no lee variables de entorno
// ni ficheros. Quien las lee es el arranque de cloud-platform, que además
// descifra la clave del tenant con el KeyProvider de los Planes 011/012.
package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/EduGoGroup/wapp-shared/llm"
)

// Proveedores soportados. Son los mismos dos valores que admite la columna
// provider de la tabla tenant_llm.
const (
	// ProviderAnthropic es la única vía completa del Plan 044.
	ProviderAnthropic = "anthropic"
	// ProviderGemini existe como stub: se construye, pero sus llamadas fallan.
	ProviderGemini = "gemini"
)

// Valores por defecto de la configuración. Se aplican solo a los campos que
// admiten uno razonable: ni el modelo ni la credencial lo tienen, porque
// inventarlos convertiría un error de configuración en una factura.
const (
	// DefaultTimeout es el tiempo máximo de una llamada al proveedor.
	DefaultTimeout = 60 * time.Second
	// DefaultMaxTokens es el tope de tokens de salida por llamada.
	DefaultMaxTokens = 4096
	// DefaultAnthropicBaseURL es el host público de la API de Anthropic.
	DefaultAnthropicBaseURL = "https://api.anthropic.com"
)

// ErrInvalidConfig indica que la Config no permite construir un provider:
// falta la credencial, falta el modelo o el proveedor no existe. Es un fallo de
// ARRANQUE, nunca de una llamada.
var ErrInvalidConfig = errors.New("llm/api: invalid config")

// ErrUnsupportedProvider indica que Config.Provider no es ninguno de los
// soportados. Envuelve a ErrInvalidConfig: también es un fallo de arranque.
var ErrUnsupportedProvider = fmt.Errorf("%w: unsupported provider", ErrInvalidConfig)

// ErrNotImplemented lo devuelve el stub de gemini en cada una de sus llamadas.
var ErrNotImplemented = errors.New("llm/api: provider not implemented")

// ErrUpstream indica que el fallo es de INFRAESTRUCTURA: el proveedor no
// respondió, tardó de más, devolvió un código de error o devolvió algo que ni
// siquiera es la envoltura que su propia API promete.
//
// Es lo contrario de llm.ErrLLMQuality y por eso son dos centinelas y no uno:
// «la API está caída» se reintenta más tarde con el mismo prompt; «el modelo
// devolvió basura» se reintenta ya, una vez, con otra temperatura. Confundirlos
// hace que un incidente del proveedor se lea como un problema de calidad del
// modelo. Lo custodia TestAnthropic_ErrorDeInfraNoEsErrorDeCalidad.
var ErrUpstream = errors.New("llm/api: upstream provider failure")

// Config es la configuración de un provider de API. Se inyecta entera: el
// provider NUNCA lee el entorno.
type Config struct {
	// Provider es ProviderAnthropic o ProviderGemini.
	Provider string
	// APIKey es la credencial del tenant, ya descifrada por quien construye.
	APIKey string
	// Model es el modelo a usar. Obligatorio y sin valor por defecto.
	Model string
	// BaseURL permite apuntar a otro host (proxy corporativo, servidor de
	// pruebas). Vacío significa el host público del proveedor.
	BaseURL string
	// Timeout es el tope de una llamada. Cero significa DefaultTimeout.
	Timeout time.Duration
	// MaxTokens es el tope de tokens de salida. Cero significa
	// DefaultMaxTokens.
	MaxTokens int
}

// New construye el provider configurado.
//
// Falla en el arranque si la configuración es imposible; en particular, un
// proveedor desconocido (o «local», que en este plan no está cableado) devuelve
// ErrUnsupportedProvider aquí y no un error tardío en la primera llamada.
//
// El proveedor se comprueba ANTES que la credencial y el modelo, y el orden no es
// cosmético: una config de proveedor inexistente falla igual en las dos formas,
// pero solo en este orden el error DICE cuál es el problema. Al revés, un tenant
// con provider «local» y sin clave se diagnosticaba como «falta la credencial», y
// quien leyera ese arranque se pondría a buscar una credencial que no arreglaba
// nada. Lo custodia TestNew_ElProveedorSeDiagnosticaAntesQueLaCredencial.
func New(cfg Config) (llm.LLMProvider, error) {
	if err := checkProvider(cfg.Provider); err != nil {
		return nil, err
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("%w: falta la credencial del proveedor", ErrInvalidConfig)
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("%w: falta el modelo", ErrInvalidConfig)
	}

	if cfg.Provider == ProviderGemini {
		return newGemini(cfg.withDefaults("")), nil
	}
	return newAnthropic(cfg.withDefaults(DefaultAnthropicBaseURL)), nil
}

// checkProvider acepta los dos únicos proveedores soportados. No es un registro
// ni un factory: es la comprobación del switch de arranque, separada solo para
// que el diagnóstico del proveedor llegue antes que el de la credencial.
func checkProvider(provider string) error {
	switch provider {
	case ProviderAnthropic, ProviderGemini:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrUnsupportedProvider, provider)
	}
}

// withDefaults devuelve una copia de la configuración con los valores por
// defecto ya aplicados.
func (c Config) withDefaults(baseURL string) Config {
	out := c
	if out.Timeout <= 0 {
		out.Timeout = DefaultTimeout
	}
	if out.MaxTokens <= 0 {
		out.MaxTokens = DefaultMaxTokens
	}
	if out.BaseURL == "" {
		out.BaseURL = baseURL
	}
	out.BaseURL = trimTrailingSlash(out.BaseURL)
	return out
}

// newHTTPClient arma el cliente con el timeout de la configuración. Es el ÚNICO
// mecanismo de corte por tiempo del provider: no hay reintentos internos, el
// retry vive en el caller (REQ-02).
func newHTTPClient(cfg Config) *http.Client {
	return &http.Client{Timeout: cfg.Timeout}
}

// trimTrailingSlash quita la barra final para que concatenar la ruta no produzca
// una doble barra.
func trimTrailingSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
