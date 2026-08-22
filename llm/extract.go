package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// envelopeKeys son las claves de envoltura espuria que los modelos chicos
// añaden de vez en cuando alrededor del objeto que se les pidió. Solo se
// desenvuelve si la clave es una de éstas Y el objeto de arriba tiene
// exactamente una.
var envelopeKeys = map[string]bool{
	"bytes":    true,
	"result":   true,
	"data":     true,
	"response": true,
	"output":   true,
	"json":     true,
}

// ExtractJSON aísla el objeto JSON que hay dentro de la salida cruda de un
// modelo y lo devuelve listo para decodificar.
//
// El orden es éste, y cada paso existe por un modo de fallo medido:
//
//  1. Si la salida trae una valla de código Markdown con algo estructural
//     dentro, MANDA el contenido de la valla y se ignora lo de fuera. Es el caso
//     real dominante y además es lo único que separa la respuesta de un modelo
//     que antes de responder hace eco del esquema que le pidió el prompt: el eco
//     vive fuera de la valla y la respuesta dentro.
//  2. Si el texto (ya sin valla) empieza por `[`, es un array de tope: se toma su
//     PRIMER objeto. Un array de tope que no cierra es salida truncada y da
//     ErrLLMQuality; nunca su primer elemento.
//  3. Si no, se prueban candidatos desde CADA `{` que abra un objeto de verdad
//     (el siguiente byte con contenido es `"` o `}`), no solo desde el primero,
//     y gana el primero que balancee y sea JSON válido. Así una prosa previa con
//     comillas descuadradas deja de tapar un JSON perfectamente bueno.
//  4. Al ganador se le aplica unwrapEnvelope UNA vez.
//
// El balanceo respeta las llaves que van dentro de una cadena y las escapadas.
// Lo custodian TestExtractJSON_LlavesDentroDeCadenaNoCuentan y
// TestExtractJSON_ProsaPreviaNoTapaElJSONBueno.
//
// Lo que ExtractJSON NO puede hacer, dicho aquí para que nadie se confíe: si el
// modelo devuelve el esquema del prompt SIN valla y sin nada más, ese eco es JSON
// válido y sale de aquí como si fuera una respuesta. Quien lo rechaza es el
// Parse* de la tarea, que valida enums y campos obligatorios. La extracción sola
// no cierra ese agujero y no pretende hacerlo.
//
// Cualquier fallo aquí es de CALIDAD, no de infraestructura: el proveedor
// respondió, lo que no sirve es lo que dijo. Todos los errores envuelven
// ErrLLMQuality.
func ExtractJSON(raw string) (json.RawMessage, error) {
	// Si hay valla con algo estructural dentro, la valla MANDA: su contenido es
	// la respuesta y lo de fuera es cháchara. No se cae de vuelta al texto entero
	// cuando la valla falla, porque ese respaldo rescataría trozos de una salida
	// truncada que la valla ya había declarado inservible.
	if fenced, ok := fencedBlock(raw); ok && strings.ContainsAny(fenced, "{[") {
		return extractFrom(fenced)
	}
	return extractFrom(raw)
}

// fencedBlock devuelve el contenido de la primera valla de código Markdown.
//
// Acepta la valla con etiqueta (```json) y sin ella, y tolera que la valla de
// cierre falte: en ese caso devuelve todo lo que hay después de la de apertura,
// que es exactamente la forma de una respuesta cortada a media generación.
func fencedBlock(s string) (string, bool) {
	abre := strings.Index(s, "```")
	if abre < 0 {
		return "", false
	}
	resto := s[abre+len("```"):]

	// La etiqueta de la valla ocupa el resto de esa línea; el contenido empieza
	// en la siguiente. Sin salto de línea no hay contenido que rescatar.
	salto := strings.IndexByte(resto, '\n')
	if salto < 0 {
		return "", false
	}
	resto = resto[salto+1:]

	if cierra := strings.Index(resto, "```"); cierra >= 0 {
		resto = resto[:cierra]
	}
	return resto, true
}

// extractFrom aplica al texto dado la política de extracción: array de tope si
// empieza por `[`, y si no, barrido de candidatos de objeto.
func extractFrom(s string) (json.RawMessage, error) {
	if trimmed := strings.TrimSpace(s); strings.HasPrefix(trimmed, "[") {
		return firstObjectOfTopArray(trimmed)
	}
	return firstObjectCandidate(s)
}

