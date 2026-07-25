package jwt

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

type VerifierKey struct {
	method jwt.SigningMethod
	key    any
}

func HS256VerifierKey(secret string) VerifierKey {
	return VerifierKey{method: jwt.SigningMethodHS256, key: []byte(secret)}
}

func ES256VerifierKey(publicKey *ecdsa.PublicKey) VerifierKey {
	return VerifierKey{method: jwt.SigningMethodES256, key: publicKey}
}

func (k VerifierKey) isZero() bool { return k.method == nil }

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

type MultiVerifier struct {
	byKid map[string]*JWTManager
	def   *JWTManager
}

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

func newVerifyOnlyManager(issuer string, vk VerifierKey) *JWTManager {
	return &JWTManager{
		issuer:       issuer,
		method:       vk.method,
		signKey:      nil,
		verifyKey:    vk.key,
		validMethods: []string{vk.method.Alg()},
	}
}

func (mv *MultiVerifier) ValidateToken(tokenString string) (*Claims, error) {
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
