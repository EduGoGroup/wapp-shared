package textmatch_test

import (
	"context"
	"errors"
	"testing"

	"github.com/EduGoGroup/wapp-shared/textmatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cascada085 es el comparador determinista de los dos escalones: exacto + fuzzy 0,85.
func cascada085() *textmatch.Cascade {
	return textmatch.NewCascade(textmatch.Exact{}, textmatch.NewFuzzy(0.85))
}

func TestMatchAnswerPedidoTipico(t *testing.T) {
	ctx := context.Background()
	expected := []string{"tequeños", "torta", "empanadas"}
	// «tequenos» sin ñ: 8 runas, 1 edición ⇒ 0,875 ≥ 0,85, lo rescata el fuzzy.
	pedido := "quiero tequenos, una torta y empanadas"

	t.Run("Lenient completa sin zona gris", func(t *testing.T) {
		m := textmatch.NewSetMatcher(cascada085(), textmatch.PolicyLenient)
		rep, err := m.MatchAnswer(ctx, expected, pedido)
		require.NoError(t, err)
		assert.Equal(t, []bool{true, true, true}, rep.Covered)
		assert.True(t, rep.Complete, "Lenient debía completar: typos rescatados, relleno ignorado")
	})

	t.Run("Strict no completa por tokens foráneos", func(t *testing.T) {
		m := textmatch.NewSetMatcher(cascada085(), textmatch.PolicyStrict)
		rep, err := m.MatchAnswer(ctx, expected, pedido)
		require.NoError(t, err)
		assert.Equal(t, []bool{true, true, true}, rep.Covered)
		assert.False(t, rep.Complete, "Strict NO debía completar: sobran «quiero» y «una»")
		assert.Len(t, rep.Leftover, 2)
	})
}

func TestMatchAnswerMultiPalabra(t *testing.T) {
	ctx := context.Background()
	m := textmatch.NewSetMatcher(cascada085(), textmatch.PolicyStrict)
	rep, err := m.MatchAnswer(ctx, []string{"torta de chocolate", "tequeños"}, "torta de chocolate y tequeños")
	require.NoError(t, err)
	assert.True(t, rep.Complete, "el ítem multi-palabra debía casar vía n-grama")
	assert.Equal(t, []int{0, 3}, rep.UsedBy)
}

func TestMatchAnswerItemForaneoStrictVsLenient(t *testing.T) {
	ctx := context.Background()
	expected := []string{"tequeños", "empanadas", "torta"}
	pedido := "tequeños empanadas torta y servilletas"

	strict := textmatch.NewSetMatcher(cascada085(), textmatch.PolicyStrict)
	repS, err := strict.MatchAnswer(ctx, expected, pedido)
	require.NoError(t, err)
	assert.False(t, repS.Complete, "«servilletas» es un ítem extra foráneo")

	lenient := textmatch.NewSetMatcher(cascada085(), textmatch.PolicyLenient)
	repL, err := lenient.MatchAnswer(ctx, expected, pedido)
	require.NoError(t, err)
	assert.True(t, repL.Complete, "Lenient ignora el sobrante")
}

func TestMatchAnswerFaltanteReal(t *testing.T) {
	ctx := context.Background()
	expected := []string{"tequeños", "empanadas", "torta"}
	for _, policy := range []textmatch.Policy{textmatch.PolicyStrict, textmatch.PolicyLenient} {
		m := textmatch.NewSetMatcher(cascada085(), policy)
		rep, err := m.MatchAnswer(ctx, expected, "tequeños empanadas")
		require.NoError(t, err)
		assert.False(t, rep.Complete, "policy %v: no debía completar con un esperado ausente", policy)
		assert.Equal(t, -1, rep.UsedBy[2])
	}
}

// TestMatchAnswerForaneosVsDuplicados cubre la semántica de foráneos de Strict: un
// sobrante es foráneo solo si no corresponde a ningún esperado.
func TestMatchAnswerForaneosVsDuplicados(t *testing.T) {
	ctx := context.Background()
	expected := []string{"tequeños", "empanadas", "torta"}
	tests := []struct {
		name         string
		pedido       string
		wantComplete bool
	}{
		{"relleno foráneo invalida", "tequeños empanadas torta porfa", false},
		{"ítem extra real invalida", "tequeños empanadas torta y servilletas", false},
		{"duplicado exacto no penaliza", "tequeños tequeños empanadas torta", true},
		{"duplicado con typo casa por fuzzy y no penaliza", "tequeños tequenos empanadas torta", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := textmatch.NewSetMatcher(cascada085(), textmatch.PolicyStrict)
			rep, err := m.MatchAnswer(ctx, expected, tt.pedido)
			require.NoError(t, err)
			assert.Equal(t, []bool{true, true, true}, rep.Covered)
			assert.Equal(t, tt.wantComplete, rep.Complete, "leftover=%v", rep.Leftover)
		})
	}
}

