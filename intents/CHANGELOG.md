# Changelog — intents

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/)
y [Versionado Semantico](https://semver.org/lang/es/).

## [Unreleased]

## [v0.1.0] - 2026-07-11

### Added

- Version inicial del modulo `intents`: contrato canonico de configuracion de
  intenciones por tenant y su validacion estructural (destino `intents/v0.1.0`).
  - Tipos `Config`, `Intent`, `Ejemplo` con las claves JSON canonicas (mezcla
    es/en deliberada, heredada del prototipo validado).
  - Constantes `DefaultThreshold` (0.6), `MaxConfigBytes` (256 KiB) y
    `ReservedUnknown` (`desconocido`, prohibido declararlo).
  - `ParseAndValidate([]byte) (*Config, error)`: valida tamano, JSON (tolerante
    a campos futuros, sin `DisallowUnknownFields`), `version` no vacia,
    normalizacion del umbral (ausente/0 ⇒ `DefaultThreshold`, presente ⇒
    rango (0,1]), >=1 intent, patron/unicidad de `name` y `params`, descripcion
    no vacia, >=1 ejemplo con mensaje no vacio y claves de ejemplo ⊆ params.
  - Error centinela `ErrInvalidConfig`; todos los rechazos lo envuelven con el
    intent/campo ofensor.
  - Solo stdlib (sin dependencias de otros modulos de wapp-shared).
