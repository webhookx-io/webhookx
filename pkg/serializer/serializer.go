package serializer

type Serializer interface {
	Serialize(val interface{}) ([]byte, error)
	Deserialize(b []byte, val interface{}) error
}
