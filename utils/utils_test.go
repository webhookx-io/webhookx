package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDefaultIfZero(t *testing.T) {
	tests := []struct {
		Input    interface{}
		Default  interface{}
		Expected interface{}
	}{
		{
			Input:    "",
			Default:  "value",
			Expected: "value",
		},
		{
			Input:    false,
			Default:  true,
			Expected: true,
		},
		{
			Input:    0,
			Default:  1,
			Expected: 1,
		},
	}

	for _, test := range tests {
		v := DefaultIfZero(test.Input, test.Default)
		assert.Equal(t, test.Expected, v)
	}
}
