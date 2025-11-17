package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUUID(t *testing.T) {
	uuid := UUID()
	assert.True(t, IsValidUUID(uuid))
}
