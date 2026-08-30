# Constitución de `wapp-shared`

> Las reglas de esta pieza. Si algo de aquí choca con lo que te apetece hacer, gana esto.
> Verificado contra el árbol en `main` a fecha **2026-08-30** (HEAD `ff741c0`, 37 tags).

---

## 0. Qué es esta pieza, en una frase

`wapp-shared` es el **monorepo Go multi-módulo de librerías compartidas del ecosistema wApp**:
**once módulos Go independientes**, cada uno con su `go.mod`, su `README.md`, su `CHANGELOG.md` y
su propia línea de versiones (`<modulo>/vX.Y.Z`). **No produce ni un binario y no sirve ni una
ruta.** Su razón de ser es que la política transversal (logging, config, cripto, JWT, middleware
web, cliente de identidad, puerto del LLM, tokens de diseño) tenga **un solo dueño** en lugar de
una copia por repo.

Módulos: `auth` · `config` · `envelope` · `health` · `iam` · `intents` · `llm` · `logger` ·
`textmatch` · `ui` · `web`.

**Cómo se comprueba:** `find . -maxdepth 2 -name go.mod | wc -l` → 11.
`find . -name 'main.go' -o -type d -name cmd` (fuera de `.git`) → **vacío**.

---

## 1. Los invariantes del ECOSISTEMA que te tocan aquí

Este repo se clona solo. Lo que sigue no está enlazado a ningún sitio a propósito: se repite aquí
porque un cambio en `envelope`, `auth` o `llm` puede violarlo sin que nada compile en rojo.

### E1 · Zero-knowledge — la nube nunca ve credenciales ni llaves privadas

El núcleo de wApp corre 24/7 en el equipo del cliente (el **Edge**) y la nube lo gobierna. La nube
**nunca** accede a credenciales de WhatsApp ni a llaves privadas. Protege **llaves**, **no** el
contenido de negocio: los pedidos, cotizaciones y mensajes **sí** suben a la nube, y eso es
deliberado.

**Qué significa aquí:** `envelope` existe para que ese modelo sea posible. Ninguna función de este
repo puede aceptar una llave privada por un canal que el servidor observe, ni registrarla, ni
devolverla en un error. **Cómo se comprueba:** `envelope/doc.go` lo declara; `envelope` solo usa
stdlib y `golang.org/x/crypto`, sin criptografía casera (`envelope/go.mod:6`).

### E2 · Doble llave: la DEK es del cliente, el Lease es del servidor

- La **DEK** (32 bytes, AES-256) descifra el almacén de `whatsmeow`. **La custodia el cliente y
  jamás cruza el contrato hacia la nube en claro.** `envelope` la mueve **sellada**: sellado
  anónimo X25519 / NaCl box (`envelope/sealing.go`), de forma que el transporte no la expone.
- El **Lease** autoriza a operar. **Lo emite y lo revoca el servidor**: es el kill-switch
  anti-clon. No vive en este repo, pero los claims que lo acompañan viajan en el JWT de `auth`.

🔴 **Prohibido**: proponer que la DEK se persista en el servidor, se registre en un log, o viaje
sin sellar. El ADR-0007 (modelo de doble llave), en el repo de documentación del ecosistema, lo
fija; contradecirlo desde una librería es peor que desde una aplicación, porque lo hereda todo el
mundo.

### E3 · Sin Redis ni broker en el Edge — la concurrencia se resuelve con Go

El Edge no tiene Redis, ni RabbitMQ, ni ningún broker. La concurrencia es goroutines, canales y
`sync`. **Consecuencia directa en este repo:** `web.KeyedRateLimiter` es **por instancia de
proceso**, sin estado compartido (`web/ratelimit.go:50-52`); con N réplicas el límite efectivo se
multiplica por N. Está aceptado por escrito como contrapartida, no es un bug — pero no lo
"arregles" metiendo un almacén externo.

### E4 · Copia-adaptación, nunca dependencia: prohibido importar `edugo-*`

Buena parte del código de wApp se **copió y adaptó** de otro producto del grupo (EduGo) y se
renombró al espacio de nombres wApp. **Está prohibido `import "github.com/EduGoGroup/edugo-..."`
en cualquier módulo.** `textmatch/doc.go:5-9` lo declara para su propio caso, y la adaptación
incluso eliminó la dependencia externa que traía el original.

