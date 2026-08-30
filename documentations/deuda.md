# Deuda viva de `wapp-shared`

> Verificado contra `main` a **2026-08-30** (HEAD `ff741c0`). Todo lo de aquí se comprobó leyendo
> el código o ejecutando un comando; lo que no, va marcado **NO VERIFICADO**.

**Antes de leer la lista, el contexto que cambia cómo se lee:** este repo está escrito
excepcionalmente bien. `grep -rnE '\b(TODO|FIXME|HACK|XXX|BUG:)\b'` sobre todo el código **no
devuelve ni un marcador**. Ninguna función pasa de 88 líneas. No hay errores tragados salvo dos
`_ = body.Close()` justificados por escrito. Los comentarios no describen el código: describen el
fallo de campo que lo trajo. Lo que sigue son cosas reales, no un inventario de desorden.

---

## 1. Deuda declarada por el propio código

### D1 · `DEUDA-044.16` — el parseo de artefactos del LLM es TODO-O-NADA

`llm/parse.go:162`. Si un ítem del lote viene mal, **se pierde el lote entero** en vez de degradar
por ítem. Ocurrió en campo el 2026-08-26: ante «entre 10 y 12 kilos de papas fritas», el modelo
emitió el `range` correcto y omitió la clave `qty`; el resto de su salida era impecable y **el
artefacto entero se perdió** por esa clave ausente. Subir el volumen del prompt **no lo arregló**:
el v0.4.2 añadió la regla explícita y el modelo siguió omitiendo el campo, porque omitirlo es
coherente con el contrato que él lee.

**Consecuencia:** una cotización con cinco ítems se pierde entera por un ítem.
**Cómo se cerraría:** degradar por ítem — marcar el ítem defectuoso y devolver el resto. Hay que
decidir antes qué significa un artefacto parcial para el consumidor.
**Dueño:** citada en `llm/parse_test.go:446`; sin plan asignado.

### D2 · `gemini` es un stub que compila y falla siempre

`llm/api/gemini.go:32-53`; construcción en `llm/api/api.go:34,60`. Los cinco métodos devuelven
`ErrNotImplemented`. **Un tenant configurado con `provider="gemini"` arranca bien y muere en la
primera petición**, no en el arranque, que es donde debería morir.
**Cómo se cerraría:** o se implementa, o `New` rechaza `ProviderGemini` con `ErrUnsupportedProvider`
en el arranque. La segunda es media hora de trabajo y convierte un fallo de ejecución en uno de
configuración.

### D3 · `INV-6` — el rate-limit es por instancia, sin broker

`web/ratelimit.go:50-52`, escrito como contrapartida asumida: con N réplicas detrás de un
balanceador **el límite efectivo se multiplica por N**. Es coherente con el invariante del
ecosistema de no meter un broker, y un límite global exigiría un backend compartido.
**No se cierra: es una decisión.** Está aquí para que nadie la "arregle" por su cuenta.

---

## 2. Bugs y asperezas verificadas en el código

### D4 · `auth` no se puede releasear con las herramientas del repo

`auth/CHANGELOG.md:1-6` — **es el único de los once sin sección `## [Unreleased]`**: empieza directo
en `## [0.5.0]` en la línea 6.
**Consecuencia comprobable:** `make changelog-module MODULE=auth VERSION=…` aborta con «El changelog
de auth no contiene la seccion ## [Unreleased]» (`scripts/update-module-changelog.sh:112-115`), y
`auto-release.sh` falla por lo mismo (`:226,237`).
**Cómo se cierra:** reponer `## [Unreleased]` justo encima de `## [0.5.0]`. Es una línea.
**Prioridad: alta** — `auth` es el módulo con más consumidores desalineados (ver D14).

### D5 · `health.Checker` indexa por nombre y no rechaza duplicados

`health/health.go:83` — `CheckAll` hace `results[check.Name()] = …` sobre el mapa, y `Register`
(`:60-67`) acepta cualquier cosa. **Dos checks con el mismo nombre se pisan en silencio** y uno de
los dos deja de vigilarse sin que nada falle.
**Consecuencia:** un componente desaparece del panel de salud y el panel sigue en verde.
**Cómo se cierra:** que `Register` devuelva error ante un nombre ya registrado, o que `CheckAll`
detecte la colisión. Rompe la firma actual, así que va con un `v0.2.0`.

### D6 · `health.Checker.Register(nil)` se traga el error en silencio