// TestMatchAtomico cubre el nivel bajo (candidatos discretos, foráneo = sobrante).
func TestMatchAtomico(t *testing.T) {
	ctx := context.Background()
	expected := []string{"tequeños", "empanadas"}
	candidates := []string{"tequenos", "empanadas", "servilletas"}

	strict := textmatch.NewSetMatcher(cascada085(), textmatch.PolicyStrict)
	repS, err := strict.Match(ctx, expected, candidates)
	require.NoError(t, err)
	assert.Equal(t, []bool{true, true}, repS.Covered)
	assert.Equal(t, []int{0, 1}, repS.UsedBy)
	assert.Equal(t, []int{2}, repS.Leftover)
	assert.False(t, repS.Complete, "«servilletas» es candidato foráneo")

	lenient := textmatch.NewSetMatcher(cascada085(), textmatch.PolicyLenient)
	repL, err := lenient.Match(ctx, expected, candidates)
	require.NoError(t, err)
	assert.True(t, repL.Complete, "Lenient ignora el candidato sobrante")
}

func TestGenerateCandidates(t *testing.T) {
	got := textmatch.GenerateCandidates([]string{"torta", "de", "chocolate"}, 2)
	want := []textmatch.Candidate{
		{Text: "torta", Start: 0, End: 1},
		{Text: "torta de", Start: 0, End: 2},
		{Text: "de", Start: 1, End: 2},
		{Text: "de chocolate", Start: 1, End: 3},
		{Text: "chocolate", Start: 2, End: 3},
	}
	assert.Equal(t, want, got)
}

// TestSetMatcherSinZonaGrisEsDeterminista: el CRITERIO 3 también al nivel de
// conjunto — sin nada inyectado el matcher funciona y repite el mismo informe.
func TestSetMatcherSinZonaGrisEsDeterminista(t *testing.T) {
	ctx := context.Background()
	m := textmatch.NewSetMatcher(cascada085(), textmatch.PolicyLenient)
	primera, err := m.MatchAnswer(ctx, []string{"tequeños", "torta de chocolate"}, "tequenos y el pastel oscuro")
	require.NoError(t, err)
	require.Equal(t, []bool{true, false}, primera.Covered, "sin zona gris el pastel no se resuelve")

	for i := 0; i < 20; i++ {
		otra, err := m.MatchAnswer(ctx, []string{"tequeños", "torta de chocolate"}, "tequenos y el pastel oscuro")
		require.NoError(t, err)
		require.Equal(t, primera, otra)
	}

	// Y con nil explícito, igual.
	conNil := m.WithGrayZone(nil)
	rep, err := conNil.MatchAnswer(ctx, []string{"tequeños", "torta de chocolate"}, "tequenos y el pastel oscuro")
	require.NoError(t, err)
	assert.Equal(t, primera, rep)
}