⚠️ **La excepción viva, y no es la misma cosa:** `auth` importa
`github.com/EduGoGroup/identity-shared/auth v0.3.1` (`auth/go.mod:6`) y
`auth/jwt/jwt_claims.go:22` hace `type Grants = rbac.Grants`. `identity-shared` **no** es
`edugo-shared`: es el motor de permisos del grupo, la decisión está justificada por escrito en
`auth/jwt/jwt_claims.go:17-20` y ata la cadencia de `auth` a la de identity. Eso es un hecho, no
una licencia para abrir más puertas.

🔴 **No hay candado.** Ni un test ni `depguard` impiden hoy un import a `edugo-*`. Lo vigila quien
revisa. **Cómo se comprueba:** `grep -rn 'EduGoGroup/edugo' --include='*.go' .` → debe salir vacío.

### E5 · El código compartido interno de wApp vive AQUÍ

Este monorepo **es** ese sitio. Un consumidor depende de `github.com/EduGoGroup/wapp-shared/<modulo>`
con una versión pineada, nunca de un `replace` al árbol de al lado (ver §3.4).

---

## 2. Los invariantes PROPIOS de este repo

### P1 · Un módulo = un `go.mod` = una línea de versiones. Los once son **nivel 0**

`scripts/module-manifest.tsv` es el **inventario único**: los once están declarados `level 0`,
`integration=false`, `coverage_validation=true`. **Nivel 0 significa que ningún módulo de
`wapp-shared` importa a otro módulo de `wapp-shared`.**

Está escrito, y varias veces, con el motivo: `web/doc.go:26-31` («lo que necesitaría de `auth`
entra como parámetro»), `iam/doc.go:36-42` («tampoco `auth`, que es lógica pura y no debe cargar
con un cliente HTTP»), `llm/doc.go` («solo stdlib, y es un invariante»), `intents/doc.go:17-18`.

🔴 **La tentación concreta ya está identificada**: P1 del pipeline clasifica contra el catálogo de
intenciones, que vive en `intents`. Pasar `*intents.Config` a `llm` ataría el puerto genérico del
modelo al contrato de negocio de un tenant **y a su cadencia de releases**. Por eso
`llm.ClassifyRequestInput` recibe el catálogo **aplanado** (`llm.IntentSpec`,
`llm/provider.go:105-131`) y quien aplana es el llamante.

**Cómo se comprueba:** `grep -rn 'wapp-shared/' --include='go.mod' .` → ningún `require` cruzado.
**No hay test que lo vigile.** Es un invariante de prosa; si lo rompes, compila.

### P2 · Este repo no produce binarios, no lee variables de entorno y no escribe ficheros

- **Sin binario:** verificado, cero `main.go` y cero `cmd/`.
- **Sin variables de entorno por nombre:** el único `os.LookupEnv` del repo está en
  `config/provider.go:20` y es el **mecanismo genérico**; la clave y el prefijo los pone el
  llamante vía `config.WithEnvPrefix` (`config/config.go:36`). `llm/api/api.go:15-17` lo declara
  explícito para su paquete. **Cómo se comprueba:**
  `grep -rn 'os.Getenv\|os.LookupEnv' --include='*.go' . | grep -v _test` → solo esas tres líneas
  de `config`.
- **Sin escritura a disco:** `grep -rn 'os.WriteFile\|os.Create\|os.OpenFile\|os.MkdirAll'` solo
  devuelve `config/config_test.go:17` y `.eval-t183/gen.go:61` (que ningún gate compila).
- **Sin base de datos, sin migraciones, sin esquema.** `find . -name '*.sql'` → vacío. El módulo
  `health` define **el contrato** de un health check y deliberadamente **no** implementa ninguno
  contra un driver (`health/doc.go:18-19`).

Si tu cambio añade lectura de entorno, escritura de fichero o un binario a este repo, **lo más
probable es que estés escribiendo en el repo equivocado**.

> 🔎 Confusión frecuente: el volcado de los prompts ajustables se hace con
> `go run ./cmd/prompts -volcar <dir>` **desde `wapp-cloud-platform`**, no desde aquí. Lo que vive
> aquí es el **texto compilado** (`llm/prompt.go`) y el validador (`llm/plantilla.go:142`).

### P3 · La regla al EXTRAER de dos forks: el default del módulo es el valor de UNO de los dos

