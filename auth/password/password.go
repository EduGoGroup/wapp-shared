package password

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// bcryptCost define el costo computacional de bcrypt (12 ≈ 250ms por hash:
// balance entre seguridad y latencia de login).
const bcryptCost = 12

// maxPasswordLength es el límite de bcrypt (72 bytes).
const maxPasswordLength = 72

// ErrInvalidInput indica que el password de entrada es inválido (hoy: excede el
// límite de bcrypt). Las funciones lo envuelven con contexto vía fmt.Errorf("%w: ...").
var ErrInvalidInput = errors.New("auth/password: invalid input")

// DefaultHasher es la implementación por defecto de [Hasher]: delega en
// [HashPassword] y [VerifyPassword]. Existe para que los consumidores puedan
// inyectar el hasher en sus casos de uso (y sustituirlo por un doble en tests)
// sin renunciar a las funciones sueltas.
type DefaultHasher struct{}

// NewHasher construye un [DefaultHasher].
func NewHasher() *DefaultHasher {
	return &DefaultHasher{}
}

// HashPassword genera un hash bcrypt del password. Equivale a la función
// [HashPassword].
func (h *DefaultHasher) HashPassword(password string) (string, error) {
	return HashPassword(password)
}

// CheckPasswordHash indica si el password corresponde al hash. Es la forma
// booleana de [VerifyPassword]: colapsa cualquier error a false.
func (h *DefaultHasher) CheckPasswordHash(password, hash string) bool {
	return VerifyPassword(hash, password) == nil
}

// HashPassword genera un hash bcrypt del password (con salt aleatorio
// automático). Devuelve error si el password excede 72 bytes.
func HashPassword(password string) (string, error) {
	if len(password) > maxPasswordLength {
		return "", fmt.Errorf("%w: password excede el máximo de %d bytes", ErrInvalidInput, maxPasswordLength)
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("no se pudo hashear el password: %w", err)
	}
	return string(bytes), nil
}

// VerifyPassword verifica que `password` coincida con `hashedPassword`.
// Devuelve nil si coincide; error en caso contrario.
func VerifyPassword(hashedPassword, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return fmt.Errorf("password inválido: %w", err)
	}
	return nil
}
