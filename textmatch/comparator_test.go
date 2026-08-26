package textmatch_test

import (
	"context"
	"errors"
	"testing"

	"github.com/EduGoGroup/wapp-shared/textmatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// zonaGrisContada es el doble de la estrategia inyectada (en producción, el LLM).
// Cuenta sus invocaciones: es el instrumento con el que se demuestra que el
// escalón caro NO se paga cuando Exact o Fuzzy ya resolvieron.
type zonaGrisContada struct {
	llamadas    int
	expuestos   []string   // el «expected» de cada llamada
	ofrecidos   [][]string // los candidatos ofrecidos en cada llamada
	elige       int        // índice que devuelve; -1 = ninguno
	eligeTexto  string     // si no es "", elige el candidato con ese texto (o ninguno)
	fallaCon    error      // si no es nil, Resolve devuelve este error
	confidencia float64
}

func (z *zonaGrisContada) Name() string { return "fake-zona-gris" }

func (z *zonaGrisContada) Resolve(_ context.Context, expected string, candidates []string) (textmatch.GrayZoneDecision, error) {
	z.llamadas++
	z.expuestos = append(z.expuestos, expected)
	z.ofrecidos = append(z.ofrecidos, candidates)
	if z.fallaCon != nil {
		return textmatch.GrayZoneDecision{}, z.fallaCon
	}
	if z.eligeTexto != "" {
		idx := -1
		for i, c := range candidates {
			if c == z.eligeTexto {
				idx = i
				break
			}
		}
		return textmatch.GrayZoneDecision{Index: idx, Confidence: z.confidencia, Evidence: "juicio del doble por texto"}, nil
	}
	return textmatch.GrayZoneDecision{Index: z.elige, Confidence: z.confidencia, Evidence: "juicio del doble"}, nil
}

func TestExactCompare(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name           string
		expected, cand string
		wantOutcome    textmatch.Outcome
	}{
		{"igual tras normalizar", "Tequeños", "tequeños", textmatch.OutcomeMatch},
		{"tildes igualan", "Café", "cafe", textmatch.OutcomeMatch},
		{"ñ distingue", "año", "ano", textmatch.OutcomeNoMatch},
		{"typo no casa exacto", "whatsapp", "whastapp", textmatch.OutcomeNoMatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := textmatch.Exact{}.Compare(ctx, tt.expected, tt.cand)
			require.NoError(t, err)
			assert.Equal(t, tt.wantOutcome, r.Outcome)
			assert.Equal(t, "exact", r.Strategy)
		})
	}
}

// TestCriterio1WhastappSimilitud es el CRITERIO 1 del Plan 044 · T3.1: «whastapp»
// ≈ «whatsapp» con similitud OSA 0,875 ≥ 0,85 ⇒ match en Fuzzy. Los números están
// escritos a mano (no derivados de las constantes del paquete): 8 runas, 1 edición
// —la transposición s↔t—, 1 - 1/8 = 0,875.
func TestCriterio1WhastappSimilitud(t *testing.T) {
	ctx := context.Background()

	require.Equal(t, 1, textmatch.EditDistance("whatsapp", "whastapp"), "la transposición es UNA edición")
	require.Len(t, []rune("whatsapp"), 8)

	r, err := textmatch.NewFuzzy(0.85).Compare(ctx, "whatsapp", "whastapp")
	require.NoError(t, err)
	assert.InDelta(t, 0.875, r.Confidence, 1e-9, "similitud esperada 1 - 1/8")
	assert.Equal(t, textmatch.OutcomeMatch, r.Outcome, "0,875 ≥ 0,85 debe casar")
	assert.Equal(t, "fuzzy", r.Strategy)

	// Y el escalón anterior NO lo resuelve: por eso hace falta el fuzzy.
	rx, err := textmatch.Exact{}.Compare(ctx, "whatsapp", "whastapp")
	require.NoError(t, err)
	require.Equal(t, textmatch.OutcomeNoMatch, rx.Outcome)
}

// TestCriterio2NoquisSinEnye es el CRITERIO 2: «ñoquis» ≠ «noquis» en Exact (la ñ
// se preserva) pero SÍ casan en Fuzzy.
//
// 🔴 ADVERTENCIA MEDIDA: la segunda mitad NO se sostiene con el umbral por defecto
// 0,85. La aritmética es fija: «ñoquis» son 6 runas y la ñ→n es 1 edición, así que
// la similitud es 1 - 1/6 = 0,8333…, por debajo de 0,85. Con 0,85 el par NO casa;
// casa a partir de un umbral ≤ 0,8333. El test fija AMBOS hechos con literales
// escritos a mano para que ninguno se pueda mover en silencio.
func TestCriterio2NoquisSinEnye(t *testing.T) {
	ctx := context.Background()

	t.Run("Exact NO iguala: la ñ se preserva", func(t *testing.T) {
		r, err := textmatch.Exact{}.Compare(ctx, "ñoquis", "noquis")
		require.NoError(t, err)
		require.Equal(t, textmatch.OutcomeNoMatch, r.Outcome)
		require.NotEqual(t, textmatch.Normalize("noquis"), textmatch.Normalize("ñoquis"))
	})

	t.Run("Fuzzy SÍ casa (umbral 0,80)", func(t *testing.T) {
		r, err := textmatch.NewFuzzy(0.80).Compare(ctx, "ñoquis", "noquis")
		require.NoError(t, err)
		assert.InDelta(t, 0.8333333, r.Confidence, 1e-6, "similitud esperada 1 - 1/6")
		assert.Equal(t, textmatch.OutcomeMatch, r.Outcome)
	})

	t.Run("con el umbral por defecto 0,85 NO llega", func(t *testing.T) {
		r, err := textmatch.NewFuzzy(0.85).Compare(ctx, "ñoquis", "noquis")
		require.NoError(t, err)
		assert.InDelta(t, 0.8333333, r.Confidence, 1e-6)
		assert.Equal(t, textmatch.OutcomeNoMatch, r.Outcome,
			"0,8333 < 0,85: el criterio del plan no se sostiene a 0,85")
	})
}

