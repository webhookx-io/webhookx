package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPointer(t *testing.T) {
	s := "string"
	assert.Equal(t, &s, Pointer(s))

	b := true
	assert.Equal(t, &b, Pointer(b))

	f := 1.1
	assert.Equal(t, &f, Pointer(f))

	i := 1
	assert.Equal(t, &i, Pointer(i))
}

func TestPointerValue(t *testing.T) {
	s := "string"
	assert.Equal(t, s, PointerValue(Pointer(s)))

	b := true
	assert.Equal(t, b, PointerValue(Pointer(b)))

	f := 1.1
	assert.Equal(t, f, PointerValue(Pointer(f)))

	i := 1
	assert.Equal(t, i, PointerValue(Pointer(i)))
}

func TestPointerValueNil(t *testing.T) {
	var s *string
	assert.Equal(t, "", PointerValue(s))

	var b *bool
	assert.Equal(t, false, PointerValue(b))

	var f *float64
	assert.Equal(t, float64(0), PointerValue(f))

	var i *int
	assert.Equal(t, 0, PointerValue(i))
}
