# Changelog — auth

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/)
y [Versionado Semantico](https://semver.org/lang/es/).

## [Unreleased]

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
