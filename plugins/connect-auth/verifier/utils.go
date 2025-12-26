package verifier

import "crypto/subtle"

func timingSafeEqual(str1 string, str2 string) bool {
	return subtle.ConstantTimeCompare([]byte(str1), []byte(str2)) == 1
}
