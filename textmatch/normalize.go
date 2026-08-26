package textmatch

import (
	"regexp"
	"strings"
	"unicode"
)

// nTildeDescompuesta es la «ñ» en forma descompuesta (NFD): una «n» seguida de la
// tilde combinante U+0303. Se recompone ANTES del barrido de marcas porque el
// barrido borraría la tilde y dejaría una «n» pelada, colapsando «año» con «ano».
const nTildeDescompuesta = "n\u0303"

// pliegueDiacritico mapea cada letra latina acentuada a su letra base. Sustituye
// a la cadena NFD→remove(Mn)→NFC de `golang.org/x/text` del original de EduGo: el
// módulo no puede tener más dependencia que testify, así que el plegado es propio
// y de stdlib.
//
// 🔴 La «ñ» NO aparece en ninguna fila a propósito (Plan 044 · T3.1): es una letra
// propia del español, no una «n» con tilde. Al no estar en la tabla y no ser una
// marca combinante, Normalize la deja intacta. El invariante lo custodia un test
// sobre la propia tabla, no un comentario.
var pliegueDiacritico = construirPliegue(map[rune]string{
	'a': "áàâäãåāăą",
	'c': "çćĉċč",
	'd': "ďđ",
	'e': "éèêëēĕėęě",
	'g': "ĝğġģ",
	'h': "ĥħ",
	'i': "íìîïĩīĭįı",
	'j': "ĵ",
	'k': "ķ",
	'l': "ĺļľłŀ",
	'n': "ńņňŉ",
	'o': "óòôöõøōŏő",
	'r': "ŕŗř",
	's': "śŝşš",
	't': "ţťŧ",
	'u': "úùûüũūŭůűų",
	'w': "ŵ",
	'y': "ýÿŷ",
	'z': "źżž",
})

// construirPliegue invierte la tabla «letra base → sus acentuadas» a la forma que
// consulta Normalize («acentuada → letra base»). Declararla por base evita que la
// tabla se desincronice por posición.
func construirPliegue(porBase map[rune]string) map[rune]rune {
	out := make(map[rune]rune)
	for base, acentuadas := range porBase {
		for _, r := range acentuadas {
			out[r] = base
		}
	}
	return out
}

// connectorTokens son las palabras conectoras que separan ítems en el texto del
// cliente pero no forman parte de ningún ítem («tequeños y una torta» → dos
// ítems). Se descartan tras el split; nunca casan un ítem.
var connectorTokens = map[string]struct{}{"y": {}, "e": {}}

// nonAlnum casa runs de caracteres que no son letra ni dígito (unicode). Toda la
// puntuación y los separadores («,» «|» «;») caen aquí y actúan como frontera de
// token, así "tequeños, torta" no pega "tequeños," ≠ "tequeños".
var nonAlnum = regexp.MustCompile(`[^\p{L}\p{N}]+`)

// Normalize es el contrato de normalización canónico: minúsculas, sin
// tildes/diéresis, PRESERVA la «ñ» (es letra, no tilde), colapsa los espacios
// internos y hace trim. Es una función pura.
//
// El barrido tiene dos patas complementarias: las marcas combinantes sueltas
// (categoría Unicode Mn, texto que llega descompuesto) se descartan, y las letras
// precompuestas se pliegan por tabla. La «ñ» no es ninguna de las dos cosas, y por
// eso sobrevive sin necesitar un caso especial.
func Normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, nTildeDescompuesta, "ñ")

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if base, ok := pliegueDiacritico[r]; ok {
			r = base
		}
		b.WriteRune(r)
	}
	// strings.Fields colapsa cualquier run de espacios y descarta los extremos.
	return strings.Join(strings.Fields(b.String()), " ")
}

// SplitTokens descompone un texto en tokens normalizados para el match de listas.
// Es pura. Reglas: aplica Normalize, usa como frontera todo carácter no
// alfanumérico unicode (incluida la puntuación), descarta las conectoras «y»/«e»
// (separan ítems, no son ítems) y colapsa los tokens vacíos.
func SplitTokens(s string) []string {
	normalized := Normalize(s)
	raw := nonAlnum.Split(normalized, -1)
	out := make([]string, 0, len(raw))
	for _, tok := range raw {
		if tok == "" {
			continue
		}
		if _, isConnector := connectorTokens[tok]; isConnector {
			continue
		}
		out = append(out, tok)
	}
	return out
}
