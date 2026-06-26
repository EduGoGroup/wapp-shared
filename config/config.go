package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

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

// GetInt devuelve la variable de entorno key (con prefijo) parseada como entero,
// o def si no esta definida o no es parseable.
func (l *Loader) GetInt(key string, def int) int {
	v, ok := l.lookup(key)
	if !ok {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// GetBool devuelve la variable de entorno key (con prefijo) parseada como
// booleano, o def si no esta definida o no es parseable. Acepta los formatos de
// strconv.ParseBool (1, t, T, TRUE, true, 0, f, FALSE, false, ...).
func (l *Loader) GetBool(key string, def bool) bool {
	v, ok := l.lookup(key)
	if !ok {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}
