package envelope

import (
	"crypto/rand"
	"errors"
	"fmt"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/nacl/box"
)

// PublicKeySize es el tamaño en bytes de una clave pública X25519.
const PublicKeySize = 32

// PrivateKeySize es el tamaño en bytes de una clave privada X25519.
const PrivateKeySize = 32

// ErrPublicKeySize indica que la clave pública del destinatario no mide [PublicKeySize] bytes.
var ErrPublicKeySize = errors.New("la clave pública debe medir exactamente 32 bytes (X25519)")

// ErrPrivateKeySize indica que la clave privada no mide [PrivateKeySize] bytes.
var ErrPrivateKeySize = errors.New("la clave privada debe medir exactamente 32 bytes (X25519)")

// ErrOpenFailed indica que el sellado no pudo abrirse: clave privada equivocada, sellado
// manipulado o truncado.
var ErrOpenFailed = errors.New("no se pudo abrir el sellado (clave privada incorrecta o datos manipulados)")

// GenerateKeyPair genera un par de claves X25519 nuevo.
//
// La pública se comparte (p. ej. Ks_pub del servidor o Kd_pub del dispositivo); la privada nunca
// sale de su titular. Ambas miden 32 bytes ([PublicKeySize] / [PrivateKeySize]).
func GenerateKeyPair() (pub, priv []byte, err error) {
	pk, sk, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("box.GenerateKey: %w", err)
	}
	// box opera sobre arrays [32]byte; los copiamos a slices para una API pública sin tamaños fijos.
	pubOut := make([]byte, PublicKeySize)
	privOut := make([]byte, PrivateKeySize)
	copy(pubOut, pk[:])
	copy(privOut, sk[:])
	return pubOut, privOut, nil
}

// SealFor sella plaintext hacia el destinatario identificado por recipientPub (su clave pública
// X25519). Solo la clave privada correspondiente puede abrirlo con [OpenWith].
//
// Usa sellado anónimo NaCl box: se genera un par efímero por llamada, así que el emisor no necesita
// identidad propia y dos sellados del mismo plaintext difieren. El sellado resultante incluye la
// clave pública efímera prefijada y el overhead de autenticación; devuelve [ErrPublicKeySize] si la
// pública del destinatario no mide 32 bytes.
func SealFor(recipientPub, plaintext []byte) ([]byte, error) {
	if len(recipientPub) != PublicKeySize {
		return nil, ErrPublicKeySize
	}
	var pubArr [32]byte
	copy(pubArr[:], recipientPub)

	sealed, err := box.SealAnonymous(nil, plaintext, &pubArr, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("box.SealAnonymous: %w", err)
	}
	return sealed, nil
}

// OpenWith abre un sellado producido por [SealFor], usando la clave privada X25519 del destinatario.
//
// Reconstruye la pública desde la privada (X25519 lo permite) porque box.OpenAnonymous necesita
// ambas. Devuelve [ErrPrivateKeySize] si la privada no mide 32 bytes, o [ErrOpenFailed] si el
// sellado no corresponde a esta clave o fue manipulado/truncado.
func OpenWith(priv, sealed []byte) ([]byte, error) {
	if len(priv) != PrivateKeySize {
		return nil, ErrPrivateKeySize
	}
	var privArr, pubArr [32]byte
	copy(privArr[:], priv)
	// La pública X25519 es la privada multiplicada por el punto base; curve25519 la deriva.
	pubDerived, err := publicFromPrivate(privArr)
	if err != nil {
		return nil, err
	}
	pubArr = pubDerived

	plaintext, ok := box.OpenAnonymous(nil, sealed, &pubArr, &privArr)
	if !ok {
		return nil, ErrOpenFailed
	}
	return plaintext, nil
}

// publicFromPrivate deriva la clave pública X25519 correspondiente a una privada, multiplicando
// el escalar por el punto base de la curva (curve25519.Basepoint). Es una operación pública y
// determinista: la misma privada produce siempre la misma pública.
func publicFromPrivate(priv [32]byte) ([32]byte, error) {
	var pub [32]byte
	derived, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		return pub, fmt.Errorf("curve25519.X25519: %w", err)
	}
	copy(pub[:], derived)
	return pub, nil
}
