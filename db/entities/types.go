package entities

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type Metadata map[string]string

func (m *Metadata) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), m)
}

func (m *Metadata) Value() (driver.Value, error) {
	return json.Marshal(m)
}

type UnixTime struct {
	time.Time
}

func NewUnixTime(t time.Time) UnixTime {
	return UnixTime{
		Time: time.Unix(t.Unix(), 0),
	}
}

func (t *UnixTime) UnmarshalJSON(b []byte) error {
	var timestamp int64
	err := json.Unmarshal(b, &timestamp)
	if err != nil {
		return err
	}
	t.Time = time.Unix(timestamp, 0)
	return nil
}

func (t UnixTime) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%d", t.Unix())), nil
}

func (t *UnixTime) Scan(src interface{}) error {
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

func (t UnixTime) Value() (driver.Value, error) {
	return t.Time, nil
}

type BaseModel struct {
	CreatedAt   UnixTime `db:"created_at" json:"created_at"`
	UpdatedAt   UnixTime `db:"updated_at" json:"updated_at"`
	WorkspaceId string   `db:"ws_id" json:"-"`
}
