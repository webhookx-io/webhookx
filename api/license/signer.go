package license

import (
	"crypto/ed25519"
	"encoding/hex"
)

type Signer struct {
	PrivateKey []byte
}

func NewSigner(privateKey string) (*Signer, error) {
	key, err := hex.DecodeString(privateKey)
	if err != nil {
		return nil, err
	}
	return &Signer{
		PrivateKey: key,
	}, nil
}

func (s *Signer) Sign(message string) string {
	signature := ed25519.Sign(s.PrivateKey, []byte(message))
	return hex.EncodeToString(signature)
}
