package intents

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// DefaultThreshold es el umbral de confianza aplicado cuando el contrato no
// declara `umbral_confianza` (o lo deja en 0): por debajo de él, el clasificador
// debe tratar el mensaje como no reconocido.
const DefaultThreshold = 0.6

// MaxConfigBytes es el tamaño máximo aceptado para el JSON de configuración.
// Actúa como cortafuegos ante payloads abusivos en el PUT del API.
const MaxConfigBytes = 256 * 1024

// ReservedUnknown es el intent reservado que el clasificador emite cuando ningún
// intent declarado supera el umbral; está prohibido declararlo en el contrato.
const ReservedUnknown = "desconocido"

// namePattern es el patrón que deben cumplir los nombres de intent y de param:
// minúscula inicial, luego minúsculas/dígitos/guion bajo, entre 2 y 64 chars.
const namePattern = `^[a-z][a-z0-9_]{1,63}$`

// nameRe compila namePattern una sola vez; se reutiliza para intents y params.
var nameRe = regexp.MustCompile(namePattern)

// Config es el contrato de configuración de intenciones de un tenant. Las claves
// JSON mezclan español e inglés de forma deliberada (contrato canónico heredado);
// no se renombran.
type Config struct {
	Version         string   `json:"version"`
	UmbralConfianza float64  `json:"umbral_confianza,omitempty"`
	Intents         []Intent `json:"intents"`
	Vocabulario     []string `json:"vocabulario,omitempty"`
}

// Intent describe una intención que el clasificador debe reconocer, con sus
// parámetros extraíbles y ejemplos de entrenamiento.
type Intent struct {
	Name        string    `json:"name"`
	Descripcion string    `json:"descripcion"`
	Params      []string  `json:"params,omitempty"`
	Ejemplos    []Ejemplo `json:"ejemplos"`
}

// Ejemplo es un mensaje de muestra para un intent, opcionalmente anotado con los
// valores de los params que ese mensaje contiene.
type Ejemplo struct {
	Mensaje string            `json:"mensaje"`
	Params  map[string]string `json:"params,omitempty"`
}

// ParseAndValidate decodifica y valida el contrato de intenciones.
//
// Devuelve el Config ya normalizado (umbral por defecto aplicado) o un error que
// envuelve ErrInvalidConfig identificando el intent/campo ofensor. El JSON es
// tolerante a campos desconocidos a propósito (no se usa DisallowUnknownFields):
// el contrato debe aceptar claves futuras sin romper a los consumidores viejos.
func ParseAndValidate(data []byte) (*Config, error) {
	if len(data) > MaxConfigBytes {
		return nil, fmt.Errorf("%w: config de %d bytes excede el máximo de %d",
			ErrInvalidConfig, len(data), MaxConfigBytes)
	}

	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%w: JSON inválido: %w", ErrInvalidConfig, err)
	}

	if strings.TrimSpace(c.Version) == "" {
		return nil, fmt.Errorf("%w: version no puede estar vacía", ErrInvalidConfig)
	}
	if err := normalizeThreshold(&c); err != nil {
		return nil, err
	}
	if len(c.Intents) == 0 {
		return nil, fmt.Errorf("%w: se requiere al menos un intent", ErrInvalidConfig)
	}

	seen := make(map[string]struct{}, len(c.Intents))
	for i := range c.Intents {
		if err := validateIntent(i, c.Intents[i], seen); err != nil {
			return nil, err
		}
	}
	return &c, nil
}

// normalizeThreshold aplica DefaultThreshold cuando el umbral viene ausente o en
// 0, y rechaza cualquier valor presente fuera del rango (0,1].
func normalizeThreshold(c *Config) error {
	if c.UmbralConfianza == 0 {
		c.UmbralConfianza = DefaultThreshold
		return nil
	}
	if c.UmbralConfianza < 0 || c.UmbralConfianza > 1 {
		return fmt.Errorf("%w: umbral_confianza %v fuera del rango (0,1]",
			ErrInvalidConfig, c.UmbralConfianza)
	}
	return nil
}

// validateIntent valida un intent y registra su nombre en seen para detectar
// duplicados entre intents del mismo contrato.
func validateIntent(i int, in Intent, seen map[string]struct{}) error {
	if !nameRe.MatchString(in.Name) {
		return fmt.Errorf("%w: intents[%d].name %q no cumple el patrón %s",
			ErrInvalidConfig, i, in.Name, namePattern)
	}
	if in.Name == ReservedUnknown {
		return fmt.Errorf("%w: intents[%d].name %q está reservado para el clasificador",
			ErrInvalidConfig, i, in.Name)
	}
	if _, dup := seen[in.Name]; dup {
		return fmt.Errorf("%w: intent name %q duplicado", ErrInvalidConfig, in.Name)
	}
	seen[in.Name] = struct{}{}

	if strings.TrimSpace(in.Descripcion) == "" {
		return fmt.Errorf("%w: intent %q sin descripción", ErrInvalidConfig, in.Name)
	}

	params, err := validateParams(in)
	if err != nil {
		return err
	}
	return validateEjemplos(in, params)
}

// validateParams valida los params declarados (patrón y unicidad) y devuelve el
// conjunto de nombres para contrastar contra las claves de los ejemplos.
func validateParams(in Intent) (map[string]struct{}, error) {
	set := make(map[string]struct{}, len(in.Params))
	for _, p := range in.Params {
		if !nameRe.MatchString(p) {
			return nil, fmt.Errorf("%w: intent %q param %q no cumple el patrón %s",
				ErrInvalidConfig, in.Name, p, namePattern)
		}
		if _, dup := set[p]; dup {
			return nil, fmt.Errorf("%w: intent %q param %q duplicado",
				ErrInvalidConfig, in.Name, p)
		}
		set[p] = struct{}{}
	}
	return set, nil
}

// validateEjemplos exige al menos un ejemplo con mensaje no vacío y verifica que
// las claves de Ejemplo.Params sean un subconjunto de los params declarados.
func validateEjemplos(in Intent, params map[string]struct{}) error {
	if len(in.Ejemplos) == 0 {
		return fmt.Errorf("%w: intent %q necesita al menos un ejemplo",
			ErrInvalidConfig, in.Name)
	}
	for j, ej := range in.Ejemplos {
		if strings.TrimSpace(ej.Mensaje) == "" {
			return fmt.Errorf("%w: intent %q ejemplos[%d] con mensaje vacío",
				ErrInvalidConfig, in.Name, j)
		}
		for k := range ej.Params {
			if _, ok := params[k]; !ok {
				return fmt.Errorf("%w: intent %q ejemplos[%d] usa param %q no declarado",
					ErrInvalidConfig, in.Name, j, k)
			}
		}
	}
	return nil
}
