# Changelog — logger

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/)
y [Versionado Semantico](https://semver.org/lang/es/).

## [Unreleased]

## [0.2.0] - 2026-08-01

### Added

- Propagación del logger por `context.Context` (Plan 038), para que las capas
  internas no reciban el logger como parámetro ni lo tomen de una global:
  - `WithContext(ctx, Logger) context.Context` — inyecta el logger; con `ctx` o
    logger `nil` devuelve el contexto tal cual.
  - `FromContext(ctx) Logger` — lo recupera; si el contexto es `nil` o no lleva
    logger, devuelve `Default()`, así que nunca entrega `nil`.

  Cambio **aditivo**: la clave del contexto es un tipo privado, sin colisión
  posible con las de otros paquetes.

## [0.1.0] - 2026-06-25

### Added

- Version inicial del modulo `logger`.
- Interfaz `Logger` con metodos `Debug/Info/Warn/Error(msg, args...)` y
  `With(args...) Logger` (semantica clave/valor de slog).
- Implementacion sobre `log/slog` con `New(opts...)` y opciones `WithLevel`,
  `WithJSON` y `WithWriter`; helper `Default()`.
