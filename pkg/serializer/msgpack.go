package serializer

import (
	"bytes"

	"github.com/vmihailenco/msgpack/v5"
)

var MsgPack MsgPackSerializer

type MsgPackSerializer struct{}

func (s MsgPackSerializer) Serialize(val interface{}) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := msgpack.NewEncoder(&buffer)
	encoder.SetCustomStructTag("json")
	err := encoder.Encode(val)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (s MsgPackSerializer) Deserialize(b []byte, val interface{}) error {
	decoder := msgpack.NewDecoder(bytes.NewReader(b))
	decoder.SetCustomStructTag("json")
	return decoder.Decode(val)
}
