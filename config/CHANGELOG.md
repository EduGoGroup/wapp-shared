# Changelog — config

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/)
y [Versionado Semantico](https://semver.org/lang/es/).

## [Unreleased]

## [0.3.0] - 2026-08-01

### Added

- Inversión de dependencias en las dos fuentes que el `Loader` toca (Plan 038),
  para poder testear la configuración sin `os.Setenv` ni archivos temporales:
  - `EnvProvider`: contrato de lectura de variables de entorno
    (`LookupEnv(key) (string, bool)`). Por defecto sigue leyendo de `os`.
  - `FileReader`: contrato de lectura del archivo de configuración
    (`ReadFile(path) ([]byte, error)`). Por defecto sigue leyendo de `os`.
  - `MapEnvProvider`: implementación de `EnvProvider` sobre un mapa en memoria,
    pensada para tests.
  - Opciones `WithEnvProvider(EnvProvider)` y `WithFileReader(FileReader)` para
    inyectarlos en `New(opts...)`.

  Cambio **aditivo**: sin opciones, el comportamiento de `New` es el de antes.

## [0.2.0] - 2026-07-10

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
