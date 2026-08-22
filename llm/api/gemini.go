package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/EduGoGroup/wapp-shared/llm"
)

// geminiProvider es el stub de Gemini.
//
// Existe para que el cableado esté hecho el día que se implemente y para que la
// forma del puerto quede probada contra más de una implementación. Se construye
// sin error —un tenant puede tener gemini configurado en tenant_llm— y falla en
// cada llamada con ErrNotImplemented, nombrando el modelo que se pidió para que
// el log diga qué se intentó y no solo que algo falló.
//
// El fallo es de INFRAESTRUCTURA, no de calidad: no hay modelo que reintentar
// con otra temperatura.
type geminiProvider struct {
	cfg Config
}

var _ llm.LLMProvider = (*geminiProvider)(nil)

// newGemini construye el stub.
func newGemini(cfg Config) *geminiProvider {
	return &geminiProvider{cfg: cfg}
}

func (p *geminiProvider) ExtractMainIdeas(_ context.Context, _ llm.ExtractMainIdeasInput, _ llm.Options) (json.RawMessage, error) {
	return nil, p.unavailable("ExtractMainIdeas")
}

func (p *geminiProvider) ExtractItemSpecs(_ context.Context, _ llm.ExtractItemSpecsInput, _ llm.Options) (json.RawMessage, error) {
	return nil, p.unavailable("ExtractItemSpecs")
}

func (p *geminiProvider) NormalizeQuantities(_ context.Context, _ llm.NormalizeQuantitiesInput, _ llm.Options) (json.RawMessage, error) {
	return nil, p.unavailable("NormalizeQuantities")
}

func (p *geminiProvider) GenerateQuoteText(_ context.Context, _ llm.GenerateQuoteTextInput, _ llm.Options) (json.RawMessage, error) {
	return nil, p.unavailable("GenerateQuoteText")
}

// unavailable arma el error nombrado del stub.
func (p *geminiProvider) unavailable(task string) error {
	return fmt.Errorf("%w: %s no está implementada para el proveedor %q (modelo %q)",
		ErrNotImplemented, task, ProviderGemini, p.cfg.Model)
}
