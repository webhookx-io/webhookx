package config

import "encoding/json"

type Map map[string]string

func (m *Map) Decode(value string) error {
	return json.Unmarshal([]byte(value), m)
}
