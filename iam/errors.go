package iam

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrUnauthorized señala un 401 del plano de identidad: la credencial presentada —la contraseña, el
// refresh o el Identity Token— no vale o ya no vale.
var ErrUnauthorized = errors.New("iam: no autorizado")

// ErrForbidden señala un 403 del System Gate de identity: el usuario existe y su contraseña es
// CORRECTA, pero esta aplicación no está entre las suyas.
//
// No se colapsa con [ErrUnauthorized] a propósito. Son dos diagnósticos distintos —«no eres tú» y
// «eres tú, pero esta aplicación no es tuya»— y de distinguirlos depende que quien lo lea sepa si
// tiene que cambiar la contraseña o pedir un alta en el catálogo. La consola de operadores los
// colapsaba y el operador veía «credenciales inválidas» con la contraseña buena.
var ErrForbidden = errors.New("iam: aplicación no autorizada para el usuario")

// ErrDualModeOff señala el 503 del canje: la plataforma no tiene construido su verificador de
// Identity Tokens, así que el modo dual está apagado de SU lado y no puede canjear nada.
//
// Se distingue del resto de fallos porque no es una avería: es un despliegue incompleto —el llamante
// delega pero la plataforma todavía no verifica— y se arregla configurándole a la plataforma su
// WAPP_IDENTITY_JWKS_URL, no reintentando.
var ErrDualModeOff = errors.New("iam: canje no disponible (modo dual apagado en la plataforma)")

// ErrInvalidOptions señala unas [Options] con las que no se puede hablar con el plano de identidad.
//
// Existe porque el `system` dejó de ser una constante del binario para ser un campo: lo que antes no
// podía estar mal ahora sí, y un `system` vacío no falla al construir el cliente sino en el login,
// con un 403 del System Gate que parece un problema de permisos del usuario.
var ErrInvalidOptions = errors.New("iam: opciones inválidas")

// APIError es un fallo del upstream que conserva la operación del contrato y el status HTTP.
//
// No lleva el cuerpo de la respuesta, y no es un olvido: todo lo que viaja por este plano son
// credenciales, y un cuerpo de error puede repetir de vuelta lo que se le mandó. El status es la
// información que el llamante necesita y la única que se conserva.
type APIError struct {
	// Op es la operación del contrato que falló ("identity login", "exchange", …).
	Op string
	// StatusCode es el status HTTP que devolvió el upstream.
	StatusCode int
	// kind es el error con nombre que este status representa (o nil si no tiene ninguno). Va sin
	// exportar para que el único modo de construir un APIError con sentido sea statusError.
	kind error
}

// Error implementa la interfaz error con la operación y el status, nunca con el cuerpo del upstream.
func (e *APIError) Error() string {
	return fmt.Sprintf("iam: %s devolvió status %d", e.Op, e.StatusCode)
}

// Unwrap expone el error con nombre del status (ErrUnauthorized, ErrForbidden) para errors.Is.
//
// Que el sentinela viaje DENTRO del APIError es lo que permite preguntar las dos cosas al mismo
// error: `errors.Is(err, ErrUnauthorized)` y `StatusCodeOf(err) == 401`. Envolver al revés —el
// sentinela fuera, con fmt.Errorf— dejaba StatusCodeOf devolviendo 0 justo en los dos status que más
// se diagnostican.
func (e *APIError) Unwrap() error { return e.kind }

// StatusCodeOf extrae el status HTTP del upstream de un error de este módulo (0 si no lo lleva).
func StatusCodeOf(err error) int {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode
	}
	return 0
}

// statusError traduce un status no-2xx al error tipado del contrato.
func statusError(op string, status int) error {
	err := &APIError{Op: op, StatusCode: status}
	switch status {
	case http.StatusUnauthorized:
		err.kind = ErrUnauthorized
	case http.StatusForbidden:
		err.kind = ErrForbidden
	}
	return err
}
