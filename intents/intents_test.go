package intents_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/EduGoGroup/wapp-shared/intents"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalJSON es el contrato válido más pequeño posible: un intent con un
// ejemplo y sin umbral (para ejercer la normalización por defecto).
const minimalJSON = `{
  "version": "1",
  "intents": [
    {
      "name": "saludar",
      "descripcion": "El usuario saluda",
      "ejemplos": [ { "mensaje": "hola" } ]
    }
  ]
}`

// completeJSON ejercita todos los campos: umbral explícito, vocabulario, params
// declarados, ejemplos con params anotados y un campo desconocido a tolerar.
const completeJSON = `{
  "version": "2026-07-10",
  "umbral_confianza": 0.8,
  "vocabulario": ["pizza", "hamburguesa"],
  "campo_futuro": "debe ignorarse",
  "intents": [
    {
      "name": "pedir_comida",
      "descripcion": "El usuario pide un plato",
      "params": ["plato", "cantidad"],
      "ejemplos": [
        { "mensaje": "quiero 2 pizzas", "params": { "plato": "pizza", "cantidad": "2" } },
        { "mensaje": "una hamburguesa" }
      ]
    },
    {
      "name": "cancelar",
      "descripcion": "El usuario cancela",
      "ejemplos": [ { "mensaje": "cancela mi pedido" } ]
    }
  ]
}`

func TestParseAndValidate_Minimal(t *testing.T) {
	cfg, err := intents.ParseAndValidate([]byte(minimalJSON))
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "1", cfg.Version)
	require.Len(t, cfg.Intents, 1)
	assert.Equal(t, "saludar", cfg.Intents[0].Name)
	// Umbral ausente ⇒ normalizado al valor por defecto.
	assert.InDelta(t, intents.DefaultThreshold, cfg.UmbralConfianza, 1e-9)
}

func TestParseAndValidate_Complete(t *testing.T) {
	cfg, err := intents.ParseAndValidate([]byte(completeJSON))
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.InDelta(t, 0.8, cfg.UmbralConfianza, 1e-9)
	assert.Equal(t, []string{"pizza", "hamburguesa"}, cfg.Vocabulario)
	require.Len(t, cfg.Intents, 2)
	assert.Equal(t, []string{"plato", "cantidad"}, cfg.Intents[0].Params)
	assert.Equal(t, "pizza", cfg.Intents[0].Ejemplos[0].Params["plato"])
}

func TestParseAndValidate_ThresholdZeroNormalized(t *testing.T) {
	// umbral_confianza presente pero en 0 se trata como ausente.
	cfg, err := intents.ParseAndValidate([]byte(`{
      "version": "1",
      "umbral_confianza": 0,
      "intents": [ { "name": "xx", "descripcion": "d", "ejemplos": [ { "mensaje": "m" } ] } ]
    }`))
	require.NoError(t, err)
	assert.InDelta(t, intents.DefaultThreshold, cfg.UmbralConfianza, 1e-9)
}

func TestParseAndValidate_ThresholdBoundaryOne(t *testing.T) {
	// 1.0 es el borde superior permitido del rango (0,1].
	cfg, err := intents.ParseAndValidate([]byte(`{
      "version": "1",
      "umbral_confianza": 1,
      "intents": [ { "name": "xx", "descripcion": "d", "ejemplos": [ { "mensaje": "m" } ] } ]
    }`))
	require.NoError(t, err)
	assert.InDelta(t, 1.0, cfg.UmbralConfianza, 1e-9)
}

