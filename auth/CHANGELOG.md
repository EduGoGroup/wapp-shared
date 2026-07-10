# Changelog — auth

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/)
y [Versionado Semantico](https://semver.org/lang/es/).

## [Unreleased]

### Added

- Soporte OPCIONAL de firma **ES256** (ECDSA P-256) en `JWTManager`, coexistiendo
  con HS256 (ADR-0019, H3 del Plan 027). API HS256 intacta:
  - `NewJWTManagerES256(privateKey, issuer)`: firma con clave EC privada y valida
    con su pública.
  - `NewJWTVerifierES256(publicKey, issuer)`: validador de SOLO pública (no puede
    firmar; `GenerateToken` devuelve `ErrInvalidInput`) — mínimo privilegio para
    validadores que no emiten.
  - Guard anti alg-confusion: cada manager valida exclusivamente su algoritmo
    (rechazo cruzado HS256↔ES256 con `ErrInvalidToken`). Sin nuevas dependencias
    (curva P-256 en stdlib).

## [0.1.1] - 2026-07-10

### Changed

- `ValidateToken`: se elimina la doble verificación de `iss`. El issuer lo valida
  únicamente el parser vía `jwt.WithIssuer` (única fuente de verdad); un issuer
  inesperado se sigue propagando como `ErrInvalidToken`. Sin cambios de API ni de
  semántica pública.

## [0.1.0] - 2026-07-03

### Added

- Version inicial del modulo `auth` (copia-adaptacion del modulo auth de EduGo,
  SIN import a repos EduGo; ADR-0004). Primitivas puras de auth/authz para wApp.
- `JWTManager` (HS256) con `NewJWTManager`, `GenerateToken` y `ValidateToken`
  (valida firma, `iss` y `exp`); `Claims{UserID, TenantID, Roles, Grants,
  TokenUse}` SIMPLIFICADO (sin jerarquia escuela/unidad/ward).
- `ServiceJWTManager` para tokens M2M: `ServiceClaims{ClientID, TenantID,
  Scopes}`, `Generate/ValidateServiceToken` (`token_use="service"`, valida
  `aud`).
- `HashPassword`/`VerifyPassword` (bcrypt cost 12).
- `GenerateRefreshToken(ttl)` (refresh token opaco) + `HashToken` (SHA256).
- `PermissionMatches`/`EvaluateGrants`: matcher glob de permisos
  (`recurso.accion`, `*`, `prefix.*`, `*.suffix`, `prefix.*.suffix`) con
  precedencia deny-sobre-allow y default DENY.
- `ResolveRoleChain` (herencia por `parent_role_id`, guard de ciclos) y
  `MergeGrantChain` (aplana el set efectivo de grants de la cadena).