`health/health.go:61-63`, documentado como «Si check es nil, se ignora silenciosamente». Un check
que el bootstrap construyó mal **desaparece del panel sin dejar rastro**.
**Cómo se cierra:** igual que D5, devolviendo error. Van juntas.

### D7 · `intents.normalizeThreshold`: el comentario y el código no dicen lo mismo

`intents/intents.go:34` (el tipo) y `:92-98` (la función). El comentario dice que «rechaza cualquier
valor presente fuera del rango (0,1]», pero `UmbralConfianza` es `float64` y no `*float64`, así que
**el código no puede distinguir «ausente» de «0 explícito»**: un contrato con
`"umbral_confianza": 0` —que está fuera de (0,1]— **no se rechaza, se sustituye por 0.6**.
**Consecuencia:** un operador que escriba 0 creyendo que desactiva el umbral obtiene 0.6 y ningún
aviso.
**Cómo se cierra:** `*float64` en el tipo, o dejar el comentario diciendo la verdad. Lo primero es
un cambio de contrato JSON; lo segundo, una línea.

### D8 · `config.Loader.lookup` muta `l.envProvider` sin candado

`config/config.go:104-110`. Es una inicialización perezosa **en el camino de lectura**. `New()`
siempre rellena el campo, pero `&config.Loader{}` es construible desde fuera del paquete, y entonces
dos goroutines llamando a `GetString` a la vez **son una carrera de datos**. El módulo `config` no
importa `sync` en ninguna parte.
**Cómo se cierra:** rellenar el default en el valor cero del tipo, o proteger con `sync.Once`.
**No hay test de carrera que lo cace** (`test-race-all` existe, pero ningún test construye un
`Loader{}` desnudo y lo usa concurrentemente).

### D9 · `iam.drainClose` vacía el cuerpo SIN tope

`iam/client.go:147-155` hace `io.Copy(io.Discard, body)` sin límite, mientras que la decodificación
sí lo tiene (`maxResponseBody = 1 MiB`, `iam/client.go:23`). Con el `http.Client` por defecto lo
acota el `DefaultTimeout` de 15 s, pero `Options.HTTPClient` (`iam/client.go:40`) permite inyectar
un cliente **sin** timeout, y ahí el drenaje de un upstream hostil no tiene freno.
**Cómo se cierra:** `io.CopyN(io.Discard, body, maxResponseBody)`. Una línea.

### D10 · `ui.FS()` entra en pánico en vez de devolver error

`ui/ui.go:14-20`. Es inalcanzable en la práctica (el `embed` garantiza que `css/` existe), pero es
una función de **librería** que puede tumbar el proceso del consumidor.
**Cómo se cierra:** devolver `(fs.FS, error)` — cambio de firma, va con un mayor de `ui`.

### D11 · `KeyedRateLimiter.Close` es idempotente pero el limitador sigue vivo

`web/ratelimit.go:128-147`. Los dos hechos están escritos y justificados: `Close` es idempotente
vía `sync.Once`, y **no inhabilita el limitador** porque hay callers que cierran y luego siguen
sirviendo. Juntos significan que **la memoria acumulada después del primer `Close` no la libera
nunca un segundo `Close`**. No está escondido; es un intercambio que el nombre `Close` no anuncia.
**Cómo se cierra:** renombrar (`ReleaseMemory`) o quitar el `sync.Once`. Ninguna es gratis.

### D12 · El orden «BodyLimit ANTES que CSRF» vive SOLO en prosa

`web/doc.go:38-44`, más `web/README.md` y `web/gin/csrf.go`: escrito **tres veces** y **ningún test
lo vigila**. En las 61 funciones de test de `web` no hay ninguna de orden de middleware. Lo compone
el consumidor, así que el módulo no puede imponerlo — pero tampoco hay candado en los consumidores.
**Hoy no muerde** porque nadie monta `webgin.BodyLimit` (ver D16), y eso es precisamente lo que lo
hace peligroso: el día que alguien lo monte, no habrá nada que le avise del orden.
**Cómo se cierra:** un test en los consumidores que compruebe la posición relativa de los dos
middlewares en la cadena montada, o un candado sobre el AST de quien construye el router.

---

## 3. Código sin ningún consumidor (medido, no supuesto)

**Método:** cada símbolo exportado se buscó en los seis repos consumidores **y** en el resto de
`wapp-shared` fuera de su propio paquete, excluyendo `_test.go`. El barrido **cuenta las
apariciones en comentarios como consumo**, así que esta lista es un **mínimo**, no un máximo.

