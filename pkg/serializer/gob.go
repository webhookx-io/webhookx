package serializer

import (
	"bytes"
	"encoding/gob"
)

var Gob GobSerializer

type GobSerializer struct{}

func (g GobSerializer) Serialize(val interface{}) ([]byte, error) {
	var buffer bytes.Buffer
	err := gob.NewEncoder(&buffer).Encode(val)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (g GobSerializer) Deserialize(b []byte, val interface{}) error {
	return gob.NewDecoder(bytes.NewReader(b)).Decode(val)
}
