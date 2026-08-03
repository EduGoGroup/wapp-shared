# Changelog — envelope

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/)
y [Versionado Semantico](https://semver.org/lang/es/).

## [Unreleased]

## [0.2.1] - 2026-08-02

### Security

- `golang.org/x/crypto` sube de `v0.51.0` a `v0.54.0`. `govulncheck` reporta
  vulnerabilidades en la `v0.51.0` con arreglo publicado desde la `v0.52.0`;
  ninguna es alcanzable desde este código hoy, pero `envelope` custodia los
  sobres cifrados y lo que arrastre lo heredan `wapp-cloud-platform` y
  `wapp-edge-agent`. Arrastra también `golang.org/x/sys` (indirecta) de la
  `v0.44.0` a la `v0.47.0`.

  Queda fuera **GO-2026-5932**, que aún no tiene versión con arreglo publicada.

### Changed

- El módulo publicado deja de declarar `go 1.26.0`: `main` ya estaba en
  `go 1.26.5` y este es el primer release que lo hace viajar. Sin cambios de API.

## [0.2.0] - 2026-08-01

### Added

- Contratos explícitos para las dos capas del módulo (Plan 038), de modo que un
  consumidor dependa de la operación y no del tipo concreto:
  - `Crypter` — capa simétrica: `Seal`, `Open`, `Overhead`. La satisface
    `*Envelope`.
  - `Sealer` — capa asimétrica: `SealFor`, `OpenWith`. La satisface el nuevo
    `BoxSealer` (constructor `NewBoxSealer()`), envoltura con receptor de las
    funciones `SealFor`/`OpenWith` ya existentes, que siguen disponibles.
- Aserciones de compilación `var _ Crypter = (*Envelope)(nil)` y
  `var _ Sealer = (*BoxSealer)(nil)`: una deriva entre interfaz e implementación
  rompe el build de este módulo, no el del consumidor.

  Cambio **aditivo**: la API previa no cambia de forma ni de semántica.

## [0.1.1] - 2026-07-10

### Changed

- El método `Envelope.Overhead()` ahora devuelve la constante `Overhead` (única
  fuente de verdad del valor) en lugar de recomputarlo; ambos siguen coincidiendo
  (nonce 12 + tag 16 = 28). Sin cambios de API.

## [0.1.0] - 2026-06-25

### Added

- Version inicial del modulo `envelope` (copia-adaptacion de
  `edugo-shared/crypto/envelope` al namespace wApp). Dos capas independientes:
  - **Simetrica (AES-256-GCM):** `Envelope` con `NewEnvelope`, `Seal`, `Open`,
    `Overhead`; DEK de 32 bytes, nonce aleatorio de 12 bytes prefijado, overhead
    fijo de 28 bytes, autenticacion por tag GCM. Constantes `DEKSize`/`Overhead`;
    errores `ErrKeySize`/`ErrBlobTooShort`.
  - **Asimetrica (sellado X25519 via NaCl box anonimo):** `GenerateKeyPair`,
    `SealFor`, `OpenWith` para el modelo zero-knowledge (ADR-0007, doble llave).
    Constantes `PublicKeySize`/`PrivateKeySize`; errores
    `ErrPublicKeySize`/`ErrPrivateKeySize`/`ErrOpenFailed`.
  - Tests de vectores: round-trip de ambas capas, fallo con clave equivocada,
    fallo ante manipulacion, no-determinismo del sellado anonimo, tamaños de
    clave invalidos, y un caso cruzado del flujo completo sellar-DEK → abrir →
    descifrar.
- Dependencia unica: `golang.org/x/crypto` (curve25519, nacl/box).
