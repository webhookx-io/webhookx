package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type Time struct {
	time.Time
}

func NewTime(t time.Time) Time {
	return Time{
		Time: t,
	}
}

func (t Time) Equal(other Time) bool {
	return t.Time.Equal(other.Time)
}

func (t *Time) UnmarshalJSON(b []byte) error {
	var timestamp int64
	err := json.Unmarshal(b, &timestamp)
	if err != nil {
		return err
	}
	if timestamp != 0 {
		t.Time = time.UnixMilli(timestamp)
	}
	return nil
}

func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("0"), nil
	}
	return []byte(fmt.Sprintf("%d", t.UnixMilli())), nil
}

func (t Time) MarshalYAML() (interface{}, error) {
	return t.UnixMilli(), nil
}

func (t *Time) Scan(src interface{}) error {
	if src == nil {
		t.Time = time.Unix(0, 0)
		return nil
	}

	if v, ok := src.(time.Time); ok {
		t.Time = v
		return nil
	} else {
		return fmt.Errorf("cannot scan %T", src)
	}
}

func (t Time) Value() (driver.Value, error) {
	return t.Time, nil
}
