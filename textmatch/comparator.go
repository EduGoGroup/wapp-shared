package textmatch

import (
	"context"
	"fmt"
)

// Outcome es el veredicto de una estrategia sobre UN par (expected, candidate).
type Outcome int

// Los valores de Outcome gobiernan el escalado de la cascada.
const (
	OutcomeNoMatch   Outcome = iota // negativo → escala a la siguiente estrategia
	OutcomeMatch                    // positivo → corta la cascada
	OutcomeUncertain                // incierto → escala a la siguiente estrategia
)

// DefaultFuzzyThreshold es el umbral de similitud por defecto del fuzzy (0,85),
// fijado por el Plan 044 · T3.1. Conservador a propósito: estricto con lo que no
// corresponde, tolerante con los typos de lo que sí.
const DefaultFuzzyThreshold = 0.85

// Result es el resultado de una comparación de un par.
type Result struct {
	Outcome    Outcome
	Confidence float64 // 0..1
	Evidence   string  // por qué (legible: para el dueño del negocio y para depurar)
	Strategy   string  // qué estrategia lo produjo (procedencia)
}

// Strategy es una estrategia de comparación de un par. Contrato mínimo y estable
// (ISP: dos métodos). El ctx + error acomodan estrategias cancelables y
// transitorias; las deterministas nunca devuelven error.
type Strategy interface {
	Name() string
	Compare(ctx context.Context, expected, candidate string) (Result, error)
}

// Comparator es lo que consume el SetMatcher para cada celda. Cascade lo
// implementa; también lo implementa una Strategy suelta (misma firma de Compare).
type Comparator interface {
	Compare(ctx context.Context, expected, candidate string) (Result, error)
}

// GrayZoneDecision es lo que devuelve la zona gris: cuál de los candidatos que se
// le ofrecieron corresponde al esperado, o ninguno.
type GrayZoneDecision struct {
	Index      int     // índice en candidates, o -1 si NINGUNO corresponde
	Confidence float64 // 0..1
	Evidence   string  // por qué, en texto legible
}

// GrayZone es el escalón caro de la cascada: el que juzga lo que ni Exact ni Fuzzy
// resolvieron. 🔴 textmatch NO lo implementa y NO importa `wapp-shared/llm` (DIP,
// Plan 044 · T3.1): el adaptador LLM vive donde vive el provider y se INYECTA aquí.
//
// Resolve recibe el esperado y TODOS los candidatos que siguen en juego, y elige
// uno o ninguno. Esa forma —conjunto, no par— es deliberada: permite que el
// SetMatcher la consulte UNA sola vez por esperado sin cubrir, fuera del bucle.
type GrayZone interface {
	Name() string
	Resolve(ctx context.Context, expected string, candidates []string) (GrayZoneDecision, error)
}

// Exact es la estrategia de igualdad de strings normalizados.
type Exact struct{}

// Name identifica la estrategia (procedencia en Result.Strategy).
func (Exact) Name() string { return "exact" }

// Compare devuelve Match/1.0 si expected y candidate normalizan al mismo texto.
func (Exact) Compare(_ context.Context, expected, candidate string) (Result, error) {
	if Normalize(expected) == Normalize(candidate) {
		return Result{Outcome: OutcomeMatch, Confidence: 1.0, Evidence: "iguales tras normalizar", Strategy: "exact"}, nil
	}
	return Result{Outcome: OutcomeNoMatch, Confidence: 0.0, Evidence: "distintos tras normalizar", Strategy: "exact"}, nil
}

// Fuzzy es la estrategia ortográfica: distancia de edición normalizada por runas
// sobre los textos normalizados; sim = 1 - dist/maxLen. Es el escalón del medio
// entre el match exacto y el juicio de la zona gris.
type Fuzzy struct {
	// Threshold es la similitud mínima para Match (0..1). Si es <= 0 se usa
	// DefaultFuzzyThreshold, así el literal Fuzzy{} también es seguro.
	Threshold float64
}

// NewFuzzy construye un Fuzzy; threshold <= 0 cae al default DefaultFuzzyThreshold.
func NewFuzzy(threshold float64) Fuzzy {
	if threshold <= 0 {
		threshold = DefaultFuzzyThreshold
	}
	return Fuzzy{Threshold: threshold}
}

