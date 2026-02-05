package serializer

import (
	"bytes"

	"github.com/vmihailenco/msgpack/v5"
)

var MsgPack MsgPackSerializer

type MsgPackSerializer struct{}

func (s MsgPackSerializer) Serialize(val interface{}) ([]byte, error) {
	var buf bytes.Buffer

	encoder := msgpack.GetEncoder()
	defer msgpack.PutEncoder(encoder)

	encoder.SetCustomStructTag("json")
	encoder.Reset(&buf)

	err := encoder.Encode(val)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s MsgPackSerializer) Deserialize(b []byte, val interface{}) error {
	decoder := msgpack.GetDecoder()
	defer msgpack.PutDecoder(decoder)

	decoder.SetCustomStructTag("json")
	decoder.Reset(bytes.NewReader(b))

	return decoder.Decode(val)
}