// TestContadorZonaGrisEnElSetMatcher es la medida que pide el plan: el escalón caro
// se paga como mucho UNA vez por esperado sin cubrir, y CERO cuando Exact/Fuzzy ya
// resolvieron. Nunca por celda del bucle.
func TestContadorZonaGrisEnElSetMatcher(t *testing.T) {
	ctx := context.Background()

	t.Run("todo cubierto por Exact/Fuzzy ⇒ 0 llamadas", func(t *testing.T) {
		fake := &zonaGrisContada{elige: -1}
		m := textmatch.NewSetMatcher(cascada085(), textmatch.PolicyLenient).WithGrayZone(fake)
		rep, err := m.MatchAnswer(ctx, []string{"tequeños", "torta"}, "quiero tequenos y una torta")
		require.NoError(t, err)
		assert.Equal(t, []bool{true, true}, rep.Covered)
		assert.Zero(t, fake.llamadas, "el escalón caro no se paga si los baratos resolvieron")
	})

	t.Run("un esperado sin cubrir ⇒ exactamente 1 llamada", func(t *testing.T) {
		fake := &zonaGrisContada{eligeTexto: "el pastel oscuro", confidencia: 0.7}
		m := textmatch.NewSetMatcher(cascada085(), textmatch.PolicyLenient).WithGrayZone(fake)
		rep, err := m.MatchAnswer(ctx, []string{"tequeños", "torta de chocolate"}, "tequenos y el pastel oscuro")
		require.NoError(t, err)

		require.Equal(t, 1, fake.llamadas, "una sola consulta para el único esperado sin cubrir")
		require.Len(t, fake.expuestos, 1)
		assert.Equal(t, "torta de chocolate", fake.expuestos[0])
		assert.Greater(t, len(fake.ofrecidos[0]), 1, "se le ofrecen todos los candidatos libres de una vez")
		assert.Contains(t, fake.ofrecidos[0], "el pastel oscuro")
		assert.NotContains(t, fake.ofrecidos[0], "tequenos", "el token ya consumido no se ofrece")

		assert.Equal(t, []bool{true, true}, rep.Covered)
		assert.Equal(t, []int{0, 1}, rep.UsedBy, "el span del pastel arranca en el token 1")
		assert.True(t, rep.Complete)
	})

	t.Run("dos esperados sin cubrir ⇒ 2 llamadas, no una por celda", func(t *testing.T) {
		fake := &zonaGrisContada{elige: -1}
		m := textmatch.NewSetMatcher(cascada085(), textmatch.PolicyLenient).WithGrayZone(fake)
		_, err := m.MatchAnswer(ctx, []string{"torta de chocolate", "bebida fria"}, "el pastel oscuro y algo helado")
		require.NoError(t, err)
		assert.Equal(t, 2, fake.llamadas)
	})

	t.Run("nivel atómico ⇒ 1 llamada con los candidatos libres", func(t *testing.T) {
		fake := &zonaGrisContada{eligeTexto: "pastel oscuro", confidencia: 0.7}
		m := textmatch.NewSetMatcher(cascada085(), textmatch.PolicyStrict).WithGrayZone(fake)
		rep, err := m.Match(ctx, []string{"torta de chocolate"}, []string{"pastel oscuro"})
		require.NoError(t, err)
		assert.Equal(t, 1, fake.llamadas)
		assert.Equal(t, []string{"pastel oscuro"}, fake.ofrecidos[0])
		assert.Equal(t, []bool{true}, rep.Covered)
		assert.Equal(t, []int{0}, rep.UsedBy)
		assert.Empty(t, rep.Leftover)
		assert.True(t, rep.Complete)
	})
}

// TestZonaGrisPropagaErrorEnElSetMatcher: el fallo del escalón caro no se traga.
func TestZonaGrisPropagaErrorEnElSetMatcher(t *testing.T) {
	ctx := context.Background()
	sentinela := errors.New("proveedor caído")
	fake := &zonaGrisContada{fallaCon: sentinela}
	m := textmatch.NewSetMatcher(cascada085(), textmatch.PolicyLenient).WithGrayZone(fake)

	_, err := m.MatchAnswer(ctx, []string{"torta de chocolate"}, "el pastel oscuro")
	require.ErrorIs(t, err, sentinela)

	_, err = m.Match(ctx, []string{"torta de chocolate"}, []string{"pastel oscuro"})
	require.ErrorIs(t, err, sentinela)
}

// TestZonaGrisIndiceFueraDeRango: una respuesta inválida del escalón caro se lee
// como «ninguno corresponde», sin panicar ni inventar cobertura.
func TestZonaGrisIndiceFueraDeRango(t *testing.T) {
	ctx := context.Background()
	for _, idx := range []int{-5, 99} {
		fake := &zonaGrisContada{elige: idx}
		m := textmatch.NewSetMatcher(cascada085(), textmatch.PolicyLenient).WithGrayZone(fake)
		rep, err := m.Match(ctx, []string{"torta de chocolate"}, []string{"pastel oscuro"})
		require.NoError(t, err)
		assert.Equal(t, []bool{false}, rep.Covered)
		assert.Equal(t, []int{-1}, rep.UsedBy)
	}
}
