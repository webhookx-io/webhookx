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

func TestMergeMap(t *testing.T) {
	dst := map[string]interface{}{
		"key": "v",
		"map": map[string]interface{}{
			"k1": "v1",
			"k2": "v2",
		},
	}
	src := map[string]interface{}{
		"key": "value",
		"map": map[string]interface{}{
			"k2": "vv2",
			"k3": "v3",
		},
	}
	MergeMap(dst, src)
	assert.EqualValues(t, map[string]interface{}{
		"key": "value",
		"map": map[string]interface{}{
			"k1": "v1",
			"k2": "vv2",
			"k3": "v3",
		},
	}, dst)
}
