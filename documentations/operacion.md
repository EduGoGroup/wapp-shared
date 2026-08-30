# Operación de `wapp-shared`

> Cómo se arranca, se prueba, se publica y se depura. Verificado contra `main` a **2026-08-30**.

---

## 1. Arranque local — no hay nada que arrancar

**Esta pieza no produce ningún binario.** No hay servicio que levantar, ni puerto, ni fichero de
configuración, ni base de datos. «Arrancar en local» aquí significa **compilar y probar**.

```bash
cd shared/wapp-shared
make build-all        # go build ./... en los once módulos
make test-all         # go test ./... en los once módulos
```

**Requisitos reales:**

| Qué | Versión | De dónde sale |
|---|---|---|
| Go | **1.26.5** | `Makefile:7` y los once `go.mod` |
| golangci-lint | **v2.12.2** | `Makefile:8`; se instala con `make tools` (que lo fija a esa versión) |
| Docker | opcional | solo para `make ci-docker` |

⚠️ **`make install-tools` NO fija la versión** (`go install …@latest`, `Makefile:272`). Usa
**`make tools`**, que instala `@$(LINT_VERSION)`. Instalar `@latest` te dará avisos que el CI real
no da, o al revés.

**El workspace.** Estos módulos se compilan dentro del `go.work` de la raíz `wApp/`, que los une
con los siete repos de aplicación para que un consumidor compile contra el código de al lado antes
del tag. Consecuencias operativas:

- Si abres solo `shared/wapp-shared/` en el editor, **gopls no ve el workspace**: hay que abrir la
  raíz `wApp/`.
- Un módulo nuevo sin su línea `use` en ese `go.work` muere con
  `directory prefix . does not contain modules listed in go.work`. Ese fichero **está en otro
  repo** y un `grep` aquí no lo encuentra.
- 🔴 **Para saber si un consumidor está de verdad en verde, apaga el workspace:**
  `GOWORK=off go build ./...` **en el repo consumidor**. Con el workspace activo estás compilando
  contra el árbol de al lado, no contra el tag publicado.
- **Prohibido el `replace`** en un `go.mod` para suplir el workspace.

---

## 2. Cómo se prueba

### 2.1 Los gates que existen de verdad

| Target | Qué hace exactamente |
|---|---|
| **`make ci-local`** | `fmt-check-all` + `vet-all` + `lint-all` + `test-all`. **Es el pre-push real.** |
| **`make ci-docker`** | Lo mismo dentro de `golang:1.26.5-bookworm`, instalando golangci-lint `v2.12.2`. Monta el `GOPATH/pkg/mod` del host y el repo en `/workspace`. Requiere Docker. |
| `make check-all` | `fmt-all` + `vet-all` + `lint-all` + `test-all` + `build-all` (⚠️ `fmt-all` **muta** el código; `fmt-check-all` no) |
| `make ci` | `fmt-all` + `vet-all` + **`test-race-all`** + `build-all` |
| `make test-all` / `test-race-all` / `build-all` / `lint-all` / `vet-all` / `fmt-all` / `tidy-all` | por módulo, sobre `scripts/list-modules.sh --set all` |
| `make build-parallel` / `test-parallel` / `lint-parallel` | versiones paralelas |
| `make test-integration-all` | **hoy no ejecuta nada**: `--set integration` está vacío porque los once módulos tienen `integration=false` en el manifest |
| `make -C <modulo> check` | `fmt` + `vet` + `lint` + `test` + `build` de un solo módulo (viene de `scripts/module-common.mk`) |

### 2.2 🔴 `ci.yml` es `workflow_dispatch`: **un PR no valida nada**

Esto aplica a **todo el ecosistema wApp** y se repite aquí porque es la fuente de errores más cara:
`.github/workflows/ci.yml:6` es `on: workflow_dispatch` **y nada más**. El propio fichero lo
explica en su cabecera: «La validación continua ahora es LOCAL (`make ci-local` / `make ci-docker`),
no en GitHub Actions». **El gate real es local, y son dos.**

