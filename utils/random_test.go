package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomString(t *testing.T) {
	assert.Equal(t, "", RandomString(0))
	assert.Equal(t, 1, len(RandomString(1)))
	assert.Equal(t, 32, len(RandomString(32)))
	assert.Panics(t, func() { RandomString(-1) }, "the code did not panic")
}