Esta pieza ya pagó esta trampa. `web` y `iam` nacieron **reconciliando dos copias que ya habían
divergido** (`web/doc.go:7-14`, `iam/doc.go:6-12`). Al extraer, cada constante que en los forks
tenía dos valores distintos acaba con **uno solo** en el módulo, y el fork perdedor **lo hereda en
silencio** — compila, pasa los tests, y en campo dos consolas servidas en el mismo host se pisan la
cookie.

Por eso, y no por gusto, **los nombres de cookie son PARÁMETRO y no constante**:
`DefaultSessionCookieName = "wapp_session"` (`web/cookies.go:29`),
`DefaultCSRFCookieName = "wapp_csrf"` (`web/csrf.go:42`),
`DefaultOneTimeCookieName = "wapp_once"` (`web/onetime.go:11`) — los tres marcados
«deliberadamente genérico: cada consola debe poner el suyo».

🔴 **Regla al extraer, obligatoria:**
1. Enumera cada valor que difiera entre los forks **antes** de escribir el módulo.
2. Para cada uno decide explícitamente: ¿gana uno, se unen, o pasa a parámetro?
3. **Anota la divergencia en el CHANGELOG del módulo.** Es lo que se hizo con la CSP de `web`, que
   es la **unión** de las dos (`object-src 'none'` y `font-src 'self'` del BFF **más** el
   `base-uri 'none'`, más estricto, de la consola de operadores) —
   `web/csp_test.go:19` `TestBuildCSP_UnionDeLasDosConsolas` lo vigila.
4. **Ningún fork gana entero.** Si te descubres copiando uno y borrando el otro, no has extraído:
   has elegido.

### P4 · Restricciones de orden de `web` (las fija el consumidor, y no hay candado)

Escritas en `web/doc.go:33-44`:

- **`BodyLimit` va ANTES que `CSRF`.** El middleware CSRF lee el formulario para comparar el token
  y con eso consume el cuerpo entero; un tope montado después llega cuando el daño está hecho.
- **El CSRF valida ANTES de sembrar.** Si primero siembra y luego rechaza, el 403 sale con un
  `Set-Cookie` que el atacante provoca a voluntad. **Sí hay test:**
  `web/gin/csrf_test.go:38` `TestCSRF_ValidaAntesDeSembrar`.
- **Las cabeceras de seguridad van antes que cualquier handler que renderice**: el nonce lo siembra
  ese middleware y el renderizador lo lee.

🔴 **El orden BodyLimit→CSRF NO lo vigila ningún test**, ni aquí ni en los consumidores: lo compone
el consumidor, así que el módulo no puede imponerlo. Está escrito tres veces (`web/doc.go:38-44`,
`web/README.md`, `web/gin/csrf.go`) y nada más.

### P5 · Fail-closed en la entropía

`web.Nonce` y `web.NewCSRFToken` leen del mismo `io.Reader` (crypto/rand por defecto, inyectable en
tests). **Si falla, no se sirve la página: 500 y sin CSP.** Nunca una respuesta con inline sin
nonce. **Cómo se comprueba:** `web/csp_test.go:71` `TestNonceYToken_FallanCerradoSinEntropia` y
`web/gin/security_test.go:30` `TestSecurityHeaders_FallaCerradoSinEntropia`.

### P6 · `llm` rescata la salida del modelo en DOS capas, y hacen falta las dos

`ExtractJSON` aísla el JSON de entre los adornos del modelo pero **no puede juzgar lo que aísla**:
los prompts imprimen el esquema, y un modelo que lo repita produce JSON perfectamente válido. La
segunda capa son los `Parse*`, que rechazan por `llm.ErrLLMQuality` los valores fuera del enum y
los campos obligatorios vacíos o con el relleno `PlaceholderEsquema`. **Sin la segunda capa, un eco
del esquema entra al pipeline SIN ERROR** — el peor modo de fallo posible.

Corolario de campo, escrito en el ecosistema y que aplica aquí: **en el esquema de un prompt no
puede haber un valor que su propio validador rechace**, porque el modelo copia el ejemplo. Un
`"package_size": 0` costó 0 de 14 en campo. Por eso una plantilla inválida **aborta el arranque**
en vez de servirse (`llm/plantilla.go:142`, `ValidarPlantilla`, exige que el esquema **relleno** lo
acepte el `Parse*` de su etapa **y** que el esquema **crudo** lo siga rechazando).

### P7 · Dos centinelas de error en `llm`, y no es redundancia

