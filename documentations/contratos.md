# Contratos de `wapp-shared`

> Todo lo que otros consumen de esta pieza, y todo lo que esta pieza le pide al mundo.
> Verificado contra `main` a **2026-08-30** (HEAD `ff741c0`).

---

## 0. Lo primero, porque cambia la forma de este documento

Esta pieza **no sirve ninguna ruta HTTP, no expone ningún gRPC, no ofrece ningún comando CLI, no
lee ninguna variable de entorno por nombre, no escribe ningún fichero y no toca ninguna base de
datos.** Los cinco están verificados abajo, cada uno con el comando que lo demuestra.

Su contrato tiene, por tanto, **cuatro caras**:

1. **La API Go** que los otros repos importan (§1–§2).
2. **Las rutas HTTP que este código LLAMA** hacia afuera (§3).
3. **Las formas de datos** que fija y que atan a los consumidores (§4).
4. **El contrato de release**: el nombre del tag y el formato del CHANGELOG (§6).

---

## 1. La superficie Go: qué se importa y desde dónde

**Fuente de la lista:** los `go.mod` de los seis repos hermanos
(`grep -n "wapp-shared" <repo>/go.mod`) cruzados con los imports reales
(`grep -rhoE '"github.com/EduGoGroup/wapp-shared/[a-z/]+"' <repo> --include='*.go'`). La regla de
conteo es **ficheros que contienen al menos un import de ese módulo**, no número de llamadas.

| Consumidor | Módulos pineados en su `go.mod` | Ficheros que importan cada uno |
|---|---|---|
| `cloud/wapp-cloud-platform` | `auth v0.5.0` · `config v0.3.0` · `envelope v0.2.1` · `health v0.1.1` · `intents v0.1.0` · `llm v0.4.5` · `logger v0.2.0` · `textmatch v0.1.0` | logger 115 · llm 51 · auth/jwt 21 · textmatch 13 · envelope 9 · health 5 · llm/api 3 · intents 3 · config 1 |
| `edge/wapp-edge-agent` | ⚠️ `auth v0.4.1` · `config v0.3.0` · `envelope v0.2.1` · `intents v0.1.0` · `logger v0.2.0` | logger 73 · envelope 23 · auth/jwt 5 · intents 3 · config 1 |
| `edge/wapp-edge-intent` | `intents v0.1.0` (**su única dependencia**) | intents 4 |
| `guardian/wapp-guardian-bff` | `auth v0.5.0` · `config v0.3.0` · `iam v0.1.0` · `logger v0.2.0` · `ui v0.4.1` · `web v0.2.0` | web 12 · web/gin 7 · iam 3 · auth/jwt 3 · ui 1 · logger 1 · config 1 |
| `guardian/wapp-client-console` | `auth v0.5.0` · `config v0.3.0` · `iam v0.1.0` · `ui v0.4.1` · `web v0.2.0` | web 13 · web/gin 9 · auth/jwt 4 · ui 2 · iam 2 · config 1 |
| `guardian/wapp-platform-console` | `auth v0.5.0` · `config v0.3.0` · `iam v0.1.0` · `ui v0.4.1` · `web v0.2.0` | web 10 · web/gin 6 · ui 2 · iam 2 · auth/jwt 2 · config 1 |
| `cloud/wapp-cloudlink` | **ninguno** — su `go.mod` solo pide gRPC y protobuf | 0 |
| `guardian/wapp-guardian-app` | KMP, sin `go.mod` | — |

Hechos que salen de la tabla y conviene no repetir mal:

- ⚠️ **`edge-agent` está pineado en `auth v0.4.1`** cuando el tag más alto es `v0.5.0`
  (`edge/wapp-edge-agent/go.mod:9`). Confirmado también en el binario desplegado en UAT. Es el
  único desalineamiento del ecosistema.
- **Ningún `require` de `wapp-shared` está marcado `// indirect`** en ningún consumidor.
- Los once módulos están dados de alta en el `go.work` de la raíz `wApp/` (fuera de este repo).

---

## 2. Los símbolos que forman el contrato, por módulo

No es la lista completa de exportados: son los que **rompen a alguien** si cambian.

