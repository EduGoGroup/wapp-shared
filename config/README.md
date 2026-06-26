# config

Carga de configuracion para el ecosistema wApp: un archivo YAML opcional mas un
overlay de variables de entorno. Dependencia minima: `gopkg.in/yaml.v3`.

## Instalacion

```bash
go get github.com/EduGoGroup/wapp-shared/config
```

## Uso

```go
package main

import (
	"fmt"

	"github.com/EduGoGroup/wapp-shared/config"
)

func main() {
	loader := config.New(
		config.WithFile("config.yaml"),
		config.WithEnvPrefix("WAPP_"),
	)

	var cfg struct {
		Host   string `yaml:"host"`
		Puerto int    `yaml:"puerto"`
	}
	if err := loader.Unmarshal(&cfg); err != nil {
		panic(err)
	}

	// Overlay de entorno: lee WAPP_LOG_LEVEL, con fallback a "info".
	nivel := loader.GetString("LOG_LEVEL", "info")
	fmt.Println(cfg.Host, cfg.Puerto, nivel)
}
```

## API

- `New(opts ...Option) *Loader`
  - `WithFile(path string)` — ruta del archivo YAML.
  - `WithEnvPrefix(prefix string)` — prefijo aplicado a las claves de entorno.
- `Unmarshal(into any) error` — vuelca el YAML en `into`. Si no hay archivo
  configurado o no existe, no falla (devuelve `nil`).
- Getters de entorno (leen `os.Getenv(prefix+key)` con fallback al default):
  - `GetString(key, def string) string`
  - `GetInt(key string, def int) int`
  - `GetBool(key string, def bool) bool`
