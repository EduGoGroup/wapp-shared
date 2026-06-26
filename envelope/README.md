# envelope

Primitivas de cifrado para el modelo zero-knowledge de wApp (ADR-0007, doble llave). Dos capas
independientes: cifrado simetrico de datos (AES-256-GCM) y sellado asimetrico hacia un destinatario
(X25519 via NaCl box).

## Instalacion

```bash
go get github.com/EduGoGroup/wapp-shared/envelope
```

Solo depende de la stdlib y `golang.org/x/crypto`. Sin algoritmos caseros.

## Capa simetrica — AES-256-GCM

Cifra blobs en reposo con una DEK (Data Encryption Key) de 32 bytes. Nonce aleatorio de 12 bytes
por valor, prefijado al ciphertext; formato `nonce(12B) || ciphertext || tag(16B)`, overhead fijo de
28 bytes. Si la DEK es incorrecta o el blob fue manipulado, `Open` falla en la verificacion del tag.

```go
env, err := envelope.NewEnvelope(dek)      // dek de 32 bytes
sealed, err := env.Seal(plaintext)         // nonce||ciphertext||tag
plaintext, err := env.Open(sealed)         // error si DEK mala o manipulado
n := env.Overhead()                        // 28
```

Constantes: `DEKSize` (32), `Overhead` (28). Errores: `ErrKeySize`, `ErrBlobTooShort`.

## Capa asimetrica — sellado X25519 (NaCl box anonimo)

Sella un blob (tipicamente una DEK) hacia un destinatario por su clave publica; solo su privada lo abre.
El emisor no necesita identidad propia: cada sello usa un par efimero interno, asi que dos sellados del
mismo dato difieren.

```go
pub, priv, err := envelope.GenerateKeyPair()   // X25519, 32B cada una
sealed, err := envelope.SealFor(pub, dek)       // cifra hacia el destinatario
dek, err := envelope.OpenWith(priv, sealed)     // solo la privada correcta abre
```

Constantes: `PublicKeySize` (32), `PrivateKeySize` (32). Errores: `ErrPublicKeySize`,
`ErrPrivateKeySize`, `ErrOpenFailed`.

### Flujo zero-knowledge (ADR-0007)

- **Subida:** el dispositivo genera una DEK, cifra sus datos con la capa simetrica y sella la DEK con
  `Ks_pub` (publica del servidor). El servidor abre la DEK con `Ks_priv` y descifra.
- **Pairing:** el servidor sella la DEK con `Kd_pub` (publica del dispositivo) para entregarsela.

## Decision de diseño

El sellado asimetrico usa `golang.org/x/crypto/nacl/box` (`SealAnonymous`/`OpenAnonymous`) en vez de
ensamblar `crypto/ecdh` + HKDF + AEAD a mano: NaCl box es una construccion X25519 auditada y estandar
que cubre exactamente el caso (sellado anonimo hacia una publica), con menos superficie de error. Ver
godoc del paquete para el detalle.

## Navegacion

- [Changelog](CHANGELOG.md)

## Comandos disponibles

```bash
make build     # Compilar
make test      # Tests
make check     # Lint y validacion
```