### `logger`
`Logger` (interfaz: `Debug`/`Info`/`Warn`/`Error`/`With`), `New(opts...)`, `WithLevel`,
`WithJSON`, y el logger en contexto (`logger/context.go`). Semántica variádica clave/valor de slog.

### `config`
`New(opts...) *Loader`, `WithFile(path)`, `WithEnvPrefix(prefix)`, `Loader.Unmarshal(&cfg)` y los
getters tipados con default (`GetString`, …). Andamiaje de inyección: `EnvProvider`, `FileReader`,
`MapEnvProvider`, `WithEnvProvider`, `WithFileReader` — hoy solo lo usan sus propios tests.

### `envelope`
`Envelope`, `NewEnvelope`, `Envelope.Seal`, `Envelope.Open`, `Envelope.Overhead`;
`GenerateKeyPair`, `SealFor`, `OpenWith`. Constantes de formato:
`Overhead = 28`, `DEKSize = 32`, nonce de 12 B, tag de 16 B (`envelope/envelope.go:13-26`); claves
X25519 de 32 B (`envelope/sealing.go:13-16`).

### `health`
`HealthCheck` (interfaz `Name`+`Check`), `Checker`, `NewChecker`, `Checker.Register`,
`Checker.CheckAll` → `map[string]CheckResult`, `Checker.IsHealthy`. `Status` con
healthy/unhealthy/degraded.

### `auth/jwt`
`Claims` (`auth/jwt/jwt_claims.go:31`) con `TokenUseAccess = "access"`; `Grants` **alias** de
`identity-shared/auth/rbac.Grants`. Constructores: `NewJWTManager` (HS256),
`NewJWTManagerES256`, `NewJWTVerifierES256`, `JWTManager.WithKid`, `JWTManager.GenerateToken`,
`GenerateTenantlessToken`, `ValidateToken`. Verificación por `kid`: `NewMultiVerifier`,
`HS256VerifierKey`, `ES256VerifierKey`. Service token M2M: `NewServiceJWTManager`,
`GenerateServiceToken`, `ValidateServiceToken`, `ServiceClaims`, `TokenUseService = "service"`.
Tolerancia de reloj: `clockLeeway = 30 * time.Second` (`auth/jwt/jwt_manager.go:15`).

### `web` (solo stdlib + `x/time/rate`)
- **CSRF:** `CSRFFieldName = "csrf_token"`, `CSRFHeaderName = "X-CSRF-Token"`
  (`web/csrf.go:15,17`), `CSRFOptions`, `CSRFCookie`, `ValidateCSRF`, `IsUnsafeMethod`,
  `DefaultCSRFMaxAge = 12h`.
- **CSP y cabeceras:** `BuildCSP(nonce)`, `CSPDirectives`, `ApplySecurityHeaders`, `Nonce(r)`,
  `NewCSRFToken(r)`, `SecurityOptions`.
- **Cookies:** `SessionCookieOptions`, `SessionCookie`, `SameSiteMode`, `SessionMaxAge`,
  `EncodeCookiePayload`/`DecodeCookiePayload`, `DefaultSessionMaxAge = 1h`.
- **Un solo uso (PRG):** `OneTimeCookieOptions`, `OneTimeCookie`, `ClearOneTimeCookie`,
  `DefaultOneTimeCookieMaxAge = 60s`.
- **Sesión:** `SessionData`, `EncodeSession`, `DecodeSession`, `SessionValid(exp)`,
  `RefreshDue(exp, margin)` — reciben el `exp` **ya extraído**, no un tipo de `auth`.
- **Rate limit:** `NewKeyedRateLimiter`, `RateLimiterOptions`, `UserRateKey`, `IPRateKey`,
  `RateKey`, `KeyedRateLimiter.Close`.
- **CORS y proxies:** `NewCORSPolicy`, `CORSOptions`, `ParseOrigins`, `AppendVary`,
  `ParseTrustedProxies`.
- **Otros:** `NewBodyLimit`, `NewFlashCatalog`, `NewRefreshGroup[T]` (single-flight genérico).

**Nombres de cookie por defecto — todos parametrizables a propósito** (§2 P3 de la constitución):
`wapp_session` (`web/cookies.go:29`), `wapp_csrf` (`web/csrf.go:42`), `wapp_once`
(`web/onetime.go:11`).

