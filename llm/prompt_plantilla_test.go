package llm_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/EduGoGroup/wapp-shared/llm"
	"github.com/stretchr/testify/require"
)

// Este fichero vigila UNA regla, la que P4 rompió y costó la etapa entera: la
// plantilla que el prompt le ENSEÑA al modelo, con sus huecos rellenos, tiene
// que ser un artefacto que el validador de esa misma etapa ACEPTE.
//
// Qué pasó, para que nadie tenga que reconstruirlo: el esquema de P4 imprimía
// `"unit_kind": "package", "package_size": 0` y `"range": {"min": 0, "max": 0}`,
// y validarPaquete y validarRango rechazan exactamente eso. El modelo no
// desobedecía: copiaba el ejemplo que se le daba. P4 fue 0 de 14 en su primer
// día en campo.
//
// La distinción que ordena todo esto —y que hace que este test NO exija quitar
// los `"..."`— es si el relleno es RECONOCIBLE:
//
//   - `"..."` (PlaceholderEsquema) y `AAAA-MM-DD` son rellenos reconocibles: si
//     el modelo los ecoa, el validador lo CAZA y lo llama por su nombre («el
//     modelo repitió el esquema»). Que los rechace es la función, no el defecto.
//   - Un `0` no es reconocible: es indistinguible de un valor real. No hay forma
//     de detectarlo como eco, solo de rechazarlo por semántica, y eso es lo que
//     se vio seis veces en campo.
//
// De ahí las dos aserciones de cada etapa: la plantilla cruda sigue siendo
// rechazable (los huecos siguen siendo reconocibles) y la plantilla con los
// huecos rellenos es válida (no queda ningún valor que el validador rechace).

// textoDeRelleno es lo que se pone en los huecos de texto de la plantilla. No es
// una trampa para que el test pase: lo que se afirma es «con los huecos de texto
// rellenos con algo plausible, la FORMA que se le enseña al modelo es un
// artefacto válido». Los huecos de texto son huecos por definición.
const textoDeRelleno = "un paquete de tequeños congelados de 30"

// formatoDeFecha es el otro relleno reconocible de la plantilla, y lo dice el
// propio prompt: «AAAA-MM-DD es el FORMATO, no un valor».
const formatoDeFecha = "AAAA-MM-DD"

// fechaDeRelleno es una fecha REAL con ese formato: el miércoles de la semana
// siguiente al mensaje del caso que gobierna el Plan 044.
const fechaDeRelleno = "2026-07-22"

// casoDePlantilla es el validador de una etapa más lo que haga falta para
// rellenar los huecos que prometen algo más estrecho que «un texto».
type casoDePlantilla struct {
	// parse es el validador REAL de la etapa, el mismo que corre en producción.
	parse func(json.RawMessage) error
	// prepara rellena los huecos cuyo valor válido NO es un texto cualquiera.
	// Hoy solo lo necesita P1: su `"intent": "..."` promete «uno de los nombres
	// del catálogo», y rellenarlo con prosa sería rellenarlo con algo que la
	// plantilla nunca prometió. Nil cuando la etapa no tiene ninguno.
	prepara func(t *testing.T, esquema string) string
}

// validadorPorEtapa empareja cada constructor de prompt con el Parse* que lee su
// salida. Las claves son las de todosLosPrompts: el test exige que coincidan, así
// que una etapa nueva sin validador aquí sale roja en vez de quedarse sin cubrir.
func validadorPorEtapa() map[string]casoDePlantilla {
	return map[string]casoDePlantilla{
		"ClassifyRequest": {
			parse: errDeClassification,
			prepara: func(t *testing.T, esquema string) string {
				t.Helper()
				const hueco = `"intent": "..."`
				require.Contains(t, esquema, hueco,
					"el esquema de P1 ya no imprime %s: actualiza este test antes de creerte el resto", hueco)
				// intake_request es el primer nombre de catalogoDePrueba, que es el
				// catálogo con el que se arma este mismo prompt.
				return strings.Replace(esquema, hueco, `"intent": "intake_request"`, 1)
			},
		},
		"ExtractMainIdeas":    {parse: errDeMainIdeas},
		"ExtractItemSpecs":    {parse: errDeItemSpecs},
		"NormalizeQuantities": {parse: errDeQuantities},
		"GenerateQuoteText":   {parse: errDeQuoteText},
	}
}

