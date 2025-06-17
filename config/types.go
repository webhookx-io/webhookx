package config

import "encoding/json"

type Map map[string]string

func (m *Map) Decode(value string) error {
	return json.Unmarshal([]byte(value), m)
}

type Password string

func (p Password) MarshalJSON() ([]byte, error) {
	return json.Marshal("******")
}
