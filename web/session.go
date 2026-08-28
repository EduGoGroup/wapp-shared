package web

import (
	"encoding/base64"
	"encoding/json"
	"time"
)

// SessionData es lo MÍNIMO que una consola custodia server-side para operar y
// refrescar: nunca material criptográfico, nunca datos de negocio. Viaja dentro
// de la cookie de sesión, que es HttpOnly.
type SessionData struct {
	AccessToken  string `json:"a"`
	RefreshToken string `json:"r"`
	ExpiresAt    string `json:"e,omitempty"`
}

// EncodeSession serializa la sesión al valor de la cookie (base64 URL-safe sin
// padding, que es seguro dentro de una cookie sin escapes).
func EncodeSession(s SessionData) (string, error) {
	// #nosec G117 -- serializar los tokens es justo lo que hace esta función: el
	// resultado va DENTRO de la cookie de sesión, que es HttpOnly. No se loguea
	// ni se persiste en ningún otro sitio.
	raw, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// DecodeSession revierte EncodeSession.
func DecodeSession(value string) (SessionData, error) {
	var s SessionData
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return s, err
	}
	return s, nil
}

// DefaultRefreshMargin es el colchón del refresh PROACTIVO: se refresca antes de
// que el token expire, para que ninguna petición del usuario llegue con uno
// caducado por los pelos.
const DefaultRefreshMargin = 2 * time.Minute

// SessionValid es la validación mínima que hace la capa web: que haya `exp` y
// que esté en el futuro. La firma NO se verifica aquí —eso es del emisor y del
// upstream—; esta capa solo decide si merece la pena intentar usar el token.
//
// exp nil significa que el token no traía `exp`: se trata como inválido.
func SessionValid(exp *time.Time) bool {
	if exp == nil {
		return false
	}
	return exp.After(time.Now())
}

// RefreshDue dice si conviene refrescar ya. Un token sin `exp` se refresca
// siempre: no hay forma de saber cuánto le queda. margin <= 0 cae a
// DefaultRefreshMargin.
func RefreshDue(exp *time.Time, margin time.Duration) bool {
	if margin <= 0 {
		margin = DefaultRefreshMargin
	}
	if exp == nil {
		return true
	}
	return time.Until(*exp) < margin
}
