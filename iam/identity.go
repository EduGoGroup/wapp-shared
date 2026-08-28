package iam

import (
	"context"
	"fmt"
)

// IdentityTokens es la respuesta de `POST /api/v1/auth/{login,refresh}` de identity-api.
//
// El Identity Token responde QUIÉN ERES y no lleva claims de negocio: el tenant no está aquí y no
// puede estarlo. Por eso este par no llega nunca a una cookie tal cual — antes se canjea por un
// Context Token de wApp (ver [Client.Exchange]).
type IdentityTokens struct {
	// SessionID es el UUID de la sesión abierta en identity para esta aplicación.
	SessionID string `json:"session_id"`
	// System es la aplicación de la sesión, tal como la devuelve el emisor.
	System string `json:"system"`
	// IdentityToken es el JWT ES256 firmado por identity-core.
	IdentityToken string `json:"identity_token"`
	// RefreshToken es el refresh opaco: se entrega una vez y rota en cada uso.
	RefreshToken string `json:"refresh_token"`
	// ExpiresIn son los segundos de vida que le quedan al Identity Token.
	ExpiresIn int64 `json:"expires_in"`
}

type identityLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	System   string `json:"system"`
}

type identityRefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type identityLogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// IdentityLogin autentica email+password contra `POST /api/v1/auth/login` de identity.
//
// El `system` de [Options] viaja en el cuerpo y es lo que el System Gate evalúa: un usuario sin
// acceso a esta aplicación recibe 403 ([ErrForbidden]) con la contraseña CORRECTA, que es un caso
// distinto del 401 de credenciales inválidas ([ErrUnauthorized]) y por eso no se colapsan.
//
// Devuelve el Identity Token en crudo: quien quiera la sesión de wApp de una vez tiene [Client.Login].
func (c *Client) IdentityLogin(ctx context.Context, email, password string) (*IdentityTokens, error) {
	return c.identityTokens(ctx, "identity login", "/api/v1/auth/login",
		identityLoginRequest{Email: email, Password: password, System: c.system})
}

// IdentityRefresh rota el refresh opaco contra `POST /api/v1/auth/refresh` y emite un Identity Token
// nuevo.
//
// El cuerpo NO lleva `system`: la aplicación sale de la fila de la sesión en identity, nunca del
// cliente. Aceptarlo permitiría canjear el refresh de una aplicación por el token de otra y el
// System Gate quedaría sorteado.
func (c *Client) IdentityRefresh(ctx context.Context, refreshToken string) (*IdentityTokens, error) {
	return c.identityTokens(ctx, "identity refresh", "/api/v1/auth/refresh",
		identityRefreshRequest{RefreshToken: refreshToken})
}

// Logout cierra en identity la sesión del refresh presentado vía `POST /api/v1/auth/logout`.
//
// Cierra UNA sesión, la de esta aplicación: las de las demás —la del Edge, sin ir más lejos—
// sobreviven. Es el modelo Google: cerrar la consola no te echa del teléfono.
//
// No lleva Bearer a propósito: identity resuelve el usuario a partir del refresh, server-side, y
// responde 2xx tanto si había sesión que revocar como si no (un 404 sería un oráculo de validez de
// refresh tokens). El Context Token que el llamante custodia no le sirve a identity: lo emitió wApp.
//
// El error del upstream se PROPAGA: si identity falla, el refresh sigue vivo allí aunque el llamante
// ya haya borrado su cookie, y eso tiene que quedar en el log en vez de perderse en un nil.
func (c *Client) Logout(ctx context.Context, refreshToken string) error {
	return c.post(ctx, "identity logout", c.identityURL+"/api/v1/auth/logout",
		identityLogoutRequest{RefreshToken: refreshToken}, nil)
}

// identityTokens ejecuta la petición y exige que la respuesta traiga el par COMPLETO: un login que
// no devuelve refresh dejaría una sesión sin forma de renovarse y el fallo aparecería quince minutos
// después, lejos de su causa.
func (c *Client) identityTokens(ctx context.Context, op, path string, payload any) (*IdentityTokens, error) {
	var out IdentityTokens
	if err := c.post(ctx, op, c.identityURL+path, payload, &out); err != nil {
		return nil, err
	}
	if out.IdentityToken == "" {
		return nil, fmt.Errorf("iam: %s: respuesta sin identity_token", op)
	}
	if out.RefreshToken == "" {
		return nil, fmt.Errorf("iam: %s: respuesta sin refresh_token", op)
	}
	return &out, nil
}
