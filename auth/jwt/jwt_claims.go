package jwt

import (
	"github.com/EduGoGroup/wapp-shared/auth/rbac"
	"github.com/golang-jwt/jwt/v5"
)

// Valores del claim `token_use` para los JWT de usuario. El service token usa
// TokenUseService (ver service_claims.go).
const (
	// TokenUseAccess identifica un access token de usuario.
	TokenUseAccess = "access"
	// TokenUseRefresh identifica un JWT de refresh (no confundir con el refresh
	// token OPACO de refresh_token.go, que es el mecanismo por defecto en wApp).
	TokenUseRefresh = "refresh"
)

// Grants es el wire format de permisos que viaja en los claims: un ALIAS de
// [rbac.Grants], no un tipo nuevo. Los dos nombres son intercambiables sin
// conversión, así que un consumidor que solo emite y valida tokens no necesita
// importar auth/rbac. La gramática glob, la precedencia deny-sobre-allow y el
// default DENY los define y aplica ese paquete; aquí solo se transportan.
type Grants = rbac.Grants

// Claims representa los claims personalizados del JWT de usuario de wApp.
//
// Está SIMPLIFICADO respecto al auth de EduGo: solo lleva la identidad
// multi-tenant ({UserID, TenantID}), el conjunto de roles y los grants
// efectivos ya resueltos (rol + cadena de herencia ⊕ overrides), de modo que
// el middleware evalúa cada request contra estos patterns SIN consultar la BD.
// No incluye la jerarquía escuela/unidad/ward de EduGo (Decisión C).
type Claims struct {
	UserID   string   `json:"user_id"`
	TenantID string   `json:"tenant_id"`
	Roles    []string `json:"roles"`
	Grants   Grants   `json:"grants"`
	TokenUse string   `json:"token_use,omitempty"`
	jwt.RegisteredClaims
}