func TestFuzzyCompare(t *testing.T) {
	ctx := context.Background()
	f := textmatch.NewFuzzy(0.85)
	tests := []struct {
		name           string
		expected, cand string
		wantOutcome    textmatch.Outcome
	}{
		{"typo transpuesto whatsapp", "whatsapp", "whastapp", textmatch.OutcomeMatch},   // sim 0,875
		{"typo inserción instagram", "instagram", "instalgram", textmatch.OutcomeMatch}, // sim 0,9
		{"dos ítems distintos", "tequeños", "empanadas", textmatch.OutcomeNoMatch},
		{"ñ no se cuela bajo umbral", "año", "ano", textmatch.OutcomeNoMatch}, // sim 0,667
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := f.Compare(ctx, tt.expected, tt.cand)
			require.NoError(t, err)
			assert.Equal(t, tt.wantOutcome, r.Outcome, "confianza %.3f", r.Confidence)
		})
	}
}

// TestNewFuzzyDefault confirma que un threshold no positivo cae al default y que
// ese default rescata el typo transpuesto.
func TestNewFuzzyDefault(t *testing.T) {
	ctx := context.Background()
	for _, f := range []textmatch.Fuzzy{textmatch.NewFuzzy(0), textmatch.NewFuzzy(-1), {}} {
		r, err := f.Compare(ctx, "whatsapp", "whastapp")
		require.NoError(t, err)
		assert.Equal(t, textmatch.OutcomeMatch, r.Outcome)
	}
	assert.InDelta(t, 0.85, textmatch.DefaultFuzzyThreshold, 1e-9)
}

// TestCriterio3CascadaSinZonaGris es el CRITERIO 3: sin estrategia inyectada la
// cascada funciona, no panica, no falla y es DETERMINISTA PURA.
func TestCriterio3CascadaSinZonaGris(t *testing.T) {
	ctx := context.Background()
	casos := []struct {
		nombre         string
		cascada        *textmatch.Cascade
		expected, cand string
		wantOutcome    textmatch.Outcome
		wantStrategy   string
	}{
		{"nunca cableada · match exacto", textmatch.NewCascade(textmatch.Exact{}, textmatch.NewFuzzy(0.85)), "Tequeños", "tequeños", textmatch.OutcomeMatch, "exact"},
		{"nunca cableada · rescate fuzzy", textmatch.NewCascade(textmatch.Exact{}, textmatch.NewFuzzy(0.85)), "whatsapp", "whastapp", textmatch.OutcomeMatch, "fuzzy"},
		{"nunca cableada · sin match", textmatch.NewCascade(textmatch.Exact{}, textmatch.NewFuzzy(0.85)), "tequeños", "empanadas", textmatch.OutcomeNoMatch, "fuzzy"},
		{"cableada a nil explícito", textmatch.NewCascade(textmatch.Exact{}, textmatch.NewFuzzy(0.85)).WithGrayZone(nil), "tequeños", "empanadas", textmatch.OutcomeNoMatch, "fuzzy"},
		{"solo Exact, sin más escalones", textmatch.NewCascade(textmatch.Exact{}), "tequeños", "empanadas", textmatch.OutcomeNoMatch, "exact"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			require.False(t, c.cascada.HasGrayZone())
			r, err := c.cascada.Compare(ctx, c.expected, c.cand)
			require.NoError(t, err, "sin zona gris la cascada no puede fallar")
			assert.Equal(t, c.wantOutcome, r.Outcome)
			assert.Equal(t, c.wantStrategy, r.Strategy)

			// Determinista: mismo par, mismo veredicto, siempre.
			for i := 0; i < 50; i++ {
				otra, err := c.cascada.Compare(ctx, c.expected, c.cand)
				require.NoError(t, err)
				require.Equal(t, r, otra, "la cascada determinista debe repetirse idéntica")
			}
		})
	}
}

// TestCascadaVaciaNoPanica: el caso degenerado (ninguna estrategia, ninguna zona
// gris) devuelve el cero de Result sin panicar.
func TestCascadaVaciaNoPanica(t *testing.T) {
	r, err := textmatch.NewCascade().Compare(context.Background(), "a", "b")
	require.NoError(t, err)
	assert.Equal(t, textmatch.Result{}, r)
}

