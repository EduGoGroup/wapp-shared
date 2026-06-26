# Changelog

Todos los cambios relevantes del repositorio `wapp-shared` se registran aqui.

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/)
y el repositorio adopta [Versionado Semantico](https://semver.org/lang/es/) por modulo.

## [Unreleased]

### Added

- Estructura inicial del monorepo multi-modulo `wapp-shared` con la ingenieria de
  releases (Makefile raiz, `scripts/` y workflows CI/release) adaptada desde
  `edugo-shared`.
- Modulo `logger`: logging estructurado sobre `log/slog`.
- Modulo `config`: carga de configuracion YAML con overlay de entorno.
