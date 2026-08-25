package llm_test

// t183_sin_params_test.go — LA RETIRADA DE `params` DE P1 (Plan 044 · Ola 1.8 · T1.8-3, D3).
//
// 🔴 POR QUÉ SE RETIRA, en una frase: la descomposición en ítems es trabajo de P2-P4, y pedírsela también
// a P1 le hacía PERDER ÍTEMS (D-044.20). Desde la Ola 1.6 (pull, ADR-0045) nadie consumía lo que P1
// extraía: `sig.Intent` sale siempre nil, el pool los descarta y en UAT hay CERO reglas `kind='llm'`.
// Era un campo que costaba tokens en cada mensaje y no alimentaba a nadie.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/EduGoGroup/wapp-shared/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// catalogoConParamsDeclarados es el catálogo que hace FALSABLE al criterio (a). Un catálogo sin `Params`
// daría un prompt sin la palabra «params» hiciera lo que hiciera el constructor: el test pasaría con el
// código viejo y con el nuevo, y no probaría nada.
func catalogoConParamsDeclarados() llm.ClassifyRequestInput {
	return llm.ClassifyRequestInput{
		Text: "cómo va el pedido 42",
		Catalog: []llm.IntentSpec{
			{
				Name:        "consulta_estado",
				Description: "el cliente pregunta por un pedido que ya hizo",
				Params:      []string{"numero_pedido", "fecha"},
				Examples: []llm.IntentExample{
					{Message: "cómo va el pedido 42", Params: map[string]string{"numero_pedido": "42"}},
				},
			},
			{
				Name:        "intake_request",
				Description: "el cliente pide un presupuesto",
				Examples:    []llm.IntentExample{{Message: "quiero encargar una torta"}},
			},
		},
		UnknownLabel: "desconocido",
	}
}

// TestT183_a_ElPromptDeP1NoPideParams es el criterio (a): se EJECUTA el constructor y se mira su salida,
// no se lee el fichero. Un grep sobre prompt.go seguiría verde con el campo declarado y no impreso —o al
// revés— y aquí lo que importa es lo que le llega al modelo.
func TestT183_a_ElPromptDeP1NoPideParams(t *testing.T) {
	in := catalogoConParamsDeclarados()
	prompt := llm.BuildClassifyRequestPrompt(in)

	// La comprobación va sobre el prompt en minúsculas para que no se cuele por una mayúscula.
	bajo := strings.ToLower(prompt)
	assert.NotContains(t, bajo, "params", "P1 sólo DETECTA (D3): ni la regla, ni el esquema, ni el few-shot "+
		"pueden nombrar params. Este catálogo DECLARA dos, así que si el constructor los anunciara, saldría aquí")
	assert.NotContains(t, bajo, "numero_pedido", "el nombre de un parámetro declarado no puede llegar al prompt")
	assert.NotContains(t, bajo, "extrae estos parámetros")

	// Y lo que P1 SÍ sigue pidiendo, para que la retirada no se lleve por delante su oficio.
	assert.Contains(t, prompt, "consulta_estado", "el catálogo sigue entero")
	assert.Contains(t, prompt, `"intent"`)
	assert.Contains(t, prompt, `"confidence"`)
	assert.Contains(t, prompt, `"evidence"`)
}

// TestT183_b_ElParserAceptaConYSinParams_YProduceLoMismo es el criterio (b), y la mitad que importa es la
// de CON params: un Edge que corra una versión anterior seguirá mandándolos, y el Cloud no puede romperse
// por eso. Tolerar los dos lados es lo que permite desplegar sin coordinar versiones.
func TestT183_b_ElParserAceptaConYSinParams_YProduceLoMismo(t *testing.T) {
	in := catalogoConParamsDeclarados()

	sin := json.RawMessage(`{"version":1,"intent":"consulta_estado","confidence":0.9,"evidence":"cómo va el pedido 42"}`)
	con := json.RawMessage(`{"version":1,"intent":"consulta_estado","confidence":0.9,` +
		`"params":{"numero_pedido":"42"},"evidence":"cómo va el pedido 42"}`)

	sinP, err := llm.ParseClassification(sin, in)
	require.NoError(t, err, "una respuesta SIN params es la forma nueva: tiene que parsear")
	conP, err := llm.ParseClassification(con, in)
	require.NoError(t, err, "una respuesta CON params es la de un Edge anterior: NO puede romper el Cloud")

	assert.Equal(t, sinP.Intent, conP.Intent, "el intent es el mismo con y sin params")
	assert.InDelta(t, sinP.Confidence, conP.Confidence, 0.0001)
	assert.Equal(t, sinP.Evidence, conP.Evidence)
	assert.Empty(t, sinP.Params, "sin params en el JSON, el campo queda vacío y no se inventa nada")
}

// TestT183_d_ElPrefijoDeP1_Encoge deja el número EN EL TEST y no sólo en el cierre del plan, para que la
// próxima vez que alguien toque P1 vea cuánto costaba antes. No es una aserción sobre un valor exacto
// —eso sería un test tautológico contra una constante—: lo que se afirma es la DIRECCIÓN.
func TestT183_d_ElPrefijoDeP1_Encoge(t *testing.T) {
	in := catalogoConParamsDeclarados()
	prompt := llm.BuildClassifyRequestPrompt(in)

	// MEDIDO el 2026-08-25 con ESTE catálogo, ejecutando el constructor en las dos versiones (no
	// estimado): **2.040 B antes de T1.8-3 → 1.697 B después = −343 B (−16,8 %)**.
	//
	// ⚠️ EN TOKENS NO SE MIDE AQUÍ, Y NO ES UN OLVIDO: un número de tokens depende del tokenizador del
	// modelo, no del texto, así que inventarlo desde los bytes sería fabricarlo. El dato real llega solo,
	// del campo `prompt_tokens` que el cajero ya loguea en cada inferencia servida contra el catálogo de
	// UAT — que valía **2.218** el 2026-08-25, antes de esta tarea.
	//
	// 🔴 Y LO QUE SE AHORRA AQUÍ ES CASI TODO PREFIJO ESTABLE, que se prefilla UNA vez y no por mensaje
	// (T1.7-1 dejó P1 con un 96,8 % de prefijo común). El ahorro por mensaje es el del few-shot; el resto
	// se cobra una sola vez por prefijo caliente. No se venda esto como −16,8 % de latencia.
	t.Logf("P1 con este catálogo: %d B (antes de T1.8-3: 2040 B)", len(prompt))
	assert.Less(t, len(prompt), 2040, "P1 medía 2.040 B con params en este catálogo; tras retirarlos debe "+
		"ser MENOR. Si esto sale rojo, o volvieron los params o el prompt creció por otro sitio: mira cuál")
}
