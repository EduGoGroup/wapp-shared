# Changelog — auth

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/)
y [Versionado Semantico](https://semver.org/lang/es/).

## [0.5.0] - 2026-08-14

### Added

- **`JWTManager.GenerateTenantlessToken(userID, ttl)`** — emite un access token
  de usuario **sin tenant**: una identidad acreditada que todavía no pertenece a
  ninguna empresa de wApp (wApp Plan 056 · T3.3 · D-056.12, el estado «en
  espera»). **No acepta roles ni grants**, y esa ausencia ES la garantía: un
  token sin tenant no puede salir de aquí llevando permisos, ni por error ni por
  un descuido futuro. Emite `roles: []` y `grants: {allow: [], deny: []}`, de
  modo que el default DENY del matcher lo cierra todo.
  > Es **aditivo**: `GenerateToken` no cambia de firma ni de comportamiento, y
  > **sigue rechazando** el `tenantID` vacío con `ErrInvalidInput`. Los claims de
  > ambos emisores los arma la MISMA función privada, así que un token con tenant
  > y uno sin él son idénticos campo a campo salvo por lo que trae el llamante.
  > ⚠️ Añade un símbolo público ⇒ **bump MINOR** (`0.5.0`) y tag
  > `auth/v0.5.0` **antes** de que `wapp-cloud-platform` mueva su `go.mod`
  > (regla «shared va primero», tasks.md del Plan 056 §Leyenda).

## [0.4.1] - 2026-08-11

Release de mantenimiento: **solo dependencias**, sin un cambio de API ni de
comportamiento. La API pública del módulo es idéntica símbolo a símbolo a la de
`0.4.0`, así que actualizar es seguro para todos sus consumidores.

### Changed

- Sube `github.com/EduGoGroup/identity-shared/auth` de **0.3.0 a 0.3.1** (patch,
  propuesto por Dependabot). Toca únicamente `auth/go.mod` y `auth/go.sum` — el
  motor de RBAC que wApp importa del SSO del grupo desde la partición de la
  `0.4.0`.

## [0.4.0] - 2026-08-02

Partición por frontera: el módulo se parte en dos y **la mitad de identidad se
va a identity-shared** (identity Plan 003, Ola 1). No es una limpieza, es un
cambio de dueño: los usuarios, sus credenciales y sus sesiones pasan a ser del
SSO del grupo, y wApp se queda **solo con lo que es suyo** — el contexto de
negocio (`tenant_id`/`roles`/`grants`) y su firma.

### Removed

- **BREAKING — `auth/rbac` deja de existir.** El motor de permisos del grupo es
  uno solo y vive en identity: `PermissionMatches`, `EvaluateGrants`, `Grants`,
  `ResolveRoleChain` y `MergeGrantChain` se importan ahora de
  `github.com/EduGoGroup/identity-shared/auth/rbac`, **sin envoltura ni
  adaptador**. La API es idéntica símbolo a símbolo y los tags JSON de `Grants`
  también (`allow`/`deny`), así que el wire format de los tokens no se mueve.
- **BREAKING — `auth/password` deja de existir.** Quien hashea y verifica una
  contraseña es el SSO. `HashPassword`/`VerifyPassword` (bcrypt cost 12, el
  mismo a ambos lados) se importan de `identity-shared/auth/password`. Con él
  cae la dependencia `golang.org/x/crypto`.
- **BREAKING — el refresh opaco sale del módulo**: `RefreshToken`,
  `GenerateRefreshToken` y `HashToken`. La sesión y su renovación son de
  identity (identity ADR-0003) — quien las emite tiene que ser quien las revoca.
  > ⚠️ **No es un reemplazo pieza a pieza.** El de identity prefija **`rft_`** y
  > su hash se llama **`HashRefreshToken`**. Al cambiar el formato, **los hashes
  > de refresh ya almacenados dejan de casar y las sesiones vivas piden
  > re-login**. Es consecuencia asumida, no efecto colateral: hoy son 3 usuarios
  > sintéticos de E2E, wApp está en alfa y los refresh viejos no migran (DUD-16).
  > 📌 Nota para el consumidor: `iam/usecase/apikeys.go` usaba
  > `GenerateRefreshToken` **solo como CSPRNG** para el secreto de una API key.
  > Al migrar decide si adopta el de identity —que prefija `rft_`, y un secreto
  > de API key llamado «refresh token» miente sobre lo que es— o un CSPRNG propio.
- **BREAKING — poda de la superficie pública sin un solo llamador** en los 6
  repos de wApp (verificado archivo a archivo, producción y tests):
  - `TokenGenerator`, `TokenVerifier` y `TokenManager` (el archivo
    `jwt/interfaces.go` entero). Ningún consumidor las adoptó: cloud-platform
    declara las suyas propias duplicando la firma exacta.
  - `TokenUseRefresh`, que describía un JWT de refresh que el módulo ya no emite.
  - `ServiceClaims.HasScope`: cloud-platform resuelve el chequeo de scopes con
    `PermissionMatches`.

### Changed

- **`jwt.Grants` sigue siendo un alias — lo que cambia es a quién apunta**: de
  `wapp-shared/auth/rbac` a `identity-shared/auth/rbac`. Con ese solo movimiento
  el módulo **gana la dependencia `identity-shared/auth v0.3.0`**, y la
  dirección es la correcta: el ecosistema depende del SSO, nunca al revés.

### Qué QUEDA, y por qué

Lo que este módulo conserva no es un resto: es **el plano de contexto de wApp**,
que identity no puede emitir porque identity no firma negocio.

- **`Claims`** con `tenant_id`/`roles`/`grants`: el kernel del **Context Token**
  que la Ola 3 emitirá desde `/auth/exchange` de cloud-platform. Son claims
  **distintos** de los de identity (`system`/`email`/`token_version`) porque son
  **dos tokens distintos**, no dos versiones del mismo.
- **`JWTManager` ES256 + `kid` y el `MultiVerifier`**: emisión y verificación de
  ese plano, incluida la distribución JWKS-push al edge (wApp ADR-0025).
- **`ServiceJWTManager` HS256 (M2M) — transitorio declarado.** Se queda **solo**
  para no arrastrar dos migraciones a la vez y **muere en la Ola 3**, cuando el
  M2M pase a Service Tokens ES256 de identity (identity ADR-0025). Queda escrito
  aquí para que no se vuelva permanente por inercia.
- `NewJWTManager` (HS256 de usuario) y `HS256VerifierKey` **sobreviven a la
  poda** pese a no tener llamador de producción: los tests de cloud-platform aún
  los usan. Salen con el resto del plano HS256 en la Ola 3.

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
