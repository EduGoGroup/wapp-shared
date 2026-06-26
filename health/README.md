# health

Framework minimo y extensible de health checks para el ecosistema wApp, construido **solo** sobre
la libreria estandar. Sin dependencias de drivers concretos: define el contrato; los checks
especificos se implementan donde se usan.

## Instalacion

```bash
go get github.com/EduGoGroup/wapp-shared/health
```

## Uso

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/EduGoGroup/wapp-shared/health"
)

// socketCheck implementa health.HealthCheck para un componente cualquiera.
type socketCheck struct{ alive bool }

func (s socketCheck) Name() string { return "whatsmeow-socket" }

func (s socketCheck) Check(_ context.Context) health.CheckResult {
	st := health.StatusHealthy
	if !s.alive {
		st = health.StatusUnhealthy
	}
	return health.CheckResult{Status: st, Component: s.Name(), Timestamp: time.Now()}
}

func main() {
	c := health.NewChecker()
	c.Register(socketCheck{alive: true})

	results := c.CheckAll(context.Background()) // map[string]health.CheckResult
	fmt.Println(c.IsHealthy(context.Background()))
	_ = results
}
```

## API

- `NewChecker() *Checker` — crea un Checker vacio.
- `(*Checker) Register(HealthCheck)` — registra un check (ignora `nil`).
- `(*Checker) CheckAll(ctx) map[string]CheckResult` — ejecuta todos y agrega por componente.
- `(*Checker) IsHealthy(ctx) bool` — `false` si algun check esta `unhealthy` (`degraded` no cuenta).
- `(*Checker) IsReady(ctx) bool` — readiness (equivale a `IsHealthy`).
- `(*Checker) IsLive() bool` — liveness basica.
- `type Status` — `StatusHealthy` / `StatusUnhealthy` / `StatusDegraded`.
- `type CheckResult` — `Status`, `Component`, `Message`, `Timestamp`, `Metadata`.
- `type HealthCheck interface` — `Name() string`, `Check(ctx) CheckResult`.

## Navegacion

- [Changelog](CHANGELOG.md)

## Comandos disponibles

```bash
make build     # Compilar
make test      # Tests
make check     # Lint y validacion
```
