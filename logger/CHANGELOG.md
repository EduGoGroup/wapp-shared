# Changelog — logger

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/)
y [Versionado Semantico](https://semver.org/lang/es/).

## [Unreleased]

## [0.1.0] - 2026-06-25

### Added

- Version inicial del modulo `logger`.
- Interfaz `Logger` con metodos `Debug/Info/Warn/Error(msg, args...)` y
  `With(args...) Logger` (semantica clave/valor de slog).
- Implementacion sobre `log/slog` con `New(opts...)` y opciones `WithLevel`,
  `WithJSON` y `WithWriter`; helper `Default()`.