// firstObjectCandidate recorre el texto probando cada apertura de objeto y
// devuelve la primera que balancee y sea JSON válido.
//
// La regla de corte es la que sostiene la promesa «truncado ⇒ ErrLLMQuality»: un
// candidato que ABRE un objeto de verdad y nunca cierra es la firma de la salida
// cortada a media generación, así que ahí se para. Seguir buscando devolvería un
// objeto interior del artefacto roto haciéndolo pasar por el artefacto entero,
// que es peor que fallar.
func firstObjectCandidate(s string) (json.RawMessage, error) {
	for i := 0; i < len(s); i++ {
		if s[i] != '{' || !opensObject(s, i) {
			continue
		}
		cand, ok := balancedSpanAt(s, i, '{', '}')
		if !ok {
			return nil, fmt.Errorf("%w: hay un objeto JSON empezado que nunca cierra (salida truncada)", ErrLLMQuality)
		}
		if json.Valid([]byte(cand)) {
			return unwrapEnvelope(json.RawMessage(cand)), nil
		}
	}
	return nil, fmt.Errorf("%w: la salida no contiene un objeto JSON completo", ErrLLMQuality)
}

// opensObject dice si el `{` que hay en s[i] abre un objeto JSON de verdad.
//
// Un objeto solo puede seguir con una clave entrecomillada o cerrarse en el acto,
// así que el siguiente byte con contenido tiene que ser `"` o `}`. Es lo que
// distingue el `{` de un artefacto del `{` suelto que aparece en la prosa del
// modelo, y sin ese filtro una prosa con llaves abiertas se lleva por delante la
// respuesta buena que viene detrás.
func opensObject(s string, i int) bool {
	for j := i + 1; j < len(s); j++ {
		switch s[j] {
		case ' ', '\t', '\r', '\n':
			continue
		case '"', '}':
			return true
		default:
			return false
		}
	}
	return false
}

// firstObjectOfTopArray devuelve el primer objeto de un array de tope.
//
// Distingue dos cosas que se parecen y no son lo mismo: un array COMPLETO del que
// se toma el primer objeto (comportamiento documentado; quien decide si sirve es
// el Parse* de la tarea, que exigirá su version) y un array A MEDIAS, que es
// salida truncada y se rechaza entera. Devolver el primer objeto de un array
// truncado sería entregar un artefacto incompleto sin avisar.
func firstObjectOfTopArray(s string) (json.RawMessage, error) {
	span, ok := balancedSpanAt(s, 0, '[', ']')
	if !ok {
		return nil, fmt.Errorf("%w: el array de tope nunca cierra (salida truncada)", ErrLLMQuality)
	}

	var elementos []json.RawMessage
	if err := json.Unmarshal([]byte(span), &elementos); err != nil {
		return nil, fmt.Errorf("%w: el array de tope no es JSON válido: %w", ErrLLMQuality, err)
	}
	for _, e := range elementos {
		if trimmed := bytes.TrimSpace(e); len(trimmed) > 0 && trimmed[0] == '{' {
			return unwrapEnvelope(json.RawMessage(trimmed)), nil
		}
	}
	return nil, fmt.Errorf("%w: el array de tope no contiene ningún objeto", ErrLLMQuality)
}

// balancedSpanAt devuelve el tramo de s que va desde el delimitador de apertura
// en s[start] hasta el de cierre que lo equilibra.
//
// Respeta los delimitadores que caen dentro de una cadena JSON y los escapados:
// el caso del escape se resuelve ANTES que el de la comilla a propósito, porque
// al revés una comilla escapada abriría una cadena fantasma y el balanceo se
// perdería. Devuelve false si el tramo nunca cierra, que es exactamente el caso
// del JSON cortado por fin de generación.
func balancedSpanAt(s string, start int, abre, cierra byte) (string, bool) {
	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(s); i++ {
		c := s[i]
		switch {
		case escaped:
			escaped = false
		case inString && c == '\\':
			escaped = true
		case c == '"':
			inString = !inString
		case inString:
			// Dentro de una cadena los delimitadores son texto, no estructura.
		case c == abre:
			depth++
		case c == cierra:
			depth--
			if depth == 0 {
				return s[start : i+1], true
			}
		}
	}
	return "", false
}

// unwrapEnvelope quita UNA envoltura espuria del tipo {"bytes":{...}}.
//
// Solo actúa si el objeto de arriba tiene exactamente una clave, esa clave es
// una de envelopeKeys, y lo que envuelve es a su vez un objeto. Si algo de eso
// no se cumple devuelve el mensaje intacto: es preferible entregar el objeto de
// más que perder un artefacto legítimo cuya única clave se llame «data».
//
// Se desenvuelve una sola vez a propósito: dos niveles de envoltura no se han
// visto, y desenvolver en bucle abriría la puerta a devolver un trozo interior
// que no es el artefacto.
func unwrapEnvelope(msg json.RawMessage) json.RawMessage {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(msg, &top); err != nil {
		return msg
	}
	if len(top) != 1 {
		return msg
	}

	var key string
	var inner json.RawMessage
	for k, v := range top {
		key, inner = k, v
	}
	if !envelopeKeys[key] {
		return msg
	}

	// Lo envuelto tiene que ser un objeto; si es un número o una cadena, la
	// clave no era una envoltura sino un campo del artefacto.
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(inner, &probe); err != nil || probe == nil {
		return msg
	}
	return inner
}
