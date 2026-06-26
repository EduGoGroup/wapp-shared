package envelope_test

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"

	"github.com/EduGoGroup/wapp-shared/envelope"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// randBytes genera n bytes aleatorios para los vectores de prueba.
func randBytes(t *testing.T, n int) []byte {
	t.Helper()
	b := make([]byte, n)
	_, err := io.ReadFull(rand.Reader, b)
	require.NoError(t, err)
	return b
}

// --- Capa simétrica (AES-256-GCM) ---

func TestEnvelope_SealOpen_RoundTrip(t *testing.T) {
	dek := randBytes(t, envelope.DEKSize)
	env, err := envelope.NewEnvelope(dek)
	require.NoError(t, err)

	cases := map[string][]byte{
		"vacío":   {},
		"corto":   []byte("hola"),
		"binario": randBytes(t, 1024),
	}
	for name, plaintext := range cases {
		t.Run(name, func(t *testing.T) {
			sealed, err := env.Seal(plaintext)
			require.NoError(t, err)
			// El ciphertext nunca es igual al plaintext y carga el overhead fijo.
			assert.Len(t, sealed, len(plaintext)+envelope.Overhead)
			assert.Equal(t, envelope.Overhead, env.Overhead())

			opened, err := env.Open(sealed)
			require.NoError(t, err)
			// GCM Open con dst nil devuelve nil para plaintext vacío; Empty cubre nil y len 0.
			if len(plaintext) == 0 {
				assert.Empty(t, opened)
			} else {
				assert.Equal(t, plaintext, opened)
			}
		})
	}
}

func TestEnvelope_FreshNoncePerSeal(t *testing.T) {
	dek := randBytes(t, envelope.DEKSize)
	env, err := envelope.NewEnvelope(dek)
	require.NoError(t, err)

	plaintext := []byte("mismo mensaje")
	a, err := env.Seal(plaintext)
	require.NoError(t, err)
	b, err := env.Seal(plaintext)
	require.NoError(t, err)
	// Dos cifrados del mismo plaintext difieren por el nonce aleatorio prefijado.
	assert.False(t, bytes.Equal(a, b), "el nonce debe ser fresco por sello")
}

func TestEnvelope_Open_WrongDEKFails(t *testing.T) {
	good, err := envelope.NewEnvelope(randBytes(t, envelope.DEKSize))
	require.NoError(t, err)
	bad, err := envelope.NewEnvelope(randBytes(t, envelope.DEKSize))
	require.NoError(t, err)

	sealed, err := good.Seal([]byte("secreto"))
	require.NoError(t, err)

	// Abrir con otra DEK debe fallar en la verificación del tag GCM.
	_, err = bad.Open(sealed)
	require.Error(t, err)
}

func TestEnvelope_Open_TamperedFails(t *testing.T) {
	env, err := envelope.NewEnvelope(randBytes(t, envelope.DEKSize))
	require.NoError(t, err)

	sealed, err := env.Seal([]byte("integridad"))
	require.NoError(t, err)
	// Voltear un bit del último byte (dentro del tag) invalida la autenticación.
	sealed[len(sealed)-1] ^= 0x01

	_, err = env.Open(sealed)
	require.Error(t, err)
}

func TestNewEnvelope_BadKeySize(t *testing.T) {
	for _, size := range []int{0, 16, 31, 33} {
		_, err := envelope.NewEnvelope(make([]byte, size))
		require.ErrorIs(t, err, envelope.ErrKeySize)
	}
}

func TestEnvelope_Open_TooShort(t *testing.T) {
	env, err := envelope.NewEnvelope(randBytes(t, envelope.DEKSize))
	require.NoError(t, err)
	// Un blob menor al nonce no puede contener ni el nonce.
	_, err = env.Open([]byte{0x00, 0x01})
	require.ErrorIs(t, err, envelope.ErrBlobTooShort)
}

// --- Capa asimétrica (sellado X25519 vía nacl/box) ---

