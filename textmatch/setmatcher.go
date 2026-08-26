package textmatch

import (
	"context"
	"strings"
)

// Policy captura la ESTRICTEZ de un match de conjunto: una decisión de negocio,
// ortogonal a cómo se compara un par.
type Policy int

const (
	// PolicyStrict exige todos los esperados cubiertos Y ningún sobrante FORÁNEO. Un
	// sobrante es foráneo solo si NO corresponde a ningún esperado vía el
	// comparador (exact/fuzzy); así un ítem extra no reconocido invalida el match,
	// pero un DUPLICADO o una VARIANTE con typo de algo esperado NO penaliza.
	PolicyStrict Policy = iota
	// PolicyLenient exige todos los esperados cubiertos; los sobrantes se ignoran (el
	// relleno de cortesía —«el famoso», «porfa»— no penaliza).
	PolicyLenient
)

// Candidate es un candidato de match con su rango sobre los tokens base del
// texto del cliente. El rango [Start, End) permite a MatchAnswer saber qué tokens
// consume un n-grama, para el chequeo de foráneos de PolicyStrict.
type Candidate struct {
	Text  string
	Start int // índice de token base inicial (inclusive)
	End   int // índice de token base final (exclusivo)
}

// GenerateCandidates produce, a partir de los tokens base, los tokens sueltos más
// los n-gramas contiguos hasta maxLen tokens. Así cada ítem encuentra su
// casi-match y los ítems multi-palabra («torta de chocolate») encuentran su
// n-grama. Orden determinista: por inicio ascendente y, dentro de cada inicio, por
// longitud ascendente (el primer Match gana, así el n-grama del largo justo se
// prefiere sin desperdiciar tokens).
func GenerateCandidates(tokens []string, maxLen int) []Candidate {
	if maxLen < 1 {
		maxLen = 1
	}
	out := make([]Candidate, 0, len(tokens)*maxLen)
	for start := 0; start < len(tokens); start++ {
		for l := 1; l <= maxLen && start+l <= len(tokens); l++ {
			out = append(out, Candidate{
				Text:  strings.Join(tokens[start:start+l], " "),
				Start: start,
				End:   start + l,
			})
		}
	}
	return out
}

// MatchReport es el resultado de un match de conjunto.
type MatchReport struct {
	Covered  []bool // por ítem esperado: ¿cubierto?
	UsedBy   []int  // por ítem esperado: índice que lo cubrió, o -1 (ver cada método)
	Leftover []int  // índices no consumidos (candidatos o tokens; ver cada método)
	Complete bool   // según la Policy
}

// SetMatcher matchea un conjunto de candidatos contra un conjunto de esperados,
// usando un Comparator (Nivel 1) por celda y una Policy de completitud.
//
// 🔴 El Comparator del bucle debe ser DETERMINISTA (típicamente
// `NewCascade(Exact{}, NewFuzzy(0)).Deterministic()`): el escalón caro nunca entra
// aquí, porque el bucle es de N×M celdas. La zona gris se cablea con WithGrayZone
// y se consulta FUERA del bucle, como mucho una vez por esperado sin cubrir.
type SetMatcher struct {
	cmp      Comparator
	policy   Policy
	grayZone GrayZone // opcional; nil ⇒ match determinista puro
}

// NewSetMatcher construye un SetMatcher con el comparador determinista
// (típicamente una Cascade) y la política de estrictez. Sin zona gris.
func NewSetMatcher(cmp Comparator, policy Policy) *SetMatcher {
	return &SetMatcher{cmp: cmp, policy: policy}
}

// WithGrayZone devuelve una COPIA del SetMatcher con gz como segunda pasada.
// Pasar nil devuelve un matcher determinista puro.
func (m *SetMatcher) WithGrayZone(gz GrayZone) *SetMatcher {
	return &SetMatcher{cmp: m.cmp, policy: m.policy, grayZone: gz}
}

