package iam

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
)

// tokenTypeBearer es el esquema con el que se presenta el Context Token. Es constante del contrato,
// no algo que el upstream elija: la plataforma emite Bearer y el llamante lo presenta como Bearer.
const tokenTypeBearer = "Bearer"

// IdentityContext son los datos de negocio que viajan en el CONTEXT Token: tenant, usuario y roles.
//
// El nombre viene de las dos implementaciones que este módulo reconcilia y se conserva para que la
// migración sea un cambio de import. Ojo con leerlo literalmente: NO sale del Identity Token —que no
// tiene claims de negocio y no puede tenerlos— sino del Context Token que emitió la plataforma.
type IdentityContext struct {
	// TenantID es el tenant de wApp en el que opera el usuario.
	TenantID string `json:"tenant_id"`
	// UserID es el usuario de wApp.
	UserID string `json:"user_id"`
	// Roles son los roles con los que la plataforma selló el Context Token.
	Roles []string `json:"roles"`
}

// AuthResult es la sesión que el llamante custodia tras un [Client.Login] o un [Client.Refresh].
type AuthResult struct {
	// AccessToken es SIEMPRE el Context Token de wApp, nunca el Identity Token. De esa regla depende
	// que el tenant se pueda seguir leyendo de los claims: un Identity Token en la cookie haría
	// desaparecer el tenant sin más aviso.
	AccessToken string `json:"access_token"`
	// RefreshToken es el refresh opaco de identity, el que rota en cada uso.
	RefreshToken string `json:"refresh_token"`
	// TokenType es siempre "Bearer".
	TokenType string `json:"token_type"`
	// ExpiresAt es el vencimiento del Context Token en RFC3339, tal como lo acotó la plataforma.
	ExpiresAt string `json:"expires_at"`
	// Context son los claims de negocio leídos del Context Token SIN verificar la firma: solo
	// alimentan la traza del llamante. Quien valida de verdad es la plataforma en cada llamada.
	Context IdentityContext `json:"context"`
}

// Login autentica en identity y canja al instante: dos saltos server-to-server, una sola sesión.
//
// El Identity Token muere aquí: no vuelve al llamante, no entra en ninguna cookie y no se registra.
func (c *Client) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	tokens, err := c.IdentityLogin(ctx, email, password)
	if err != nil {
		return nil, err
	}
	return c.session(ctx, tokens)
}

// Refresh rota el refresh en identity y vuelve a canjear, con el mismo trato al Identity Token.
func (c *Client) Refresh(ctx context.Context, refreshToken string) (*AuthResult, error) {
	tokens, err := c.IdentityRefresh(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	return c.session(ctx, tokens)
}

// session canjea el Identity Token y arma la sesión: Context Token como access, refresh de identity,
// y el vencimiento que la plataforma acotó con el del Identity Token.
func (c *Client) session(ctx context.Context, tokens *IdentityTokens) (*AuthResult, error) {
	exchanged, err := c.Exchange(ctx, tokens.IdentityToken)
	if err != nil {
		return nil, err
	}
	return &AuthResult{
		AccessToken:  exchanged.ContextToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokenTypeBearer,
		ExpiresAt:    exchanged.ExpiresAt,
		Context:      contextOf(exchanged.ContextToken),
	}, nil
}

// contextOf lee tenant, usuario y roles del CONTEXT Token, que es de donde salen siempre.
//
// SIN verificar la firma y a mano, con la stdlib: el token acaba de llegar por un canal
// server-to-server y quien lo valida de verdad es la plataforma en cada llamada, así que aquí solo
// alimenta la traza. Hacerlo con la stdlib es lo que mantiene este módulo sin dependencias y sin
// importar `wapp-shared/auth` (que es lógica pura y no debe cargar con un cliente HTTP).
//
// Un token ilegible devuelve un contexto vacío en vez de tumbar el login: la sesión es válida —la
// emitió la plataforma— aunque este módulo no sepa leerle los claims.
func contextOf(contextToken string) IdentityContext {
	parts := strings.Split(contextToken, ".")
	if len(parts) != 3 {
		return IdentityContext{}
	}
	// El payload de un JWT es base64url SIN relleno, pero hay emisores que lo mandan con él: se
	// recorta para aceptar los dos y no depender de cuál nos toque.
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return IdentityContext{}
	}
	var claims IdentityContext
	if err := json.Unmarshal(payload, &claims); err != nil {
		return IdentityContext{}
	}
	return claims
}
