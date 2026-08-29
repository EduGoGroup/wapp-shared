# Changelog — web

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/)
y [Versionado Semantico](https://semver.org/lang/es/).

## [Unreleased]

## [0.2.0] - 2026-08-28

### Added

- **Cookie efimera de un solo uso** (`onetime.go`): el transporte que faltaba para
  un POST-Redirect-GET que emite un SECRETO. `OneTimeCookieOptions`,
  `OneTimeCookie`, `ClearOneTimeCookie` y, en el adaptador Gin,
  `SetOneTimeCookie` y `TakeOneTimeCookie`.
  - Nace de M-10 de `CODE-REVIEW-2026-08-15` (consola de plataforma): la pantalla
    del codigo de enrolamiento se renderizaba SOBRE el POST, asi que un F5
    reenviaba el POST y emitia un codigo nuevo dejando el anterior huerfano y
    vivo 24 h. Con PRG el secreto tiene que viajar del POST al GET siguiente, y
    las dos vias que ya habia no servian: `FlashCatalog` va por query string (el
    secreto acabaria en el log de acceso, en el `Referer` y en el historial) y
    `SessionData` viaja en la cookie de sesion, que dura horas y va en todas las
    peticiones del sitio.
  - **NO cifra ni firma el valor, y es deliberado**: el destinatario del secreto
    es exactamente quien tiene la cookie, y el secreto se le pinta en pantalla de
    todas formas. Cifrar lo protegeria del propio destinatario; firmar detectaria
    que se enganio a si mismo. Lo unico que compra la cookie es que el secreto no
    pase por la URL, y eso lo da el transporte. Una llave aqui seria una llave que
    custodiar y rotar sin cerrar ninguna fuga. La razon esta escrita en el
    doc-comment del tipo, no solo aqui.
  - `HttpOnly` va SIEMPRE y no es parametro (mismo criterio que `SessionCookie`).
    `Secure` y `SameSite` siguen la config de cada consola via `SameSiteMode`.
  - `Path` se acota a la PANTALLA destino y vacio NO se rellena a `/`: rellenarlo
    convertiria un olvido del llamante en una cookie con secreto enviada en todas
    las peticiones del sitio. Hay test que lo fija.
  - `MaxAge` corto (60 s por defecto) como TOPE DE SEGURIDAD; quien la retira de
    verdad es el GET que la consume. `TakeOneTimeCookie` lee y borra en UN SOLO
    GESTO, y emite el borrado aunque no hubiera nada que leer: si colgara de una
    rama, cualquier salida temprana del handler dejaria el secreto vivo en el
    navegador. El criterio se prueba con un `cookiejar` real, no leyendo cabeceras.
- **`EncodeCookiePayload` / `DecodeCookiePayload`** (`cookies.go`): empaquetado
  JSON + base64 URL-safe sin padding para el valor de cualquier cookie con
  contenido estructurado. El alfabeto no es decorativo: los adaptadores de
  framework aplican `url.QueryUnescape` al leer, y un `+` del base64 estandar
  volveria como espacio.

### Changed

- `EncodeSession`/`DecodeSession` pasan a delegar en `EncodeCookiePayload`/
  `DecodeCookiePayload`. El empaquetado era el mismo y ahora vive una sola vez;
  el valor de la cookie de sesion no cambia ni un byte.

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

