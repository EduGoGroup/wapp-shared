package llm_test

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/EduGoGroup/wapp-shared/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Este fichero vigila el invariante I6 del ADR-0046: en un prompt, todo lo
// estable va delante y todo lo variable al final.
//
// Por qué importa: Ollama reutiliza el KV-cache del PREFIJO del prompt. Lo que
// queda detrás del primer byte que cambia entre dos llamadas se re-prefilla
// entero, a ~21,6 ms/token. Un dato variable a mitad de camino hace que la mitad
// de atrás se pague en CADA llamada.
//
// El test mide DOS cosas distintas, y el mensaje de fallo dice cuál se rompió:
//
//  1. EL ORDEN (guarda dura, `andamiaje`). Detrás del primer byte que cambia no
//     puede quedar ni un literal del andamiaje del prompt. Un literal no depende
//     de la entrada, así que si aparece ahí es porque está colocado DESPUÉS de un
//     dato variable. Esta guarda NO depende del tamaño de las entradas: es la que
//     caza un reordenamiento.
//
//  2. EL RATIO (`prefijoEstableMinimo`). Cuánto del prompt es prefijo cacheable.
//
// ⚠️ LIMITACIÓN CONOCIDA DEL RATIO — léela antes de tocar nada si ves un rojo.
// El ratio es prefijo/total, así que NO depende solo del orden del prompt:
// depende también del TAMAÑO DE LAS ENTRADAS de este test. El prefijo estable de
// P2 son 1341 B, así que P2 cumple el 90 % mientras el mensaje más corto del par
// no pase de ~149 B; con el par de hoy (70 B) va al 95,0 %. Si alguien mete aquí
// un hilo real largo, el ratio bajará del umbral SIN QUE HAYA NINGÚN DEFECTO DE
// CÓDIGO. Por eso la guarda 1 existe y por eso el mensaje de fallo distingue las
// dos causas.
//
// Qué hacer ante cada rojo:
//   - Falla la guarda 1 → el orden está roto de verdad: mueve el dato variable al
//     FINAL del prompt en prompt.go.
//   - Falla solo el ratio → el orden está bien. Mira el tamaño de las entradas
//     del par (o si el andamiaje estable del prompt se ha acortado). NO reordenes
//     el prompt y NO bajes el umbral.
//
// Prefijos estables medidos el 2026-08-24, tras T1.7-1 (referencia, no aserción):
// P1 2106 B · P2 1341 B · P3 1738 B · P4 2252 B · P5 1191 B.

// prefijoEstableMinimo es la fracción del prompt más corto que dos llamadas
// consecutivas de la MISMA etapa tienen que compartir byte a byte por delante.
//
// Bajarlo es apagar el detector, no arreglar el defecto: si un constructor no
// llega, lo que se mueve es el dato variable, no el número.
const prefijoEstableMinimo = 0.90

// otroTextoDelCliente es un segundo mensaje del MISMO tenant: lo único que
// cambia entre dos llamadas de P1/P2 es el texto del cliente.
const otroTextoDelCliente = "buenas, necesito 40 empanadas para el sábado y una torta de chocolate"

// andamiajeComun son los literales que los cinco prompts comparten. Ninguno
// depende de la entrada, así que ninguno tiene derecho a aparecer detrás del
// primer byte que cambia.
var andamiajeComun = []string{
	"Reglas de salida, sin excepciones",
	"Esquema de la respuesta:",
}

