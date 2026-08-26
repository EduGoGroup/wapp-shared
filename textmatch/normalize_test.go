package textmatch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"minúsculas", "TeQueños", "tequeños"},
		{"tildes fuera", "Café Perú", "cafe peru"},
		{"diéresis fuera", "Pingüino", "pinguino"},
		{"ñ preservada minúscula", "año", "año"},
		{"ñ preservada mayúscula", "AÑO", "año"},
		{"ano sin ñ intacto", "ano", "ano"},
		{"ñ descompuesta se recompone", "año", "año"},
		{"acento descompuesto se barre", "café", "cafe"},
		{"colapsa espacios y trim", "  torta   de   chocolate  ", "torta de chocolate"},
		{"cedilla y otros latinos", "Garçon", "garcon"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Normalize(tt.in))
		})
	}
}

// TestNormalizeAnioNoEsAno es la regresión canónica de la «ñ»: se preserva, así
// «año» y «ano» NO normalizan al mismo texto.
func TestNormalizeAnioNoEsAno(t *testing.T) {
	require.NotEqual(t, Normalize("ano"), Normalize("año"),
		"«año» y «ano» no deben normalizar igual: la ñ es letra, no tilde")
}

// TestPliegueNoContieneEnye custodia el invariante en su ORIGEN (la tabla), no en
// una conducta: si algún día alguien añade la «ñ» a una fila del plegado, este test
// cae antes que ningún caso de negocio. La «n» sí pliega a otras variantes (ń, ň).
func TestPliegueNoContieneEnye(t *testing.T) {
	_, plegada := pliegueDiacritico['ñ']
	require.False(t, plegada, "la «ñ» NO puede estar en la tabla de plegado de diacríticos")

	base, ok := pliegueDiacritico['ń']
	require.True(t, ok, "la «ń» (n con acento agudo) sí debe plegar")
	require.Equal(t, 'n', base)
}

func TestSplitTokens(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "descarta conector y, el relleno queda como tokens",
			in:   "tequeños y la famosa torta",
			want: []string{"tequeños", "la", "famosa", "torta"},
		},
		{
			name: "puntuación como frontera y conector y",
			in:   "quiero: tequeños, empanadas y torta",
			want: []string{"quiero", "tequeños", "empanadas", "torta"},
		},
		{
			name: "conector e descartado",
			in:   "café e historia",
			want: []string{"cafe", "historia"},
		},
		{
			name: "barras y comas",
			in:   "a,b|c",
			want: []string{"a", "b", "c"},
		},
		{
			name: "números sobreviven",
			in:   "30 tequeños",
			want: []string{"30", "tequeños"},
		},
		{
			name: "vacío",
			in:   "   ",
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, SplitTokens(tt.in))
		})
	}
}