// Name identifica la estrategia (procedencia en Result.Strategy).
func (Fuzzy) Name() string { return "fuzzy" }

// Compare produce Match si la similitud alcanza el umbral (Confidence = sim), o
// NoMatch en caso contrario (Confidence = sim igualmente, para depurar). No emite
// OutcomeUncertain: la banda incierta queda para la zona gris.
func (f Fuzzy) Compare(_ context.Context, expected, candidate string) (Result, error) {
	threshold := f.Threshold
	if threshold <= 0 {
		threshold = DefaultFuzzyThreshold
	}
	e, c := Normalize(expected), Normalize(candidate)
	if e == c {
		return Result{Outcome: OutcomeMatch, Confidence: 1.0, Evidence: "iguales tras normalizar", Strategy: "fuzzy"}, nil
	}
	maxLen := len([]rune(e))
	if n := len([]rune(c)); n > maxLen {
		maxLen = n
	}
	if maxLen == 0 {
		return Result{Outcome: OutcomeMatch, Confidence: 1.0, Evidence: "ambos vacíos", Strategy: "fuzzy"}, nil
	}
	sim := 1.0 - float64(EditDistance(e, c))/float64(maxLen)
	outcome := OutcomeNoMatch
	if sim >= threshold {
		outcome = OutcomeMatch
	}
	return Result{
		Outcome:    outcome,
		Confidence: sim,
		Evidence:   fmt.Sprintf("similitud %.3f (umbral %.3f)", sim, threshold),
		Strategy:   "fuzzy",
	}, nil
}

// Cascade orquesta una lista ordenada de estrategias deterministas (barata→cara)
// con escalado explícito —positivo corta; incierto/negativo escala; un error se
// propaga— y, opcionalmente, un ÚLTIMO escalón de zona gris inyectado (WithGrayZone).
//
// Sin zona gris la cascada es DETERMINISTA PURA: no hay tercer escalón, no panica
// y no falla. Si se agota sin positivo devuelve el último Result (negativo o
// incierto): qué significa ese no-match lo decide el caller, no el motor.
// Implementa Comparator.
type Cascade struct {
	strategies []Strategy
	grayZone   GrayZone // opcional; nil ⇒ cascada determinista pura
}

// NewCascade construye una Cascade con las estrategias deterministas en orden
// barata→cara. Sin zona gris (ver WithGrayZone).
func NewCascade(strategies ...Strategy) *Cascade {
	return &Cascade{strategies: strategies}
}

// WithGrayZone devuelve una COPIA de la cascada con gz como último escalón. Pasar
// nil devuelve una cascada determinista pura, así el caller puede cablear
// «con LLM si hay proveedor, sin él si no» sin ramas.
func (c *Cascade) WithGrayZone(gz GrayZone) *Cascade {
	return &Cascade{strategies: c.strategies, grayZone: gz}
}

// Deterministic devuelve una COPIA de la cascada SIN el escalón de zona gris. Es
// lo que se le pasa al SetMatcher para el bucle: el escalón caro se consulta
// aparte, una vez por esperado sin cubrir (ver SetMatcher.WithGrayZone).
func (c *Cascade) Deterministic() *Cascade {
	return &Cascade{strategies: c.strategies}
}

// HasGrayZone informa si la cascada tiene el tercer escalón cableado.
func (c *Cascade) HasGrayZone() bool { return c.grayZone != nil }

// Compare recorre las estrategias hasta el primer Match y, si ninguna acertó y hay
// zona gris, la consulta con ese único candidato. Ver Cascade.
func (c *Cascade) Compare(ctx context.Context, expected, candidate string) (Result, error) {
	var last Result
	for _, s := range c.strategies {
		r, err := s.Compare(ctx, expected, candidate)
		if err != nil {
			return Result{}, err
		}
		if r.Outcome == OutcomeMatch {
			return r, nil
		}
		last = r
	}
	if c.grayZone == nil {
		return last, nil
	}
	d, err := c.grayZone.Resolve(ctx, expected, []string{candidate})
	if err != nil {
		return Result{}, err
	}
	outcome := OutcomeNoMatch
	if d.Index == 0 {
		outcome = OutcomeMatch
	}
	return Result{
		Outcome:    outcome,
		Confidence: d.Confidence,
		Evidence:   d.Evidence,
		Strategy:   c.grayZone.Name(),
	}, nil
}
