package envelope

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

// DEKSize es el tamaño exacto en bytes de la DEK (Data Encryption Key) para AES-256.
const DEKSize = 32

// Overhead es el costo fijo en bytes que GCM añade a cualquier plaintext: nonce(12) + tag(16) = 28.
const Overhead = 12 + 16

// ErrKeySize indica que la DEK no mide [DEKSize] bytes.
var ErrKeySize = errors.New("la DEK debe medir exactamente 32 bytes (AES-256)")

// ErrBlobTooShort indica que el blob a abrir no contiene ni siquiera el nonce.
var ErrBlobTooShort = errors.New("blob demasiado corto: no contiene el nonce")

// Envelope cifra/descifra blobs con AES-256-GCM.
//
// Formato en disco: nonce(12B) || ciphertext || tag(16B), todo concatenado. El nonce aleatorio
// se genera por valor y se prefija al ciphertext (nunca se reusa). La DEK es de 32 bytes
// ([DEKSize], AES-256) y se inyecta por construcción.
type Envelope struct {
	aead cipher.AEAD
}

// NewEnvelope construye un [Envelope] con la DEK dada (32 bytes, [DEKSize]).
// Devuelve [ErrKeySize] si la DEK no mide exactamente 32 bytes.
func NewEnvelope(dek []byte) (*Envelope, error) {
	if len(dek) != DEKSize {
		return nil, ErrKeySize
	}
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("aes.NewCipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("cipher.NewGCM: %w", err)
	}
	return &Envelope{aead: aead}, nil
}

// Seal cifra plaintext y devuelve nonce||ciphertext||tag.
//
// Un plaintext nil se cifra igual (resultado: nonce+tag, longitud 28) para no perder información:
// la distinción nil/no-nil aguas arriba se delega al consumidor (aquí siempre se cifra lo que llega).
func (e *Envelope) Seal(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, e.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("no se pudo generar nonce: %w", err)
	}
	// Seal añade el ciphertext+tag al final del nonce que pasamos como dst.
	return e.aead.Seal(nonce, nonce, plaintext, nil), nil
}

// Open descifra un blob con formato nonce||ciphertext||tag.
//
// Si la DEK es incorrecta o el blob fue manipulado, GCM falla en la verificación del tag de
// autenticidad y se devuelve error (justamente esa es la garantía de integridad). Devuelve
// [ErrBlobTooShort] si el blob ni siquiera alcanza para el nonce.
func (e *Envelope) Open(blob []byte) ([]byte, error) {
	ns := e.aead.NonceSize()
	if len(blob) < ns {
		return nil, ErrBlobTooShort
	}
	nonce, ct := blob[:ns], blob[ns:]
	pt, err := e.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("GCM Open falló (DEK incorrecta o blob manipulado): %w", err)
	}
	return pt, nil
}

// Overhead es el costo en bytes que GCM añade a cualquier plaintext: nonce(12) + tag(16) = 28.
// Coincide con la constante [Overhead]; se expone como método por conveniencia del consumidor.
func (e *Envelope) Overhead() int {
	return e.aead.NonceSize() + e.aead.Overhead()
}
