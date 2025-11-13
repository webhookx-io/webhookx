package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

func Sha256(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
