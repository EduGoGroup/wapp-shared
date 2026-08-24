package api_test

import (
	"context"
	"testing"

	"github.com/EduGoGroup/wapp-shared/llm"
	"github.com/EduGoGroup/wapp-shared/llm/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// configValida es una configuración completa a la que cada test le cambia lo
// que quiere probar.
func configValida() api.Config {
	// #nosec G101 -- «clave-de-prueba» es un literal de test: New solo comprueba
	// que el campo no esté vacío, no valida la clave contra ningún servicio.
	return api.Config{
		Provider: api.ProviderAnthropic,
		APIKey:   "clave-de-prueba",
		Model:    "modelo-de-prueba",
	}
}

func TestNew_ProveedorDesconocidoFallaEnBootstrap(t *testing.T) {
	// «local» no es un caso especial: en este plan la vía local no está
	// cableada, así que falla exactamente igual que cualquier otro nombre que no
	// exista. Su rechazo con mensaje propio vive en la capa de configuración del
	// tenant, no aquí.
	desconocidos := []string{"local", "openai", "ollama", "", "Anthropic"}
	for _, nombre := range desconocidos {
		cfg := configValida()
		cfg.Provider = nombre

		p, err := api.New(cfg)
		require.Error(t, err, "provider %q debería fallar al construir", nombre)
		assert.Nil(t, p)
		assert.ErrorIs(t, err, api.ErrUnsupportedProvider)
		assert.ErrorIs(t, err, api.ErrInvalidConfig)
	}
}

func TestNew_ElProveedorSeDiagnosticaAntesQueLaCredencial(t *testing.T) {
	// Falla en el arranque de las dos formas, así que el criterio se cumplía
	// igual; lo que no se cumplía era DECIR la verdad. Con la credencial validada
	// primero, un tenant con provider «local» y sin clave se diagnosticaba como
	// «falta la credencial del proveedor», y quien leyera ese arranque se pondría
	// a buscar una clave que no arregla nada: la vía local no está cableada.
	//
	// Mutación que lo pone rojo: en api.New, mover la llamada a checkProvider por
	// debajo de las dos comprobaciones de cfg.APIKey y cfg.Model.
	incompletas := []api.Config{
		{Provider: "local"},
		{Provider: "local", Model: "modelo-de-prueba"},
		{Provider: "ollama"},
		{Provider: ""},
	}
	for _, cfg := range incompletas {
		p, err := api.New(cfg)
		require.Error(t, err)
		assert.Nil(t, p)
		assert.ErrorIs(t, err, api.ErrUnsupportedProvider)
		assert.NotContains(t, err.Error(), "credencial",
			"el diagnóstico tiene que señalar el proveedor, no una credencial que no arregla nada")
	}
}

func TestNew_FaltaCredencialOModelo(t *testing.T) {
	sinClave := configValida()
	sinClave.APIKey = ""
	_, err := api.New(sinClave)
	require.Error(t, err)
	assert.ErrorIs(t, err, api.ErrInvalidConfig)

	sinModelo := configValida()
	sinModelo.Model = ""
	_, err = api.New(sinModelo)
	require.Error(t, err)
	assert.ErrorIs(t, err, api.ErrInvalidConfig)
}

func TestNew_AnthropicSeConstruye(t *testing.T) {
	p, err := api.New(configValida())
	require.NoError(t, err)
	require.NotNil(t, p)
}

func TestGemini_SeConstruyeYFallaNombrado(t *testing.T) {
	cfg := configValida()
	cfg.Provider = api.ProviderGemini

	p, err := api.New(cfg)
	require.NoError(t, err, "el stub debe construirse: un tenant puede tenerlo configurado")
	require.NotNil(t, p)

	ctx := context.Background()
	llamadas := map[string]func() error{
		"ClassifyRequest": func() error {
			_, err := p.ClassifyRequest(ctx, llm.ClassifyRequestInput{}, llm.Options{})
			return err
		},
		"ExtractMainIdeas": func() error {
			_, err := p.ExtractMainIdeas(ctx, llm.ExtractMainIdeasInput{}, llm.Options{})
			return err
		},
		"ExtractItemSpecs": func() error {
			_, err := p.ExtractItemSpecs(ctx, llm.ExtractItemSpecsInput{}, llm.Options{})
			return err
		},
		"NormalizeQuantities": func() error {
			_, err := p.NormalizeQuantities(ctx, llm.NormalizeQuantitiesInput{}, llm.Options{})
			return err
		},
		"GenerateQuoteText": func() error {
			_, err := p.GenerateQuoteText(ctx, llm.GenerateQuoteTextInput{}, llm.Options{})
			return err
		},
	}

	for nombre, llamar := range llamadas {
		t.Run(nombre, func(t *testing.T) {
			err := llamar()
			require.Error(t, err)
			assert.ErrorIs(t, err, api.ErrNotImplemented)
			// No es un fallo de calidad: no hay modelo al que reintentar.
			assert.NotErrorIs(t, err, llm.ErrLLMQuality)
			assert.Contains(t, err.Error(), nombre)
		})
	}
}
