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
| `auth` | JWT (HS256) + JWT service M2M + RBAC glob + bcrypt + refresh opaco. | [README](auth/README.md) | [CHANGELOG](auth/CHANGELOG.md) |

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
- golangci-lint **v2.4.0** (`make tools`)
