# Changelog — envelope

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/)
y [Versionado Semantico](https://semver.org/lang/es/).

## [Unreleased]

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