// TestContadorZonaGrisEnLaCascada mide con el contador del doble que el escalón
// caro NO se invoca cuando Exact o Fuzzy ya resolvieron, y que se invoca UNA vez
// cuando ninguno de los dos acertó.
func TestContadorZonaGrisEnLaCascada(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name           string
		expected, cand string
		elige          int
		wantLlamadas   int
		wantOutcome    textmatch.Outcome
		wantStrategy   string
	}{
		{"resuelto por Exact ⇒ 0 llamadas", "Tequeños", "tequeños", 0, 0, textmatch.OutcomeMatch, "exact"},
		{"resuelto por Fuzzy ⇒ 0 llamadas", "whatsapp", "whastapp", 0, 0, textmatch.OutcomeMatch, "fuzzy"},
		{"zona gris acepta ⇒ 1 llamada", "torta de chocolate", "el pastel oscuro", 0, 1, textmatch.OutcomeMatch, "fake-zona-gris"},
		{"zona gris rechaza ⇒ 1 llamada", "torta de chocolate", "servilletas", -1, 1, textmatch.OutcomeNoMatch, "fake-zona-gris"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &zonaGrisContada{elige: tt.elige, confidencia: 0.7}
			c := textmatch.NewCascade(textmatch.Exact{}, textmatch.NewFuzzy(0.85)).WithGrayZone(fake)
			require.True(t, c.HasGrayZone())

			r, err := c.Compare(ctx, tt.expected, tt.cand)
			require.NoError(t, err)
			assert.Equal(t, tt.wantLlamadas, fake.llamadas, "invocaciones del escalón caro")
			assert.Equal(t, tt.wantOutcome, r.Outcome)
			assert.Equal(t, tt.wantStrategy, r.Strategy)
		})
	}
}

// TestCascadaZonaGrisPropagaError: un fallo transitorio del escalón caro se
// propaga; el caller decide si reintenta.
func TestCascadaZonaGrisPropagaError(t *testing.T) {
	sentinela := errors.New("fallo transitorio del proveedor")
	fake := &zonaGrisContada{fallaCon: sentinela}
	c := textmatch.NewCascade(textmatch.Exact{}, textmatch.NewFuzzy(0.85)).WithGrayZone(fake)

	_, err := c.Compare(context.Background(), "torta", "servilletas")
	require.ErrorIs(t, err, sentinela)
	assert.Equal(t, 1, fake.llamadas)
}

// TestDeterministicRetiraLaZonaGris: la copia determinista de una cascada cableada
// no consulta al escalón caro. Es lo que se le pasa al SetMatcher.
func TestDeterministicRetiraLaZonaGris(t *testing.T) {
	fake := &zonaGrisContada{elige: 0}
	c := textmatch.NewCascade(textmatch.Exact{}, textmatch.NewFuzzy(0.85)).WithGrayZone(fake)
	det := c.Deterministic()

	require.False(t, det.HasGrayZone())
	r, err := det.Compare(context.Background(), "torta", "servilletas")
	require.NoError(t, err)
	assert.Equal(t, textmatch.OutcomeNoMatch, r.Outcome)
	assert.Equal(t, 0, fake.llamadas)
	assert.True(t, c.HasGrayZone(), "la cascada original no se muta")
}

// errStrategy es un doble determinista que falla, para probar la propagación de
// error y el corte de la cascada.
type errStrategy struct{ llamadas *int }

func (errStrategy) Name() string { return "err-stub" }
func (s errStrategy) Compare(_ context.Context, _, _ string) (textmatch.Result, error) {
	*s.llamadas++
	return textmatch.Result{}, errors.New("fallo de la estrategia")
}

func TestCascadaEscalado(t *testing.T) {
	ctx := context.Background()

	t.Run("positivo corta sin llamar a la siguiente", func(t *testing.T) {
		llamadas := 0
		c := textmatch.NewCascade(textmatch.Exact{}, errStrategy{llamadas: &llamadas})
		r, err := c.Compare(ctx, "tequeños", "tequeños")
		require.NoError(t, err)
		assert.Equal(t, textmatch.OutcomeMatch, r.Outcome)
		assert.Equal(t, "exact", r.Strategy)
		assert.Zero(t, llamadas, "la estrategia siguiente no debía invocarse tras un positivo")
	})

	t.Run("error se propaga", func(t *testing.T) {
		llamadas := 0
		c := textmatch.NewCascade(errStrategy{llamadas: &llamadas})
		_, err := c.Compare(ctx, "a", "b")
		require.Error(t, err)
		assert.Equal(t, 1, llamadas)
	})

	t.Run("agotada devuelve el último negativo", func(t *testing.T) {
		c := textmatch.NewCascade(textmatch.Exact{}, textmatch.NewFuzzy(0.99))
		r, err := c.Compare(ctx, "tequeños", "empanadas")
		require.NoError(t, err)
		assert.Equal(t, textmatch.OutcomeNoMatch, r.Outcome)
		assert.Equal(t, "fuzzy", r.Strategy)
	})
}
