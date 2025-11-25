package license

import (
	"crypto/ed25519"
	"encoding/hex"
)

type Verifier struct {
	PublicKey []byte
}

func NewVerifier(publicKey string) (*Verifier, error) {
	key, err := hex.DecodeString(publicKey)
	if err != nil {
		return nil, err
	}
	return &Verifier{
		PublicKey: key,
	}, nil
}

func (v *Verifier) Verify(message, signature []byte) bool {
	return ed25519.Verify(v.PublicKey, message, signature)
}