`llm.ErrLLMQuality` (`llm/errors.go:18`) = «el modelo devolvió basura» → se reintenta **una** vez
con `TemperatureRetry = 0.3` y luego se aísla la unidad.
`api.ErrUpstream` (`llm/api/api.go:71`) = «el proveedor está caído» → transitorio, lo reintenta el
job. **Colapsarlos en uno cambia el comportamiento del pipeline entero.**

Temperaturas: solo existen dos, `TemperatureGreedy = 0.0` (el valor cero de `Options`) y
`TemperatureRetry = 0.3` (`llm/provider.go:16,21`). No hay barrido ni ajuste por tarea.

### P8 · El prefijo del prompt es estable a propósito, y hay candado

El dato variable va al **FINAL** del prompt: dos llamadas consecutivas de la misma etapa comparten
al menos el **90 %** del prompt más corto byte a byte (`prefijoEstableMinimo = 0.90`,
`llm/prompt_prefijo_test.go:56`). Es lo que permite que el prefijo se cachee.

🔴 Si ese test se pone rojo, **la salida es mover el dato variable al final de `prompt.go`, NO
bajar el umbral**. El propio fichero lo dice: «bajarlo es apagar el detector, no arreglar el
defecto».

### P9 · `textmatch` no importa `llm`: la zona gris se INYECTA

El escalón caro (el LLM) se define como la interfaz `GrayZone` y entra por inyección. Sin inyectar
nada, el motor sigue funcionando y es **determinista puro** (`textmatch/doc.go:20-26`). En el
`SetMatcher` la zona gris se consulta **fuera** del bucle: como mucho una vez por esperado sin
cubrir, nunca por celda.

### P10 · El `system` de `iam` es un CAMPO, no una constante

El System Gate de identity autoriza **aplicaciones**, no ecosistemas: la clave es namespaced
(`wapp.bff`, `wapp.platform`) y `wapp` a secas **no vale**. Viaja en `iam.Options`
(`iam/doc.go:29-35`). **Solo el login lo declara: el refresh NO lo lleva** — aceptarlo permitiría
canjear el refresh de una aplicación por el token de otra.

Y: `Client.Login` / `Client.Refresh` devuelven **siempre** el Context Token. El Identity Token
**muere dentro** del cliente, no vuelve al llamante y no se registra. *Lo que no sale, no se
filtra.*

### P11 · `ui` sirve CSS, y su orden de carga es contrato

Cinco hojas embebidas con `//go:embed css/*.css` (`ui/ui.go:11`): `wapp-tokens.css`,
`wapp-components.css`, `theme-bff.css`, `theme-edge.css`, `theme-platform.css`.
**Orden obligatorio: tokens → componentes → tema.**

Los tests de `ui` **no prueban Go, prueban el CSS**: `TestGetCSS_ParesDeColorPorComponente`
(`ui/ui_test.go:114`), `TestGetCSS_TokensDeTextoDefinidosEnLosDosTemas` (`:211`),
`TestGetCSS_SnackbarSuccessLegible` (`:93`), `TestGetCSS_EncabezadoDentroDeSnackbarHereda`
(`:287`). Son un candado estructural sobre las hojas, no sobre las 25 líneas de `ui/ui.go`.

🔴 **El par mixto es el fallo recurrente de esta hoja**: un color que sigue al tema sobre un fondo
que no. Ya ocurrió cinco veces y dos las introdujo un arreglo nuestro. La regla: **se mide el par
color+fondo, no la regla CSS suelta**, y el candado deriva la lista de componentes del propio CSS
en vez de llevarla escrita a mano.

---

## 3. Reglas de release y de alta de módulo

### 3.1 🔴 El CHANGELOG va `## [0.1.0]` — **SIN la «v»**

`scripts/module-release.sh:84` valida con `grep -q "^## \[$VERSION_NO_V\]"`. Escribirlo
`## [v0.1.0]` **aborta el release**. Esta trampa ya abortó un release de verdad porque la guía del
ecosistema documentaba el formato equivocado; se corrigió el 2026-08-28.

### 3.2 🔴 El CHANGELOG necesita además una sección `## [Unreleased]` viva

`scripts/update-module-changelog.sh:112-115` aborta con «El changelog de `<m>` no contiene la
seccion `## [Unreleased]`». **Hoy `auth` no la tiene** (`auth/CHANGELOG.md:6` empieza directo en
`## [0.5.0]`): es el único de los once, y mientras siga así **`auth` no se puede releasear con las
herramientas del repo**. Ver `documentations/deuda.md`.

