package jwt

import (
	"github.com/EduGoGroup/wapp-shared/auth/rbac"
	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenUseAccess  = "access"
	TokenUseRefresh = "refresh"
)

type Grants = rbac.Grants

type Claims struct {
	UserID   string   `json:"user_id"`
	TenantID string   `json:"tenant_id"`
	Roles    []string `json:"roles"`
	Grants   Grants   `json:"grants"`
	TokenUse string   `json:"token_use,omitempty"`
	jwt.RegisteredClaims
}
