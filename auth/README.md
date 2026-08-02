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

El módulo **no expone paquete raíz**: se importa siempre el subpaquete concreto
(`auth/jwt`, `auth/password`, `auth/rbac`). Desde `v0.3.0` el `package auth` a
secas ya no existe.

## Uso

```go
package main

import (
	"time"

	"github.com/EduGoGroup/wapp-shared/auth/jwt"
	"github.com/EduGoGroup/wapp-shared/auth/rbac"
)

func main() {
	mgr := jwt.NewJWTManager("secret-hs256", "wapp-iam")

	grants := rbac.Grants{
		Allow: []string{"flows.*", "messages.send"},
		Deny:  []string{"crypto.rekey"},
	}
	token, _, _ := mgr.GenerateToken("user-1", "tenant-1", []string{"operator"}, grants, time.Hour)

	claims, _ := mgr.ValidateToken(token)
	_ = rbac.EvaluateGrants(claims.Grants, "messages.send") // true
	_ = rbac.EvaluateGrants(claims.Grants, "crypto.rekey")  // false (deny gana)
}
```

`jwt.Grants` es un alias de `rbac.Grants`, así que los claims se construyen con
cualquiera de los dos nombres sin conversión.

## Piezas

| Paquete / archivo | Qué expone |
|---|---|
| `jwt/jwt_manager.go` · `jwt/jwt_claims.go` | `JWTManager`, `Claims`, `Grants` (alias de `rbac.Grants`), `TokenUseAccess`/`TokenUseRefresh` |
| `jwt/jwt_multiverifier.go` | `MultiVerifier`, `VerifierKey`, `HS256VerifierKey`, `ES256VerifierKey` |
| `jwt/service_claims.go` | `ServiceJWTManager`, `ServiceClaims`, `TokenUseService` (M2M por scopes) |
| `jwt/refresh_token.go` | `RefreshToken`, `GenerateRefreshToken`, `HashToken` (refresh opaco + SHA256) |
| `jwt/interfaces.go` · `jwt/errors.go` | `TokenGenerator`/`TokenVerifier`/`TokenManager`; `ErrInvalidInput`, `ErrTokenExpired`, `ErrInvalidToken` |
| `password/password.go` · `password/interfaces.go` | `HashPassword`, `VerifyPassword` (bcrypt cost 12), `Hasher`, `DefaultHasher`, `NewHasher` |
| `rbac/permission_matcher.go` | `Grants`, `PermissionMatches`, `EvaluateGrants` (glob, deny-precede-allow, default DENY) |
| `rbac/role_chain.go` · `rbac/interfaces.go` | `ResolveRoleChain`, `MergeGrantChain` (herencia de roles); `PermissionEvaluator` |

## RBAC — gramática de permisos

`recurso.accion` con comodines: `*` (todo), `prefix.*` (subárbol),
`*.suffix` (cualquier `<algo>.suffix`), `prefix.*.suffix`. La evaluación es
**default DENY** y **deny precede a allow**.

Roles canónicos: `tenant_admin` = `*`; `operator` = `flows.*, messages.send,
media.*, contacts.read, integrations.read`; `viewer` = `*.read`.
