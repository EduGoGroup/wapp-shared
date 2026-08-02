# Changelog — auth

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/)
y [Versionado Semantico](https://semver.org/lang/es/).

## [0.3.0] - 2026-08-01

### Removed

- **BREAKING — el paquete raíz `auth` deja de existir.** Ya no hay
  `package auth`: `github.com/EduGoGroup/wapp-shared/auth` es solo la raíz del
  módulo, y todo se importa desde `auth/jwt`, `auth/password` o `auth/rbac`
  (Plan 038, separación por responsabilidad). Sus errores centinela
  (`ErrInvalidInput`, `ErrTokenExpired`, `ErrInvalidToken`, con texto
  `auth: …`) se eliminan: los equivalentes de `auth/jwt` son **valores
  distintos** (texto `auth/jwt: …`), así que un `errors.Is` contra los viejos
  compilaría y devolvería `false` en runtime. Borrarlos hace que el compilador
  delate cada uso pendiente en vez de fallar en silencio.

### Changed

- **BREAKING — migración de import paths, símbolo a símbolo.** La API no cambia
  de forma ni de semántica; solo cambia el paquete que la aloja:

  | Antes (`auth`) | Ahora |
  | --- | --- |
  | `JWTManager`, `NewJWTManager`, `NewJWTManagerES256`, `NewJWTVerifierES256`, `(*JWTManager).WithKid` | `auth/jwt` |
  | `MultiVerifier`, `NewMultiVerifier`, `VerifierKey`, `HS256VerifierKey`, `ES256VerifierKey` | `auth/jwt` |
  | `Claims`, `TokenUseAccess`, `TokenUseRefresh` | `auth/jwt` |
  | `RefreshToken`, `GenerateRefreshToken`, `HashToken` | `auth/jwt` |
  | `ServiceClaims`, `ServiceJWTManager`, `NewServiceJWTManager`, `TokenUseService` | `auth/jwt` |
  | `ErrInvalidInput`, `ErrTokenExpired`, `ErrInvalidToken` | `auth/jwt` (texto nuevo: `auth/jwt: …`) |
  | `HashPassword`, `VerifyPassword` | `auth/password` |
  | `Grants`, `PermissionMatches`, `EvaluateGrants` | `auth/rbac` |
  | `ResolveRoleChain`, `MergeGrantChain` | `auth/rbac` |

  `auth/jwt` declara `type Grants = rbac.Grants` (**alias**, no tipo nuevo): los
  claims se construyen indistintamente con `jwt.Grants` o `rbac.Grants`, sin
  conversión. Un consumidor que solo emite y valida tokens no necesita importar
  `auth/rbac`.

### Added

- Interfaces para inyección de dependencias (DIP) en los consumidores:
  - `auth/jwt`: `TokenGenerator`, `TokenVerifier` y `TokenManager` (composición
    de las dos anteriores). `*JWTManager` satisface `TokenManager`;
    `*MultiVerifier`, que es solo-verificación, satisface `TokenVerifier`.
  - `auth/password`: `Hasher` con su implementación `DefaultHasher` y el
    constructor `NewHasher()`; las funciones `HashPassword`/`VerifyPassword`
    siguen disponibles para el uso directo.
  - `auth/rbac`: `PermissionEvaluator`, contrato para los evaluadores de
    permisos que cada consumidor implementa sobre su propio contexto.
- `auth/password` estrena su propio `ErrInvalidInput` (texto
  `auth/password: invalid input`).
- Doc de paquete (`doc.go`) en cada subpaquete, heredando y repartiendo el del
  paquete raíz eliminado.

## [0.2.0] - 2026-07-10

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
- Selección de llave por `kid` para la coexistencia/rotación de algoritmos
  (ADR-0019, T0 del Plan 028). API HS256/ES256 previa intacta:
  - `(*JWTManager).WithKid(kid)`: devuelve una copia que estampa el header `kid`
    en cada token emitido (aditivo, sirve para cualquier algoritmo del manager).
  - `MultiVerifier` (solo-verificación) con `NewMultiVerifier(issuer, byKid, def)`:
    valida seleccionando la llave por el `kid` del token; una entrada "default"
    valida los tokens SIN `kid` (legacy HS256). Cada entrada FIJA su algoritmo y
    su llave (`HS256VerifierKey(secret)` / `ES256VerifierKey(publicKey)`),
    extendiendo el guard anti alg-confusion por entrada: `kid` desconocido, token
    sin `kid` sin default, o `alg` que no coincide con la entrada ⇒
    `ErrInvalidToken`; `exp` vencido ⇒ `ErrTokenExpired`. Misma semántica de
    issuer/exp que `JWTManager.ValidateToken`.

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