// TestPlantillaDelPrompt_PasaSuPropioValidador es un test de REGLA sobre las
// cinco etapas, no cinco tests de conducta: el invariante es el mismo en los
// cinco sitios.
//
// Mutaciones que lo ponen rojo (ejecutadas, no prometidas): devolver
// `"package_size": 0` al esquema de P4, y devolver `{"min": 0, "max": 0}` a su
// range.
func TestPlantillaDelPrompt_PasaSuPropioValidador(t *testing.T) {
	prompts := todosLosPrompts(t)
	casos := validadorPorEtapa()
	require.Len(t, casos, len(prompts),
		"cada constructor de prompt tiene que tener su validador en la tabla: son %d prompts y %d validadores",
		len(prompts), len(casos))

	for nombre, prompt := range prompts {
		caso, cubierta := casos[nombre]
		require.True(t, cubierta, "la etapa %q no tiene validador en validadorPorEtapa", nombre)

		t.Run(nombre, func(t *testing.T) {
			esquema := esquemaDeLaPlantilla(t, prompt)

			// (a) Cruda, la plantilla tiene que seguir siendo RECHAZABLE: es lo
			// que garantiza que sus huecos siguen siendo reconocibles. Si esto
			// sale rojo, alguien puso en el esquema un valor que pasa por bueno,
			// y entonces el eco del esquema entra al pipeline como si fuera una
			// respuesta.
			requiereErrorDeCalidad(t, caso.parse(json.RawMessage(esquema)))

			// (b) Con los huecos rellenos, la plantilla tiene que ser VÁLIDA.
			relleno := esquema
			if caso.prepara != nil {
				relleno = caso.prepara(t, relleno)
			}
			relleno = rellenarHuecosReconocibles(relleno)
			require.NotContains(t, relleno, llm.PlaceholderEsquema,
				"el relleno no sustituyó nada: la aserción de abajo pasaría sin mirar")

			require.NoError(t, caso.parse(json.RawMessage(relleno)),
				"%s: el validador de esta etapa RECHAZA la plantilla que su propio prompt le enseña al "+
					"modelo. Un modelo que copia el ejemplo produce un artefacto inválido: eso es lo que "+
					"dejó P4 en 0 de 14 en campo. Lo que se arregla es la PLANTILLA en prompt.go, NO el "+
					"validador: en el esquema no puede quedar ningún valor que el Parse* rechace. Los "+
					"`\"...\"` sí pueden quedarse —son reconocibles—; un número no, porque no hay forma de "+
					"distinguirlo de una respuesta de verdad.\nPlantilla rellena: %s", nombre, relleno)
		})
	}
}

// esquemaDeLaPlantilla aísla el objeto JSON que el prompt imprime bajo «Esquema
// de la respuesta:».
//
// Lo aísla con el MISMO ExtractJSON que corre en producción sobre la salida del
// modelo, y no con un recorte a mano, a propósito: si el modelo ecoa el esquema,
// esto es EXACTAMENTE lo que el pipeline va a extraer y pasarle al Parse*.
func esquemaDeLaPlantilla(t *testing.T, prompt string) string {
	t.Helper()
	const marca = "Esquema de la respuesta:"
	i := strings.Index(prompt, marca)
	require.GreaterOrEqual(t, i, 0,
		"el prompt ya no imprime %q; actualiza este test antes de creerte el resto", marca)

	raw, err := llm.ExtractJSON(prompt[i+len(marca):])
	require.NoError(t, err, "el esquema que imprime el prompt no es ni siquiera JSON extraíble")
	return string(raw)
}

// rellenarHuecosReconocibles sustituye los DOS rellenos reconocibles de las
// plantillas por valores plausibles. No toca ningún número, que es justo el
// punto: un número de la plantilla tiene que ser válido tal cual está impreso,
// porque el modelo lo copia y nadie puede saber que lo copió.
func rellenarHuecosReconocibles(esquema string) string {
	conTexto := strings.ReplaceAll(esquema, llm.PlaceholderEsquema, textoDeRelleno)
	return strings.ReplaceAll(conTexto, formatoDeFecha, fechaDeRelleno)
}

