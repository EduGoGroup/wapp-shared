# logger

Logging estructurado para el ecosistema wApp, construido **solo** sobre la
libreria estandar (`log/slog`). Sin dependencias externas.

## Instalacion

```bash
go get github.com/EduGoGroup/wapp-shared/logger
```

## Uso

```go
package main

import (
	"log/slog"

	"github.com/EduGoGroup/wapp-shared/logger"
)

func main() {
	log := logger.New(
		logger.WithLevel(slog.LevelDebug),
		logger.WithJSON(true),
	)

	log.Info("servidor iniciado", "puerto", 8080)

	// Logger hijo que arrastra campos en cada emision.
	reqLog := log.With("request_id", "abc-123")
	reqLog.Warn("latencia alta", "ms", 250)
}
```

## API

- `New(opts ...Option) Logger` — construye un logger. Por defecto: nivel `Info`,
  formato texto, salida a `os.Stdout`.
  - `WithLevel(slog.Level)` — nivel minimo; los mensajes por debajo se descartan.
  - `WithJSON(bool)` — `true` para salida JSON, `false` para texto.
  - `WithWriter(io.Writer)` — destino de escritura.
- `Default() Logger` — usa el `slog.Logger` por defecto del proceso.
- Interfaz `Logger`:
  - `Debug/Info/Warn/Error(msg string, args ...any)` — `args` son pares
    clave/valor (semantica de slog).
  - `With(args ...any) Logger` — deriva un hijo que arrastra esos campos.
