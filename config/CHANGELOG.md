# Changelog — config

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/)
y [Versionado Semantico](https://semver.org/lang/es/).

## [Unreleased]

## [0.1.0] - 2026-06-25

### Added

- Version inicial del modulo `config`.
- `Loader` con `New(opts...)` y opciones `WithEnvPrefix` y `WithFile`.
- `Unmarshal(into any)` desde archivo YAML (no falla si el archivo no existe).
- Getters tipados de entorno `GetString`, `GetInt` y `GetBool` con fallback a
  default y prefijo configurable.
