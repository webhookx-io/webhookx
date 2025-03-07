package store

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test(t *testing.T) {
	v, ok := Get("key")
	assert.False(t, ok)
	assert.Nil(t, v)

	Set("key", "value")
	v, ok = Get("key")
	assert.True(t, ok)
	assert.Equal(t, "value", v)

	Remove("key")
	v, ok = Get("key")
	assert.False(t, ok)
	assert.Nil(t, v)
}
