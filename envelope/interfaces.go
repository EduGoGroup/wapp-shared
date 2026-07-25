package envelope

// Crypter abstrae las operaciones de cifrado y descifrado simétrico (p. ej. AES-256-GCM).
type Crypter interface {
	Seal(plaintext []byte) ([]byte, error)
	Open(blob []byte) ([]byte, error)
	Overhead() int
}

// Sealer abstrae el sellado y apertura asimétrica (p. ej. NaCl Box / X25519).
type Sealer interface {
	SealFor(recipientPub, plaintext []byte) ([]byte, error)
	OpenWith(priv, sealed []byte) ([]byte, error)
}