| # | Qué está muerto | Tamaño | Evidencia |
|---|---|---|---|
| D13 | **`textmatch` Nivel 2 completo**: `SetMatcher`, `NewSetMatcher`, `MatchReport`, `Policy`, `GenerateCandidates`, `Candidate`. Los consumidores usan solo el Nivel 1 (`Normalize` 22 usos, `SplitTokens` 9, `Result` 6, `OutcomeMatch` 6, `NewFuzzy` 3…) y **cero** `SetMatcher` | 373 líneas + 240 de test | `textmatch/setmatcher.go:38,70,78` |
| D14 | **El service token M2M entero**: `NewServiceJWTManager`, `ServiceJWTManager`, `ServiceClaims`, `GenerateServiceToken`, `ValidateServiceToken`. Nadie lo construye | 140 líneas | `auth/jwt/service_claims.go:44,60,106` |
| D15 | **`NewJWTManager` (HS256) y `NewJWTVerifierES256`**: los consumidores usan solo `NewJWTManagerES256` y `NewMultiVerifier` | — | `auth/jwt/jwt_manager.go:46,77` |
| D16 | **`webgin.BodyLimit`**: sus cuatro apariciones en el ecosistema son comentarios y un test, ninguna llamada. El helper de decisión `web.NewBodyLimit` **sí** se usa | — | `web/gin/bodylimit.go:23` |
| D17 | **`theme-edge.css` no lo sirve nadie**: el Edge Agent **no importa `ui`** en absoluto. Las tres consolas sirven `theme-bff.css` y `theme-platform.css` | 11 líneas | `ui/css/theme-edge.css` |
| D18 | **`iam.IdentityLogin` / `IdentityRefresh` / `Exchange`, `IdentityTokens`, `APIError`**: los «escalones por separado» que ofrece `iam/doc.go:29-31`. El único consumidor usa `Login`/`Refresh`/`Logout` | — | `iam/identity.go:47,58`; `iam/exchange.go:38` |
| D19 | **`envelope.BoxSealer` / `NewBoxSealer` / `Sealer`**: la envoltura de interfaz sobre el sellado; los consumidores llaman a `SealFor`/`OpenWith` directamente | — | `envelope/sealing.go:29,32`; `envelope/interfaces.go:11` |
| D20 | **`config.EnvProvider`, `FileReader`, `MapEnvProvider`, `WithEnvProvider`, `WithFileReader`**: el andamiaje de inyección solo lo usan sus propios tests | — | `config/provider.go:7,12,34`; `config/config.go:50,59` |
| D21 | **`.eval-t183/gen.go`**: herramienta de un solo experimento (A/B de prompts), en un directorio oculto **sin `go.mod`** y con `//go:build ignore`. **Ningún gate la compila**, así que puede pudrirse indefinidamente sin que nada avise. Entró en `7a4acbf` | 60+ líneas | `.eval-t183/gen.go:1` |

**Cómo se cerraría, en general:** borrar. La excepción es D14 y D15, que son capacidades pensadas
para un consumidor futuro; ahí la decisión es *«se borra o se le pone dueño y fecha»*, y hoy no
tienen ni una cosa ni la otra. D21 es borrado puro: es basura de un experimento cerrado.

---

## 4. Deuda de proceso y de documentación interna

### D22 · Deriva de toolchain entre `main` y cuatro tags publicados

Los tags de `config`, `health`, `intents` y `logger` declaran `go 1.26.0`; `main` dice `1.26.5`. El
commit que subió el toolchain (`6129caf`, «chore(toolchain): bump Go to 1.26.5 across all modules»,
2026-08-01) quedó **por encima** de esos cuatro tags — `git rev-list --count <tag>..HEAD -- <mod>/`
da 1 en los cuatro y 0 en los demás. **Inocuo pero real**: no digas «todo el repo publica 1.26.5».
**Se cierra** con un release de patch de esos cuatro, que hoy no aporta nada más.

### D23 · `scripts/module-manifest.tsv` no es TSV

`scripts/list-modules.sh:58` lo parsea con `awk -F'|'`. **El nombre miente sobre el formato**: quien
lo edite con un editor que respete la extensión `.tsv` (convirtiendo `|` en tabuladores, o
alineando columnas) lo rompe y con él todo el `Makefile`, que deriva las listas de módulos de ahí.
**Se cierra** renombrándolo a `.psv` o `.txt` y actualizando las referencias.

### D24 · Dos historias de tags conviviendo

