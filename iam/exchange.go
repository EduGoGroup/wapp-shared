package iam

import (
	"context"
	"errors"
	"net/http"
)

// ExchangeResult es la respuesta de `POST /api/v1/auth/exchange` de la plataforma: SOLO el Context
// Token.
//
// El canje no emite refresh, y no es un descuido: el refresh es de identity y vive donde vive la
// sesión. Aquí se renueva presentando otra vez un Identity Token fresco.
type ExchangeResult struct {
	// ContextToken es el JWT de wApp con los claims del negocio (tenant, roles). Es el que la cookie
	// custodia y el único que el llamante vuelve a presentar.
	ContextToken string `json:"context_token"`
	// ExpiresAt es el vencimiento del Context Token en RFC3339. La plataforma lo acota con el del
	// Identity Token que se canjeó: la visa nunca dura más que el pasaporte.
	ExpiresAt string `json:"expires_at"`
}

type exchangeRequest struct {
	IdentityToken string `json:"identity_token"`
}

// Exchange presenta un Identity Token a la plataforma y recibe el Context Token del tenant del
// usuario.
//
// Los errores con nombre del contrato:
//   - **503** — modo dual apagado en la plataforma ([ErrDualModeOff]): no es una avería, es un
//     despliegue a medias.
//   - **401** — el Identity Token es inválido, vencido o su `sub` no corresponde a ningún usuario de
//     wApp ([ErrUnauthorized]).
//   - **409** — el usuario tiene más de un tenant. NO viaja como [ErrUnauthorized] a propósito: «tu
//     sesión ya no vale» pide limpiar la cookie, y «tu sesión vale pero wApp no sabe en qué tenant
//     ponerte» no se arregla echando al usuario. El status se preserva vía [StatusCodeOf].
func (c *Client) Exchange(ctx context.Context, identityToken string) (*ExchangeResult, error) {
	var out ExchangeResult
	if err := c.post(ctx, "exchange", c.platformURL+"/api/v1/auth/exchange",
		exchangeRequest{IdentityToken: identityToken}, &out); err != nil {
		if StatusCodeOf(err) == http.StatusServiceUnavailable {
			return nil, ErrDualModeOff
		}
		return nil, err
	}
	if out.ContextToken == "" {
		return nil, errors.New("iam: exchange: respuesta sin context_token")
	}
	return &out, nil
}