### `web/gin` (paquete `webgin`)
Middlewares: `SecurityHeaders`, `CSRF`, `CORS`, `RateLimit`, `RateLimitKey`, `RequestDeadline`,
`BodyLimit`, `SlogLogger`. Cookies: `SetSessionCookie`, `ClearSessionCookie`,
`SessionCookieValue`, `SetOneTimeCookie`, `TakeOneTimeCookie`. Render: `NewRenderer(layout)`.
Infra: `SetTrustedProxies`, `SetReleaseMode`.
**Claves compartidas de `gin.Context`** (`web/gin/context.go:8-37`): `ContextCSPNonce`,
`ContextCSRFToken`, `ContextUserID`, `ContextTenantID`, `ContextAccessToken`,
`ContextRefreshToken`, con sus lectores `NonceFromContext`, `CSRFTokenFromContext`,
`UserIDFromContext`, `TenantIDFromContext`, `AccessTokenFromContext`, `RefreshTokenFromContext`,
`IsAuthenticated`.

### `iam`
`NewClient(Options) (*Client, error)`, `Client.Login`, `Client.Refresh`, `Client.Logout` (los que
se usan) y los escalones sueltos `IdentityLogin`, `IdentityRefresh`, `Exchange`. Tipos:
`Options` (`System`, `IdentityBaseURL`, `PlatformBaseURL`, `Timeout`, `HTTPClient`), `AuthResult`,
`IdentityTokens`, `APIError`. `DefaultTimeout = 15s` (`iam/client.go:19`); tope de decodificación
`maxResponseBody = 1 MiB` (`iam/client.go:23`).

### `intents`
`Config`, `Intent`, `Ejemplo`, `ParseAndValidate(raw)`, `ErrInvalidConfig`,
`MaxConfigBytes = 256 KiB`, `DefaultThreshold = 0.6`.

### `llm`
`LLMProvider` (5 métodos), los `*Input` de cada etapa, `Options{Temperature}`,
`TemperatureGreedy = 0.0`, `TemperatureRetry = 0.3`, `IntentSpec`, `IntentExample`,
`ExtractJSON`, los `Parse*`, los artefactos (`ArtifactVersion = 1`), `ErrLLMQuality`,
`Etapa` (`"p2"`…`"p5"`), `ValidarPlantilla`, los `Build*Prompt`.

### `llm/api`
`New(Config)`, `Config`, `ProviderAnthropic = "anthropic"`, `ProviderGemini = "gemini"`,
`DefaultTimeout = 60s`, `DefaultMaxTokens = 4096`,
`DefaultAnthropicBaseURL = "https://api.anthropic.com"`, y los centinelas `ErrInvalidConfig`,
`ErrUnsupportedProvider`, `ErrNotImplemented`, `ErrUpstream`.

### `textmatch`
Nivel 1 (**el que se usa**): `Normalize`, `SplitTokens`, `Result`, `OutcomeMatch`, `NewFuzzy`,
`Cascade`, `Exact`, `EditDistance`, `Comparator`, `GrayZone`.
Nivel 2 (**sin consumidores hoy**): `SetMatcher`, `NewSetMatcher`, `MatchReport`, `Policy`,
`GenerateCandidates`, `Candidate`.

### `ui`
`ui.Assets` (`embed.FS`), `ui.FS()`, `ui.GetCSS(name)`. Las cinco hojas y su orden de carga
obligatorio (tokens → componentes → tema) son parte del contrato.

---

## 3. Rutas HTTP — este repo **no sirve ninguna**; estas son las que **llama**

**Cómo se obtuvo:** no hay fichero de registro de rutas porque no hay servidor. La lista sale de
`grep -rnE '"/[a-zA-Z0-9/_.{}-]*"' iam/*.go llm/api/*.go`, más los constructores de URL de cada
cliente. La regla de conteo es **una fila por combinación método+ruta que el código compone**.

**Cómo se comprueba que no sirve nada:** `grep -rn 'http.ListenAndServe\|gin.Default()\|\.GET(\|\.POST('`
sobre ficheros de producción → nada fuera de `web/gin`, que **recibe** el `*gin.Engine` del
consumidor y **no registra rutas propias**.

### 3.1 Plano de identidad — módulo `iam`

