# Changelog — health

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/)
y [Versionado Semantico](https://semver.org/lang/es/).

## [Unreleased]

## [0.1.1] - 2026-07-10

### Changed

- `Checker` ahora es seguro para uso concurrente: `Register` y las lecturas
  (`CheckAll`/`IsHealthy`/`IsReady`) se sincronizan con un `sync.RWMutex`.
  `CheckAll` toma una instantánea de los checks bajo lock de lectura y los
  ejecuta fuera del lock. Sin cambios de API.

## [0.1.0] - 2026-06-25

### Added

- Version inicial del modulo `health` (copia-adaptacion del patron Checker de
  `edugo-shared/health` al namespace wApp, sin acoplamientos a drivers).
- Tipo `Status` (`healthy`/`unhealthy`/`degraded`) y `CheckResult`
  (Status, Component, Message, Timestamp, Metadata).
- Interfaz `HealthCheck` (Name + Check) y `Checker` con `NewChecker`,
  `Register` (ignora nil), `CheckAll` (agrega resultados por componente) y
  helpers `IsHealthy`/`IsReady`/`IsLive`.
- Solo stdlib; los checks concretos (Postgres, Mongo, socket, etc.) se
  implementan en cada consumidor.
