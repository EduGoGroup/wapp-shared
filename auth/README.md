# auth

Primitivas **puras** de autenticación y autorización del ecosistema wApp: JWT
de usuario **ES256** (ECDSA P-256, con `kid` y `MultiVerifier` para coexistencia
de algoritmos; ADR-0019), service tokens M2M (HS256), bcrypt, refresh tokens
opacos y RBAC por grants glob con cadena de roles. La firma HS256 sigue disponible
en `JWTManager` para compatibilidad. Sin base de datos ni HTTP.

Es una **copia-adaptación** del módulo auth de EduGo (ADR-0004): se toma el
patrón y se **simplifica** a `{tenant_id, user_id, roles, grants}` — sin la
jerarquía escuela/unidad/ward. **No** importa ningún paquete de EduGo.

## Instalacion

```bash
go get github.com/EduGoGroup/wapp-shared/auth
```

## Uso

```go
package main

import (
	"time"

	"github.com/EduGoGroup/wapp-shared/auth"
)

func main() {
	mgr := auth.NewJWTManager("secret-hs256", "wapp-iam")

	grants := auth.Grants{
		Allow: []string{"flows.*", "messages.send"},
		Deny:  []string{"crypto.rekey"},
	}
	token, _, _ := mgr.GenerateToken("user-1", "tenant-1", []string{"operator"}, grants, time.Hour)

	claims, _ := mgr.ValidateToken(token)
	_ = auth.EvaluateGrants(claims.Grants, "messages.send") // true
	_ = auth.EvaluateGrants(claims.Grants, "crypto.rekey")  // false (deny gana)
}
```

## Piezas

| Archivo | Qué expone |
|---|---|
| `jwt_manager.go` / `jwt_claims.go` | `JWTManager`, `Claims`, `Grants` (access token de usuario) |
| `service_claims.go` | `ServiceJWTManager`, `ServiceClaims` (M2M por scopes) |
| `password.go` | `HashPassword`, `VerifyPassword` (bcrypt cost 12) |
| `refresh_token.go` | `GenerateRefreshToken`, `HashToken` (refresh opaco + SHA256) |
| `permission_matcher.go` | `PermissionMatches`, `EvaluateGrants` (glob, deny-precede-allow, default DENY) |
| `role_chain.go` | `ResolveRoleChain`, `MergeGrantChain` (herencia de roles) |

## RBAC — gramática de permisos

`recurso.accion` con comodines: `*` (todo), `prefix.*` (subárbol),
`*.suffix` (cualquier `<algo>.suffix`), `prefix.*.suffix`. La evaluación es
**default DENY** y **deny precede a allow**.

Roles canónicos: `tenant_admin` = `*`; `operator` = `flows.*, messages.send,
media.*, contacts.read, integrations.read`; `viewer` = `*.read`.