⚠️ Ojo adicional: el cuerpo de `ci.yml:14-21` describe una lógica de scope por PR («PR → dev: solo
módulos modificados…»). Es **letra muerta**: no hay PRs que la disparen. No la cites como si
corriera.

El único workflow que corre solo en este repo es `sync-main-to-dev.yml`.

### 2.3 🔴 Un `rc=0` no significa «todo pasó»: hay que contar los SKIP

`go test` devuelve **0** tanto con `--- PASS` como con `--- SKIP`. Un gate que solo mira el código
de salida no distingue «pasó» de «no se ejecutó». En otros repos del ecosistema los tests de
integración **se saltan solos** cuando la variable de base de datos (`WAPP_TEST_DB_DSN`) no está
puesta, y el `rc=0` resultante se ha leído como verde más de una vez.

**En `wapp-shared` hoy no hay ningún skip**, y eso es un hecho medido, no una suposición:
`grep -rn "t.Skip\|testing.Short()\|//go:build" --include='*_test.go' .` → **vacío**. No hay build
tags, ni skips condicionales, ni `WAPP_TEST_DB_DSN`, ni Docker. Los 271 tests son unitarios puros
con `httptest` para el cable.

Aun así, **cuenta los SKIP siempre**, porque el día que alguien añada el primero nadie se enterará:

```bash
cd shared/wapp-shared
for m in $(./scripts/list-modules.sh --set all); do
  (cd "$m" && go test ./... 2>&1) | tee /tmp/out-$m.txt
done
grep -c -- "--- SKIP" /tmp/out-*.txt      # tiene que dar 0 en todos
```

Y **lee el `rc` sin pipe**: un `go test ./... | tee` devuelve el rc de `tee`, no el de `go test`.

### 2.4 Qué hay cubierto

271 funciones `Test*`/`Fuzz*`/`Benchmark*`/`Example*` en 45 ficheros; **7.314 líneas de test frente
a 6.830 de producción**.

| Módulo | Funciones de test | Ficheros |
|---|---|---|
| `llm` | 65 | 9 |
| `web` | 61 | 17 |
| `iam` | 30 | 5 |
| `textmatch` | 28 | 4 |
| `auth` | 27 | 4 |
| `config` | 16 | 1 |
| `envelope` | 13 | 1 |
| `ui` | 9 | 1 |
| `health` / `intents` | 8 / 8 | 1 / 1 |
| `logger` | 6 | 1 |

**Los tests que no prueban lo que parece:**

- Los de `ui` **prueban el CSS**, no las 25 líneas de Go: pares de color por componente, tokens de
  texto en los dos temas, legibilidad del snackbar (`ui/ui_test.go:93,114,211,287`).
- `llm/prompt_prefijo_test.go:56` vigila que el **prefijo del prompt sea estable al 90 %**. Si se
  pone rojo, se mueve el dato variable al final de `prompt.go`; **no se baja el umbral**.
- `web/csp_test.go:19` `TestBuildCSP_UnionDeLasDosConsolas` vigila que la CSP siga siendo la
  **unión** de los dos forks reconciliados, no la de uno.

**Cobertura: NO VERIFICADO.** Existen `scripts/validate-coverage.sh` y `analyze-coverage.sh` y el
manifest marca `coverage_validation=true` en los once módulos, pero no se ejecutó la medición en
este levantamiento.

**Lint limpio: NO VERIFICADO.** Se ejecutó `go test` en los once (once verdes, ninguno saltado);
`make lint-all` no.

---

## 3. Cómo se publica una versión

### 3.1 El procedimiento, paso a paso

Un release aquí es **por módulo**. No hay módulo raíz y no existe un tag `vX.Y.Z` suelto.

```bash
cd shared/wapp-shared
git checkout main && git pull            # los tags se cortan SOBRE main

# 1. Escribe a mano lo que cambió bajo `## [Unreleased]` del CHANGELOG del módulo.
$EDITOR llm/CHANGELOG.md

# 2. Promueve Unreleased → la versión. Deja `## [Unreleased]` vivo y vacío.
make changelog-module MODULE=llm VERSION=v0.4.6

