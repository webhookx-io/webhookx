package utils

import (
	"strings"

	uuid "github.com/satori/go.uuid"
)

func UUID() string {
	return uuid.NewV4().String()
}

func UUIDShort() string {
	return strings.ReplaceAll(UUID(), "-", "")
}

func IsValidUUID(id string) bool {
	_, err := uuid.FromString(id)
	return err == nil
}