### 3.3 🔴 El alta de un módulo nuevo toca un TERCER registro que vive en OTRO repo

Dar de alta un módulo son tres cosas, no dos:

1. `<modulo>/go.mod` + `Makefile` (dos líneas: `MODULE_NAME` e `include ../scripts/module-common.mk`)
   + `README.md` + `CHANGELOG.md` con `## [Unreleased]`.
2. La fila en `scripts/module-manifest.tsv`.
3. 🔴 **La línea `use ./shared/wapp-shared/<modulo>` en el `go.work` de la raíz `wApp/`** — que
   está **fuera de este repo**. Un `grep` dentro de este monorepo **no puede encontrarlo**.

Sin (3), el `make check` del módulo nuevo muere con
`directory prefix . does not contain modules listed in go.work`. Hoy las once líneas están puestas.

### 3.4 🔴 Prohibido el `replace` en un `go.mod`, y el verde se comprueba con `GOWORK=off`

El `go.work` de la raíz existe para que un consumidor compile contra el código de al lado **antes**
del tag. **No sustituye al release**: el CI y cualquier otro clon resuelven por `go.mod`. Por eso:

- **Está prohibido meter un `replace` en un `go.mod` para suplir el workspace.** Verificado hoy:
  `grep -E '^\s*replace ' */go.mod` → ninguno.
- **Un puerto de `shared` se verifica contra el TAG PUBLICADO, no contra el árbol de al lado.** Lo
  que dice si un repo consumidor está de verdad en verde es
  `GOWORK=off go build ./...` **en el consumidor**. Con el workspace activo, un consumidor puede
  estar compilando contra código que aún no existe para nadie más.

### 3.5 Cadencia de ramas

Toda ola aterriza en `dev`; a `main` se pasa al final del plan. **`wapp-shared` es la excepción
declarada**: necesita `main` para cortar los tags. `sync-main-to-dev.yml` deja `dev` alineada tras
cada publicación y es **el único workflow que se dispara solo** en este repo.

⚠️ Deuda de proceso: `ui/v0.4.0` y `ui/v0.4.1` se cortaron sobre commits hechos **directamente en
`main`** (`1116178` y `ff741c0`), no sobre un merge desde `dev`, a diferencia de todos los
anteriores.

---

## 4. Tecnología y versiones **reales**

| Qué | Valor | Evidencia |
|---|---|---|
| Go | **1.26.5** en los once `go.mod` | `grep -h '^go ' */go.mod` |
| Toolchain de CI | Go **1.26.5**, golangci-lint **v2.12.2** | `Makefile:7-8` |
| Lint | `.golangci.yml` v2: `errcheck` (con `check-type-assertions` y `check-blank` en `true`), `govet`, `staticcheck`, `unused`, `ineffassign`, `nakedret`, `prealloc`, `gosec`, `gocyclo` (min-complexity **15**), `gocritic`, `revive`, `errorlint`, `errname`, `contextcheck`, `nilerr`, `nilnil`; formateadores `gofmt` + `goimports` | `.golangci.yml` |
| Tamaño | **6.830** líneas Go de producción, **7.314** de test, **452** de CSS | `find`+`wc` |
| Tests | **271** funciones `Test*`/`Fuzz*`/`Benchmark*`/`Example*` en 45 ficheros | `grep -rhE '^func (Test\|Fuzz\|Benchmark\|Example)'` |
| Tags publicados | **37** | `git tag \| wc -l` |
| Remoto | `github.com/EduGoGroup/wapp-shared` — repo **público** | `git remote -v` |

**Dependencias externas: solo seis en todo el repo.** Siete de los once módulos no tienen ninguna
de producción.

| Módulo | Dependencias directas de producción |
|---|---|
| `auth` | `EduGoGroup/identity-shared/auth v0.3.1`, `golang-jwt/jwt/v5 v5.3.1`, `google/uuid v1.6.0` |
| `config` | `gopkg.in/yaml.v3 v3.0.1` |
| `envelope` | `golang.org/x/crypto v0.54.0` |
| `web` | `gin-gonic/gin v1.10.0`, `golang.org/x/time v0.6.0` |
| `health` `intents` `llm` `textmatch` `ui` | ninguna (solo `testify` de test) |
| `iam` `logger` | **cero dependencias**, ni de test — su `go.mod` son tres líneas |

