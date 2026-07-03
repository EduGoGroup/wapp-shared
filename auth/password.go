package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// bcryptCost define el costo computacional de bcrypt (12 ≈ 250ms por hash:
// balance entre seguridad y latencia de login).
const bcryptCost = 12

// maxPasswordLength es el límite de bcrypt (72 bytes).
const maxPasswordLength = 72

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
