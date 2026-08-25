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

// etiquetaDesconocido es la etiqueta de escape que declara el caller. En wApp la
// fija wapp-shared/intents (ReservedUnknown); este paquete no la conoce, y por eso
// aquí es una constante del test y no del código.
const etiquetaDesconocido = "desconocido"

// catalogoDePrueba es el catálogo CERRADO que el caller le ofrece al modelo.
//
// El primero es el intent que está publicado EN CAMPO, con sus params vacíos
// (D-044.20); el segundo existe para ejercitar la rama de los parámetros
// declarados, que hoy ningún tenant usa pero el contrato admite.
func catalogoDePrueba() []llm.IntentSpec {
	return []llm.IntentSpec{
		{
			Name:        "intake_request",
			Description: "el cliente pide un presupuesto o quiere encargar algo",
			Examples: []llm.IntentExample{
				{Message: "quiero encargar una torta"},
			},
		},
		{
			Name:        "consulta_estado",
			Description: "el cliente pregunta por un pedido que ya hizo",
			Params:      []string{"numero_pedido"},
			Examples: []llm.IntentExample{
				{Message: "cómo va el pedido 42", Params: map[string]string{"numero_pedido": "42"}},
			},
		},
	}
}

// entradaDeClasificacion es la entrada completa de P1 del caso de prueba: la misma
// que se le pasa al Build y al Parse, que es justo lo que el puerto exige.
func entradaDeClasificacion() llm.ClassifyRequestInput {
	return llm.ClassifyRequestInput{
		Text:         textoAmbar,
		Catalog:      catalogoDePrueba(),
		UnknownLabel: etiquetaDesconocido,
		Vocabulary:   []string{"tequeños", "torta húmeda"},
	}
}

// todosLosPrompts devuelve los cinco prompts del puerto, ya construidos, para
// las afirmaciones que valen para todos.
func todosLosPrompts(t *testing.T) map[string]string {
	t.Helper()
	ref := time.Date(2026, time.July, 13, 10, 0, 0, 0, time.UTC)
	return map[string]string{
		"ClassifyRequest": llm.BuildClassifyRequestPrompt(entradaDeClasificacion()),
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

func TestBuildClassifyRequestPrompt_CatalogoVocabularioYEscape(t *testing.T) {
	prompt := llm.BuildClassifyRequestPrompt(entradaDeClasificacion())

	// El catálogo entero, con la descripción de cada intención: sin ella el
	// modelo elige por el nombre, que es una etiqueta y no una definición.
	assert.Contains(t, prompt, "- intake_request: el cliente pide un presupuesto")
	assert.Contains(t, prompt, "- consulta_estado: el cliente pregunta por un pedido")
	// 🔴 LOS PARAMS DECLARADOS YA NO SE ANUNCIAN (T1.8-3, D3), y esta aserción se INVIRTIÓ: antes exigía
	// "(extrae estos parámetros: numero_pedido)". El catálogo de este caso SÍ declara `numero_pedido`, así
	// que si el constructor volviera a anunciarlos esto saldría rojo — que es justo lo que hace falta para
	// que el criterio (a) sea falsable y no una tautología sobre un catálogo sin params.
	assert.NotContains(t, prompt, "extrae estos parámetros")
	assert.NotContains(t, prompt, "numero_pedido")

	assert.Contains(t, prompt, "Vocabulario del negocio (pistas de dominio): tequeños, torta húmeda")
	assert.Contains(t, prompt, "el nombre que devuelvas tiene que estar en la lista")
	// El rango se pide explícito: es lo único que hace comparable la confianza
	// contra el umbral del tenant.
	assert.Contains(t, prompt, "confidence es un número entre 0 y 1")
	assert.Contains(t, prompt, `responde "desconocido"`)
	assert.Contains(t, prompt, textoAmbar)
}

func TestBuildClassifyRequestPrompt_FewShotConLaFormaDeLaRespuesta(t *testing.T) {
	// El few-shot es lo que sostiene el mapeo en un modelo de 1–2B —medido en
	// wapp-edge-intent—, y la vía local ejecuta uno de ese tamaño. Los ejemplos
	// van con la MISMA forma que se le pide al modelo, no con una parecida.
	// Mutación que lo pone rojo: en fewShotDeIntents, quitar la Version del shot.
	prompt := llm.BuildClassifyRequestPrompt(entradaDeClasificacion())

	assert.Contains(t, prompt, `"cómo va el pedido 42" -> `)
	assert.Contains(t, prompt, `{"version":1,"intent":"consulta_estado","confidence":0.9,`+
		`"evidence":"cómo va el pedido 42"}`)
	// 🔴 EL EJEMPLO YA NO IMPRIME `"params":{}` (T1.8-3, D3). El ejemplo del catálogo QUE SÍ TIENE params
	// declarados (`consulta_estado` → `numero_pedido:42`) es el que hace falsable esta aserción: si el
	// few-shot volviera a serializarlos, aparecerían aquí.
	assert.NotContains(t, prompt, `"params"`)
	assert.Contains(t, prompt, `"intent":"intake_request","confidence":0.9,"evidence":`)
}

func TestBuildClassifyRequestPrompt_LoOpcionalDesaparece(t *testing.T) {
	// Un catálogo pelado —sin ejemplos, sin vocabulario y sin etiqueta de escape—
	// da un prompt más flojo, pero no da un prompt roto con secciones vacías
	// colgando ni una regla que ofrezca una etiqueta que el caller no declaró.
	prompt := llm.BuildClassifyRequestPrompt(llm.ClassifyRequestInput{
		Text:    textoAmbar,
		Catalog: []llm.IntentSpec{{Name: "intake_request", Description: "pide algo"}},
	})
	assert.NotContains(t, prompt, "Ejemplos (mensaje del cliente")
	assert.NotContains(t, prompt, "Vocabulario del negocio")
	assert.NotContains(t, prompt, "no encaja en ninguna intención")
	assert.Contains(t, prompt, "- intake_request: pide algo")
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