// TestPromptDeP4_DiceQueHacerConElPaqueteSinTamano vigila la regla GEMELA de «si
// el cliente no dijo cuántos, qty vale 1»: la que dice qué hacer cuando sí es un
// paquete pero el cliente no dijo de cuántas unidades.
//
// No es una aserción de que cierta frase esté escrita: es que la frase mande
// hacer lo ÚNICO que el contrato acepta. La evidencia está en validarPaquete
// (parse.go): sale por la puerta de arriba solo si unit_kind viene VACÍO, y con
// unit_kind "package" exige package_size >= 1. No hay tercera salida —ni un
// centinela, ni un package_size nulo—, así que la instrucción correcta es omitir
// el campo, con el coste conocido de perder que era un paquete.
//
// Mutación que lo pone rojo (ejecutada): quitar esa regla de la instrucción de
// BuildNormalizeQuantitiesPrompt.
func TestPromptDeP4_DiceQueHacerConElPaqueteSinTamano(t *testing.T) {
	prompt := llm.BuildNormalizeQuantitiesPrompt(llm.NormalizeQuantitiesInput{
		SourceText: textoAmbar,
		Items:      []llm.ItemSpec{{Product: "tequeños congelados"}},
		MessageTS:  time.Date(2026, time.July, 13, 10, 0, 0, 0, time.UTC),
	})

	require.Contains(t, prompt, "Si es un paquete pero el cliente no dijo cuántas unidades trae, omite unit_kind",
		"al modelo se le dice qué hacer si falta la cantidad y nada si falta el tamaño del paquete: "+
			"esa regla es la gemela que faltaba")
	require.Contains(t, prompt, "no pongas 0 ni te inventes el tamaño",
		"un 0 lo rechaza validarPaquete y un tamaño inventado pasa el validador y miente: "+
			"la regla tiene que cerrar las dos puertas")

	// Lo que la regla manda hacer, ACEPTADO.
	require.NoError(t, errDeQuantities(json.RawMessage(
		`{"version": 1, "items": [{"product": "tequeños", "qty": 1,
		                           "evidence": "un paquete de tequeños congelados"}]}`)),
		"omitir unit_kind y package_size es la única salida que el contrato ofrece; si esto se cayera, "+
			"la regla estaría mandando hacer algo inválido")

	// La alternativa que la regla prohíbe, RECHAZADA: declarar el paquete sin
	// poder decir de cuánto es.
	requiereErrorDeCalidad(t, errDeQuantities(json.RawMessage(
		`{"version": 1, "items": [{"product": "tequeños", "qty": 1, "unit_kind": "package",
		                           "evidence": "un paquete de tequeños congelados"}]}`)))
}

// TestPromptDeP4_DiceQueValeQtyConUnRango vigila el TERCER hueco de la misma
// familia: la instrucción enunciaba «si el cliente no dijo cuántos, qty vale 1» y
// «los rangos se conservan como rango», pero NO decía qué vale qty cuando la
// cantidad ES el rango. El cliente sí dijo cuántos —«entre 10 y 12 kilos»—, así
// que la primera regla no aplica, y el hueco quedaba a interpretación del modelo.
//
// Lo que pasó en campo el 2026-08-26, y es la razón de que este test exista: dos
// modelos INDEPENDIENTES (gemma4:e2b y gemma4:e4b) rellenaron ese hueco de la
// misma forma, `"qty": 0`, con el sentido de «no procede» — y validarQuantities
// lo rechaza en `it.Qty < 1`. Cuando dos modelos distintos coinciden en la
// respuesta equivocada, el que está mal es el prompt: no había nada que
// desobedecer. Los dos habían descompuesto BIEN los tres productos del mensaje;
// se perdió el artefacto entero por este campo, porque el parseo es todo-o-nada.
//
// Mutación que lo pone rojo (ejecutada): quitar de la instrucción de
// BuildNormalizeQuantitiesPrompt la frase que fija qty a 1 ante un rango.
func TestPromptDeP4_DiceQueValeQtyConUnRango(t *testing.T) {
	prompt := llm.BuildNormalizeQuantitiesPrompt(llm.NormalizeQuantitiesInput{
		SourceText: textoAmbar,
		Items:      []llm.ItemSpec{{Product: "papas fritas", Variant: "10 o 12 kilos"}},
		MessageTS:  time.Date(2026, time.July, 13, 10, 0, 0, 0, time.UTC),
	})

	require.Contains(t, prompt, "el rango lleva el cuánto y qty vale 1",
		"sin esto, «entre 10 y 12 kilos» deja qty sin definir: el cliente SÍ dijo cuántos, "+
			"así que la regla de la cantidad omitida no cubre el caso")
	require.Contains(t, prompt, "qty es un entero de 1 en adelante y NUNCA vale 0",
		"el 0 es la respuesta que dieron los dos modelos en campo; hay que cerrarla por su nombre")

	// Lo que la regla manda hacer, ACEPTADO.
	require.NoError(t, errDeQuantities(json.RawMessage(
		`{"version": 1, "items": [{"product": "papas fritas", "qty": 1,
		                           "range": {"min": 10, "max": 12, "unit": "kilos"},
		                           "evidence": "entre 10 y 12 kilos de papas fritas"}]}`)),
		"qty 1 junto al rango es la única salida que el contrato ofrece; si esto se cayera, "+
			"la regla estaría mandando hacer algo inválido")

	// Lo que la regla prohíbe, RECHAZADO: es el artefacto EXACTO que mataron
	// gemma4:e2b y gemma4:e4b el 2026-08-26.
	requiereErrorDeCalidad(t, errDeQuantities(json.RawMessage(
		`{"version": 1, "items": [{"product": "papas fritas", "qty": 0,
		                           "range": {"min": 10, "max": 12, "unit": "kilos"},
		                           "evidence": "entre 10 y 12 kilos de papas fritas"}]}`)))
}