Dos destinos distintos, configurados en `iam.Options` como `IdentityBaseURL` y `PlatformBaseURL`.

| Método | Ruta | Contra | Función | Evidencia |
|---|---|---|---|---|
| `POST` | `{identity}/api/v1/auth/login` | identity-api | `Client.IdentityLogin` | `iam/identity.go:48` |
| `POST` | `{identity}/api/v1/auth/refresh` | identity-api | `Client.IdentityRefresh` | `iam/identity.go:59` |
| `POST` | `{identity}/api/v1/auth/logout` | identity-api | `Client.Logout` | `iam/identity.go:75` |
| `POST` | `{platform}/api/v1/auth/exchange` | wapp-cloud-platform | `Client.Exchange` | `iam/exchange.go:40` |

Cabeceras fijas en las cuatro: `Content-Type: application/json` y `Accept: application/json`
(`iam/client.go:120-121`). `Login` y `Refresh` encadenan dos de estas llamadas y devuelven
**siempre** el Context Token. **Ninguna respuesta de este cliente se registra ni se incluye en un
error**: son credenciales.

### 3.2 Proveedor de modelo — módulo `llm/api`

| Método | Ruta | Host por defecto | Evidencia |
|---|---|---|---|
| `POST` | `/v1/messages` (Anthropic Messages API) | `https://api.anthropic.com` | `llm/api/anthropic.go:16`; `llm/api/api.go:47` |

Cabeceras: `x-api-key: <credencial del tenant>` y `anthropic-version: 2023-06-01`
(`llm/api/anthropic.go:18-23`). Topes: `maxResponseBytes = 4 MiB` de cuerpo leído,
`DefaultMaxTokens = 4096` de salida, `DefaultTimeout = 60s` por llamada.
🔴 **`gemini` es un stub**: se construye sin error y devuelve `ErrNotImplemented` en sus cinco
métodos (`llm/api/gemini.go:32-53`).

---

## 4. Contratos de datos (esto es lo que de verdad ata a los consumidores)

### 4.1 `intents` — la configuración de intenciones por tenant

Forma (`intents/intents.go:33-54`). Las claves **mezclan español e inglés a propósito**; no se
renombran.

```
Config  { version, umbral_confianza?, intents[], vocabulario[]? }
Intent  { name, descripcion, params[]?, ejemplos[] }
Ejemplo { mensaje, params{}? }
```

Reglas duras de `ParseAndValidate` (`intents/intents.go:62-120`): `MaxConfigBytes = 256 KiB`;
`version` no vacía; ≥ 1 intent; `name` contra `^[a-z][a-z0-9_]{1,63}$`; `"desconocido"`
**reservado**; nombres únicos; `umbral_confianza` en (0,1] o `DefaultThreshold = 0.6`.
**Tolera campos desconocidos a propósito.**

### 4.2 `llm` — el puerto, los artefactos y las plantillas

Cinco métodos, uno por etapa; todos devuelven `json.RawMessage`. Artefactos versionados con
`ArtifactVersion = 1` (`llm/parse.go:17`): `Classification`, `MainIdeas`, `ItemSpecs`,
`Quantities`, `QuoteText`, `NormalizedItem`, `Range`.

**Plantillas de prompt ajustables sin release** (`llm/plantilla.go`): `Etapa` es `"p2"`, `"p3"`,
`"p4"`, `"p5"`, y **solo el prefijo `pN-` del nombre de fichero es contrato** — es lo que empareja
fichero con etapa (`llm/plantilla.go:49-51`). `ValidarPlantilla` (`:142`) exige **dos** cosas: que
el esquema **relleno** lo acepte el `Parse*` de su etapa, y que el esquema **crudo** lo siga
rechazando. **P1 no va por aquí**: su prompt lo gobierna el catálogo de intenciones.

### 4.3 `web` — nombres, campos y claves de contexto

Ver §2. Lo que es literal de contrato y no se puede cambiar sin coordinar con las plantillas de las
tres consolas: `CSRFFieldName = "csrf_token"` y `CSRFHeaderName = "X-CSRF-Token"`.

### 4.4 `envelope` — el formato en disco

