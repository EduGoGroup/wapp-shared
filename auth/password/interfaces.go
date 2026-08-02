package password

// Hasher define el contrato para hashing y verificación de contraseñas.
type Hasher interface {
	HashPassword(pwd string) (string, error)
	CheckPasswordHash(pwd, hash string) bool
}
