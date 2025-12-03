package utils

import (
	"github.com/zeebo/xxh3"
)

func XXHash3(s string) uint64 {
	return xxh3.HashString(s)
}
