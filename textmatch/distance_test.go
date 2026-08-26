package textmatch_test

import (
	"testing"

	"github.com/EduGoGroup/wapp-shared/textmatch"
	"github.com/stretchr/testify/assert"
)

func TestEditDistance(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{"iguales", "tequeños", "tequeños", 0},
		{"vacío contra algo", "", "abc", 3},
		{"algo contra vacío", "abc", "", 3},
		{"clásico kitten/sitting", "kitten", "sitting", 3},
		{"inserción instalgram/instagram", "instalgram", "instagram", 1},
		// Transposición adyacente s<->t: OSA la cuenta 1, no 2 como el Levenshtein
		// puro. Es lo que deja «whastapp» a similitud 0,875 y no 0,75.
		{"transposición whastapp/whatsapp", "whastapp", "whatsapp", 1},
		{"por runas año/ano", "año", "ano", 1},
		{"por runas ñoquis/noquis", "ñoquis", "noquis", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, textmatch.EditDistance(tt.a, tt.b))
		})
	}
}

// TestEditDistanceSimetrica confirma que la distancia no depende del orden.
func TestEditDistanceSimetrica(t *testing.T) {
	pairs := [][2]string{{"whastapp", "whatsapp"}, {"instalgram", "instagram"}, {"ñoquis", "noquis"}}
	for _, p := range pairs {
		assert.Equal(t, textmatch.EditDistance(p[0], p[1]), textmatch.EditDistance(p[1], p[0]),
			"EditDistance no simétrica para %q/%q", p[0], p[1])
	}
}
