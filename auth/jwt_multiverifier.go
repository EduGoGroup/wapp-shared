package auth

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// VerifierKey describe UNA entrada de verificación del [MultiVerifier]: fija su
// algoritmo y su llave pública/secreto. Constrúyela con [HS256VerifierKey] o
// [ES256VerifierKey]; una VerifierKey de valor cero significa "sin entrada".
type VerifierKey struct {
	method jwt.SigningMethod
	key    any // []byte (HS256) o *ecdsa.PublicKey (ES256).
}

// HS256VerifierKey construye una entrada de verificación HS256 (simétrica) a
// partir del secreto compartido. Sirve para validar tokens legacy firmados con
// HS256 durante la coexistencia de algoritmos (ADR-0019).
func HS256VerifierKey(secret string) VerifierKey {
	return VerifierKey{method: jwt.SigningMethodHS256, key: []byte(secret)}
}

// ES256VerifierKey construye una entrada de verificación ES256 (ECDSA P-256) a
// partir de la clave pública. El [MultiVerifier] nunca firma, así que solo
// necesita la pública (mínimo privilegio).
func ES256VerifierKey(publicKey *ecdsa.PublicKey) VerifierKey {
	return VerifierKey{method: jwt.SigningMethodES256, key: publicKey}
}

// isZero indica que la VerifierKey no se configuró (valor cero).
func (k VerifierKey) isZero() bool { return k.method == nil }

// validate comprueba que la llave asociada al algoritmo sea utilizable.
func (k VerifierKey) validate() error {
	switch key := k.key.(type) {
	case []byte:
		if len(key) == 0 {
			return fmt.Errorf("%w: secreto HS256 vacío", ErrInvalidInput)
		}
	case *ecdsa.PublicKey:
		if key == nil {
			return fmt.Errorf("%w: clave pública ES256 nil", ErrInvalidInput)
		}
	default:
		return fmt.Errorf("%w: tipo de llave no soportado para %s", ErrInvalidInput, k.method.Alg())
	}
	return nil
}

// MultiVerifier valida JWT de usuario seleccionando la llave de verificación por
// el header `kid` del token. Cada entrada FIJA su propio algoritmo y su llave,
// de modo que un token cuyo `alg` no coincide con el de la entrada elegida se
// rechaza (guard anti alg-confusion extendido, ADR-0019): un token HS256 con el
// `kid` de una entrada ES256 —o viceversa— se rechaza con ErrInvalidToken.
//
// Un token SIN `kid` se valida con la entrada "default" (típicamente los tokens
// legacy HS256 emitidos antes del corte a ES256). Sin default, un token sin
// `kid` se rechaza.
//
// Es SOLO de verificación: no emite tokens. La validación de issuer y expiración
// es idéntica a [JWTManager.ValidateToken] (reusa esa lógica por entrada).
type MultiVerifier struct {
	byKid map[string]*JWTManager
	def   *JWTManager // verificador para tokens sin `kid`; nil = se rechazan.
}

// NewMultiVerifier construye un MultiVerifier para el issuer dado. `byKid` mapea
// cada `kid` a su VerifierKey; `def` es la entrada para tokens sin `kid` (pásala
// en cero para rechazar los tokens sin `kid`). Devuelve ErrInvalidInput si una
// llave es inutilizable o si no queda ninguna forma de validar (mapa vacío y sin
// default).
func NewMultiVerifier(issuer string, byKid map[string]VerifierKey, def VerifierKey) (*MultiVerifier, error) {
	mv := &MultiVerifier{byKid: make(map[string]*JWTManager, len(byKid))}

	for kid, vk := range byKid {
		if kid == "" {
			return nil, fmt.Errorf("%w: el kid de una entrada no puede estar vacío", ErrInvalidInput)
		}
		if err := vk.validate(); err != nil {
			return nil, fmt.Errorf("entrada %q: %w", kid, err)
		}
		mv.byKid[kid] = newVerifyOnlyManager(issuer, vk)
	}

	if !def.isZero() {
		if err := def.validate(); err != nil {
			return nil, fmt.Errorf("entrada default: %w", err)
		}
		mv.def = newVerifyOnlyManager(issuer, def)
	}

	if len(mv.byKid) == 0 && mv.def == nil {
		return nil, fmt.Errorf("%w: MultiVerifier sin entradas ni default", ErrInvalidInput)
	}

	return mv, nil
}

// newVerifyOnlyManager arma un JWTManager de solo-verificación (sin clave de
// firma) con el algoritmo y la llave de la entrada. Reusa toda la lógica de
// validación de [JWTManager.ValidateToken], incluido el guard anti
// alg-confusion, la verificación de issuer y `exp`.
func newVerifyOnlyManager(issuer string, vk VerifierKey) *JWTManager {
	return &JWTManager{
		issuer:       issuer,
		method:       vk.method,
		signKey:      nil,
		verifyKey:    vk.key,
		validMethods: []string{vk.method.Alg()},
	}
}

// ValidateToken selecciona la entrada por el header `kid` del token y delega en
// su verificador. Semántica de retorno idéntica a [JWTManager.ValidateToken]:
// devuelve *Claims o ErrTokenExpired/ErrInvalidToken. Un `kid` desconocido, un
// token sin `kid` cuando no hay default, o un token malformado devuelven
// ErrInvalidToken.
func (mv *MultiVerifier) ValidateToken(tokenString string) (*Claims, error) {
	// Extraemos el header sin verificar solo para seleccionar la entrada; la
	// verificación real (firma/alg/issuer/exp) la hace el JWTManager elegido.
	unverified, _, err := jwt.NewParser().ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}

	m := mv.def
	if raw, ok := unverified.Header["kid"]; ok {
		kid, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("%w: header kid no es una cadena", ErrInvalidToken)
		}
		m = mv.byKid[kid]
		if m == nil {
			return nil, fmt.Errorf("%w: kid desconocido", ErrInvalidToken)
		}
	}
	if m == nil {
		return nil, fmt.Errorf("%w: token sin kid y sin verificador default", ErrInvalidToken)
	}

	return m.ValidateToken(tokenString)
}
