package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/EduGoGroup/wapp-shared/llm"
)

const (
	// anthropicMessagesPath es la ruta de la Messages API.
	anthropicMessagesPath = "/v1/messages"
	// anthropicAPIVersion es el valor fijo del header de versión de la API.
	anthropicAPIVersion = "2023-06-01"
	// headerAnthropicKey es el header que lleva la credencial del tenant.
	headerAnthropicKey = "x-api-key"
	// headerAnthropicVersion es el header de versión de la API.
	headerAnthropicVersion = "anthropic-version"
	// blockTypeText es el único tipo de bloque de contenido que se concatena.
	blockTypeText = "text"
	// maxResponseBytes acota lo que se lee del proveedor: la respuesta esperada
	// son unos pocos KiB de JSON, y un cuerpo sin tope es una vía de agotar
	// memoria por culpa de un tercero.
	maxResponseBytes = 4 << 20
	// maxSnippetRunes acota el trozo de cuerpo que se cita en un error.
	maxSnippetRunes = 256
)

// anthropicProvider habla con la Messages API de Anthropic.
type anthropicProvider struct {
	cfg    Config
	client *http.Client
}

var _ llm.LLMProvider = (*anthropicProvider)(nil)

// newAnthropic construye el provider de Anthropic con la configuración ya
// normalizada.
func newAnthropic(cfg Config) *anthropicProvider {
	return &anthropicProvider{cfg: cfg, client: newHTTPClient(cfg)}
}

func (p *anthropicProvider) ClassifyRequest(ctx context.Context, in llm.ClassifyRequestInput, opts llm.Options) (json.RawMessage, error) {
	return p.run(ctx, llm.BuildClassifyRequestPrompt(in), opts)
}

func (p *anthropicProvider) ExtractMainIdeas(ctx context.Context, in llm.ExtractMainIdeasInput, opts llm.Options) (json.RawMessage, error) {
	return p.run(ctx, llm.BuildExtractMainIdeasPrompt(in), opts)
}

func (p *anthropicProvider) ExtractItemSpecs(ctx context.Context, in llm.ExtractItemSpecsInput, opts llm.Options) (json.RawMessage, error) {
	return p.run(ctx, llm.BuildExtractItemSpecsPrompt(in), opts)
}

func (p *anthropicProvider) NormalizeQuantities(ctx context.Context, in llm.NormalizeQuantitiesInput, opts llm.Options) (json.RawMessage, error) {
	return p.run(ctx, llm.BuildNormalizeQuantitiesPrompt(in), opts)
}

func (p *anthropicProvider) GenerateQuoteText(ctx context.Context, in llm.GenerateQuoteTextInput, opts llm.Options) (json.RawMessage, error) {
	return p.run(ctx, llm.BuildGenerateQuoteTextPrompt(in), opts)
}

// run es el camino común de las cinco tareas: pedir, y aislar el JSON de lo que
// venga. Los dos pasos fallan con centinelas distintos a propósito —ErrUpstream
// el primero, llm.ErrLLMQuality el segundo— y esa es toda la lógica que el
// provider tiene: no reintenta, no decide, no enruta.
func (p *anthropicProvider) run(ctx context.Context, prompt string, opts llm.Options) (json.RawMessage, error) {
	text, err := p.complete(ctx, prompt, opts)
	if err != nil {
		return nil, err
	}
	return llm.ExtractJSON(text)
}

// complete hace UNA llamada a la Messages API y devuelve el texto concatenado de
// los bloques de contenido.
func (p *anthropicProvider) complete(ctx context.Context, prompt string, opts llm.Options) (text string, err error) {
	mensajes := []anthropicMessage{{Role: "user", Content: prompt}}
	payload, err := json.Marshal(anthropicRequest{
		Model:       p.cfg.Model,
		MaxTokens:   p.cfg.MaxTokens,
		Temperature: opts.Temperature,
		Messages:    mensajes,
	})
	if err != nil {
		return "", fmt.Errorf("%w: serializando la petición: %w", ErrUpstream, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.BaseURL+anthropicMessagesPath, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("%w: construyendo la petición: %w", ErrUpstream, err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set(headerAnthropicKey, p.cfg.APIKey)
	req.Header.Set(headerAnthropicVersion, anthropicAPIVersion)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrUpstream, err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("%w: cerrando la respuesta: %w", ErrUpstream, cerr)
		}
	}()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", fmt.Errorf("%w: leyendo la respuesta: %w", ErrUpstream, err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: HTTP %d: %s", ErrUpstream, resp.StatusCode, snippet(body))
	}

	var decoded anthropicResponse
	if uerr := json.Unmarshal(body, &decoded); uerr != nil {
		return "", fmt.Errorf("%w: la respuesta no tiene la forma de la Messages API: %w", ErrUpstream, uerr)
	}
	return decoded.text(), nil
}

// anthropicRequest es el cuerpo de la Messages API.
type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature"`
	Messages    []anthropicMessage `json:"messages"`
}

// anthropicMessage es un turno de la conversación que se le manda al modelo.
type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicResponse es la parte de la respuesta que interesa: los bloques de
// contenido. El resto de campos se ignora a propósito.
type anthropicResponse struct {
	Content []anthropicContentBlock `json:"content"`
}

// anthropicContentBlock es un bloque de contenido de la respuesta.
type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// text concatena los bloques de tipo texto, en orden y sin separador: la
// respuesta puede venir partida en varios bloques y el JSON que se busca puede
// quedar a caballo entre dos, así que meter un separador lo rompería.
func (r anthropicResponse) text() string {
	parts := make([]string, 0, len(r.Content))
	for _, block := range r.Content {
		if block.Type == blockTypeText {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "")
}

// snippet recorta un cuerpo de respuesta para citarlo en un error sin volcar
// kilobytes en el log. Corta por runas para no partir un carácter por la mitad.
func snippet(body []byte) string {
	runes := []rune(strings.TrimSpace(string(body)))
	if len(runes) > maxSnippetRunes {
		return string(runes[:maxSnippetRunes]) + "..."
	}
	return string(runes)
}
