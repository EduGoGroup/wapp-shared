# Changelog — config

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/)
y [Versionado Semantico](https://semver.org/lang/es/).

## [Unreleased]

### Added

- Getters con error explícito `GetIntE`, `GetBoolE` y `GetDurationE`: devuelven
  `ErrInvalid` (envuelto) ante un valor presente-pero-inválido, en vez de tragarlo.
- `GetDuration`/`GetDurationE` tipados (formato de `time.ParseDuration`).
- Getters "required" `RequireString`, `RequireInt`, `RequireBool` y
  `RequireDuration`: `ErrMissing` si la clave no está definida, `ErrInvalid` si
  su valor no parsea.
- Errores centinela exportados `ErrMissing` y `ErrInvalid`.

### Changed

- `GetInt`/`GetBool` ya no fallan en silencio ante un valor presente-pero-inválido:
  registran un `slog.Warn` (clave, sin volcar el valor) antes de caer al default.
  La firma y el fallback a default se mantienen (retrocompatible).

## [0.1.0] - 2026-06-25

### Added

- Version inicial del modulo `config`.
- `Loader` con `New(opts...)` y opciones `WithEnvPrefix` y `WithFile`.
- `Unmarshal(into any)` desde archivo YAML (no falla si el archivo no existe).
- Getters tipados de entorno `GetString`, `GetInt` y `GetBool` con fallback a
  default y prefijo configurable.
