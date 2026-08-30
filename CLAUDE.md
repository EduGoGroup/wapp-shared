# CLAUDE.md — `wapp-shared`

Monorepo Go **multi-módulo** de librerías compartidas del ecosistema **wApp** (mensajería sobre
WhatsApp: el núcleo corre 24/7 en el equipo del cliente —el **Edge**— y lo gobierna una plataforma
cloud modular). **Once módulos independientes** —`auth`, `config`, `envelope`, `health`, `iam`,
`intents`, `llm`, `logger`, `textmatch`, `ui`, `web`—, cada uno con su `go.mod`, su `CHANGELOG.md`
y su línea de versiones (`<modulo>/vX.Y.Z`).

🔴 **No produce ni un binario y no sirve ni una ruta.** Si tu cambio añade un `main.go`, lectura de
variables de entorno o escritura de ficheros, casi seguro estás en el repo equivocado.

## Las cinco reglas innegociables

1. **Nivel 0: ningún módulo importa a otro módulo de `wapp-shared`.** Lo que necesitarías de otro
   entra **como parámetro** (`web` recibe el `exp` ya extraído; `llm` recibe el catálogo aplanado en
   `IntentSpec`, sin tocar `intents`). 🔴 **No hay candado: si lo rompes, compila.**

2. **Zero-knowledge y doble llave.** La nube **nunca** accede a credenciales ni llaves privadas. La
   **DEK** (descifra el almacén de whatsmeow) la custodia el cliente y **jamás cruza el contrato en
   claro**: `envelope` la mueve **sellada** (X25519/NaCl box). El **Lease** lo emite y revoca el
   servidor — es el kill-switch anti-clon. Protege **llaves**, no el contenido de negocio, que sí
   sube a la nube a propósito. Lo fija el ADR-0007 (modelo de doble llave), en el repo de
   documentación del ecosistema.

3. **Copia-adaptación, nunca dependencia.** Se copió código de otro producto del grupo (EduGo) y se
   adaptó al espacio de nombres de wApp: **prohibido `import "github.com/EduGoGroup/edugo-..."`**
   (la dependencia de `auth` a `identity-shared` es otra cosa y está justificada por escrito). Y
   **sin Redis ni broker en el Edge**: la concurrencia se resuelve con Go — por eso el rate-limit de
   `web` es **por instancia**, contrapartida aceptada y no un bug.

4. **El CHANGELOG de un módulo va `## [0.1.0]`, SIN la «v».** Con la «v» el release **aborta**
   (`scripts/module-release.sh:84`). Necesita además una sección `## [Unreleased]` viva.

5. **Dar de alta un módulo nuevo toca un TERCER registro que vive en OTRO repo:** la línea
   `use ./shared/wapp-shared/<modulo>` en el `go.work` de la raíz `wApp/`. Un `grep` aquí **no puede
   encontrarlo**, y sin ella `make check` muere con «directory prefix . does not contain modules
   listed in go.work». **Prohibido el `replace`** para suplirlo: un consumidor solo está de verdad
   en verde con **`GOWORK=off go build ./...`**, que resuelve contra el **tag publicado**.

## Antes de correr un gate

`ci.yml` es **`workflow_dispatch`**: **un PR no valida nada**. El gate real es local y son dos,
`make ci-local` y `make ci-docker` (Go 1.26.5 + golangci-lint v2.12.2, `Makefile:7-8`). Un `rc=0`
cuenta igual un `--- SKIP` que un `--- PASS`: **cuenta los SKIP** y lee el `rc` sin pipe.

## La verdad vive en `documentations/`

- **[`documentations/README.md`](documentations/README.md)** — portal de la pieza.
- 🔴 **[`documentations/constitucion.md`](documentations/constitucion.md)** — **empieza aquí**:
  invariantes con cómo se comprueban, versiones reales, convenciones y las **14 trampas conocidas**.
- **[`documentations/arquitectura.md`](documentations/arquitectura.md)** — un módulo por sección: qué
  hace, versión publicada y **quién lo consume**.
- **[`documentations/contratos.md`](documentations/contratos.md)** — superficie Go, rutas que este
  código **llama**, contratos de datos y contrato de release.
- **[`documentations/operacion.md`](documentations/operacion.md)** — compilar, probar, **publicar por
  módulo** y depurar.
- **[`documentations/deuda.md`](documentations/deuda.md)** — deuda viva y código muerto verificado.

⚠️ Los `README.md` de la raíz y de los módulos sirven como referencia de API, pero **varias de sus
afirmaciones sobre quién consume qué están caducadas** (listadas en `documentations/deuda.md` §4).