func TestSealFor_OpenWith_RoundTrip(t *testing.T) {
	pub, priv, err := envelope.GenerateKeyPair()
	require.NoError(t, err)
	assert.Len(t, pub, envelope.PublicKeySize)
	assert.Len(t, priv, envelope.PrivateKeySize)

	plaintext := randBytes(t, envelope.DEKSize) // caso real: sellar una DEK de 32B
	sealed, err := envelope.SealFor(pub, plaintext)
	require.NoError(t, err)
	assert.False(t, bytes.Equal(sealed, plaintext))

	opened, err := envelope.OpenWith(priv, sealed)
	require.NoError(t, err)
	assert.Equal(t, plaintext, opened)
}

func TestSealFor_AnonymousIsNonDeterministic(t *testing.T) {
	pub, _, err := envelope.GenerateKeyPair()
	require.NoError(t, err)

	msg := []byte("misma DEK sellada dos veces")
	a, err := envelope.SealFor(pub, msg)
	require.NoError(t, err)
	b, err := envelope.SealFor(pub, msg)
	require.NoError(t, err)
	// El par efímero por sello hace que dos sellados del mismo dato difieran.
	assert.False(t, bytes.Equal(a, b))
}

func TestOpenWith_WrongPrivateKeyFails(t *testing.T) {
	pub, _, err := envelope.GenerateKeyPair()
	require.NoError(t, err)
	_, otherPriv, err := envelope.GenerateKeyPair()
	require.NoError(t, err)

	sealed, err := envelope.SealFor(pub, []byte("solo para el dueño de priv"))
	require.NoError(t, err)

	// Abrir con la privada equivocada no debe revelar nada.
	_, err = envelope.OpenWith(otherPriv, sealed)
	require.ErrorIs(t, err, envelope.ErrOpenFailed)
}

func TestOpenWith_TamperedSealFails(t *testing.T) {
	pub, priv, err := envelope.GenerateKeyPair()
	require.NoError(t, err)

	sealed, err := envelope.SealFor(pub, []byte("autenticado"))
	require.NoError(t, err)
	sealed[len(sealed)-1] ^= 0x01

	_, err = envelope.OpenWith(priv, sealed)
	require.ErrorIs(t, err, envelope.ErrOpenFailed)
}

func TestSealFor_BadPublicKeySize(t *testing.T) {
	for _, size := range []int{0, 16, 31, 33} {
		_, err := envelope.SealFor(make([]byte, size), []byte("x"))
		require.ErrorIs(t, err, envelope.ErrPublicKeySize)
	}
}

func TestOpenWith_BadPrivateKeySize(t *testing.T) {
	for _, size := range []int{0, 16, 31, 33} {
		_, err := envelope.OpenWith(make([]byte, size), []byte("x"))
		require.ErrorIs(t, err, envelope.ErrPrivateKeySize)
	}
}

// TestCrossLayer_SealDEKThenEncrypt simula el flujo zero-knowledge (ADR-0007):
// se sella una DEK hacia el servidor, el servidor la abre y con ella descifra un blob de datos.
func TestCrossLayer_SealDEKThenEncrypt(t *testing.T) {
	// El servidor tiene su par Ks.
	ksPub, ksPriv, err := envelope.GenerateKeyPair()
	require.NoError(t, err)

	// El dispositivo genera una DEK y cifra datos en reposo con ella.
	dek := randBytes(t, envelope.DEKSize)
	dataEnv, err := envelope.NewEnvelope(dek)
	require.NoError(t, err)
	blob, err := dataEnv.Seal([]byte("store de whatsmeow"))
	require.NoError(t, err)

	// El dispositivo sella la DEK hacia Ks_pub y la manda al servidor.
	sealedDEK, err := envelope.SealFor(ksPub, dek)
	require.NoError(t, err)

	// El servidor abre la DEK con Ks_priv y descifra el blob.
	recoveredDEK, err := envelope.OpenWith(ksPriv, sealedDEK)
	require.NoError(t, err)
	require.Equal(t, dek, recoveredDEK)

	serverEnv, err := envelope.NewEnvelope(recoveredDEK)
	require.NoError(t, err)
	plaintext, err := serverEnv.Open(blob)
	require.NoError(t, err)
	assert.Equal(t, []byte("store de whatsmeow"), plaintext)
}
