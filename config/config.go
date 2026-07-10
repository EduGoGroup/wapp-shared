package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// ErrMissing indica que una clave requerida no está definida en el entorno.
var ErrMissing = errors.New("config: variable de entorno requerida no definida")

// ErrInvalid indica que una clave está definida pero su valor no se puede
// parsear al tipo pedido (p. ej. WAPP_PORT=81OO para un entero).
var ErrInvalid = errors.New("config: valor de entorno inválido")

// Loader carga configuracion desde un archivo YAML opcional y expone getters
// tipados sobre variables de entorno.
type Loader struct {
	envPrefix string
	file      string
}

// Option configura la construccion de un [Loader] mediante [New].
type Option func(*Loader)

// WithEnvPrefix fija el prefijo aplicado a las claves al leer variables de
// entorno. Por ejemplo, con prefijo "WAPP_", GetString("PORT", ...) lee
// la variable de entorno "WAPP_PORT".
func WithEnvPrefix(prefix string) Option {
	return func(l *Loader) {
		l.envPrefix = prefix
	}
}

// WithFile define la ruta del archivo YAML a leer en [Loader.Unmarshal].
func WithFile(path string) Option {
	return func(l *Loader) {
		l.file = path
	}
}

// New construye un [Loader] aplicando las opciones dadas.
func New(opts ...Option) *Loader {
	l := &Loader{}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Unmarshal vuelca la configuracion del archivo YAML en into.
//
// Si no se configuro archivo o el archivo no existe, no hace nada y devuelve
// nil. Cualquier otro error de lectura o de parseo YAML se propaga.
func (l *Loader) Unmarshal(into any) error {
	if l.file == "" {
		return nil
	}

	data, err := os.ReadFile(l.file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("leyendo archivo de configuracion %q: %w", l.file, err)
	}

	if err := yaml.Unmarshal(data, into); err != nil {
		return fmt.Errorf("parseando YAML de %q: %w", l.file, err)
	}

	return nil
}

// lookup devuelve el valor de la variable de entorno (con prefijo) y si existe.
func (l *Loader) lookup(key string) (string, bool) {
	return os.LookupEnv(l.envPrefix + key)
}

// GetString devuelve la variable de entorno key (con prefijo) o def si no esta
// definida.
func (l *Loader) GetString(key, def string) string {
	if v, ok := l.lookup(key); ok {
		return v
	}
	return def
}

// warnInvalid registra de forma ruidosa un valor presente-pero-inválido antes
// de caer al default. La clave se registra con prefijo; el valor NO se registra
// para no arriesgar el volcado de un secreto mal ubicado.
func (l *Loader) warnInvalid(key string, err error) {
	slog.Warn("config: valor de entorno inválido, se usa el default",
		"key", l.envPrefix+key,
		"err", err,
	)
}

// GetInt devuelve la variable de entorno key (con prefijo) parseada como entero,
// o def si no esta definida. Un valor presente-pero-inválido NO se traga en
// silencio: se registra un warning y se devuelve def (usa [Loader.GetIntE] o
// [Loader.RequireInt] si necesitas el error).
func (l *Loader) GetInt(key string, def int) int {
	v, err := l.GetIntE(key, def)
	if err != nil {
		l.warnInvalid(key, err)
	}
	return v
}

// GetIntE es como [Loader.GetInt] pero devuelve [ErrInvalid] (envuelto) cuando
// la clave está definida con un valor no parseable. Si la clave no existe,
// devuelve (def, nil).
func (l *Loader) GetIntE(key string, def int) (int, error) {
	v, ok := l.lookup(key)
	if !ok {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def, fmt.Errorf("%w: %s no es un entero: %w", ErrInvalid, l.envPrefix+key, err)
	}
	return n, nil
}

// GetBool devuelve la variable de entorno key (con prefijo) parseada como
// booleano, o def si no esta definida. Acepta los formatos de strconv.ParseBool
// (1, t, T, TRUE, true, 0, f, FALSE, false, ...). Un valor presente-pero-inválido
// NO se traga en silencio: se registra un warning y se devuelve def (usa
// [Loader.GetBoolE] o [Loader.RequireBool] si necesitas el error).
func (l *Loader) GetBool(key string, def bool) bool {
	v, err := l.GetBoolE(key, def)
	if err != nil {
		l.warnInvalid(key, err)
	}
	return v
}

// GetBoolE es como [Loader.GetBool] pero devuelve [ErrInvalid] (envuelto) cuando
// la clave está definida con un valor no parseable. Si la clave no existe,
// devuelve (def, nil).
func (l *Loader) GetBoolE(key string, def bool) (bool, error) {
	v, ok := l.lookup(key)
	if !ok {
		return def, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def, fmt.Errorf("%w: %s no es un booleano: %w", ErrInvalid, l.envPrefix+key, err)
	}
	return b, nil
}

// GetDuration devuelve la variable de entorno key (con prefijo) parseada como
// [time.Duration] (formato de [time.ParseDuration], p. ej. "30s", "5m"), o def
// si no esta definida. Un valor presente-pero-inválido NO se traga en silencio:
// se registra un warning y se devuelve def (usa [Loader.GetDurationE] o
// [Loader.RequireDuration] si necesitas el error).
func (l *Loader) GetDuration(key string, def time.Duration) time.Duration {
	v, err := l.GetDurationE(key, def)
	if err != nil {
		l.warnInvalid(key, err)
	}
	return v
}

// GetDurationE es como [Loader.GetDuration] pero devuelve [ErrInvalid]
// (envuelto) cuando la clave está definida con un valor no parseable. Si la
// clave no existe, devuelve (def, nil).
func (l *Loader) GetDurationE(key string, def time.Duration) (time.Duration, error) {
	v, ok := l.lookup(key)
	if !ok {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def, fmt.Errorf("%w: %s no es una duración: %w", ErrInvalid, l.envPrefix+key, err)
	}
	return d, nil
}

// RequireString devuelve el valor de la variable de entorno key (con prefijo) o
// [ErrMissing] (envuelto) si no está definida.
func (l *Loader) RequireString(key string) (string, error) {
	v, ok := l.lookup(key)
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrMissing, l.envPrefix+key)
	}
	return v, nil
}

// RequireInt devuelve el entero de la variable de entorno key (con prefijo).
// Devuelve [ErrMissing] si no está definida y [ErrInvalid] si su valor no es
// parseable como entero.
func (l *Loader) RequireInt(key string) (int, error) {
	v, ok := l.lookup(key)
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrMissing, l.envPrefix+key)
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%w: %s no es un entero: %w", ErrInvalid, l.envPrefix+key, err)
	}
	return n, nil
}

// RequireBool devuelve el booleano de la variable de entorno key (con prefijo).
// Devuelve [ErrMissing] si no está definida y [ErrInvalid] si su valor no es
// parseable como booleano.
func (l *Loader) RequireBool(key string) (bool, error) {
	v, ok := l.lookup(key)
	if !ok {
		return false, fmt.Errorf("%w: %s", ErrMissing, l.envPrefix+key)
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%w: %s no es un booleano: %w", ErrInvalid, l.envPrefix+key, err)
	}
	return b, nil
}

// RequireDuration devuelve la [time.Duration] de la variable de entorno key (con
// prefijo). Devuelve [ErrMissing] si no está definida y [ErrInvalid] si su valor
// no es parseable como duración.
func (l *Loader) RequireDuration(key string) (time.Duration, error) {
	v, ok := l.lookup(key)
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrMissing, l.envPrefix+key)
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%w: %s no es una duración: %w", ErrInvalid, l.envPrefix+key, err)
	}
	return d, nil
}
