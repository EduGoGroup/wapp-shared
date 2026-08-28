package web

import (
	"crypto/rand"
	"encoding/base64"
	"io"
)

// NonceBytes es el tamaño en bytes del nonce CSP (128 bits): suficiente para que
// no sea adivinable dentro de la vida de una respuesta.
const NonceBytes = 16

// CSRFTokenBytes es el tamaño en bytes del token CSRF (256 bits).
const CSRFTokenBytes = 32

// randomToken devuelve n bytes aleatorios en base64 URL-safe SIN padding.
//
// El alfabeto URL-safe (A-Za-z0-9-_) no es un capricho: el base64 estándar trae
// '+', '/' y '=', y html/template ESCAPA esos caracteres dentro del atributo
// `nonce` (p. ej. '+' -> "&#43;"). Con el escape, el nonce del atributo dejaría
// de ser byte-idéntico al de la cabecera CSP y el navegador bloquearía el bloque
// inline propio. Con el alfabeto URL-safe cabecera y atributo coinciden, y el
// valor sigue siendo un base64 válido para CSP.
//
// r nil significa crypto/rand.Reader. Es un parámetro para que los tests puedan
// forzar el agotamiento de entropía y verificar que el sistema falla CERRADO.
func randomToken(r io.Reader, n int) (string, error) {
	if r == nil {
		r = rand.Reader
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Nonce genera el nonce CSP de una petición (128 bits, base64 URL-safe sin
// padding). r nil significa crypto/rand.Reader.
func Nonce(r io.Reader) (string, error) {
	return randomToken(r, NonceBytes)
}

// NewCSRFToken genera un token CSRF (256 bits, base64 URL-safe sin padding).
// Comparte la fuente de entropía con Nonce a propósito: un solo punto de fallo,
// un solo camino de fail-closed. r nil significa crypto/rand.Reader.
func NewCSRFToken(r io.Reader) (string, error) {
	return randomToken(r, CSRFTokenBytes)
}
