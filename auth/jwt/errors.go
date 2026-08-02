package jwt

import "errors"

// ErrInvalidInput indica que un argumento de entrada es inválido (vacío,
// fuera de rango, etc.). Los constructores de tokens lo envuelven con
// contexto mediante fmt.Errorf("%w: ...").
var ErrInvalidInput = errors.New("auth/jwt: invalid input")

// ErrTokenExpired indica que el token está bien firmado pero su claim `exp`
// ya venció.
var ErrTokenExpired = errors.New("auth/jwt: token expired")

// ErrInvalidToken indica que el token es inválido: firma incorrecta, método
// de firma inesperado, issuer/audience no coincide, o claims corruptos.
var ErrInvalidToken = errors.New("auth/jwt: invalid token")