`ui/v0.1.0` y `ui/v0.2.0` son tags **anotados**; `ui/v0.3.0`, `v0.4.0` y `v0.4.1` son **ligeros**.
`scripts/module-release.sh:108` hace `git tag "$TAG"` sin `-a`, así que los ligeros son los que
produce la herramienta y los anotados vinieron de otra vía.
**Se cierra** decidiendo uno de los dos y poniéndolo en el script.

### D25 · Dos tags de `ui` cortados directamente sobre `main`

`ui/v0.4.0` (`1116178`) y `ui/v0.4.1` (`ff741c0`) se cortaron sobre commits hechos **directamente en
`main`**, no sobre un merge desde `dev`, a diferencia de todos los anteriores
(`ui/v0.1.0` → «Merge pull request #10 from dev», `web/v0.2.0` → «merge(dev): …»). Hoy `dev` y
`main` están alineadas porque `sync-main-to-dev.yml` corrió, así que **no hay daño**: es una grieta
en la cadencia, no un estado roto.

### D26 · La dependencia a `identity-shared` rompe la propiedad de «cero dependencias externas»

`auth/go.mod:6` requiere `github.com/EduGoGroup/identity-shared/auth v0.3.1`, y
`auth/jwt/jwt_claims.go:22` hace `type Grants = rbac.Grants`. **No es una violación de la
prohibición de importar `edugo-*`** —`identity-shared` es otro repo, es el motor de permisos del
grupo, y la decisión está justificada por escrito en `auth/jwt/jwt_claims.go:17-20`— pero **ata la
cadencia de releases de `auth` a la de identity** y rompe la propiedad de «solo stdlib» que el resto
del monorepo sí cumple. **Es una decisión, no una deuda con dueño**; se anota para que nadie la
descubra como sorpresa.

### D27 · Contradicciones vivas en los README de este repo

Ninguna afecta al comportamiento, todas engañan a quien lee:

| Fichero | Lo que dice | Lo que es |
|---|---|---|
| `ui/README.md` y `README.md` raíz | «los consumen la Consola Cloud BFF y **la Edge Agent UI**» | El **Edge Agent no consume `ui`**: no está en su `go.mod`. Los consumidores son **tres consolas** (`guardian-bff`, `client-console`, `platform-console`) |
| `ui/README.md`, tabla de ficheros | lista **cuatro** CSS | Hay **cinco**: falta `theme-platform.css`, que existe desde `ui/v0.2.0` y **sí se sirve** |
| `README.md` raíz, fila de `web` | «Paquete raíz **solo stdlib**» | `web/go.mod:6` requiere `golang.org/x/time v0.6.0`. `web/doc.go:26` sí lo dice bien («solo stdlib **(+ golang.org/x/time/rate)**») |
| `CHANGELOG.md` raíz, `## [Unreleased]` | enumera **tres** módulos (`logger`, `config`, `llm`) | Hay **once** y 37 tags. Sin tocarse desde el 2026-08-22; describe un repo que ya no existe. Los CHANGELOG **de módulo** sí están al día |
| `README.md` raíz, Toolchain | «Go **1.26**» | `Makefile:7` y los `go.mod` dicen `1.26.5`; cuatro tags publicados dicen `1.26.0` (D22) |
| `.github/workflows/ci.yml:14-21` | describe una lógica de scope por PR | El workflow es `workflow_dispatch` y **el propio fichero lo dice dos líneas antes**. Esa lógica es letra muerta |
| `web/README.md`, ejemplo de montaje | enseña `engine.Use(webgin.BodyLimit(...))` | **Ninguna consola lo monta hoy** (D16). El README enseña un montaje que el ecosistema abandonó |

**Cómo se cierra:** una pasada por los README. La verdad de esta pieza vive en
`documentations/`; los README de módulo siguen siendo útiles como referencia de API, pero sus
afirmaciones sobre **quién consume qué** están caducadas y no deben heredarse.

---

## 5. Lo que NO se pudo verificar

- **Si los GitHub Releases existen de verdad** para los 37 tags. `release.yml` los crea, pero no se
  consultó la API de GitHub. **NO VERIFICADO**.
- **Si `origin` tiene tags que no estén en local**: se comparó `main` con `origin/main` (idénticos)
  pero no se hizo `git ls-remote --tags`. **NO VERIFICADO**.
- **Cobertura de tests**: los scripts existen y el manifest marca `coverage_validation=true` en los
  once módulos, pero **no se ejecutó la medición**. **NO VERIFICADO**.
- **Si `golangci-lint v2.12.2` pasa limpio**: solo se verificó que `go test` pasa en los once
  módulos (once verdes, ninguno saltado). `make lint-all` no se ejecutó. **NO VERIFICADO**.
