package utils

import (
	"math/rand"
	"time"
)

const charset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandomString(n int) string {
	if n < 0 {
		panic("invalid args n")
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	result := make([]byte, n)
	for i := range result {
		result[i] = charset[r.Intn(len(charset))]
	}
	return string(result)
}
