package types

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestA(t *testing.T) {
	time, err := time.Parse(time.RFC3339Nano, "2006-01-02T15:04:05.999Z")
	assert.Nil(t, err)
	t1 := NewTime(time)
	s, err := json.Marshal(t1)
	assert.Nil(t, err)
	assert.EqualValues(t, "1136214245999", string(s))

	var t2 Time
	json.Unmarshal([]byte("1136214245999"), &t2)
	assert.True(t, t1.Equal(t2))
}
