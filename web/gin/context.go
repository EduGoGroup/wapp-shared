package webgin

import "github.com/gin-gonic/gin"

// Claves del gin.Context que comparte todo el middleware. Los literales son los
// que ya usaban las dos consolas antes de extraer el módulo, así que migrar a él
// no obliga a tocar plantillas ni handlers.
const (
	// ContextAccessToken es el access token de la sesión.
	ContextAccessToken = "access_token"
	// ContextRefreshToken es el refresh token de la sesión.
	ContextRefreshToken = "refresh_token"
	// ContextUserID identifica al usuario autenticado.
	ContextUserID = "user_id"
	// ContextTenantID identifica a la empresa de la sesión.
	ContextTenantID = "tenant_id"
	// ContextCSPNonce es el nonce CSP de ESTA petición.
	ContextCSPNonce = "csp_nonce"
	// ContextCSRFToken es el token CSRF que las plantillas incrustan.
	ContextCSRFToken = "csrf_token"
)

// stringFromContext lee una clave del contexto como cadena (vacía si no está o
// si no es una cadena).
func stringFromContext(c *gin.Context, key string) string {
	v, ok := c.Get(key)
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// NonceFromContext devuelve el nonce CSP sembrado por SecurityHeaders (cadena
// vacía si el middleware no corrió).
func NonceFromContext(c *gin.Context) string { return stringFromContext(c, ContextCSPNonce) }

// CSRFTokenFromContext devuelve el token CSRF sembrado por CSRF.
func CSRFTokenFromContext(c *gin.Context) string { return stringFromContext(c, ContextCSRFToken) }

// UserIDFromContext devuelve el user_id de la sesión.
func UserIDFromContext(c *gin.Context) string { return stringFromContext(c, ContextUserID) }

// TenantIDFromContext devuelve el tenant_id de la sesión.
func TenantIDFromContext(c *gin.Context) string { return stringFromContext(c, ContextTenantID) }

// AccessTokenFromContext devuelve el access token de la sesión.
func AccessTokenFromContext(c *gin.Context) string { return stringFromContext(c, ContextAccessToken) }

// RefreshTokenFromContext devuelve el refresh token de la sesión.
func RefreshTokenFromContext(c *gin.Context) string {
	return stringFromContext(c, ContextRefreshToken)
}

// IsAuthenticated dice si esta petición trae sesión (hay access token sembrado).
func IsAuthenticated(c *gin.Context) bool {
	_, ok := c.Get(ContextAccessToken)
	return ok
}
