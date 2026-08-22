package llm_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/EduGoGroup/wapp-shared/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// textoAmbar es un recorte del caso real que gobierna el Plan 044.
const textoAmbar = "hola, quería encargar una torta para el miércoles de la semana que viene " +
	"y un paquete de tequeños congelados de 30"

// todosLosPrompts devuelve los cinco prompts del puerto, ya construidos, para
// las afirmaciones que valen para todos.
func todosLosPrompts(t *testing.T) map[string]string {
	t.Helper()
	ref := time.Date(2026, time.July, 13, 10, 0, 0, 0, time.UTC)
	return map[string]string{
		"ClassifyRequest": llm.BuildClassifyRequestPrompt(llm.ClassifyRequestInput{
			Text:       textoAmbar,
			Categories: []string{"intake_request", "saludo", "otro"},
		}),
		"ExtractMainIdeas": llm.BuildExtractMainIdeasPrompt(llm.ExtractMainIdeasInput{
			SourceText: textoAmbar,
		}),
		"ExtractItemSpecs": llm.BuildExtractItemSpecsPrompt(llm.ExtractItemSpecsInput{
			SourceText: textoAmbar,
			Idea:       "paquete de tequeños congelados de 30",
		}),
		"NormalizeQuantities": llm.BuildNormalizeQuantitiesPrompt(llm.NormalizeQuantitiesInput{
			SourceText: textoAmbar,
			Items:      []llm.ItemSpec{{Product: "torta", Variant: "10 o 12 porciones"}},
			MessageTS:  ref,
		}),
		"GenerateQuoteText": llm.BuildGenerateQuoteTextPrompt(llm.GenerateQuoteTextInput{
			Quote: json.RawMessage(`{"lines":[{"label":"tequeños x30","total":"490"}]}`),
		}),
	}
}

func TestPrompts_ExigenSoloJSON(t *testing.T) {
	for nombre, prompt := range todosLosPrompts(t) {
		t.Run(nombre, func(t *testing.T) {
			assert.Contains(t, prompt, "Responde ÚNICAMENTE con un objeto JSON válido")
			assert.Contains(t, prompt, "No escribas absolutamente nada antes ni después del JSON")
			assert.Contains(t, prompt, "No envuelvas la respuesta en otro objeto")
			// El prompt no puede llevar vallas de código: pedirlas y prohibirlas
			// a la vez es la mejor forma de que el modelo las use.
			assert.NotContains(t, prompt, "```")
		})
	}
}

func TestPrompts_EstanEnEspanol(t *testing.T) {
	for nombre, prompt := range todosLosPrompts(t) {
		t.Run(nombre, func(t *testing.T) {
			assert.Contains(t, prompt, "Reglas de salida, sin excepciones")
			assert.Contains(t, prompt, "evidencia")
		})
	}
}

func TestPrompts_LlevanLaVersionDelArtefacto(t *testing.T) {
	for nombre, prompt := range todosLosPrompts(t) {
		t.Run(nombre, func(t *testing.T) {
			assert.Contains(t, prompt, `"version": 1`)
		})
	}
}

func TestBuildClassifyRequestPrompt_ListaLasCategorias(t *testing.T) {
	prompt := llm.BuildClassifyRequestPrompt(llm.ClassifyRequestInput{
		Text:       textoAmbar,
		Categories: []string{"intake_request", "saludo"},
	})
	assert.Contains(t, prompt, "- intake_request")
	assert.Contains(t, prompt, "- saludo")
	assert.Contains(t, prompt, "no inventes categorías nuevas")
	// Categorías, no score: el número de confianza se pide EXPLÍCITAMENTE que no venga.
	assert.Contains(t, prompt, "No devuelvas un número de confianza")
}

func TestBuildExtractItemSpecsPrompt_UnSoloItemConContexto(t *testing.T) {
	prompt := llm.BuildExtractItemSpecsPrompt(llm.ExtractItemSpecsInput{
		SourceText: textoAmbar,
		Idea:       "paquete de tequeños congelados de 30",
	})
	assert.Contains(t, prompt, "Especifica UN SOLO ítem")
	assert.Contains(t, prompt, "paquete de tequeños congelados de 30")
	assert.Contains(t, prompt, textoAmbar)
	// La distinción que decide si algo se cobra aparte va escrita en el prompt.
	assert.Contains(t, prompt, "addon_candidates")
	assert.Contains(t, prompt, "customizations")
}

func TestBuildNormalizeQuantitiesPrompt_FechaDelMensajeNoDeHoy(t *testing.T) {
	// El lunes 13/07/2026 es la fecha del mensaje del caso real (D-044.9): las
	// expresiones relativas se resuelven contra ella, no contra hoy.
	prompt := llm.BuildNormalizeQuantitiesPrompt(llm.NormalizeQuantitiesInput{
		SourceText: textoAmbar,
		Items:      []llm.ItemSpec{{Product: "tequeños congelados"}},
		MessageTS:  time.Date(2026, time.July, 13, 23, 30, 0, 0, time.UTC),
	})
	assert.Contains(t, prompt, "lunes 2026-07-13")
	assert.Contains(t, prompt, "la fecha del mensaje, no la de hoy")
	assert.Contains(t, prompt, "message_ts=2026-07-13")
	assert.Contains(t, prompt, `es qty 1 con unit_kind "package" y package_size 30`)
	assert.Contains(t, prompt, "tequeños congelados")
}

func TestBuildGenerateQuoteTextPrompt_FewShotOpcional(t *testing.T) {
	entrada := llm.GenerateQuoteTextInput{
		Quote: json.RawMessage(`{"lines":[]}`),
	}
	sinEjemplos := llm.BuildGenerateQuoteTextPrompt(entrada)
	require.NotContains(t, sinEjemplos, "--- ejemplo 1 ---")

	entrada.Examples = []string{"Hola! Te paso lo de la torta", "Buenas, quedó así"}
	conEjemplos := llm.BuildGenerateQuoteTextPrompt(entrada)
	assert.Contains(t, conEjemplos, "--- ejemplo 1 ---")
	assert.Contains(t, conEjemplos, "--- ejemplo 2 ---")
	assert.Contains(t, conEjemplos, "Buenas, quedó así")
	// La regla que impide que el modelo toque los importes va siempre.
	assert.Contains(t, conEjemplos, "se copian EXACTAMENTE del borrador")
}
