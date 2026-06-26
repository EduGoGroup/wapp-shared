// Package envelope ofrece dos primitivas de cifrado para el modelo zero-knowledge de wApp
// (ADR-0007, doble llave): cifrado simétrico de datos y sellado asimétrico hacia un destinatario.
//
// # Capa simétrica (cifrado de datos)
//
// AES-256-GCM con nonce aleatorio de 12 bytes por valor, prefijado al ciphertext. El formato
// en disco es nonce(12B) || ciphertext || tag(16B); el overhead fijo es de 28 bytes. La DEK
// (Data Encryption Key) es de 32 bytes y se inyecta por construcción. Útil para cifrar blobs
// en reposo (p. ej. el store de whatsmeow) con una clave que el servidor no conoce en claro.
//
// Tipos: [Envelope], [NewEnvelope]. Métodos: [Envelope.Seal], [Envelope.Open], [Envelope.Overhead].
//
// # Capa asimétrica (sellado hacia un destinatario)
//
// Sellado anónimo X25519 vía NaCl box ([golang.org/x/crypto/nacl/box]): cualquiera que conozca
// la clave pública del destinatario puede sellar un blob, y solo la clave privada correspondiente
// puede abrirlo. El emisor no necesita identidad propia (se usa un par efímero interno por sello).
// Es la pieza que mueve una DEK entre dispositivo y servidor sin que el transporte la exponga:
// el dispositivo sella la DEK con Ks_pub (pública del servidor) y el servidor la abre con Ks_priv;
// en el pairing, el servidor sella la DEK con Kd_pub (pública del dispositivo).
//
// Funciones: [GenerateKeyPair], [SealFor], [OpenWith].
//
// Solo se usa criptografía de la stdlib y golang.org/x/crypto; no hay construcciones caseras.
package envelope
