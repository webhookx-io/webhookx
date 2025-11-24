package reference

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	tests := []struct {
		scenario  string
		reference string
		expect    Reference
	}{
		{
			reference: "{secret://aws/path/to/value?k1=v1&k2=v2}",
			expect: Reference{
				Provider: "aws",
				Name:     "path/to/value",
				Properties: map[string]string{
					"k1": "v1",
					"k2": "v2",
				},
			},
		},
		{
			reference: "{secret://aws//mysecret}",
			expect: Reference{
				Provider:   "aws",
				Name:       "/mysecret",
				Properties: map[string]string{},
			},
		},
		{
			reference: "{secret://aws/json.}",
			expect: Reference{
				Provider:    "aws",
				Name:        "json",
				JsonPointer: "",
				Properties:  map[string]string{},
			},
		},
		{
			reference: "{secret://aws/path/to/json.credentials.0.password?k1=v1&k2=v2}",
			expect: Reference{
				Provider:    "aws",
				Name:        "path/to/json",
				JsonPointer: "credentials.0.password",
				Properties: map[string]string{
					"k1": "v1",
					"k2": "v2",
				},
			},
		},
	}
	for _, test := range tests {
		ref, err := Parse(test.reference)
		assert.NoError(t, err)
		test.expect.Reference = test.reference
		assert.EqualValues(t, test.expect, *ref)
	}
}

func TestParseError(t *testing.T) {
	tests := []struct {
		scenario  string
		reference string
		expect    Reference
		expectErr error
	}{
		{
			scenario:  "",
			reference: "{:}",
			expectErr: fmt.Errorf("invalid reference: parse \":\": missing protocol scheme"),
		},
		{
			scenario:  "",
			reference: "{}",
			expectErr: fmt.Errorf(`invalid reference: "invalid reference scheme"`),
		},
		{
			scenario:  "",
			reference: "{secret:///name}",
			expectErr: fmt.Errorf(`invalid reference: "invalid reference provider"`),
		},
		{
			scenario:  "",
			reference: "{secret://aws/}",
			expectErr: fmt.Errorf(`invalid reference: "invalid reference name"`),
		},
		{
			scenario:  "",
			reference: "{secret://aws/name?key;}",
			expectErr: fmt.Errorf(`invalid reference: "invalid reference properties"`),
		},
	}

	for _, test := range tests {
		_, err := Parse(test.reference)
		assert.EqualError(t, err, test.expectErr.Error())
	}

}