// prefijoComun cuenta los bytes iniciales que dos cadenas comparten.
//
// Byte a byte y no rune a rune a propósito: lo que el modelo cachea son tokens
// sobre bytes, y un prefijo que diverge a mitad de un carácter multibyte ya está
// invalidado.
func prefijoComun(a, b string) int {
	n := min(len(a), len(b))
	for i := range n {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

// casoDeEtapa son dos prompts de la misma etapa construidos con entradas que
// difieren SOLO en lo que tiene derecho a variar, más el andamiaje estable que
// esa etapa tiene que dejar POR DELANTE del primer byte que cambia.
type casoDeEtapa struct {
	a, b      string
	andamiaje []string
}

// casosPorEtapa arma, para cada constructor del puerto, ese par de prompts.
//
// Qué varía en cada etapa, y por qué justo eso:
//   - P1 y P2: dos mensajes distintos del mismo tenant (mismo catálogo, mismo
//     vocabulario, misma etiqueta de escape).
//   - P3: dos ítems del MISMO pedido, con el mismo SourceText. Es el caso que
//     manda: P3 se llama una vez por ítem, así que el hilo se repite N veces.
//   - P4: dos fechas de mensaje distintas.
//   - P5: dos borradores distintos del mismo tenant (mismos ejemplos de voz).
func casosPorEtapa() map[string]casoDeEtapa {
	unaFecha := time.Date(2026, time.July, 13, 10, 0, 0, 0, time.UTC)
	otraFecha := time.Date(2026, time.August, 22, 18, 45, 0, 0, time.UTC)
	items := []llm.ItemSpec{{Product: "torta", Variant: "10 o 12 porciones"}}
	ejemplosDeVoz := []string{"Hola! Te paso lo de la torta"}

	classify := func(texto string) string {
		in := entradaDeClasificacion()
		in.Text = texto
		return llm.BuildClassifyRequestPrompt(in)
	}

	return map[string]casoDeEtapa{
		"ClassifyRequest": {
			a: classify(textoAmbar),
			b: classify(otroTextoDelCliente),
			andamiaje: []string{
				"Reglas de la clasificación:",
				"Vocabulario del negocio",
				"Ejemplos (mensaje del cliente -> respuesta esperada):",
				"\nMensaje del cliente:\n",
			},
		},
		"ExtractMainIdeas": {
			a:         llm.BuildExtractMainIdeasPrompt(llm.ExtractMainIdeasInput{SourceText: textoAmbar}),
			b:         llm.BuildExtractMainIdeasPrompt(llm.ExtractMainIdeasInput{SourceText: otroTextoDelCliente}),
			andamiaje: []string{"\nTexto del cliente:\n"},
		},
		"ExtractItemSpecs": {
			a: llm.BuildExtractItemSpecsPrompt(llm.ExtractItemSpecsInput{
				SourceText: textoAmbar, Idea: "una torta para el miércoles de la semana que viene",
			}),
			b: llm.BuildExtractItemSpecsPrompt(llm.ExtractItemSpecsInput{
				SourceText: textoAmbar, Idea: "un paquete de tequeños congelados de 30",
			}),
			andamiaje: []string{
				"Especifica UN SOLO ítem",
				// El hilo va DELANTE del ítem: es lo estable entre las N llamadas.
				"Texto completo del cliente (contexto y fuente de la evidencia):",
				"\nÍtem que debes especificar:\n",
			},
		},
		"NormalizeQuantities": {
			a: llm.BuildNormalizeQuantitiesPrompt(llm.NormalizeQuantitiesInput{
				SourceText: textoAmbar, Items: items, MessageTS: unaFecha,
			}),
			b: llm.BuildNormalizeQuantitiesPrompt(llm.NormalizeQuantitiesInput{
				SourceText: textoAmbar, Items: items, MessageTS: otraFecha,
			}),
			andamiaje: []string{
				// La fecha se emite AL FINAL: ni la regla ni el esquema pueden
				// llevarla interpolada, así que los dos van por delante.
				"Normaliza las cantidades de los ítems que se te dan",
				"la fecha del mensaje, no la de hoy",
				`"delivery_date_basis": "message_ts=AAAA-MM-DD"`,
				"\nÍtems a normalizar:\n",
				"Texto completo del cliente (fuente de la evidencia):",
				"Fecha de referencia (la fecha del mensaje, no la de hoy): ",
			},
		},
		"GenerateQuoteText": {
			a: llm.BuildGenerateQuoteTextPrompt(llm.GenerateQuoteTextInput{
				Quote:    json.RawMessage(`{"lines":[{"label":"tequeños x30","total":"490"}]}`),
				Examples: ejemplosDeVoz,
			}),
			b: llm.BuildGenerateQuoteTextPrompt(llm.GenerateQuoteTextInput{
				Quote:    json.RawMessage(`{"lines":[{"label":"empanadas x40","total":"820"}]}`),
				Examples: ejemplosDeVoz,
			}),
			andamiaje: []string{
				"se copian EXACTAMENTE del borrador",
				"Así escribe este negocio (imita el tono, no el contenido):",
				"\nBorrador a redactar:\n",
			},
		},
	}
}

// TestPrompts_LoEstableVaDelante es el test de REGLA de I6: uno solo que recorre
// los CINCO constructores en vez de cinco tests de conducta. El invariante es el
// mismo en los cinco sitios, así que lo que se testea es la regla.
//
// Mutaciones que lo ponen rojo POR LA GUARDA DEL ORDEN (ejecutadas, no
// prometidas):
//   - devolver la fecha de referencia de P4 al cuerpo de la instrucción;
//   - poner Idea delante de SourceText en P3.
func TestPrompts_LoEstableVaDelante(t *testing.T) {
	casos := casosPorEtapa()
	require.Len(t, casos, 5, "el test de regla tiene que cubrir los cinco constructores del puerto")

	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			corto := min(len(caso.a), len(caso.b))
			require.Positive(t, corto)

			comun := prefijoComun(caso.a, caso.b)
			fraccion := float64(comun) / float64(corto)
			t.Logf("%s: prefijo común %d B de %d B (%.1f %%)", nombre, comun, corto, fraccion*100)

			// Guarda 1 — EL ORDEN. Independiente del tamaño de las entradas.
			cola := caso.a[comun:]
			for _, etiqueta := range slices.Concat(andamiajeComun, caso.andamiaje) {
				require.Contains(t, caso.a, etiqueta,
					"%s: la etiqueta %q ya no está en el prompt; actualiza este test antes de creerte el resto",
					nombre, etiqueta)
				require.NotContains(t, cola, etiqueta,
					"%s: SE ROMPIÓ EL ORDEN, no el tamaño. Detrás del primer byte que cambia entre dos "+
						"llamadas (byte %d) quedó el literal estable %q, que no depende de la entrada: "+
						"está colocado DESPUÉS de un dato variable y se re-prefilla en cada llamada. "+
						"Mueve el dato variable al FINAL del prompt (I6, ADR-0046); no toques el umbral "+
						"ni las entradas de este test",
					nombre, comun, etiqueta)
			}

			// Guarda 2 — EL RATIO. Sí depende del tamaño de las entradas: ver la
			// limitación conocida documentada arriba.
			assert.GreaterOrEqual(t, fraccion, prefijoEstableMinimo,
				"%s: EL ORDEN ESTÁ BIEN —detrás del prefijo común no queda andamiaje—, lo que se quedó "+
					"corto es el TAMAÑO: el prefijo cacheable es %d de %d B (%.1f %%), por debajo del %.0f %%. "+
					"El ratio es prefijo/total, así que lo baja una ENTRADA más larga en este test o un "+
					"andamiaje estable más corto en prompt.go. Mira eso antes de reordenar nada, y NO bajes "+
					"prefijoEstableMinimo",
				nombre, comun, corto, fraccion*100, prefijoEstableMinimo*100)
		})
	}
}