// Match asigna cada esperado al mejor candidato NO usado que dé OutcomeMatch vía el
// Comparator (greedy, marca usado, empate = primer candidato para determinismo).
// Trata cada candidato como una unidad ATÓMICA: no genera n-gramas. En el
// MatchReport: UsedBy[e] = índice de candidato o -1; Leftover = índices de
// candidatos no consumidos. Para PolicyStrict, un candidato sobrante es foráneo
// solo si NO matchea ningún ESPERADO vía el comparador. Un error se propaga.
func (m *SetMatcher) Match(ctx context.Context, expected, candidates []string) (MatchReport, error) {
	used := make([]bool, len(candidates))
	rep := newReport(len(expected))
	for e, exp := range expected {
		for j, cand := range candidates {
			if used[j] {
				continue
			}
			r, err := m.cmp.Compare(ctx, exp, cand)
			if err != nil {
				return MatchReport{}, err
			}
			if r.Outcome == OutcomeMatch {
				rep.Covered[e] = true
				rep.UsedBy[e] = j
				used[j] = true
				break
			}
		}
	}
	// Segunda pasada: la zona gris, FUERA del bucle de celdas.
	if err := m.grayZoneAtomic(ctx, expected, candidates, used, &rep); err != nil {
		return MatchReport{}, err
	}
	rep.Leftover = leftoverIndices(used)
	complete, err := m.completeUnderPolicy(ctx, rep.Covered, textsAt(candidates, rep.Leftover), expected)
	if err != nil {
		return MatchReport{}, err
	}
	rep.Complete = complete
	return rep, nil
}

// MatchAnswer es el helper de alto nivel: toma el texto CRUDO del cliente, deriva
// sus tokens base, arma los candidatos (tokens + n-gramas contiguos hasta la
// longitud del esperado más largo) y hace el match greedy. A diferencia de Match,
// los índices del MatchReport son de TOKEN base: UsedBy[e] = índice del token
// inicial del span que cubrió al esperado (o -1); Leftover = índices de tokens
// base no consumidos. Para PolicyStrict, un token sobrante es foráneo solo si NO
// matchea ningún TOKEN de ningún esperado vía el comparador. Un error se propaga.
func (m *SetMatcher) MatchAnswer(ctx context.Context, expected []string, rawAnswer string) (MatchReport, error) {
	tokens := SplitTokens(rawAnswer)
	maxLen := 1
	for _, e := range expected {
		if n := len(SplitTokens(e)); n > maxLen {
			maxLen = n
		}
	}
	candidates := GenerateCandidates(tokens, maxLen)
	used := make([]bool, len(tokens))
	rep := newReport(len(expected))
	for e, exp := range expected {
		for _, c := range candidates {
			if spanUsed(used, c) {
				continue
			}
			r, err := m.cmp.Compare(ctx, exp, c.Text)
			if err != nil {
				return MatchReport{}, err
			}
			if r.Outcome == OutcomeMatch {
				rep.Covered[e] = true
				rep.UsedBy[e] = c.Start
				markSpan(used, c)
				break
			}
		}
	}
	// Segunda pasada: la zona gris, FUERA del bucle de celdas. Los candidatos que
	// se le ofrecen son los spans cuyos tokens siguen libres.
	if err := m.grayZoneSpans(ctx, expected, candidates, used, &rep); err != nil {
		return MatchReport{}, err
	}
	rep.Leftover = leftoverIndices(used)
	// El chequeo de foráneos de Strict compara cada token sobrante contra los
	// TOKENS de los esperados (no los ítems completos): así un token de un ítem
	// multi-palabra corresponde, y un duplicado casa su propio token.
	complete, err := m.completeUnderPolicy(ctx, rep.Covered, textsAt(tokens, rep.Leftover), expectedTokenSet(expected))
	if err != nil {
		return MatchReport{}, err
	}
	rep.Complete = complete
	return rep, nil
}

// grayZoneAtomic es la segunda pasada de Match: por cada esperado que quedó sin
// cubrir consulta la zona gris UNA vez, ofreciéndole los candidatos que siguen
// libres. Si no hay zona gris cableada no hace nada (determinista puro). Un índice
// fuera de rango o negativo se lee como «ninguno corresponde». Coste acotado: como
// mucho una llamada por esperado sin cubrir, nunca por celda.
func (m *SetMatcher) grayZoneAtomic(ctx context.Context, expected, candidates []string, used []bool, rep *MatchReport) error {
	if m.grayZone == nil {
		return nil
	}
	for e, covered := range rep.Covered {
		if covered {
			continue
		}
		free := leftoverIndices(used)
		if len(free) == 0 {
			return nil // no queda nada que ofrecer: preguntar sería gastar por gastar
		}
		d, err := m.grayZone.Resolve(ctx, expected[e], textsAt(candidates, free))
		if err != nil {
			return err
		}
		if d.Index < 0 || d.Index >= len(free) {
			continue
		}
		pick := free[d.Index]
		rep.Covered[e] = true
		rep.UsedBy[e] = pick
		used[pick] = true
	}
	return nil
}