# 3. Commitea el CHANGELOG. El árbol tiene que quedar LIMPIO.
git add llm/CHANGELOG.md && git commit -m "chore(llm): changelog v0.4.6" && git push

# 4. SIMULACRO (recomendado). Corre `check` y valida todo, sin tag ni push.
make -C llm release-dry-run VERSION=v0.4.6

# 5. El release de verdad: corre `check` y luego crea y empuja el tag.
make release-module MODULE=llm VERSION=v0.4.6
```

El push del tag `llm/v0.4.6` dispara `release.yml`, que crea el GitHub Release con la sección del
CHANGELOG. El push a `main` del paso 3 dispara `sync-main-to-dev.yml`, que realinea `dev`.

### 3.2 🔴 Las tres trampas que ya han abortado un release de verdad

**T1 · El CHANGELOG va `## [0.4.6]`, SIN la «v».**
`scripts/module-release.sh:84` valida literalmente con `grep -q "^## \[$VERSION_NO_V\]"`. Escribir
`## [v0.4.6]` **aborta el release** con «Falta la seccion ## [0.4.6]…». La guía del ecosistema
documentó el formato equivocado durante meses y el release abortó de verdad; se corrigió el
2026-08-28.

**T2 · El alta de un módulo nuevo toca un TERCER registro que vive en OTRO repo.**
Son tres cosas: (a) el `go.mod` + `Makefile` + `README.md` + `CHANGELOG.md` del módulo, (b) la fila
en `scripts/module-manifest.tsv`, y (c) **la línea `use ./shared/wapp-shared/<modulo>` en el
`go.work` de la raíz `wApp/`**. Un `grep` dentro de este monorepo **no puede encontrar (c)**, y sin
ella el `make -C <modulo> check` muere con
`directory prefix . does not contain modules listed in go.work` — un error que no menciona el
`go.work` de nadie en concreto y manda a buscar en el sitio equivocado.

**T3 · Un puerto de `shared` se verifica contra el TAG PUBLICADO, no contra el árbol de al lado.**
El `go.work` hace que un consumidor compile contra tu código sin tag. Eso es útil mientras
desarrollas y **mentiroso al cerrar**: `GOWORK=off go build ./...` en el consumidor es lo que dice
si está de verdad en verde. Sin release de `shared`, un consumidor «verde» no compila para nadie
más.

### 3.3 Otras cosas que conviene saber antes de publicar

- **`make release-module` NO tiene simulacro desde el `Makefile` raíz.** `Makefile:253-258` hace
  `@$(MAKE) -C $(MODULE) release VERSION=$(VERSION)` sin propagar ninguna variable de dry-run: un
  `DRY=1` **se ignora y publicas un tag de verdad**. El simulacro existe, pero **por módulo**:
  `make -C <modulo> release-dry-run VERSION=vX.Y.Z` y `make -C <modulo> changelog-dry-run`, los dos
  definidos en `scripts/module-common.mk`. No existe ningún target llamado `release-dry-run` en la
  raíz.
- **`release` corre `check` antes**, y `check` incluye `lint`: sin golangci-lint instalado no
  publicas.
- **El árbol tiene que estar limpio.** El script lo exige (`git status --porcelain` vacío), salvo
  en `--dry-run`, donde avisa y continúa.
- **El script comprueba que el tag no exista ni en local ni en el remoto** antes de crearlo.
- **`make auto-release`** detecta los CHANGELOG modificados y publica en cadena.
  `make auto-release-dry-run` es su simulacro (con verbose) y `make auto-release-help`, su ayuda.
  ⚠️ `auto-release.sh` también exige la sección `## [Unreleased]` (`:226,237`).
- **Los tags no son todos del mismo tipo.** `ui/v0.1.0` y `ui/v0.2.0` son **anotados**; los demás
  que produjo la herramienta son **ligeros** (`scripts/module-release.sh:108` hace `git tag` sin
  `-a`). Conviven dos historias de release.
- **`auth` no se puede releasear hoy** con estas herramientas: le falta la sección
  `## [Unreleased]`. Ver `documentations/deuda.md`.

---

## 4. Cómo se depura cuando falla