⚠️ **Deriva real de toolchain**: los tags publicados de `config`, `health`, `intents` y `logger`
declaran `go 1.26.0`; `main` dice `1.26.5`. El commit que subió el toolchain (`6129caf`,
2026-08-01) quedó **por encima** de esos cuatro tags. Es inocuo pero real: no afirmes «todo el repo
publica 1.26.5».

---

## 5. Convenciones de código

- **Cada módulo tiene `doc.go`** y ahí vive el porqué. Los comentarios de este repo **no describen
  el código: describen el fallo de campo que lo trajo**. Mantén ese registro.
- **Cero marcadores `TODO`/`FIXME`/`HACK`/`XXX`.** Verificado: `grep -rnE '\b(TODO|FIXME|HACK|XXX|BUG:)\b'`
  sobre el código no devuelve **ni uno**. La deuda se escribe con nombre (`DEUDA-044.16`, `INV-6`)
  o se anota en `documentations/deuda.md`.
- **Ninguna función pasa de 88 líneas**; `gocyclo` corta en 15.
- **Errores: centinelas con `errors.Is`**, envueltos con `%w`. `errorlint` y `errname` están
  activos. No se traga ningún error salvo dos `_ = body.Close()` justificados por escrito.
- **Nombres y prosa en español** (con acentos) en los comentarios; identificadores en el idioma que
  ya usa el paquete. Las claves JSON de `intents` **mezclan español e inglés a propósito**
  (heredado del prototipo validado, `intents/doc.go:15-16`): **no se renombran**.
- **`revive: exported` está activo**: todo símbolo exportado lleva comentario.

---

## 6. Trampas conocidas — lo que un agente hace mal aquí si nadie se lo dice

| # | La trampa | Qué pasa |
|---|---|---|
| T1 | Escribir el CHANGELOG como `## [v0.1.0]` | El release **aborta** (`scripts/module-release.sh:84`) |
| T2 | Dar de alta un módulo sin tocar el `go.work` de la raíz | `make check` muere con «directory prefix . does not contain modules listed in go.work», y el `grep` local no lo encuentra |
| T3 | Meter un `replace` para que compile | Prohibido; oculta que el consumidor no está de verdad en verde |
| T4 | Verificar un consumidor con el workspace activo | Compila contra el árbol de al lado, no contra el tag. Usa `GOWORK=off go build ./...` |
| T5 | Importar otro módulo de `wapp-shared` desde un módulo | Rompe el nivel 0 y **compila sin quejarse**: no hay candado |
| T6 | Pasar `*intents.Config` a `llm` | Ata el puerto del modelo al contrato de negocio de un tenant. Se aplana a `llm.IntentSpec` |
| T7 | Al extraer de dos forks, tomar el default de uno | El otro lo hereda en silencio: dos consolas se pisan la cookie. Ver §2 P3 |
| T8 | Bajar `prefijoEstableMinimo` para poner en verde `llm` | Apaga el detector. Mueve el dato variable al final del prompt |
| T9 | Convertir en constante un nombre de cookie | Une las tres consolas en la misma cookie. Son parámetro a propósito |
| T10 | Auditar el CSS leyendo reglas | Gana la última hoja cargada; el estático ve la regla que crea el peligro, no la que la tapa. Se mide el **par color+fondo** |
| T11 | Buscar `cmd/prompts` aquí | No está: vive en `wapp-cloud-platform`. Aquí está el texto compilado y el validador |
| T12 | Creer que `make release-dry-run` no existe | **Sí existe, pero por módulo**: `make -C <modulo> release-dry-run VERSION=…`. Lo que no existe es un target de simulacro en el `Makefile` raíz |
| T13 | Editar `scripts/module-manifest.tsv` con un editor que respete el `.tsv` | **No es TSV**: el separador es `\|` (`scripts/list-modules.sh:58`) |
| T14 | Leer un `rc=0` de los tests como «todo pasó» | Un `rc=0` cuenta igual un `--- SKIP` que un `--- PASS`. Ver `documentations/operacion.md` |

---

## 7. Índice

- `documentations/README.md` — portal de la pieza.
- `documentations/arquitectura.md` — un módulo por sección, quién lo consume, diagramas.
- `documentations/contratos.md` — API Go, rutas que este código **llama**, contratos de datos.
- `documentations/operacion.md` — arranque, pruebas, release por módulo, depuración.
- `documentations/deuda.md` — deuda viva y código muerto verificado.
