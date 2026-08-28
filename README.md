# WApp Shared

Monorepo Go **multi-modulo** de librerias compartidas del ecosistema **wApp**.
Cada modulo es un modulo Go independiente con su propio `go.mod`, `README.md` y
`CHANGELOG.md`, versionado y liberado de forma autonoma mediante tags
`modulo/vX.Y.Z`.

## Instalacion

Cada modulo se consume por separado con su ruta completa:

```bash
go get github.com/EduGoGroup/wapp-shared/<modulo>
```

Ejemplos:

```bash
go get github.com/EduGoGroup/wapp-shared/logger
go get github.com/EduGoGroup/wapp-shared/config
```

## Modulos

| Modulo | Foco | README | CHANGELOG |
| --- | --- | --- | --- |
| `logger` | Logging estructurado sobre `log/slog` (stdlib, sin dependencias). | [README](logger/README.md) | [CHANGELOG](logger/CHANGELOG.md) |
| `config` | Carga de configuracion desde archivo YAML con overlay de variables de entorno. | [README](config/README.md) | [CHANGELOG](config/CHANGELOG.md) |
| `envelope` | Cifrado AES-256-GCM (DEK 32B) y sellado anonimo X25519 para blobs. | [README](envelope/README.md) | [CHANGELOG](envelope/CHANGELOG.md) |
| `health` | Registro y agregacion de health checks (liveness/readiness). | [README](health/README.md) | [CHANGELOG](health/CHANGELOG.md) |
| `auth` | El **plano de contexto** de wApp: JWT `{tenant_id, roles, grants}` **ES256** (kid + MultiVerifier) + service token M2M HS256. Desde v0.4.0 el RBAC, bcrypt y el refresh opaco son de `identity-shared/auth`. Unico paquete: `auth/jwt`. | [README](auth/README.md) | [CHANGELOG](auth/CHANGELOG.md) |
| `intents` | Contrato canónico de configuración de intenciones por tenant del clasificador LLM y su validación estructural (tag `intents/v0.1.0`); lo consumen `wapp-edge-intent` y el Edge. | [README](intents/README.md) | [CHANGELOG](intents/CHANGELOG.md) |
| `ui` | Design tokens y componentes CSS del ecosistema servidos desde Go con `embed.FS`; los consumen la Consola Cloud BFF y la Edge Agent UI. | [README](ui/README.md) | [CHANGELOG](ui/CHANGELOG.md) |
| `llm` | Puerto unico `LLMProvider` (un metodo por tarea del pipeline), prompts compartidos en espanol, parsers de artefactos versionados y la implementacion `llm/api` contra API externa (anthropic completo, gemini stub). | [README](llm/README.md) | [CHANGELOG](llm/CHANGELOG.md) |
| `textmatch` | Motor determinista de comparacion de textos (normalizacion que preserva la ñ, distancia OSA, cascada Exact/Fuzzy y `SetMatcher`); la zona gris (LLM) se **inyecta**, el modulo no importa `llm`. | [README](textmatch/README.md) | [CHANGELOG](textmatch/CHANGELOG.md) |
| `web` | Middleware web endurecido y compartido: nonce, CSP, CSRF, rate-limit, deadline, sesion, single-flight generico, CORS, trusted-proxies, body-limit y flash. Paquete raiz **solo stdlib**; `web/gin` (paquete `webgin`) es el adaptador delgado a Gin. Reconcilia los forks divergidos del BFF y la consola (Plan 047 · O0.5). | [README](web/README.md) | [CHANGELOG](web/CHANGELOG.md) |
| `iam` | Cliente del plano de identidad: login contra identity, refresh, logout y canje a Context Token. El `system` es campo del cliente, no constante: el mismo cliente sirve a `wapp.bff` y a `wapp.platform` sin ramas. | [README](iam/README.md) | [CHANGELOG](iam/CHANGELOG.md) |

## Ingenieria de releases

El inventario unico de modulos vive en `scripts/module-manifest.tsv` y alimenta el
`Makefile` raiz y los workflows de CI/release.

Operaciones globales (desde la raiz):

```bash
make build-all      # compila todos los modulos
make test-all       # tests unitarios de todos los modulos
make vet-all        # go vet en todos los modulos
make check-all      # fmt + vet + lint + test + build
```

Release de un modulo (ver flujo detallado mas abajo):

```bash
make changelog-module MODULE=logger VERSION=v0.1.0   # versiona el CHANGELOG del modulo
make release-module   MODULE=logger VERSION=v0.1.0   # valida, crea y empuja el tag logger/v0.1.0
```

El push del tag `modulo/vX.Y.Z` dispara `.github/workflows/release.yml`, que valida
el modulo y publica el GitHub Release usando el `CHANGELOG.md` del modulo.

## Toolchain

- Go **1.26**
- golangci-lint **v2.12.2** (`make tools`; el `Makefile` fija `LINT_VERSION := v2.12.2`)
