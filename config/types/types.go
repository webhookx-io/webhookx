package types

import (
	"encoding/json"
	"time"

	"github.com/webhookx-io/webhookx/utils"
)

type Config interface {
	Validate() error
	PostProcess() error
}

type Map map[string]string

func (m *Map) Decode(value string) error {
	return json.Unmarshal([]byte(value), m)
}

type Password string

func (p Password) MarshalJSON() ([]byte, error) {
	return json.Marshal("******")
}

type Duration time.Duration

func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

func (d Duration) String() string {
	if d < 0 {
		return time.Duration(d).String()
	}
	return utils.FormatDuration(time.Duration(d))
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(value []byte) error {
	var text string
	if err := json.Unmarshal(value, &text); err != nil {
		return err
	}
	return d.Decode(text)
}

func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

func (d *Duration) UnmarshalText(value []byte) error {
	return d.Decode(string(value))
}

func (d *Duration) Decode(value string) error {
	parsed, err := utils.ParseDuration(value)
	if err != nil {
		return err
	}
	*d = Duration(parsed)
	return nil
}
