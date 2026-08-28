package iam

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultTimeout acota cada llamada al plano de identidad cuando [Options] no trae una propia.
//
// Un timeout cero en el http.Client significa «sin límite», no «el que venga»: por eso Timeout<=0
// cae aquí en vez de dejar el cliente colgado ante un upstream que nunca responde.
const DefaultTimeout = 15 * time.Second

// maxResponseBody acota lo que se decodifica de una respuesta. Un par de tokens son unos pocos KB;
// el tope está para que un upstream roto o malicioso no pueda hacer crecer la memoria del llamante.
const maxResponseBody = 1 << 20

// Options son los datos con los que se construye un [Client]. Es un struct estrecho a propósito: el
// módulo no conoce los tipos de configuración de ningún repo, solo lo que necesita para el cable.
type Options struct {
	// System es la clave de la aplicación LLAMANTE en el catálogo de identity (`iam.systems`), en
	// formato namespaced `<ecosistema>.<app>`: "wapp.bff", "wapp.platform". Es un identificador de
	// contrato, no infraestructura —identity conoce la aplicación con el mismo nombre en todos sus
	// ambientes—, pero es del llamante, no de este módulo: aquí no hay ninguna rama por su valor.
	System string
	// IdentityBaseURL es la raíz de identity-api, el único emisor de Identity Tokens del grupo.
	IdentityBaseURL string
	// PlatformBaseURL es la raíz de la plataforma de wApp, que es quien verifica el Identity Token y
	// emite el Context Token. Son dos destinos distintos: dos servicios, dos URLs, dos contratos.
	PlatformBaseURL string
	// Timeout acota cada llamada. Timeout<=0 cae a [DefaultTimeout]. Se ignora si HTTPClient viene
	// puesto: entonces el plazo es el de ese cliente.
	Timeout time.Duration
	// HTTPClient permite inyectar un http.Client propio (TLS, proxy, instrumentación). Si es nil se
	// construye uno con Timeout.
	HTTPClient *http.Client
}

// Client habla con los dos servicios del plano de identidad: identity-api para las credenciales y la
// plataforma de wApp para el canje.
//
// Todo lo que sale de aquí son credenciales, así que ninguna respuesta se registra ni se incluye en
// los errores: los tokens y la contraseña no aparecen en los logs de quien lo use.
type Client struct {
	system      string
	identityURL string
	platformURL string
	httpClient  *http.Client
}

// NewClient construye el cliente del plano de identidad y valida las opciones ANTES de la primera
// llamada: un `system` vacío o una URL sin esquema fallan aquí, no dentro de un login.
func NewClient(opts Options) (*Client, error) {
	if strings.TrimSpace(opts.System) == "" {
		return nil, fmt.Errorf("%w: System es obligatorio (la clave de la aplicación en identity)", ErrInvalidOptions)
	}
	identityURL, err := normalizeBaseURL(opts.IdentityBaseURL, "IdentityBaseURL")
	if err != nil {
		return nil, err
	}
	platformURL, err := normalizeBaseURL(opts.PlatformBaseURL, "PlatformBaseURL")
	if err != nil {
		return nil, err
	}
	httpClient := opts.HTTPClient
	if httpClient == nil {
		timeout := opts.Timeout
		if timeout <= 0 {
			timeout = DefaultTimeout
		}
		httpClient = &http.Client{Timeout: timeout}
	}
	return &Client{
		system:      strings.TrimSpace(opts.System),
		identityURL: identityURL,
		platformURL: platformURL,
		httpClient:  httpClient,
	}, nil
}

// System devuelve la clave de la aplicación con la que este cliente se presenta ante identity.
func (c *Client) System() string { return c.system }

// normalizeBaseURL exige una raíz absoluta con esquema y host, y le quita la barra final.
func normalizeBaseURL(raw, field string) (string, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(raw), "/")
	if trimmed == "" {
		return "", fmt.Errorf("%w: %s es obligatorio", ErrInvalidOptions, field)
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("%w: %s no es una URL: %w", ErrInvalidOptions, field, err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", fmt.Errorf("%w: %s debe ser una URL absoluta http(s) con host", ErrInvalidOptions, field)
	}
	return trimmed, nil
}

// post manda un POST JSON y decodifica el 2xx en out (out nil = respuesta sin cuerpo, como el 204
// del logout). Cierra siempre el cuerpo y traduce cualquier no-2xx con [statusError].
//
// El ctx llega hasta http.NewRequestWithContext, así que una cancelación corta la llamada de
// verdad: el timeout del http.Client es el tope, no el único plazo.
func (c *Client) post(ctx context.Context, op, endpoint string, payload, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("iam: %s: serializar la petición: %w", op, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("iam: %s: construir la petición: %w", op, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("iam: %s: %w", op, err)
	}
	defer drainClose(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return statusError(op, resp.StatusCode)
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBody)).Decode(out); err != nil {
		return fmt.Errorf("iam: %s: decodificar la respuesta: %w", op, err)
	}
	return nil
}

// drainClose vacía y cierra el cuerpo para que el transporte pueda reutilizar la conexión.
//
// Los dos fallos posibles se descartan a la vista: un cuerpo que no se vació entero no cambia el
// resultado de la llamada —que ya está decidido— y un Close que falla sobre una conexión que nadie
// va a volver a usar no le añade nada al llamante.
func drainClose(body io.ReadCloser) {
	if _, err := io.Copy(io.Discard, body); err != nil {
		_ = body.Close()
		return
	}
	_ = body.Close()
}