`nonce(12B) || ciphertext || tag(16B)`, `Overhead = 28`, `DEKSize = 32`. Cambiarlo invalida todo lo
cifrado que ya existe en el Edge.

### 4.5 `auth/jwt` — los claims

`{tenant_id, user_id, roles, grants}` + `token_use`. `TokenUseAccess = "access"`,
`TokenUseService = "service"`. El `kid` viaja en la cabecera del JWT.

---

## 5. Variables de entorno, ficheros y esquemas: los tres a cero

### 5.1 Variables de entorno — **este repo no lee ninguna por nombre**

**Cómo se comprueba:** `grep -rn "os.Getenv\|os.LookupEnv" --include='*.go' . | grep -v _test`
devuelve exactamente tres líneas, todas en `config/provider.go:16,20,36`, y son el **mecanismo
genérico**: `osEnvProvider.LookupEnv`. La clave la pone el llamante y el prefijo también, vía
`config.WithEnvPrefix` (`config/config.go:36`). `llm/api/api.go:15-17` lo declara explícito para su
paquete: «este paquete no lee variables de entorno ni ficheros».

🔴 **Nombre efectivo, para quien lea el código de un consumidor:** con
`config.WithEnvPrefix("WAPP_")` —que es lo que usan las tres consolas— el literal `LOG_LEVEL` del
código es la variable **`WAPP_LOG_LEVEL`** en el entorno. No busques `WAPP_` en este repo: aquí no
está.

### 5.2 Ficheros que lee o escribe — **ninguno en producción**

**Lee:** `config` puede leer un YAML, pero **la ruta la da el llamante** (`WithFile`) y `Unmarshal`
no falla si no existe. `ui` "lee" sus cinco CSS, pero están **embebidos en el binario** con
`//go:embed` (`ui/ui.go:11`): no hay E/S de disco.
**Escribe:** nada. `grep -rn "os.WriteFile\|os.Create\|os.OpenFile\|os.MkdirAll"` solo devuelve
`config/config_test.go:17` y `.eval-t183/gen.go:61` (que ningún gate compila).

### 5.3 Tablas y esquemas — **ninguno**

Sin base de datos, sin migraciones, sin versión de esquema. `find . -name '*.sql'` → vacío.
`grep -rniE "database/sql|pgx|sqlite|postgres|CREATE TABLE|migrat"` sobre `*.go` solo devuelve el
nombre de un stub de test (`health/health_test.go:33`) y un comentario (`health/doc.go:19`).

El único nombre de tabla que aparece en el repo es una **referencia documental**: `llm/api/api.go`
menciona la tabla `tenant_llm` del cloud como origen del campo `provider`. La tabla vive en
`wapp-cloud-platform`, no aquí.

### 5.4 gRPC y CLI — **ninguno**

No hay `.proto`, ni servidor, ni cliente gRPC, ni comandos. `find . -name 'main.go'` → vacío.
El binario `cmd/prompts` que vuelca los prompts ajustables **está en `wapp-cloud-platform`**.

---

## 6. El contrato de release (lo que consumen las herramientas, no las personas)

| Elemento | Forma exacta | Quién la exige |
|---|---|---|
| Nombre del tag | `<modulo>/vX.Y.Z` | `scripts/module-release.sh`; `release.yml` dispara con `'*/v*'` y `'*/*/v*'` |
| Sección del CHANGELOG | 🔴 `## [X.Y.Z]` — **SIN la «v»** | `scripts/module-release.sh:84` (`grep -q "^## \[$VERSION_NO_V\]"`) |
| Sección viva previa | `## [Unreleased]` | `scripts/update-module-changelog.sh:112-115` |
| Inventario de módulos | `scripts/module-manifest.tsv`, campos `module\|level\|integration\|coverage_validation` separados por **`\|`** | `scripts/list-modules.sh:58` |
| Alta de módulo nuevo | + la línea `use ./shared/wapp-shared/<modulo>` en el `go.work` de la raíz `wApp/`, **que está en otro repo** | el toolchain de Go |
| Estado del árbol | limpio (`git status --porcelain` vacío) salvo en `--dry-run` | `scripts/module-release.sh` |
| Módulo raíz | **no existe**: no hay `go.mod` en la raíz y por eso el trigger ignora los tags `v*` sueltos | `release.yml:4-9` |
