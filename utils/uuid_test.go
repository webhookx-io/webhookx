package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUUID(t *testing.T) {
	uuid := UUID()
	assert.True(t, IsValidUUID(uuid))
}
