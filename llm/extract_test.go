package llm_test

import (
	"testing"

	"github.com/EduGoGroup/wapp-shared/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// casoExtract describe una salida cruda de modelo y el objeto que debe salir.
type casoExtract struct {
	nombre string
	raw    string
	quiere string
}

// correrCasosExtract ejecuta una tabla de casos que deben extraerse sin error.
// Existe para que los tests de ExtractJSON crezcan por la tabla y no por el
// cuerpo de cada función.
func correrCasosExtract(t *testing.T, casos []casoExtract) {
	t.Helper()
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			got, err := llm.ExtractJSON(c.raw)
			require.NoError(t, err)
			assert.JSONEq(t, c.quiere, string(got))
		})
	}
}

// requiereErrorDeCalidad afirma que el error es de calidad y no otro.
func requiereErrorDeCalidad(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	assert.ErrorIs(t, err, llm.ErrLLMQuality)
}

func TestExtractJSON_ValladoJSON(t *testing.T) {
	correrCasosExtract(t, []casoExtract{
		{
			nombre: "valla con etiqueta json",
			raw:    "```json\n{\"version\": 1, \"text\": \"hola\"}\n```",
			quiere: `{"version": 1, "text": "hola"}`,
		},
		{
			nombre: "valla sin etiqueta",
			raw:    "```\n{\"version\": 1}\n```",
			quiere: `{"version": 1}`,
		},
		{
			nombre: "valla con cháchara alrededor",
			raw:    "Claro, aquí tienes:\n```json\n{\"version\": 1}\n```\n¿Necesitas algo más?",
			quiere: `{"version": 1}`,
		},
		{
			nombre: "sin valla y sin adornos",
			raw:    `{"version": 1}`,
			quiere: `{"version": 1}`,
		},
	})
}

func TestExtractJSON_DesenvuelveEnvolturaEspuria(t *testing.T) {
	correrCasosExtract(t, []casoExtract{
		{
			nombre: "envoltura bytes",
			raw:    `{"bytes": {"version": 1, "wants": []}}`,
			quiere: `{"version": 1, "wants": []}`,
		},
		{
			nombre: "envoltura result",
			raw:    `{"result": {"version": 1}}`,
			quiere: `{"version": 1}`,
		},
		{
			nombre: "envoltura json dentro de valla",
			raw:    "```json\n{\"json\": {\"version\": 1}}\n```",
			quiere: `{"version": 1}`,
		},
	})
}

func TestExtractJSON_NoDesenvuelveLoQueNoEsEnvoltura(t *testing.T) {
	correrCasosExtract(t, []casoExtract{
		{
			nombre: "clave desconocida con un solo campo",
			raw:    `{"items": {"version": 1}}`,
			quiere: `{"items": {"version": 1}}`,
		},
		{
			nombre: "clave conocida pero con hermanos",
			raw:    `{"data": {"version": 1}, "otro": 2}`,
			quiere: `{"data": {"version": 1}, "otro": 2}`,
		},
		{
			nombre: "clave conocida que no envuelve un objeto",
			raw:    `{"output": "un texto"}`,
			quiere: `{"output": "un texto"}`,
		},
		{
			nombre: "solo se desenvuelve una vez",
			raw:    `{"bytes": {"data": {"version": 1}}}`,
			quiere: `{"data": {"version": 1}}`,
		},
	})
}

func TestExtractJSON_LlavesDentroDeCadenaNoCuentan(t *testing.T) {
	correrCasosExtract(t, []casoExtract{
		{
			nombre: "llave de cierre dentro de una cadena",
			raw:    `{"evidence": "dijo } y siguió", "version": 1}`,
			quiere: `{"evidence": "dijo } y siguió", "version": 1}`,
		},
		{
			nombre: "comilla escapada dentro de la cadena",
			raw:    `{"evidence": "dijo \"vale}\" y siguió", "version": 1}`,
			quiere: `{"evidence": "dijo \"vale}\" y siguió", "version": 1}`,
		},
	})
}

func TestExtractJSON_TruncadoEsErrorDeCalidad(t *testing.T) {
	// La salida cortada a media generación es el fallo más común de un modelo
	// que se queda sin tokens: hay un objeto empezado que nunca cierra.
	//
	// El segundo caso es el que fija la regla de corte del barrido de candidatos:
	// el objeto de arriba está truncado pero el de dentro cierra y es JSON
	// válido, así que un barrido que siguiera buscando devolvería el ítem
	// interior haciéndolo pasar por el artefacto entero.
	// Mutación que lo pone rojo: en firstObjectCandidate, cambiar el
	// `return nil, fmt.Errorf(...)` del candidato sin cerrar por un `continue`.
	truncados := []string{
		`{"version": 1, "wants": [{"idea": "torta"`,
		`{"version": 1, "wants": [{"idea": "torta", "evidence": "una torta"}]`,
		"```json\n{\"version\": 1, \"items\": [\n",
	}
	for _, raw := range truncados {
		got, err := llm.ExtractJSON(raw)
		requiereErrorDeCalidad(t, err)
		assert.Nil(t, got)
	}
}

