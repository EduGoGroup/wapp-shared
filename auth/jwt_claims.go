package auth

import "github.com/golang-jwt/jwt/v5"

// Valores del claim `token_use` para los JWT de usuario. El service token usa
// TokenUseService (ver service_claims.go).
const (
	// TokenUseAccess identifica un access token de usuario.
	TokenUseAccess = "access"
	// TokenUseRefresh identifica un JWT de refresh (no confundir con el refresh
	// token OPACO de refresh_token.go, que es el mecanismo por defecto en wApp).
	TokenUseRefresh = "refresh"
)

// Grants es el wire format de permisos: una lista de patterns allow y una de
// patterns deny en gramática glob (ver PermissionMatches). La precedencia
// deny-sobre-allow y el default DENY los aplica EvaluateGrants, no esta
// estructura.
type Grants struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
}

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