### 4.1 Fallos de build y de workspace

| Síntoma | Causa casi segura | Salida |
|---|---|---|
| `directory prefix . does not contain modules listed in go.work` | falta la línea `use` del módulo en el `go.work` de la raíz `wApp/` | añádela; está en **otro repo** |
| gopls no resuelve nada / imports en rojo | abriste `shared/wapp-shared/` en vez de la raíz `wApp/` | abre la raíz |
| El consumidor compila en local y rompe en otro clon | estás resolviendo por workspace, no por tag | `GOWORK=off go build ./...` en el consumidor; publica el módulo |
| Alguien metió un `replace` para arreglarlo | prohibido | quítalo y publica |

### 4.2 Fallos de release

| Síntoma | Causa | Salida |
|---|---|---|
| «Falta la seccion `## [X.Y.Z]` en …/CHANGELOG.md» | escribiste `## [vX.Y.Z]` con la «v», o no corriste `make changelog-module` | escríbelo sin la «v» |
| «El changelog de `<m>` no contiene la seccion `## [Unreleased]`» | el CHANGELOG perdió esa sección (le pasa a `auth`) | repón `## [Unreleased]` justo encima de la última versión |
| «El repositorio tiene cambios sin confirmar» | árbol sucio | commitea; el CHANGELOG va en su propio commit |
| «El tag … ya existe» (local o remoto) | ese número ya se publicó | sube el número; **no borres un tag publicado** |
| El release publica pero no aparece el GitHub Release | el tag no encaja con `'*/v*'` | el nombre es `<modulo>/vX.Y.Z`; un `vX.Y.Z` suelto **no dispara nada a propósito** |

### 4.3 Fallos de test que confunden

- **`llm/prompt_prefijo_test.go` en rojo:** si falla la guarda de orden, el dato variable se coló a
  media plantilla — muévelo al final de `prompt.go`. Si falla solo el ratio, el orden está bien:
  mira el tamaño de las entradas. **No reordenes el prompt ni bajes el umbral.**
- **Tests de `ui` en rojo tras tocar CSS:** casi siempre es el **par mixto** — un color que sigue al
  tema sobre un fondo que no. Se mide el **par color+fondo**, no la regla suelta; y gana la
  **última hoja cargada**, así que auditar leyendo reglas te dará la respuesta contraria a la real.
- **Un test verde que depende del orden alfabético de los paquetes:** ya ocurrió en el ecosistema.
  Es invisible en `./...` y muerde al depurar con `-run` suelto.

### 4.4 Qué se registra en ejecución

Este repo no tiene proceso propio, así que **no emite logs por sí mismo**. Lo que sí gobierna:

- `logger` define **la forma** de los registros del ecosistema: `log/slog`, pares clave/valor,
  loggers hijos derivados con `With` que arrastran campos. El nivel y el formato (texto o JSON) los
  decide el consumidor al construirlo (`WithLevel`, `WithJSON`).
- `webgin.SlogLogger()` es el middleware que registra **una línea por petición HTTP** en las
  consolas que lo montan.
- 🔴 **`iam` no registra NADA de lo que pasa por él**, y es deliberado: todo lo que sale de ese
  cliente son credenciales, así que ninguna respuesta se registra ni se incluye en un error
  (`iam/client.go`, doc del tipo `Client`). Si depurando te tienta añadir un log ahí, **no**: es un
  invariante, no un olvido.
- `llm/api/anthropic.go` sí cita un **trozo acotado** del cuerpo en los errores de upstream
  (`maxSnippetRunes`), nunca la credencial.

### 4.5 Verificar qué versión corre de verdad

Un binario Go lleva dentro las versiones de sus módulos. Desde la máquina donde corre el
consumidor:

```bash
go version -m /ruta/al/binario | grep wapp-shared
```

Es así como se detectó que el Edge lleva `auth v0.4.1` con `v0.5.0` publicado. Y recuerda que **la
revisión del fichero instalado no es la del proceso vivo**: instalar y reiniciar son dos pasos.
Pregunta por `/proc/$(systemctl show -p MainPID --value <unidad>)/exe`.