func TestExtractJSON_SinObjetoEsErrorDeCalidad(t *testing.T) {
	sinObjeto := []string{
		"",
		"   ",
		"Lo siento, no puedo ayudarte con eso.",
		"12345",
	}
	for _, raw := range sinObjeto {
		_, err := llm.ExtractJSON(raw)
		requiereErrorDeCalidad(t, err)
	}
}

func TestExtractJSON_ArrayDeTopeDevuelveSuPrimerObjeto(t *testing.T) {
	// Documentado, no accidental: un artefacto envuelto en un array COMPLETO
	// devuelve el primer objeto. Quien decide si eso sirve es el Parse* de la
	// tarea, que exigirá su version.
	correrCasosExtract(t, []casoExtract{
		{
			nombre: "array con un objeto",
			raw:    `[{"version": 1}]`,
			quiere: `{"version": 1}`,
		},
		{
			nombre: "array con varios objetos",
			raw:    `[{"version": 1}, {"version": 2}]`,
			quiere: `{"version": 1}`,
		},
	})
}

// TestExtractJSON_ArrayDeTopeTruncadoEsErrorDeCalidad separa el array completo
// del array a medias.
//
// Es la mitad que faltaba del caso anterior: hasta este arreglo, un array
// cortado a media generación devolvía su primer objeto SIN error, o sea un
// artefacto incompleto disfrazado de artefacto entero. La promesa «truncado ⇒
// ErrLLMQuality» solo se cumplía cuando el tope era un objeto.
func TestExtractJSON_ArrayDeTopeTruncadoEsErrorDeCalidad(t *testing.T) {
	// Mutación que lo pone rojo: en extract.go, borrar de extractFrom la rama del
	// array de tope, dejando `return firstObjectCandidate(s)` como único camino.
	// Los tres casos vuelven a devolver el primer objeto del array a medias.
	truncados := []string{
		`[{"version": 1},{"version": 1`,
		`[{"version": 1},`,
		"```json\n[{\"version\": 1},{\"vers",
	}
	for _, raw := range truncados {
		got, err := llm.ExtractJSON(raw)
		requiereErrorDeCalidad(t, err)
		assert.Nil(t, got)
	}
}

// TestExtractJSON_ProsaPreviaNoTapaElJSONBueno cubre el reverso del arranque en
// la primera llave: hasta este arreglo, cualquier `{` suelta o cualquier comilla
// descuadrada en la cháchara previa del modelo hacía fallar una respuesta
// perfectamente buena, porque el candidato pasaba a ser la prosa.
func TestExtractJSON_ProsaPreviaNoTapaElJSONBueno(t *testing.T) {
	// Mutación que lo pone rojo: en extract.go, cambiar el bucle de
	// firstObjectCandidate por un solo intento en la primera llave del texto —
	//     i := strings.IndexByte(s, '{'); cand, ok := balancedSpanAt(s, i, '{', '}')
	// — que es justo el comportamiento anterior. Los tres casos se caen.
	correrCasosExtract(t, []casoExtract{
		{
			nombre: "llave sin cerrar en la prosa previa",
			raw:    "Lo intento { pero no prometo nada:\n{\"version\": 1}",
			quiere: `{"version": 1}`,
		},
		{
			nombre: "número impar de comillas en la prosa previa",
			raw:    `El cliente dijo "hola: {"version": 1}`,
			quiere: `{"version": 1}`,
		},
		{
			nombre: "objeto previo que balancea pero no es JSON",
			raw:    `El modelo dijo {"no": } y después {"version": 1}`,
			quiere: `{"version": 1}`,
		},
	})
}

// TestExtractJSON_LaVallaGanaAlEcoDelEsquema es la primera de las dos capas que
// hacen falta contra el eco del esquema: cuando el modelo repite el esquema del
// prompt ANTES de responder y mete la respuesta en una valla, la valla manda.
//
// La segunda capa —la que caza el eco cuando NO hay valla— vive en los Parse*,
// porque ninguna heurística de extracción puede distinguir un esquema repetido
// de una respuesta: los dos son JSON válido. Ver
// TestParseClassification_EcoDelEsquemaEsErrorDeCalidad.
func TestExtractJSON_LaVallaGanaAlEcoDelEsquema(t *testing.T) {
	// Mutación que lo pone rojo: en ExtractJSON, borrar la rama de fencedBlock y
	// dejar `return extractFrom(raw)`. Entonces gana el eco, que va primero.
	correrCasosExtract(t, []casoExtract{
		{
			nombre: "eco del esquema antes de la valla",
			raw: "Voy a responder con este esquema:\n" +
				`{"version": 1, "category": "...", "evidence": "..."}` + "\n" +
				"Aquí va:\n```json\n" +
				`{"version": 1, "category": "intake_request", "evidence": "quiero dos tortas"}` +
				"\n```",
			quiere: `{"version": 1, "category": "intake_request", "evidence": "quiero dos tortas"}`,
		},
	})
}

func TestExtractJSON_ObjetoMalFormadoEsErrorDeCalidad(t *testing.T) {
	// Balancea llaves pero no es JSON: el modelo se inventó la sintaxis.
	_, err := llm.ExtractJSON(`{"version": 1, "text": }`)
	requiereErrorDeCalidad(t, err)
}
