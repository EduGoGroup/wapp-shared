# auth

El **plano de contexto** de wApp: emisión y verificación de los JWT que llevan
`{tenant_id, user_id, roles, grants}` — **ES256** (ECDSA P-256, con `kid` y
`MultiVerifier` para coexistencia de algoritmos; ADR-0019) — más los service
tokens M2M (HS256). Lógica pura: sin base de datos ni HTTP.

Desde **v0.4.0 este módulo ya no hace identidad**. Los usuarios, sus
credenciales y sus sesiones son de identity, el SSO del grupo, y su cerrajería
vive en [`identity-shared/auth`](https://github.com/EduGoGroup/identity-shared)
(identity ADR-0001):

| Necesitas… | Impórtalo de |
|---|---|
| RBAC: `Grants`, `PermissionMatches`, `EvaluateGrants`, `ResolveRoleChain`, `MergeGrantChain` | `identity-shared/auth/rbac` |
| Contraseñas: `HashPassword`, `VerifyPassword` | `identity-shared/auth/password` |
| Refresh opaco: `GenerateRefreshToken`, `HashRefreshToken` | `identity-shared/auth/jwt` |

## Instalacion

```bash
go get github.com/EduGoGroup/wapp-shared/auth
```

El módulo **no expone paquete raíz** (desde `v0.3.0` el `package auth` a secas
ya no existe) y desde `v0.4.0` su único subpaquete es **`auth/jwt`**.

## Uso

```go
package main

import (
	"time"

	"github.com/EduGoGroup/identity-shared/auth/rbac"
	"github.com/EduGoGroup/wapp-shared/auth/jwt"
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

| Archivo | Qué expone |
|---|---|
| `jwt/jwt_manager.go` · `jwt/jwt_claims.go` | `JWTManager`, `Claims`, `Grants` (alias de `rbac.Grants`), `TokenUseAccess` |
| `jwt/jwt_multiverifier.go` | `MultiVerifier`, `VerifierKey`, `HS256VerifierKey`, `ES256VerifierKey` |
| `jwt/service_claims.go` | `ServiceJWTManager`, `ServiceClaims`, `TokenUseService` (M2M por scopes) |
| `jwt/errors.go` | `ErrInvalidInput`, `ErrTokenExpired`, `ErrInvalidToken` |

## Lo transitorio, declarado

`ServiceJWTManager` (HS256, M2M) **muere en la Ola 3** de la migración a
identity, cuando el M2M pase a Service Tokens ES256 canjeados contra el SSO
(identity ADR-0025). Sigue aquí solo para no arrastrar dos migraciones a la vez.
Lo mismo vale para `NewJWTManager` y `HS256VerifierKey`, que ya no firman ni
verifican tokens de usuario en producción.

## RBAC — gramática de permisos

La define y la aplica `identity-shared/auth/rbac`; aquí solo se **transportan**
los grants dentro de los claims. `recurso.accion` con comodines: `*` (todo),
`prefix.*` (subárbol), `*.suffix`, `prefix.*.suffix`; evaluación **default
DENY** y **deny precede a allow**.

Roles canónicos de wApp: `tenant_admin` = `*`; `operator` = `flows.*,
messages.send, media.*, contacts.read, integrations.read`; `viewer` = `*.read`.
