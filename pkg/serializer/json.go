package serializer

import (
	"encoding/json"
)

var JSON JSONSerializer

type JSONSerializer struct{}

func (s JSONSerializer) Serialize(val interface{}) ([]byte, error) {
	return json.Marshal(val)
}

func (s JSONSerializer) Deserialize(b []byte, val interface{}) error {
	return json.Unmarshal(b, val)
}
