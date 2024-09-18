package utils

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

type NestB struct {
	Timeout int `validate:"gt=0"`
}

type NestA struct {
	Gender string `validate:"oneof=male female"`
	NestB  NestB
}

type Struct struct {
	ID   string `json:"id"`
	Name string `validate:"required"`
	Nest NestA
	Age  int      `validate:"gte=0,lte=100"`
	Pets []string `validate:"min=1"`
}

func TestValidate(t *testing.T) {
	err := Validate(&Struct{
		Name: "",
		Nest: NestA{
			Gender: "x",
			NestB: NestB{
				Timeout: 0,
			},
		},
		Age:  -1,
		Pets: nil,
	})
	bytes, err := json.MarshalIndent(err, "", "   ")
	expected := `
{
   "message": "request validation",
   "fields": {
      "Age": "value must be >= 0",
      "Name": "required field missing",
      "Nest": {
         "Gender": "invalid value: x",
         "NestB": {
            "Timeout": "value must be > 0"
         }
      },
      "Pets": "length must be at least 1"
   }
}
`
	assert.JSONEq(t, expected, string(bytes))
}
