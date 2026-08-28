# Changelog — web

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/)
y [Versionado Semantico](https://semver.org/lang/es/).

## [Unreleased]

## [0.1.0] - 2026-08-28

### Added

- Version inicial del modulo `web`: el middleware web endurecido, extraido de las
  DOS copias que ya habian divergido (`wapp-guardian-bff/internal/web` y
  `wapp-platform-console/internal/web`) para que la UI del Plan 047 no sea la
  tercera.
  - Paquete `web` (solo stdlib + `golang.org/x/time/rate`): `BuildCSP`,
    `CSPDirectives`, `ApplySecurityHeaders`, `Nonce`, `NewCSRFToken`,
    `CORSPolicy`/`ParseOrigins`/`AppendVary`, `ValidateCSRF`/`CSRFCookie`/
    `IsUnsafeMethod`, `KeyedRateLimiter`/`RateKey`, `SameSiteMode`/
    `SessionCookie`/`SessionMaxAge`, `SessionData`/`SessionValid`/`RefreshDue`,
    `RefreshGroup[T]`, `FlashCatalog`, `BodyLimit` y `ParseTrustedProxies`.
  - Paquete `webgin` (directorio `gin/`): adaptador delgado a Gin —
    `SecurityHeaders`, `CORS`, `CSRF`, `RateLimit`, `RequestDeadline`,
    `BodyLimit`, `SlogLogger`, `SetTrustedProxies`, `SetReleaseMode`, `Renderer`
    y los accesores del contexto. El paquete NO se llama `gin` para no colisionar
    con `github.com/gin-gonic/gin` en el bloque de imports de cada consumidor.

### Changed

Las divergencias entre las dos copias, y con que se quedo el modulo:

- **CSP**: la union de las dos. Del BFF, `object-src 'none'` y `font-src 'self'`;
  de la consola de plataforma, `base-uri 'none'` (mas estricto que el `'self'`
  del BFF).
- **Nonce y token CSRF**: comparten fuente de entropia inyectable, asi que hay un
  unico punto de fallo y un unico camino de fail-closed.
- **CSRF**: TTL de 12 h configurable, aviso por `slog` al rechazar, cookie
  HttpOnly + SameSite=Lax siempre, y **validar ANTES de sembrar** (la copia que
  sembraba primero devolvia el 403 con un `Set-Cookie` que el atacante podia
  provocar a voluntad).
- **CORS**: del BFF, `Access-Control-Max-Age` y el `Vary: Origin`; de la consola,
  el **403 en el preflight** de un origen no permitido y admitir `Authorization` y
  `X-CSRF-Token`. El `Vary` pasa a emitirse tambien cuando el origen se niega: la
  respuesta depende del `Origin` en los dos casos. Las dos rechazaban `"*"`, y se
  conserva.
- **Rate-limit**: gana la **purga PEREZOSA y amortizada dentro de `Allow`** (la
  forma que la consola de plataforma acaba de dejar verificada en campo), NO el
  `sweepLoop` con goroutine del BFF. Las dos implementaciones anteriores estaban
  rotas por el mismo sitio: el BFF descartaba el limitador en `NewRouter` y dejaba
  su goroutine viva para siempre, una por router; la consola llamaba al cleanup de
  inmediato y dejaba el mapa creciendo sin techo. Sin goroutine no ocurre ninguna
  de las dos. Contrapartida asumida y escrita en el codigo: si el trafico cesa del
  todo, el mapa se queda con lo que hubiera hasta la siguiente peticion.
  Del BFF se conservan los prefijos `u:`/`ip:` en las claves (sin ellos un
  `user_id` podia colisionar con una IP), el `Retry-After` en el 429, el log del
  rechazo SIN la clave cruda y la idempotencia del cierre. `Close()` libera el
  mapa, es idempotente (`sync.Once`) y **no inhabilita** el limitador: `Allow`
  sigue atendiendo y purgando despues de cerrar. `TTL` y `PurgeEvery` pasan a ser
  configurables (eran constantes de paquete, y con eso la purga no se podia
  probar), con el reloj inyectable por campo —no por variable global— para que los
  tests corran en paralelo y con `-race`.
- **Single-flight**: gana el `defer` de la consola de plataforma (sobrevive a un
  panico en `fn()`), pero **generico** — ninguna de las dos formas previas
  (`any` en una, tipo concreto en la otra) servia para las dos. Nuevo: los que
  esperan reciben `ErrRefreshPanic` en vez del valor cero con error nil.
- **`gin.SetMode`**: bajo `sync.Once` y en una funcion explicita, no en un
  `func init()` — un modulo compartido no puede tocar el modo global de Gin por el
  mero hecho de ser importado.
- **Nombres de cookie**: dejan de ser constantes de paquete y pasan a ser
  parametros (`CSRFOptions.CookieName`, `SessionCookieOptions.Name`). Eran
  distintos en cada copia; fijar uno haria que las consolas compartieran cookie.
- **`*config.Config`**: desaparece. Cada pieza recibe tipos estrechos
  (`SecurityOptions`, `CORSOptions`, `CSRFOptions`, `RateLimiterOptions`,
  `SessionCookieOptions`, una `time.Duration`), nunca un dios-config.
- **Sin JWT**: `SessionValid` y `RefreshDue` reciben el `exp` ya extraido. El
  modulo es de nivel 0 y no importa `wapp-shared/auth`.
- **`FlashCatalog`**: sube el mecanismo (codigos estables -> mensajes fijos, nunca
  texto crudo del upstream) sin el vocabulario, que es de cada consola.

