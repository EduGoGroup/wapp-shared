# web

Middleware web **endurecido** de wApp: la capa transversal que toda consola SSR
del ecosistema necesita —nonce y CSP, CSRF double-submit, rate-limit por clave,
allowlist CORS, politica de cookies, deadline por peticion, single-flight de
refresh y traduccion de flashes— en un solo sitio.

Nace **reconciliando** dos copias que ya habian divergido: la del BFF del cliente
(`wapp-guardian-bff/internal/web`) y la de la consola de operadores
(`wapp-platform-console/internal/web`). **Ninguna de las dos gano entera**: lo que
se sirve aqui es la union de lo mejor de cada una, y cada divergencia esta anotada
en el CHANGELOG.

Es un modulo de **nivel 0**: no importa ningun otro modulo de `wapp-shared`.

## Instalacion

```bash
go get github.com/EduGoGroup/wapp-shared/web
```

## Los dos paquetes

| Paquete | Import | Que trae | Dependencias |
| --- | --- | --- | --- |
| `web` | `github.com/EduGoGroup/wapp-shared/web` | Las decisiones: politica CSP, comparacion CSRF, algoritmo del limitador, allowlist CORS, cookies, sesion, single-flight, flash | stdlib + `golang.org/x/time/rate` |
| `webgin` | `github.com/EduGoGroup/wapp-shared/web/gin` | El adaptador a Gin: un middleware por pieza, delgado | + `github.com/gin-gonic/gin` |

El paquete del directorio `gin/` se llama **`webgin`**, no `gin`: si se llamara
`gin` colisionaria con `github.com/gin-gonic/gin` —que todo consumidor importa
tambien— y obligaria a cada uno de ellos a poner un alias. El alias iria en N
sitios; el nombre distinto va en uno.

La particion no es estetica: este es el primer sitio del monorepo que trae Gin, y
quien solo quiere el algoritmo (por ejemplo para un servidor que no use Gin) no
tiene por que arrastrar los handlers.

## Montaje (el orden NO es negociable)

```go
webgin.SetReleaseMode()
engine := gin.New()
if err := webgin.SetTrustedProxies(engine, cfg.TrustedProxies); err != nil {
    return err // allowlist malformada: no arrancar
}
engine.Use(gin.Recovery())
engine.Use(webgin.SlogLogger())
engine.Use(webgin.SecurityHeaders(web.SecurityOptions{HSTS: cfg.HSTSEnabled}))
engine.Use(webgin.CORS(web.CORSOptions{AllowedOrigins: web.ParseOrigins(cfg.AllowedOrigins)}))

if cfg.RateLimitEnabled {
    limiter := web.NewKeyedRateLimiter(web.RateLimiterOptions{
        RPS: cfg.RateLimitRPS, Burst: int(cfg.RateLimitBurst),
    })
    engine.Use(webgin.RateLimit(limiter))
}

// ... /static y /healthz aqui: no renderizan formularios ni mutan estado ...

engine.Use(webgin.BodyLimit(maxImportBody, "/catalog/import")) // ANTES del CSRF
engine.Use(webgin.CSRF(web.CSRFOptions{
    CookieName: "wapp_guardian_csrf", Secure: cfg.CookieSecure,
}))
```

- **`BodyLimit` va ANTES del `CSRF`.** El CSRF lee el formulario para comparar el
  token y con eso consume el cuerpo entero (a memoria hasta `MaxMultipartMemory` y
  a disco lo que sobre). Un tope montado despues llega cuando el daño ya esta
  hecho.
- **Las cabeceras de seguridad, antes de cualquier handler que renderice**: el
  nonce lo siembra ese middleware y el renderizador lo lee.
- **El CSRF valida antes de sembrar.** Si sembrase primero, el 403 saldria con un
  `Set-Cookie` que el atacante puede provocar a voluntad.

## Fail-closed

La entropia es un unico punto de fallo compartido: `web.Nonce` y
`web.NewCSRFToken` leen del mismo `io.Reader` (`crypto/rand` por defecto,
inyectable por `Rand` en las opciones para los tests). Si falla, se responde 500 y
**no** se emite CSP: servir una pagina con bloques inline sin nonce seria servir
sin defensa justo cuando el sistema esta peor.

## Nada de nombres de cookie fijos

`CSRFOptions.CookieName` y `SessionCookieOptions.Name` son **parametros**. Eran
constantes de paquete en las dos copias, y distintas entre si; si el modulo
hubiera fijado una, las consolas del ecosistema se pisarian la cookie entre ellas.

## El limitador no arranca ninguna goroutine

`web.KeyedRateLimiter` purga las claves inactivas **dentro de `Allow`**, amortizado
(como mucho un barrido cada `PurgeEvery`, bajo el mutex que ya se toma). No hay
goroutine de fondo.

Es la salida a un defecto que estaba en **las dos** implementaciones anteriores:
una descartaba el limitador al construir el router y filtraba su goroutine de
barrido para siempre; la otra llamaba a su `cleanup` de inmediato, con lo que
mataba el barrido dejando el middleware montado y el mapa crecia **sin techo**. La
regla del modulo es que no se expone un constructor que arranque un barrido que
nadie pueda parar.

**Contrapartida asumida**: si el trafico cesa por completo, el mapa se queda con
las entradas que hubiera —no crece mas, pero tampoco se vacia— hasta la siguiente
peticion, que las barre.

`Close()` existe para el dueño del ciclo de vida (el bootstrap, en un `defer` al
apagar): libera el mapa de golpe. Es **idempotente** (`sync.Once`) y **no
inhabilita** el limitador — `Allow` sigue atendiendo y purgando despues de
`Close`, porque hay callers que cierran y luego siguen sirviendo peticiones con el
mismo router. Cerrar es soltar memoria, no apagar la defensa.

`TTL` y `PurgeEvery` son configurables a proposito: con constantes de paquete la
purga solo se podia probar con sleeps largos, o no probarse. Y el desalojo se
prueba **por comportamiento**, no espiando el mapa: con la recarga a ~0 la unica
forma de que una clave ya cortada vuelva a pasar es que su entrada se desalojara.

## El single-flight es generico

`web.RefreshGroup[T]` serializa por clave los refresh concurrentes. Es generico
porque las dos copias lo escribieron con tipos distintos —una con su propio tipo
concreto, la otra con `any`— y ninguna de las dos formas servia para las dos.

La limpieza va en un `defer`, y eso no es estilo: si `fn()` entra en panico y la
limpieza estuviera al final del cuerpo, la clave no se liberaria y **toda**
peticion posterior con esa misma clave se quedaria colgada para siempre. Ademas,
los que esperaban reciben `ErrRefreshPanic` en vez del valor cero con error nil,
que seria un refresh fallido disfrazado de exito.

## Flash: codigos, nunca texto crudo

`web.FlashCatalog` traduce codigos estables (`?error=`/`?success=`) a mensajes
fijos. El texto que se pinta sale **siempre** de la tabla y **nunca** del query
string ni del upstream: un codigo desconocido cae al mensaje generico. Los codigos
son de cada consola, asi que el modulo aporta el mecanismo y la invariante, no el
vocabulario.

## Que NO hay aqui

- **Nada de JWT.** `SessionValid` y `RefreshDue` reciben el `exp` ya extraido; el
  modulo no importa `wapp-shared/auth` ni ninguna libreria de JWT.
- **Nada de negocio**: ni rutas, ni plantillas, ni clientes de API, ni el
  `AuthMiddleware` de cada consola (que depende de su upstream).
- **Nada de config global**: cada pieza recibe lo minimo que usa.