// TestParseAndValidate_Rejections cubre una regla de rechazo por caso; todas
// deben envolver ErrInvalidConfig.
func TestParseAndValidate_Rejections(t *testing.T) {
	cases := []struct {
		name string
		json string
		want string // fragmento esperado en el mensaje de error
	}{
		{
			name: "json corrupto",
			json: `{ not json `,
			want: "JSON inválido",
		},
		{
			name: "version vacía",
			json: `{ "version": "  ", "intents": [ { "name": "xx", "descripcion": "d", "ejemplos": [ { "mensaje": "m" } ] } ] }`,
			want: "version",
		},
		{
			name: "umbral fuera de rango alto",
			json: `{ "version": "1", "umbral_confianza": 1.5, "intents": [ { "name": "xx", "descripcion": "d", "ejemplos": [ { "mensaje": "m" } ] } ] }`,
			want: "umbral_confianza",
		},
		{
			name: "umbral negativo",
			json: `{ "version": "1", "umbral_confianza": -0.1, "intents": [ { "name": "xx", "descripcion": "d", "ejemplos": [ { "mensaje": "m" } ] } ] }`,
			want: "umbral_confianza",
		},
		{
			name: "sin intents",
			json: `{ "version": "1", "intents": [] }`,
			want: "al menos un intent",
		},
		{
			name: "name con mayúscula",
			json: `{ "version": "1", "intents": [ { "name": "Saludar", "descripcion": "d", "ejemplos": [ { "mensaje": "m" } ] } ] }`,
			want: "no cumple el patrón",
		},
		{
			name: "name de un solo carácter",
			json: `{ "version": "1", "intents": [ { "name": "a", "descripcion": "d", "ejemplos": [ { "mensaje": "m" } ] } ] }`,
			want: "no cumple el patrón",
		},
		{
			name: "name reservado desconocido",
			json: `{ "version": "1", "intents": [ { "name": "desconocido", "descripcion": "d", "ejemplos": [ { "mensaje": "m" } ] } ] }`,
			want: "reservado",
		},
		{
			name: "names duplicados",
			json: `{ "version": "1", "intents": [
				{ "name": "saludar", "descripcion": "d", "ejemplos": [ { "mensaje": "m" } ] },
				{ "name": "saludar", "descripcion": "d2", "ejemplos": [ { "mensaje": "m2" } ] }
			] }`,
			want: "duplicado",
		},
		{
			name: "descripción vacía",
			json: `{ "version": "1", "intents": [ { "name": "xx", "descripcion": "   ", "ejemplos": [ { "mensaje": "m" } ] } ] }`,
			want: "sin descripción",
		},
		{
			name: "sin ejemplos",
			json: `{ "version": "1", "intents": [ { "name": "xx", "descripcion": "d", "ejemplos": [] } ] }`,
			want: "al menos un ejemplo",
		},
		{
			name: "mensaje de ejemplo vacío",
			json: `{ "version": "1", "intents": [ { "name": "xx", "descripcion": "d", "ejemplos": [ { "mensaje": "  " } ] } ] }`,
			want: "mensaje vacío",
		},
		{
			name: "param con patrón inválido",
			json: `{ "version": "1", "intents": [ { "name": "xx", "descripcion": "d", "params": ["Plato"], "ejemplos": [ { "mensaje": "m" } ] } ] }`,
			want: "no cumple el patrón",
		},
		{
			name: "params duplicados",
			json: `{ "version": "1", "intents": [ { "name": "xx", "descripcion": "d", "params": ["plato", "plato"], "ejemplos": [ { "mensaje": "m" } ] } ] }`,
			want: "duplicado",
		},
		{
			name: "ejemplo usa param no declarado",
			json: `{ "version": "1", "intents": [ { "name": "xx", "descripcion": "d", "params": ["plato"], "ejemplos": [ { "mensaje": "m", "params": { "cantidad": "2" } } ] } ] }`,
			want: "no declarado",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := intents.ParseAndValidate([]byte(tc.json))
			require.Error(t, err)
			assert.Nil(t, cfg)
			assert.ErrorIs(t, err, intents.ErrInvalidConfig)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestParseAndValidate_TooLarge(t *testing.T) {
	// Relleno con un vocabulario enorme para superar MaxConfigBytes sin
	// depender de la validez del resto (el tamaño se chequea primero).
	big := strings.Repeat("a", intents.MaxConfigBytes+1)
	_, err := intents.ParseAndValidate([]byte(big))
	require.Error(t, err)
	assert.ErrorIs(t, err, intents.ErrInvalidConfig)
	assert.Contains(t, err.Error(), "excede el máximo")
}

func TestParseAndValidate_ExactlyMaxBytesNotRejectedBySize(t *testing.T) {
	// Un payload de exactamente MaxConfigBytes no debe rechazarse por tamaño;
	// fallará más adelante por JSON inválido, no por el límite.
	data := make([]byte, intents.MaxConfigBytes)
	for i := range data {
		data[i] = ' '
	}
	_, err := intents.ParseAndValidate(data)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "excede el máximo")
}

func TestParseAndValidate_ErrorIsSentinel(t *testing.T) {
	_, err := intents.ParseAndValidate([]byte(`{`))
	require.Error(t, err)
	assert.True(t, errors.Is(err, intents.ErrInvalidConfig))
}
