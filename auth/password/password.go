package password

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12
const maxPasswordLength = 72

var ErrInvalidInput = errors.New("auth/password: invalid input")

// DefaultHasher es una estructura por defecto que implementa la interfaz Hasher.
type DefaultHasher struct{}

func NewHasher() *DefaultHasher {
	return &DefaultHasher{}
}

func (h *DefaultHasher) HashPassword(password string) (string, error) {
	return HashPassword(password)
}

func (h *DefaultHasher) CheckPasswordHash(password, hash string) bool {
	return VerifyPassword(hash, password) == nil
}

// HashPassword genera un hash bcrypt del password.
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
func VerifyPassword(hashedPassword, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return fmt.Errorf("password inválido: %w", err)
	}
	return nil
}