// grayZoneSpans es la segunda pasada de MatchAnswer: igual que grayZoneAtomic,
// pero los candidatos son spans de tokens y consumir uno marca todo su rango.
func (m *SetMatcher) grayZoneSpans(ctx context.Context, expected []string, candidates []Candidate, used []bool, rep *MatchReport) error {
	if m.grayZone == nil {
		return nil
	}
	for e, covered := range rep.Covered {
		if covered {
			continue
		}
		free := freeSpans(candidates, used)
		if len(free) == 0 {
			return nil
		}
		d, err := m.grayZone.Resolve(ctx, expected[e], spanTexts(free))
		if err != nil {
			return err
		}
		if d.Index < 0 || d.Index >= len(free) {
			continue
		}
		c := free[d.Index]
		rep.Covered[e] = true
		rep.UsedBy[e] = c.Start
		markSpan(used, c)
	}
	return nil
}

// freeSpans devuelve los candidatos cuyos tokens siguen todos sin consumir.
func freeSpans(candidates []Candidate, used []bool) []Candidate {
	out := make([]Candidate, 0, len(candidates))
	for _, c := range candidates {
		if !spanUsed(used, c) {
			out = append(out, c)
		}
	}
	return out
}

// spanTexts proyecta el texto de cada candidato.
func spanTexts(candidates []Candidate) []string {
	out := make([]string, len(candidates))
	for i, c := range candidates {
		out[i] = c.Text
	}
	return out
}

// newReport inicializa un MatchReport con UsedBy en -1 (nadie cubre aún).
func newReport(n int) MatchReport {
	rep := MatchReport{Covered: make([]bool, n), UsedBy: make([]int, n)}
	for e := range rep.UsedBy {
		rep.UsedBy[e] = -1
	}
	return rep
}

// spanUsed indica si algún token del rango del candidato ya fue consumido.
func spanUsed(used []bool, c Candidate) bool {
	for k := c.Start; k < c.End; k++ {
		if used[k] {
			return true
		}
	}
	return false
}

// markSpan marca como consumidos todos los tokens del rango del candidato.
func markSpan(used []bool, c Candidate) {
	for k := c.Start; k < c.End; k++ {
		used[k] = true
	}
}

// leftoverIndices devuelve los índices marcados como no usados.
func leftoverIndices(used []bool) []int {
	var out []int
	for i, u := range used {
		if !u {
			out = append(out, i)
		}
	}
	return out
}

// completeUnderPolicy aplica la Policy. Lenient: basta con todos los esperados
// cubiertos (sobrantes ignorados). Strict: además ningún sobrante puede ser
// FORÁNEO; un sobrante es foráneo si no matchea NINGUNA referencia vía el
// comparador (DETERMINISTA: la zona gris no participa de este chequeo, para que su
// coste siga acotado a los esperados sin cubrir). references son los ítems
// esperados (en Match) o sus tokens (en MatchAnswer).
func (m *SetMatcher) completeUnderPolicy(ctx context.Context, covered []bool, leftoverTexts, references []string) (bool, error) {
	if !allCovered(covered) {
		return false, nil
	}
	if m.policy != PolicyStrict {
		return true, nil
	}
	for _, text := range leftoverTexts {
		matched, err := m.matchesAny(ctx, text, references)
		if err != nil {
			return false, err
		}
		if !matched {
			return false, nil // sobrante que no corresponde a ningún esperado: foráneo
		}
	}
	return true, nil
}

// matchesAny reporta si target da OutcomeMatch contra alguna referencia vía el
// comparador (la referencia es el "expected", target el "candidate"). Corta en el
// primer match; un error se propaga.
func (m *SetMatcher) matchesAny(ctx context.Context, target string, references []string) (bool, error) {
	for _, ref := range references {
		r, err := m.cmp.Compare(ctx, ref, target)
		if err != nil {
			return false, err
		}
		if r.Outcome == OutcomeMatch {
			return true, nil
		}
	}
	return false, nil
}

// allCovered reporta si todos los esperados quedaron cubiertos.
func allCovered(covered []bool) bool {
	for _, c := range covered {
		if !c {
			return false
		}
	}
	return true
}

// textsAt proyecta los textos de all en los índices dados.
func textsAt(all []string, idx []int) []string {
	out := make([]string, len(idx))
	for i, j := range idx {
		out[i] = all[j]
	}
	return out
}

// expectedTokenSet es la unión (sin duplicados, orden de aparición) de los tokens
// de todos los ítems esperados. Es la referencia del chequeo de foráneos por token
// de MatchAnswer.
func expectedTokenSet(expected []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, e := range expected {
		for _, t := range SplitTokens(e) {
			if _, ok := seen[t]; ok {
				continue
			}
			seen[t] = struct{}{}
			out = append(out, t)
		}
	}
	return out
}
